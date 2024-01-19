/*
Copyright 2022-2024 EscherCloud.

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
	"net/http"
	"slices"

	"github.com/gophercloud/utils/openstack/clientconfig"

	unikornv1 "github.com/spjmurray/unikorn/pkg/apis/unikorn/v1alpha1"
	"github.com/spjmurray/unikorn/pkg/provisioners/helmapplications/clusteropenstack"
	"github.com/spjmurray/unikorn/pkg/provisioners/helmapplications/vcluster"
	"github.com/spjmurray/unikorn/pkg/server/authorization"
	"github.com/spjmurray/unikorn/pkg/server/errors"
	"github.com/spjmurray/unikorn/pkg/server/generated"
	"github.com/spjmurray/unikorn/pkg/server/handler/controlplane"
	"github.com/spjmurray/unikorn/pkg/server/handler/providers/openstack"

	coreclient "github.com/spjmurray/unikorn-core/pkg/client"

	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
)

// Client wraps up cluster related management handling.
type Client struct {
	// client allows Kubernetes API access.
	client client.Client

	// request is the http request that invoked this client.
	request *http.Request

	// authenticator provides access to authentication services.
	authenticator *authorization.Authenticator

	openstack *openstack.Openstack
}

// NewClient returns a new client with required parameters.
func NewClient(client client.Client, request *http.Request, authenticator *authorization.Authenticator, openstack *openstack.Openstack) *Client {
	return &Client{
		client:        client,
		request:       request,
		authenticator: authenticator,
		openstack:     openstack,
	}
}

// List returns all clusters owned by the implicit control plane.
func (c *Client) List(ctx context.Context, controlPlaneName generated.ControlPlaneNameParameter) ([]*generated.KubernetesCluster, error) {
	controlPlane, err := controlplane.NewClient(c.client).GetMetadata(ctx, controlPlaneName)
	if err != nil {
		return nil, err
	}

	result := &unikornv1.KubernetesClusterList{}

	if err := c.client.List(ctx, result, &client.ListOptions{Namespace: controlPlane.Namespace}); err != nil {
		return nil, errors.OAuth2ServerError("failed to list control planes").WithError(err)
	}

	slices.SortStableFunc(result.Items, unikornv1.CompareKubernetesCluster)

	out, err := c.convertList(ctx, result)
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

// Get returns the cluster.
func (c *Client) Get(ctx context.Context, controlPlaneName generated.ControlPlaneNameParameter, name generated.ClusterNameParameter) (*generated.KubernetesCluster, error) {
	controlPlane, err := controlplane.NewClient(c.client).GetMetadata(ctx, controlPlaneName)
	if err != nil {
		return nil, err
	}

	result, err := c.get(ctx, controlPlane.Namespace, name)
	if err != nil {
		return nil, err
	}

	out, err := c.convert(ctx, result)
	if err != nil {
		return nil, err
	}

	return out, nil
}

// GetKubeconfig returns the kubernetes configuation associated with a cluster.
func (c *Client) GetKubeconfig(ctx context.Context, controlPlaneName generated.ControlPlaneNameParameter, name generated.ClusterNameParameter) ([]byte, error) {
	controlPlane, err := controlplane.NewClient(c.client).GetMetadata(ctx, controlPlaneName)
	if err != nil {
		return nil, err
	}

	// TODO: propagate the client like we do in the controllers, then code sharing
	// becomes a lot easier!
	ctx = coreclient.NewContextWithDynamicClient(ctx, c.client)

	vc := vcluster.NewControllerRuntimeClient()

	vclusterConfig, err := vc.RESTConfig(ctx, controlPlane.Namespace, false)
	if err != nil {
		return nil, errors.OAuth2ServerError("failed to get control plane rest config").WithError(err)
	}

	vclusterClient, err := client.New(vclusterConfig, client.Options{})
	if err != nil {
		return nil, errors.OAuth2ServerError("failed to get control plane client").WithError(err)
	}

	clusterObjectKey := client.ObjectKey{
		Namespace: controlPlane.Namespace,
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

// createClientConfig creates an Openstack client configuration from the API.
func (c *Client) createClientConfig(controlPlane *controlplane.Meta, name string) ([]byte, string, error) {
	// Name is fully qualified to avoid namespace clashes with control planes sharing
	// the same project.
	applicationCredentialName := controlPlane.Name + "-" + name

	// Find and delete and existing credential.
	if _, err := c.openstack.GetApplicationCredential(c.request, applicationCredentialName); err != nil {
		if !errors.IsHTTPNotFound(err) {
			return nil, "", err
		}
	} else {
		if err := c.openstack.DeleteApplicationCredential(c.request, applicationCredentialName); err != nil {
			return nil, "", err
		}
	}

	ac, err := c.openstack.CreateApplicationCredential(c.request, applicationCredentialName, c.openstack.ApplicationCredentialRoles())
	if err != nil {
		return nil, "", err
	}

	cloud := "cloud"

	clientConfig := &clientconfig.Clouds{
		Clouds: map[string]clientconfig.Cloud{
			cloud: {
				AuthType: clientconfig.AuthV3ApplicationCredential,
				AuthInfo: &clientconfig.AuthInfo{
					AuthURL:                     c.authenticator.Keystone.Endpoint(),
					ApplicationCredentialID:     ac.ID,
					ApplicationCredentialSecret: ac.Secret,
				},
			},
		},
	}

	clientConfigYAML, err := yaml.Marshal(clientConfig)
	if err != nil {
		return nil, "", errors.OAuth2ServerError("unable to create cloud config").WithError(err)
	}

	return clientConfigYAML, cloud, nil
}

// createServerGroup creates an OpenStack server group.
func (c *Client) createServerGroup(controlPlane *controlplane.Meta, name, kind string) (string, error) {
	// Name is fully qualified to avoid namespace clashes with control planes sharing
	// the same project.
	serverGroupName := controlPlane.Name + "-" + name + "-" + kind

	// Reuse the server group if it exists, otherwise create a new one.
	sg, err := c.openstack.GetServerGroup(c.request, serverGroupName)
	if err != nil {
		if !errors.IsHTTPNotFound(err) {
			return "", err
		}
	}

	if sg == nil {
		if sg, err = c.openstack.CreateServerGroup(c.request, serverGroupName); err != nil {
			return "", err
		}
	}

	return sg.ID, nil
}

// Create creates the implicit cluster indentified by the JTW claims.
func (c *Client) Create(ctx context.Context, controlPlaneName generated.ControlPlaneNameParameter, options *generated.KubernetesCluster) error {
	controlPlane, err := controlplane.NewClient(c.client).GetOrCreateMetadata(ctx, controlPlaneName)
	if err != nil {
		return err
	}

	if controlPlane.Deleting {
		return errors.OAuth2InvalidRequest("control plane is being deleted")
	}

	cluster, err := c.createCluster(controlPlane, options)
	if err != nil {
		return err
	}

	clientConfig, cloud, err := c.createClientConfig(controlPlane, options.Name)
	if err != nil {
		return err
	}

	serverGroupID, err := c.createServerGroup(controlPlane, options.Name, "control-plane")
	if err != nil {
		return err
	}

	// TODO: should allow a private/self-signed CA via the API, or perhaps provide a
	// default.
	cluster.Spec.Openstack.Cloud = &cloud
	cluster.Spec.Openstack.CloudConfig = &clientConfig

	cluster.Spec.ControlPlane.ServerGroupID = &serverGroupID

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
func (c *Client) Delete(ctx context.Context, controlPlaneName generated.ControlPlaneNameParameter, name generated.ClusterNameParameter) error {
	controlPlane, err := controlplane.NewClient(c.client).GetMetadata(ctx, controlPlaneName)
	if err != nil {
		return err
	}

	if controlPlane.Deleting {
		return errors.OAuth2InvalidRequest("control plane is being deleted")
	}

	cluster := &unikornv1.KubernetesCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: controlPlane.Namespace,
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
func (c *Client) Update(ctx context.Context, controlPlaneName generated.ControlPlaneNameParameter, name generated.ClusterNameParameter, request *generated.KubernetesCluster) error {
	controlPlane, err := controlplane.NewClient(c.client).GetMetadata(ctx, controlPlaneName)
	if err != nil {
		return err
	}

	if controlPlane.Deleting {
		return errors.OAuth2InvalidRequest("control plane is being deleted")
	}

	resource, err := c.get(ctx, controlPlane.Namespace, name)
	if err != nil {
		return err
	}

	required, err := c.createCluster(controlPlane, request)
	if err != nil {
		return err
	}

	// Experience has taught me that modifying caches by accident is a bad thing
	// so be extra safe and deep copy the existing resource.
	temp := resource.DeepCopy()
	temp.Spec = required.Spec

	temp.Spec.Openstack.CACert = resource.Spec.Openstack.CACert
	temp.Spec.Openstack.Cloud = resource.Spec.Openstack.Cloud
	temp.Spec.Openstack.CloudConfig = resource.Spec.Openstack.CloudConfig

	temp.Spec.ControlPlane.ServerGroupID = resource.Spec.ControlPlane.ServerGroupID

	if err := c.client.Patch(ctx, temp, client.MergeFrom(resource)); err != nil {
		return errors.OAuth2ServerError("failed to patch cluster").WithError(err)
	}

	return nil
}
