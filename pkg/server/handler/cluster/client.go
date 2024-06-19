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
	"encoding/base64"
	goerrors "errors"
	"net"
	"net/http"
	"slices"

	"github.com/spf13/pflag"

	coreclient "github.com/unikorn-cloud/core/pkg/client"
	"github.com/unikorn-cloud/core/pkg/constants"
	"github.com/unikorn-cloud/core/pkg/server/errors"
	"github.com/unikorn-cloud/core/pkg/util"
	unikornv1 "github.com/unikorn-cloud/kubernetes/pkg/apis/unikorn/v1alpha1"
	"github.com/unikorn-cloud/kubernetes/pkg/openapi"
	"github.com/unikorn-cloud/kubernetes/pkg/provisioners/helmapplications/clusteropenstack"
	"github.com/unikorn-cloud/kubernetes/pkg/provisioners/helmapplications/vcluster"
	"github.com/unikorn-cloud/kubernetes/pkg/server/handler/clustermanager"
	"github.com/unikorn-cloud/kubernetes/pkg/server/handler/common"
	"github.com/unikorn-cloud/kubernetes/pkg/server/handler/scoping"
	regionapi "github.com/unikorn-cloud/region/pkg/openapi"

	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

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

	// namespace the controller runs in.
	namespace string

	// options control various defaults and the like.
	options *Options

	// region is a client to access regions.
	region regionapi.ClientWithResponsesInterface
}

// NewClient returns a new client with required parameters.
func NewClient(client client.Client, namespace string, options *Options, region regionapi.ClientWithResponsesInterface) *Client {
	return &Client{
		client:    client,
		namespace: namespace,
		options:   options,
		region:    region,
	}
}

// List returns all clusters owned by the implicit control plane.
func (c *Client) List(ctx context.Context, organizationID string) (openapi.KubernetesClusters, error) {
	scoper := scoping.New(ctx, c.client, organizationID)

	selector, err := scoper.GetSelector(ctx)
	if err != nil {
		if goerrors.Is(err, scoping.ErrNoScope) {
			return openapi.KubernetesClusters{}, nil
		}

		return nil, errors.OAuth2ServerError("failed to apply scoping rules").WithError(err)
	}

	options := &client.ListOptions{
		LabelSelector: selector,
	}

	result := &unikornv1.KubernetesClusterList{}

	if err := c.client.List(ctx, result, options); err != nil {
		return nil, errors.OAuth2ServerError("failed to list control planes").WithError(err)
	}

	slices.SortStableFunc(result.Items, unikornv1.CompareKubernetesCluster)

	return c.convertList(result), nil
}

// get returns the cluster.
func (c *Client) get(ctx context.Context, namespace, clusterID string) (*unikornv1.KubernetesCluster, error) {
	result := &unikornv1.KubernetesCluster{}

	if err := c.client.Get(ctx, client.ObjectKey{Namespace: namespace, Name: clusterID}, result); err != nil {
		if kerrors.IsNotFound(err) {
			return nil, errors.HTTPNotFound().WithError(err)
		}

		return nil, errors.OAuth2ServerError("unable to get cluster").WithError(err)
	}

	return result, nil
}

