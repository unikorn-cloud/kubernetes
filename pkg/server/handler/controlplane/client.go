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

package controlplane

import (
	"context"
	goerrors "errors"
	"slices"

	unikornv1core "github.com/unikorn-cloud/core/pkg/apis/unikorn/v1alpha1"
	"github.com/unikorn-cloud/core/pkg/constants"
	"github.com/unikorn-cloud/core/pkg/util/retry"
	unikornv1 "github.com/unikorn-cloud/unikorn/pkg/apis/unikorn/v1alpha1"
	"github.com/unikorn-cloud/unikorn/pkg/server/errors"
	"github.com/unikorn-cloud/unikorn/pkg/server/generated"
	"github.com/unikorn-cloud/unikorn/pkg/server/handler/applicationbundle"
	"github.com/unikorn-cloud/unikorn/pkg/server/handler/common"
	"github.com/unikorn-cloud/unikorn/pkg/server/handler/organization"
	"github.com/unikorn-cloud/unikorn/pkg/server/handler/project"

	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"

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

// Meta describes the control plane.
type Meta struct {
	// Project is the owning project's metadata.
	Project *project.Meta

	// Name is the project's Kubernetes name, so a higher level resource
	// can reference it.
	Name string

	// Namespace is the namespace that is provisioned by the control plane.
	// Should be usable and set when the project is active.
	Namespace string

	// Deleting tells us if we should allow new child objects to be created
	// in this resource's namespace.
	Deleting bool
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

// active returns true if the project is usable.
func active(c *unikornv1.ControlPlane) error {
	// No namespace created yet, you cannot provision any child resources.
	if c.Status.Namespace == "" {
		return ErrNamespaceUnset
	}

	return nil
}

// provisionDefaultControlPlane is called when a cluster creation call is made and the
// control plane does not exist.
func (c *Client) provisionDefaultControlPlane(ctx context.Context, projectName, name string) error {
	log := log.FromContext(ctx)

	log.Info("creating implicit control plane", "name", name)

	applicationBundles, err := applicationbundle.NewClient(c.client).ListControlPlane(ctx)
	if err != nil {
		return err
	}

	var applicationBundle *generated.ApplicationBundle

	for _, bundle := range applicationBundles {
		if bundle.Preview != nil && *bundle.Preview {
			continue
		}

		if bundle.EndOfLife != nil {
			continue
		}

		applicationBundle = bundle

		break
	}

	if applicationBundle == nil {
		return ErrApplicationBundle
	}

	// GetMetadata should be called by descendents of the control
	// plane e.g. clusters. Rather than delegate creation to each
	// and every client implicitly create it.
	defaultControlPlane := &generated.ControlPlane{
		Name:                         name,
		ApplicationBundle:            *applicationBundle,
		ApplicationBundleAutoUpgrade: &generated.ApplicationBundleAutoUpgrade{},
	}

	if err := c.Create(ctx, projectName, defaultControlPlane); err != nil {
		return err
	}

	return nil
}

// GetMetadata retrieves the control plane metadata.
func (c *Client) GetMetadata(ctx context.Context, projectName, name string) (*Meta, error) {
	project, err := project.NewClient(c.client).GetMetadata(ctx, projectName)
	if err != nil {
		return nil, err
	}

	result, err := c.get(ctx, project.Namespace, name)
	if err != nil {
		return nil, err
	}

	metadata := &Meta{
		Project:   project,
		Name:      name,
		Namespace: result.Status.Namespace,
		Deleting:  result.DeletionTimestamp != nil,
	}

	return metadata, nil
}

func (c *Client) GetOrCreateMetadata(ctx context.Context, projectName, name string) (*Meta, error) {
	project, err := project.NewClient(c.client).GetMetadata(ctx, projectName)
	if err != nil {
		return nil, err
	}

	result, err := c.get(ctx, project.Namespace, name)
	if err != nil {
		if !errors.IsHTTPNotFound(err) {
			return nil, err
		}

		if err := c.provisionDefaultControlPlane(ctx, projectName, name); err != nil {
			return nil, err
		}
	}

	waitCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Allow a grace period for the project to become active to avoid client
	// errors and retries.  The namespace creation should be ostensibly instant
	// and likewise show up due to non-blocking yields.
	callback := func() error {
		result, err = c.get(waitCtx, project.Namespace, name)
		if err != nil {
			// Short cut deleting errors.
			if goerrors.Is(err, ErrResourceDeleting) {
				cancel()

				return nil
			}

			return err
		}

		if err := active(result); err != nil {
			return err
		}

		return nil
	}

	if err := retry.Forever().DoWithContext(waitCtx, callback); err != nil {
		return nil, err
	}

	metadata := &Meta{
		Project:   project,
		Name:      name,
		Namespace: result.Status.Namespace,
		Deleting:  result.DeletionTimestamp != nil,
	}

	return metadata, nil
}

func convertMetadata(in *unikornv1.ControlPlane) (*generated.ResourceMetadata, error) {
	labels, err := in.ResourceLabels()
	if err != nil {
		return nil, err
	}

	// Validated to exist by ResourceLabels()
	project := labels[constants.ProjectLabel]

	out := &generated.ResourceMetadata{
		Project:      &project,
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
func (c *Client) convert(ctx context.Context, in *unikornv1.ControlPlane) (*generated.ControlPlane, error) {
	metadata, err := convertMetadata(in)
	if err != nil {
		return nil, err
	}

	bundle, err := applicationbundle.NewClient(c.client).GetControlPlane(ctx, *in.Spec.ApplicationBundle)
	if err != nil {
		return nil, err
	}

	out := &generated.ControlPlane{
		Metadata:                     metadata,
		Name:                         in.Name,
		ApplicationBundle:            *bundle,
		ApplicationBundleAutoUpgrade: common.ConvertApplicationBundleAutoUpgrade(in.Spec.ApplicationBundleAutoUpgrade),
	}

	return out, nil
}

// convertList converts from Kubernetes into OpenAPI types.
func (c *Client) convertList(ctx context.Context, in *unikornv1.ControlPlaneList) ([]*generated.ControlPlane, error) {
	out := make([]*generated.ControlPlane, len(in.Items))

	for i := range in.Items {
		item, err := c.convert(ctx, &in.Items[i])
		if err != nil {
			return nil, err
		}

		out[i] = item
	}

	return out, nil
}

// List returns all control planes.
func (c *Client) List(ctx context.Context) ([]*generated.ControlPlane, error) {
	selector := labels.NewSelector()

	// TODO: a super-admin isn't scoped to a single organization!
	// TODO: RBAC - filter projects based on user membership here.
	organization, err := organization.NewClient(c.client).GetMetadata(ctx)
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

	result := &unikornv1.ControlPlaneList{}

	if err := c.client.List(ctx, result, options); err != nil {
		return nil, errors.OAuth2ServerError("failed to list control planes").WithError(err)
	}

	slices.SortStableFunc(result.Items, unikornv1.CompareControlPlane)

	out, err := c.convertList(ctx, result)
	if err != nil {
		return nil, err
	}

	return out, nil
}

// get returns the control plane.
func (c *Client) get(ctx context.Context, namespace, name string) (*unikornv1.ControlPlane, error) {
	result := &unikornv1.ControlPlane{}

	if err := c.client.Get(ctx, client.ObjectKey{Namespace: namespace, Name: name}, result); err != nil {
		if kerrors.IsNotFound(err) {
			return nil, errors.HTTPNotFound().WithError(err)
		}

		return nil, errors.OAuth2ServerError("failed to get control plane").WithError(err)
	}

	return result, nil
}

// generate is a common function to create a Kubernetes type from an API one.
func generate(project *project.Meta, request *generated.ControlPlane) *unikornv1.ControlPlane {
	// TODO: common with CLI tools.
	controlPlane := &unikornv1.ControlPlane{
		ObjectMeta: metav1.ObjectMeta{
			Name:      request.Name,
			Namespace: project.Namespace,
			Labels: map[string]string{
				constants.VersionLabel:      constants.Version,
				constants.OrganizationLabel: project.Organization.Name,
				constants.ProjectLabel:      project.Name,
			},
		},
		Spec: unikornv1.ControlPlaneSpec{
			ApplicationBundle:            &request.ApplicationBundle.Name,
			ApplicationBundleAutoUpgrade: common.CreateApplicationBundleAutoUpgrade(request.ApplicationBundleAutoUpgrade),
		},
	}

	return controlPlane
}

// Create creates a control plane.
func (c *Client) Create(ctx context.Context, projectName generated.ProjectNameParameter, request *generated.ControlPlane) error {
	project, err := project.NewClient(c.client).GetMetadata(ctx, projectName)
	if err != nil {
		return err
	}

	if project.Deleting {
		return errors.OAuth2InvalidRequest("project is being deleted")
	}

	resource := generate(project, request)

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
func (c *Client) Delete(ctx context.Context, projectName generated.ProjectNameParameter, name generated.ControlPlaneNameParameter) error {
	project, err := project.NewClient(c.client).GetMetadata(ctx, projectName)
	if err != nil {
		return err
	}

	if project.Deleting {
		return errors.OAuth2InvalidRequest("project is being deleted")
	}

	controlPlane := &unikornv1.ControlPlane{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: project.Namespace,
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
func (c *Client) Update(ctx context.Context, projectName generated.ProjectNameParameter, name generated.ControlPlaneNameParameter, request *generated.ControlPlane) error {
	project, err := project.NewClient(c.client).GetMetadata(ctx, projectName)
	if err != nil {
		return err
	}

	if project.Deleting {
		return errors.OAuth2InvalidRequest("project is being deleted")
	}

	resource, err := c.get(ctx, project.Namespace, name)
	if err != nil {
		return err
	}

	required := generate(project, request)

	// Experience has taught me that modifying caches by accident is a bad thing
	// so be extra safe and deep copy the existing resource.
	temp := resource.DeepCopy()
	temp.Spec = required.Spec

	if err := c.client.Patch(ctx, temp, client.MergeFrom(resource)); err != nil {
		return errors.OAuth2ServerError("failed to patch control plane").WithError(err)
	}

	return nil
}
