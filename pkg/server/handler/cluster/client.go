/*
Copyright 2022-2024 EscherCloud.
Copyright 2024 the Unikorn Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package cluster

import (
	"context"
	"net"
	"slices"

	"github.com/spf13/pflag"

	coreclient "github.com/unikorn-cloud/core/pkg/client"
	"github.com/unikorn-cloud/core/pkg/constants"
	"github.com/unikorn-cloud/core/pkg/server/errors"
	unikornv1 "github.com/unikorn-cloud/unikorn/pkg/apis/unikorn/v1alpha1"
	"github.com/unikorn-cloud/unikorn/pkg/provisioners/helmapplications/clusteropenstack"
	"github.com/unikorn-cloud/unikorn/pkg/provisioners/helmapplications/vcluster"
	"github.com/unikorn-cloud/unikorn/pkg/server/generated"
	"github.com/unikorn-cloud/unikorn/pkg/server/handler/clustermanager"
	"github.com/unikorn-cloud/unikorn/pkg/server/handler/organization"
	"github.com/unikorn-cloud/unikorn/pkg/server/handler/project"
	"github.com/unikorn-cloud/unikorn/pkg/server/handler/region"

	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Options struct {
	NodeNetwork    net.IPNet
	ServiceNetwork net.IPNet
	PodNetwork     net.IPNet
	DNSNameservers []net.IP
}

func (o *Options) AddFlags(f *pflag.FlagSet) {
	_, nodeNetwork, _ := net.ParseCIDR("192.168.0.0/24")
	_, serviceNetwork, _ := net.ParseCIDR("172.16.0.0/12")
	_, podNetwork, _ := net.ParseCIDR("10.0.0.0/8")

	dnsNameservers := []net.IP{net.ParseIP("8.8.8.8")}

	f.IPNetVar(&o.NodeNetwork, "default-node-network", *nodeNetwork, "Default node network to use when creating a cluster")
	f.IPNetVar(&o.ServiceNetwork, "default-service-network", *serviceNetwork, "Default service network to use when creating a cluster")
	f.IPNetVar(&o.PodNetwork, "default-pod-network", *podNetwork, "Default pod network to use when creating a cluster")
	f.IPSliceVar(&o.DNSNameservers, "default-dns-nameservers", dnsNameservers, "Default DNS nameserver to use when creating a cluster")
}

// Client wraps up cluster related management handling.
type Client struct {
	// client allows Kubernetes API access.
	client client.Client

	// options control various defaults and the like.
	options *Options
}

// NewClient returns a new client with required parameters.
func NewClient(client client.Client, options *Options) *Client {
	return &Client{
		client:  client,
		options: options,
	}
}

// List returns all clusters owned by the implicit control plane.
func (c *Client) List(ctx context.Context, organizationName string) ([]*generated.KubernetesCluster, error) {
	selector := labels.NewSelector()

	// TODO: a super-admin isn't scoped to a single organization!
	// TODO: RBAC - filter projects based on user membership here.
	organization, err := organization.NewClient(c.client).GetMetadata(ctx, organizationName)
	if err != nil {
		return nil, err
	}

	organizationReq, err := labels.NewRequirement(constants.OrganizationLabel, selection.Equals, []string{organization.Name})
	if err != nil {
		return nil, err
	}

	selector = selector.Add(*organizationReq)

	options := &client.ListOptions{
		LabelSelector: selector,
	}

	result := &unikornv1.KubernetesClusterList{}

	if err := c.client.List(ctx, result, options); err != nil {
		return nil, errors.OAuth2ServerError("failed to list control planes").WithError(err)
	}

	slices.SortStableFunc(result.Items, unikornv1.CompareKubernetesCluster)

	out, err := c.convertList(result)
	if err != nil {
		return nil, err
	}

	return out, nil
}

// get returns the cluster.
func (c *Client) get(ctx context.Context, namespace, name string) (*unikornv1.KubernetesCluster, error) {
	result := &unikornv1.KubernetesCluster{}

	if err := c.client.Get(ctx, client.ObjectKey{Namespace: namespace, Name: name}, result); err != nil {
		if kerrors.IsNotFound(err) {
			return nil, errors.HTTPNotFound().WithError(err)
		}

		return nil, errors.OAuth2ServerError("unable to get cluster").WithError(err)
	}

	return result, nil
}

// GetKubeconfig returns the kubernetes configuation associated with a cluster.
func (c *Client) GetKubeconfig(ctx context.Context, organizationName, projectName, name string) ([]byte, error) {
	project, err := project.NewClient(c.client).GetMetadata(ctx, organizationName, projectName)
	if err != nil {
		return nil, err
	}

	// TODO: propagate the client like we do in the controllers, then code sharing
	// becomes a lot easier!
	ctx = coreclient.NewContextWithDynamicClient(ctx, c.client)

	vc := vcluster.NewControllerRuntimeClient()

	vclusterConfig, err := vc.RESTConfig(ctx, project.Namespace, false)
	if err != nil {
		return nil, errors.OAuth2ServerError("failed to get control plane rest config").WithError(err)
	}

	vclusterClient, err := client.New(vclusterConfig, client.Options{})
	if err != nil {
		return nil, errors.OAuth2ServerError("failed to get control plane client").WithError(err)
	}

	clusterObjectKey := client.ObjectKey{
		Namespace: project.Namespace,
		Name:      name,
	}

	cluster := &unikornv1.KubernetesCluster{}

	if err := c.client.Get(ctx, clusterObjectKey, cluster); err != nil {
		return nil, errors.HTTPNotFound().WithError(err)
	}

	objectKey := client.ObjectKey{
		Namespace: name,
		Name:      clusteropenstack.KubeconfigSecretName(cluster),
	}

	secret := &corev1.Secret{}

	if err := vclusterClient.Get(ctx, objectKey, secret); err != nil {
		if kerrors.IsNotFound(err) {
			return nil, errors.HTTPNotFound().WithError(err)
		}

		return nil, errors.OAuth2ServerError("unable to get cluster configuration").WithError(err)
	}

	return secret.Data["value"], nil
}

/*
// createServerGroup creates an OpenStack server group.
func (c *Client) createServerGroup(ctx context.Context, provider *openstack.Openstack, project *project.Meta, name, kind string) (string, error) {
	// Name is fully qualified to avoid namespace clashes with control planes sharing
	// the same project.
	serverGroupName := project.Name + "-" + name + "-" + kind

	// Reuse the server group if it exists, otherwise create a new one.
	sg, err := provider.GetServerGroup(ctx, serverGroupName)
	if err != nil {
		if !errors.IsHTTPNotFound(err) {
			return "", err
		}
	}

	if sg == nil {
		if sg, err = provider.CreateServerGroup(ctx, serverGroupName); err != nil {
			return "", err
		}
	}

	return sg.ID, nil
}
*/

