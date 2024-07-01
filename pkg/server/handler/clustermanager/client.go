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
	"github.com/unikorn-cloud/core/pkg/constants"
	coreopenapi "github.com/unikorn-cloud/core/pkg/openapi"
	"github.com/unikorn-cloud/core/pkg/server/conversion"
	"github.com/unikorn-cloud/core/pkg/server/errors"
	"github.com/unikorn-cloud/core/pkg/util"
	unikornv1 "github.com/unikorn-cloud/kubernetes/pkg/apis/unikorn/v1alpha1"
	"github.com/unikorn-cloud/kubernetes/pkg/openapi"
	"github.com/unikorn-cloud/kubernetes/pkg/server/handler/applicationbundle"
	"github.com/unikorn-cloud/kubernetes/pkg/server/handler/common"
	"github.com/unikorn-cloud/kubernetes/pkg/server/handler/scoping"

	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"

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

// CreateImplicit is called when a cluster creation call is made and a control plane is not specified.
func (c *Client) CreateImplicit(ctx context.Context, organizationID, projectID string) (*unikornv1.ClusterManager, error) {
	log := log.FromContext(ctx)

	namespace, err := common.New(c.client).ProjectNamespace(ctx, organizationID, projectID)
	if err != nil {
		return nil, err
	}

	var existing unikornv1.ClusterManagerList

	options := &client.ListOptions{
		Namespace:     namespace.Name,
		LabelSelector: labels.SelectorFromSet(labels.Set{constants.NameLabel: "default"}),
	}

	if err := c.client.List(ctx, &existing, options); err != nil {
		return nil, err
	}

	if len(existing.Items) != 0 {
		return &existing.Items[0], nil
	}

	log.Info("creating implicit control plane")

	request := &openapi.ClusterManagerWrite{
		Metadata: coreopenapi.ResourceWriteMetadata{
			Name:        "default",
			Description: util.ToPointer("Implicitly provisioned cluster controller"),
		},
	}

	resource, err := c.Create(ctx, organizationID, projectID, request)
	if err != nil {
		return nil, err
	}

	return resource, nil
}

// convert converts from Kubernetes into OpenAPI types.
func (c *Client) convert(in *unikornv1.ClusterManager) *openapi.ClusterManagerRead {
	provisioningStatus := coreopenapi.ResourceProvisioningStatusUnknown

	if condition, err := in.StatusConditionRead(unikornv1core.ConditionAvailable); err == nil {
		provisioningStatus = conversion.ConvertStatusCondition(condition)
	}

	out := &openapi.ClusterManagerRead{
		Metadata: conversion.ProjectScopedResourceReadMetadata(in, provisioningStatus),
	}

	return out
}

// convertList converts from Kubernetes into OpenAPI types.
func (c *Client) convertList(in *unikornv1.ClusterManagerList) openapi.ClusterManagers {
	out := make(openapi.ClusterManagers, len(in.Items))

	for i := range in.Items {
		out[i] = *c.convert(&in.Items[i])
	}

	return out
}

// List returns all control planes.
func (c *Client) List(ctx context.Context, organizationID string) (openapi.ClusterManagers, error) {
	scoper := scoping.New(ctx, c.client, organizationID)

	selector, err := scoper.GetSelector(ctx)
	if err != nil {
		if goerrors.Is(err, scoping.ErrNoScope) {
			return openapi.ClusterManagers{}, nil
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

	return c.convertList(result), nil
}

// get returns the control plane.
func (c *Client) get(ctx context.Context, namespace, clusterManagerID string) (*unikornv1.ClusterManager, error) {
	result := &unikornv1.ClusterManager{}

	if err := c.client.Get(ctx, client.ObjectKey{Namespace: namespace, Name: clusterManagerID}, result); err != nil {
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
func (c *Client) generate(ctx context.Context, namespace *corev1.Namespace, organizationID, projectID string, request *openapi.ClusterManagerWrite) (*unikornv1.ClusterManager, error) {
	applicationBundle, err := c.defaultApplicationBundle(ctx)
	if err != nil {
		return nil, err
	}

	out := &unikornv1.ClusterManager{
		ObjectMeta: conversion.NewObjectMetadata(&request.Metadata, namespace.Name).WithOrganization(organizationID).WithProject(projectID).Get(ctx),
		Spec: unikornv1.ClusterManagerSpec{
			ApplicationBundle:            &applicationBundle.Name,
			ApplicationBundleAutoUpgrade: &unikornv1.ApplicationBundleAutoUpgradeSpec{},
		},
	}

	return out, nil
}

// Create creates a control plane.
func (c *Client) Create(ctx context.Context, organizationID, projectID string, request *openapi.ClusterManagerWrite) (*unikornv1.ClusterManager, error) {
	namespace, err := common.New(c.client).ProjectNamespace(ctx, organizationID, projectID)
	if err != nil {
		return nil, err
	}

	if namespace.DeletionTimestamp != nil {
		return nil, errors.OAuth2InvalidRequest("project is being deleted")
	}

	resource, err := c.generate(ctx, namespace, organizationID, projectID, request)
	if err != nil {
		return nil, err
	}

	if err := c.client.Create(ctx, resource); err != nil {
		// TODO: we can do a cached lookup to save the API traffic.
		if kerrors.IsAlreadyExists(err) {
			return nil, errors.HTTPConflict()
		}

		return nil, errors.OAuth2ServerError("failed to create control plane").WithError(err)
	}

	return resource, nil
}

// Delete deletes the control plane.
func (c *Client) Delete(ctx context.Context, organizationID, projectID, clusterManagerID string) error {
	namespace, err := common.New(c.client).ProjectNamespace(ctx, organizationID, projectID)
	if err != nil {
		return err
	}

	if namespace.DeletionTimestamp != nil {
		return errors.OAuth2InvalidRequest("project is being deleted")
	}

	controlPlane := &unikornv1.ClusterManager{
		ObjectMeta: metav1.ObjectMeta{
			Name:      clusterManagerID,
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
func (c *Client) Update(ctx context.Context, organizationID, projectID, clusterManagerID string, request *openapi.ClusterManagerWrite) error {
	namespace, err := common.New(c.client).ProjectNamespace(ctx, organizationID, projectID)
	if err != nil {
		return err
	}

	if namespace.DeletionTimestamp != nil {
		return errors.OAuth2InvalidRequest("project is being deleted")
	}

	current, err := c.get(ctx, namespace.Name, clusterManagerID)
	if err != nil {
		return err
	}

	required, err := c.generate(ctx, namespace, organizationID, projectID, request)
	if err != nil {
		return err
	}

	updated := current.DeepCopy()
	updated.Spec = required.Spec

	if err := c.client.Patch(ctx, updated, client.MergeFrom(current)); err != nil {
		return errors.OAuth2ServerError("failed to patch control plane").WithError(err)
	}

	return nil
}