// GetKubeconfig returns the kubernetes configuation associated with a cluster.
func (c *Client) GetKubeconfig(ctx context.Context, organizationID, projectID, clusterID string) ([]byte, error) {
	project, err := common.New(c.client).ProjectNamespace(ctx, organizationID, projectID)
	if err != nil {
		return nil, err
	}

	// TODO: propagate the client like we do in the controllers, then code sharing
	// becomes a lot easier!
	ctx = coreclient.NewContextWithDynamicClient(ctx, c.client)

	vc := vcluster.NewControllerRuntimeClient()

	vclusterConfig, err := vc.RESTConfig(ctx, project.Name, false)
	if err != nil {
		return nil, errors.OAuth2ServerError("failed to get control plane rest config").WithError(err)
	}

	vclusterClient, err := client.New(vclusterConfig, client.Options{})
	if err != nil {
		return nil, errors.OAuth2ServerError("failed to get control plane client").WithError(err)
	}

	clusterObjectKey := client.ObjectKey{
		Namespace: project.Name,
		Name:      clusterID,
	}

	cluster := &unikornv1.KubernetesCluster{}

	if err := c.client.Get(ctx, clusterObjectKey, cluster); err != nil {
		return nil, errors.HTTPNotFound().WithError(err)
	}

	objectKey := client.ObjectKey{
		Namespace: clusterID,
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

func (c *Client) createIdentity(ctx context.Context, organizationID, projectID, regionID, clusterID string) (*regionapi.IdentityRead, error) {
	request := regionapi.PostApiV1OrganizationsOrganizationIDProjectsProjectIDRegionsRegionIDIdentitiesJSONRequestBody{
		ClusterId: clusterID,
	}

	resp, err := c.region.PostApiV1OrganizationsOrganizationIDProjectsProjectIDRegionsRegionIDIdentitiesWithResponse(ctx, organizationID, projectID, regionID, request)
	if err != nil {
		return nil, errors.OAuth2ServerError("unable to create identity").WithError(err)
	}

	if resp.StatusCode() != http.StatusCreated {
		return nil, errors.OAuth2ServerError("unable to create identity")
	}

	return resp.JSON201, nil
}

func (c *Client) getExternalNetworks(ctx context.Context, organizationID, projectID, regionID string) (regionapi.ExternalNetworks, error) {
	resp, err := c.region.GetApiV1OrganizationsOrganizationIDProjectsProjectIDRegionsRegionIDExternalnetworksWithResponse(ctx, organizationID, projectID, regionID)
	if err != nil {
		return nil, errors.OAuth2ServerError("unable to get external networks").WithError(err)
	}

	if resp.StatusCode() != http.StatusOK {
		return nil, errors.OAuth2ServerError("unable to get external networks")
	}

	return *resp.JSON200, nil
}

func (c *Client) applyCloudSpecificConfiguration(ctx context.Context, organizationID, projectID, regionID string, identity *regionapi.IdentityRead, cluster *unikornv1.KubernetesCluster) error {
	// Save the identity ID for later cleanup.
	if cluster.Annotations == nil {
		cluster.Annotations = map[string]string{}
	}

	cluster.Annotations[constants.CloudIdentityAnnotation] = identity.Metadata.Id

	// Setup the provider specific stuff, this should be as minial as possible!
	switch identity.Spec.Type {
	case regionapi.Openstack:
		externalNetworks, err := c.getExternalNetworks(ctx, organizationID, projectID, regionID)
		if err != nil {
			return err
		}

		if len(externalNetworks) == 0 {
			return errors.OAuth2ServerError("no external networks present")
		}

		cloudConfig, err := base64.URLEncoding.DecodeString(identity.Spec.Openstack.CloudConfig)
		if err != nil {
			return errors.OAuth2ServerError("failed to decode cloud config").WithError(err)
		}

		cluster.Spec.Openstack = &unikornv1.KubernetesClusterOpenstackSpec{
			Cloud:             &identity.Spec.Openstack.Cloud,
			CloudConfig:       &cloudConfig,
			ExternalNetworkID: &externalNetworks[0].Id,
		}
	default:
		return errors.OAuth2ServerError("unhandled provider type")
	}

	return nil
}

// Create creates the implicit cluster indentified by the JTW claims.
func (c *Client) Create(ctx context.Context, organizationID, projectID string, request *openapi.KubernetesClusterWrite) error {
	namespace, err := common.New(c.client).ProjectNamespace(ctx, organizationID, projectID)
	if err != nil {
		return err
	}

	// Implicitly create the controller manager.
	if request.Spec.ClusterManagerId == nil {
		clusterManager, err := clustermanager.NewClient(c.client).CreateImplicit(ctx, organizationID, projectID)
		if err != nil {
			return err
		}

		request.Spec.ClusterManagerId = util.ToPointer(clusterManager.Name)
	}

	cluster, err := c.generate(ctx, namespace, organizationID, projectID, request)
	if err != nil {
		return err
	}

	identity, err := c.createIdentity(ctx, organizationID, projectID, request.Spec.RegionId, cluster.Name)
	if err != nil {
		return err
	}

	if err := c.applyCloudSpecificConfiguration(ctx, organizationID, projectID, request.Spec.RegionId, identity, cluster); err != nil {
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
func (c *Client) Delete(ctx context.Context, organizationID, projectID, clusterID string) error {
	namespace, err := common.New(c.client).ProjectNamespace(ctx, organizationID, projectID)
	if err != nil {
		return err
	}

	if namespace.DeletionTimestamp != nil {
		return errors.OAuth2InvalidRequest("control plane is being deleted")
	}

	cluster := &unikornv1.KubernetesCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      clusterID,
			Namespace: namespace.Name,
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
func (c *Client) Update(ctx context.Context, organizationID, projectID, clusterID string, request *openapi.KubernetesClusterWrite) error {
	namespace, err := common.New(c.client).ProjectNamespace(ctx, organizationID, projectID)
	if err != nil {
		return err
	}

	if namespace.DeletionTimestamp != nil {
		return errors.OAuth2InvalidRequest("control plane is being deleted")
	}

	current, err := c.get(ctx, namespace.Name, clusterID)
	if err != nil {
		return err
	}

	required, err := c.generate(ctx, namespace, organizationID, projectID, request)
	if err != nil {
		return err
	}

	// Experience has taught me that modifying caches by accident is a bad thing
	// so be extra safe and deep copy the existing resource.
	updated := current.DeepCopy()
	updated.Spec = required.Spec

	/*
		temp.Spec.ControlPlane.ServerGroupID = resource.Spec.ControlPlane.ServerGroupID
	*/

	if err := c.client.Patch(ctx, updated, client.MergeFrom(current)); err != nil {
		return errors.OAuth2ServerError("failed to patch cluster").WithError(err)
	}

	return nil
}