// Create creates the implicit cluster indentified by the JTW claims.
func (c *Client) Create(ctx context.Context, organizationName, projectName string, options *generated.KubernetesCluster) error {
	project, err := project.NewClient(c.client).GetMetadata(ctx, organizationName, projectName)
	if err != nil {
		return err
	}

	// Implicitly create the controller manager.
	clusterManagerName := "default"

	if options.ClusterManager == nil {
		clusterManagerName = *options.ClusterManager
	}

	if _, err := clustermanager.NewClient(c.client).GetOrCreateMetadata(ctx, organizationName, project.Name, clusterManagerName); err != nil {
		return err
	}

	if project.Deleting {
		return errors.OAuth2InvalidRequest("control plane is being deleted")
	}

	provider, err := region.NewClient(c.client).Provider(ctx, options.Region)
	if err != nil {
		return err
	}

	cluster, err := c.generate(ctx, provider, project, options)
	if err != nil {
		return err
	}

	if err := provider.ConfigureCluster(ctx, cluster); err != nil {
		return err
	}

	if err := c.client.Create(ctx, cluster); err != nil {
		// TODO: we can do a cached lookup to save the API traffic.
		if kerrors.IsAlreadyExists(err) {
			return errors.HTTPConflict()
		}

		return errors.OAuth2ServerError("failed to create cluster").WithError(err)
	}

	return nil
}

// Delete deletes the implicit cluster indentified by the JTW claims.
func (c *Client) Delete(ctx context.Context, organizationName, projectName, name string) error {
	project, err := project.NewClient(c.client).GetMetadata(ctx, organizationName, projectName)
	if err != nil {
		return err
	}

	if project.Deleting {
		return errors.OAuth2InvalidRequest("control plane is being deleted")
	}

	cluster := &unikornv1.KubernetesCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: project.Namespace,
		},
	}

	if err := c.client.Delete(ctx, cluster); err != nil {
		if kerrors.IsNotFound(err) {
			return errors.HTTPNotFound().WithError(err)
		}

		return errors.OAuth2ServerError("failed to delete cluster").WithError(err)
	}

	return nil
}

// Update implements read/modify/write for the cluster.
func (c *Client) Update(ctx context.Context, organizationName, projectName, name string, request *generated.KubernetesCluster) error {
	project, err := project.NewClient(c.client).GetMetadata(ctx, organizationName, projectName)
	if err != nil {
		return err
	}

	if project.Deleting {
		return errors.OAuth2InvalidRequest("control plane is being deleted")
	}

	resource, err := c.get(ctx, project.Namespace, name)
	if err != nil {
		return err
	}

	provider, err := region.NewClient(c.client).Provider(ctx, request.Region)
	if err != nil {
		return err
	}

	required, err := c.generate(ctx, provider, project, request)
	if err != nil {
		return err
	}

	// Experience has taught me that modifying caches by accident is a bad thing
	// so be extra safe and deep copy the existing resource.
	temp := resource.DeepCopy()
	temp.Spec = required.Spec

	/*
		temp.Spec.ControlPlane.ServerGroupID = resource.Spec.ControlPlane.ServerGroupID
	*/

	if err := c.client.Patch(ctx, temp, client.MergeFrom(resource)); err != nil {
		return errors.OAuth2ServerError("failed to patch cluster").WithError(err)
	}

	return nil
}
