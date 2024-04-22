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

package clustermanager

import (
	"context"
	goerrors "errors"
	"slices"

	unikornv1core "github.com/unikorn-cloud/core/pkg/apis/unikorn/v1alpha1"
	coreconstants "github.com/unikorn-cloud/core/pkg/constants"
	"github.com/unikorn-cloud/core/pkg/server/errors"
	"github.com/unikorn-cloud/core/pkg/util/retry"
	unikornv1 "github.com/unikorn-cloud/unikorn/pkg/apis/unikorn/v1alpha1"
	"github.com/unikorn-cloud/unikorn/pkg/constants"
	"github.com/unikorn-cloud/unikorn/pkg/server/generated"
	"github.com/unikorn-cloud/unikorn/pkg/server/handler/applicationbundle"
	"github.com/unikorn-cloud/unikorn/pkg/server/handler/common"
	"github.com/unikorn-cloud/unikorn/pkg/server/handler/scoping"

	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// Client wraps up control plane related management handling.
type Client struct {
	// client allows Kubernetes API access.
	client client.Client
}

// NewClient returns a new client with required parameters.
func NewClient(client client.Client) *Client {
	return &Client{
		client: client,
	}
}

var (
	// ErrResourceDeleting is raised when the resource is being deleted.
	ErrResourceDeleting = goerrors.New("resource is being deleted")

	// ErrNamespaceUnset is raised when the namespace hasn't been created
	// yet.
	ErrNamespaceUnset = goerrors.New("resource namespace is unset")

	// ErrApplicationBundle is raised when no suitable application
	// bundle is found.
	ErrApplicationBundle = goerrors.New("no application bundle found")
)

// provisionDefaultClusterManager is called when a cluster creation call is made and the
// control plane does not exist.
func (c *Client) provisionDefaultClusterManager(ctx context.Context, organizationName, projectName, name string) error {
	log := log.FromContext(ctx)

	log.Info("creating implicit control plane", "name", name)

	// GetMetadata should be called by descendents of the control
	// plane e.g. clusters. Rather than delegate creation to each
	// and every client implicitly create it.
	defaultClusterManager := &generated.ClusterManagerSpec{
		Name: name,
	}

	if err := c.Create(ctx, organizationName, projectName, defaultClusterManager); err != nil {
		return err
	}

	return nil
}

func (c *Client) CreateImplicit(ctx context.Context, organizationName, projectName, name string) error {
	namespace, err := common.New(c.client).ProjectNamespace(ctx, organizationName, projectName)
	if err != nil {
		return err
	}

	if _, err := c.get(ctx, namespace.Name, name); err != nil {
		if !errors.IsHTTPNotFound(err) {
			return err
		}

		if err := c.provisionDefaultClusterManager(ctx, organizationName, projectName, name); err != nil {
			return err
		}
	}

	waitCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Allow a grace period for the project to become active to avoid client
	// errors and retries.  The namespace creation should be ostensibly instant
	// and likewise show up due to non-blocking yields.
	callback := func() error {
		if _, err := c.get(waitCtx, namespace.Name, name); err != nil {
			// Short cut deleting errors.
			if goerrors.Is(err, ErrResourceDeleting) {
				cancel()

				return nil
			}

			return err
		}

		return nil
	}

	if err := retry.Forever().DoWithContext(waitCtx, callback); err != nil {
		return err
	}

	return nil
}

func convertMetadata(in *unikornv1.ClusterManager) (*generated.ClusterManagerMetadata, error) {
	labels, err := in.ResourceLabels()
	if err != nil {
		return nil, err
	}

	// Validated to exist by ResourceLabels()
	project := labels[coreconstants.ProjectLabel]

	out := &generated.ClusterManagerMetadata{
		Project:      project,
		CreationTime: in.CreationTimestamp.Time,
		Status:       "Unknown",
	}

	if in.DeletionTimestamp != nil {
		out.DeletionTime = &in.DeletionTimestamp.Time
	}

	if condition, err := in.StatusConditionRead(unikornv1core.ConditionAvailable); err == nil {
		out.Status = string(condition.Reason)
	}

	return out, nil
}

// convert converts from Kubernetes into OpenAPI types.
func (c *Client) convert(in *unikornv1.ClusterManager) (*generated.ClusterManager, error) {
	metadata, err := convertMetadata(in)
	if err != nil {
		return nil, err
	}

	out := &generated.ClusterManager{
		Metadata: *metadata,
		Spec: generated.ClusterManagerSpec{
			Name: in.Name,
		},
	}

	return out, nil
}

// convertList converts from Kubernetes into OpenAPI types.
func (c *Client) convertList(in *unikornv1.ClusterManagerList) (generated.ClusterManagers, error) {
	out := make(generated.ClusterManagers, len(in.Items))

	for i := range in.Items {
		item, err := c.convert(&in.Items[i])
		if err != nil {
			return nil, err
		}

		out[i] = *item
	}

	return out, nil
}

// List returns all control planes.
func (c *Client) List(ctx context.Context, organizationName string) (generated.ClusterManagers, error) {
	scoper := scoping.New(ctx, c.client, organizationName)

	selector, err := scoper.GetSelector(ctx)
	if err != nil {
		if goerrors.Is(err, scoping.ErrNoScope) {
			return generated.ClusterManagers{}, nil
		}

		return nil, errors.OAuth2ServerError("failed to apply scoping rules").WithError(err)
	}

	options := &client.ListOptions{
		LabelSelector: selector,
	}

	result := &unikornv1.ClusterManagerList{}

	if err := c.client.List(ctx, result, options); err != nil {
		return nil, errors.OAuth2ServerError("failed to list control planes").WithError(err)
	}

	slices.SortStableFunc(result.Items, unikornv1.CompareClusterManager)

	out, err := c.convertList(result)
	if err != nil {
		return nil, err
	}

	return out, nil
}

// get returns the control plane.
func (c *Client) get(ctx context.Context, namespace, name string) (*unikornv1.ClusterManager, error) {
	result := &unikornv1.ClusterManager{}

	if err := c.client.Get(ctx, client.ObjectKey{Namespace: namespace, Name: name}, result); err != nil {
		if kerrors.IsNotFound(err) {
			return nil, errors.HTTPNotFound().WithError(err)
		}

		return nil, errors.OAuth2ServerError("failed to get control plane").WithError(err)
	}

	return result, nil
}

// defaultApplicationBundle returns a default application bundle.
func (c *Client) defaultApplicationBundle(ctx context.Context) (*unikornv1.ClusterManagerApplicationBundle, error) {
	applicationBundles, err := applicationbundle.NewClient(c.client).ListClusterManager(ctx)
	if err != nil {
		return nil, err
	}

	applicationBundles.Items = slices.DeleteFunc(applicationBundles.Items, func(bundle unikornv1.ClusterManagerApplicationBundle) bool {
		if bundle.Spec.Preview != nil && *bundle.Spec.Preview {
			return true
		}

		if bundle.Spec.EndOfLife != nil {
			return true
		}

		return false
	})

	if len(applicationBundles.Items) == 0 {
		return nil, errors.OAuth2ServerError("unable to select an application bundle")
	}

	return &applicationBundles.Items[0], nil
}

// generate is a common function to create a Kubernetes type from an API one.
func (c *Client) generate(ctx context.Context, namespace *corev1.Namespace, organizationName, projectName string, parameters *generated.ClusterManagerSpec) (*unikornv1.ClusterManager, error) {
	applicationBundle, err := c.defaultApplicationBundle(ctx)
	if err != nil {
		return nil, err
	}

	controlPlane := &unikornv1.ClusterManager{
		ObjectMeta: metav1.ObjectMeta{
			Name:      parameters.Name,
			Namespace: namespace.Name,
			Labels: map[string]string{
				coreconstants.VersionLabel:      constants.Version,
				coreconstants.OrganizationLabel: organizationName,
				coreconstants.ProjectLabel:      projectName,
			},
		},
		Spec: unikornv1.ClusterManagerSpec{
			ApplicationBundle:            &applicationBundle.Name,
			ApplicationBundleAutoUpgrade: &unikornv1.ApplicationBundleAutoUpgradeSpec{},
		},
	}

	return controlPlane, nil
}

// Create creates a control plane.
func (c *Client) Create(ctx context.Context, organizationName, projectName string, request *generated.ClusterManagerSpec) error {
	namespace, err := common.New(c.client).ProjectNamespace(ctx, organizationName, projectName)
	if err != nil {
		return err
	}

	if namespace.DeletionTimestamp != nil {
		return errors.OAuth2InvalidRequest("project is being deleted")
	}

	resource, err := c.generate(ctx, namespace, organizationName, projectName, request)
	if err != nil {
		return err
	}

	if err := c.client.Create(ctx, resource); err != nil {
		// TODO: we can do a cached lookup to save the API traffic.
		if kerrors.IsAlreadyExists(err) {
			return errors.HTTPConflict()
		}

		return errors.OAuth2ServerError("failed to create control plane").WithError(err)
	}

	return nil
}

// Delete deletes the control plane.
func (c *Client) Delete(ctx context.Context, organizationName, projectName, name string) error {
	namespace, err := common.New(c.client).ProjectNamespace(ctx, organizationName, projectName)
	if err != nil {
		return err
	}

	if namespace.DeletionTimestamp != nil {
		return errors.OAuth2InvalidRequest("project is being deleted")
	}

	controlPlane := &unikornv1.ClusterManager{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace.Name,
		},
	}

	if err := c.client.Delete(ctx, controlPlane); err != nil {
		if kerrors.IsNotFound(err) {
			return errors.HTTPNotFound().WithError(err)
		}

		return errors.OAuth2ServerError("failed to delete control plane").WithError(err)
	}

	return nil
}

// Update implements read/modify/write for the control plane.
func (c *Client) Update(ctx context.Context, organizationName, projectName, name string, request *generated.ClusterManagerSpec) error {
	namespace, err := common.New(c.client).ProjectNamespace(ctx, organizationName, projectName)
	if err != nil {
		return err
	}

	if namespace.DeletionTimestamp != nil {
		return errors.OAuth2InvalidRequest("project is being deleted")
	}

	resource, err := c.get(ctx, namespace.Name, name)
	if err != nil {
		return err
	}

	required, err := c.generate(ctx, namespace, organizationName, projectName, request)
	if err != nil {
		return err
	}

	// Experience has taught me that modifying caches by accident is a bad thing
	// so be extra safe and deep copy the existing resource.
	temp := resource.DeepCopy()
	temp.Spec = required.Spec

	if err := c.client.Patch(ctx, temp, client.MergeFrom(resource)); err != nil {
		return errors.OAuth2ServerError("failed to patch control plane").WithError(err)
	}

	return nil
}
