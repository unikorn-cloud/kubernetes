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

package project

import (
	"context"
	goerrors "errors"
	"slices"

	unikornv1core "github.com/unikorn-cloud/core/pkg/apis/unikorn/v1alpha1"
	"github.com/unikorn-cloud/core/pkg/constants"
	"github.com/unikorn-cloud/core/pkg/server/errors"
	unikornv1 "github.com/unikorn-cloud/unikorn/pkg/apis/unikorn/v1alpha1"
	"github.com/unikorn-cloud/unikorn/pkg/server/generated"
	"github.com/unikorn-cloud/unikorn/pkg/server/handler/organization"
	"github.com/unikorn-cloud/unikorn/pkg/server/handler/scoping"

	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Client wraps up project related management handling.
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

// Meta describes the project.
type Meta struct {
	// Organization is the owning organization;s metadata.
	Organization *organization.Meta

	// Name is the project's Kubernetes name, so a higher level resource
	// can reference it.
	Name string

	// Namespace is the namespace that is provisioned by the project.
	// Should be usable set when the project is active.
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
)

// GetMetadata retrieves the project metadata.
// Clients should consult at least the Active status before doing anything
// with the project.
func (c *Client) GetMetadata(ctx context.Context, organizationName, name string) (*Meta, error) {
	organization, err := organization.NewClient(c.client).GetMetadata(ctx, organizationName)
	if err != nil {
		return nil, err
	}

	result, err := c.get(ctx, organization.Namespace, name)
	if err != nil {
		return nil, err
	}

	metadata := &Meta{
		Organization: organization,
		Name:         name,
		Namespace:    result.Status.Namespace,
		Deleting:     result.DeletionTimestamp != nil,
	}

	return metadata, nil
}

func convertMetadata(in *unikornv1.Project) generated.ProjectMetadata {
	out := generated.ProjectMetadata{
		CreationTime: in.CreationTimestamp.Time,
		Status:       "Unknown",
	}

	if in.DeletionTimestamp != nil {
		out.DeletionTime = &in.DeletionTimestamp.Time
	}

	if condition, err := in.StatusConditionRead(unikornv1core.ConditionAvailable); err == nil {
		out.Status = string(condition.Reason)
	}

	return out
}

func convert(in *unikornv1.Project) *generated.Project {
	out := &generated.Project{
		Metadata: convertMetadata(in),
		Spec: generated.ProjectSpec{
			Name: in.Name,
		},
	}

	return out
}

func convertList(in *unikornv1.ProjectList) generated.Projects {
	out := make(generated.Projects, len(in.Items))

	for i := range in.Items {
		out[i] = *convert(&in.Items[i])
	}

	return out
}

func (c *Client) List(ctx context.Context, organizationName string) (generated.Projects, error) {
	scoper := scoping.New(ctx, c.client, organizationName)

	result, err := scoper.ListProjects(ctx)
	if err != nil {
		return nil, errors.OAuth2ServerError("failed to list projects").WithError(err)
	}

	slices.SortStableFunc(result.Items, unikornv1.CompareProject)

	return convertList(result), nil
}

// get returns the implicit project identified by the JWT claims.
func (c *Client) get(ctx context.Context, namespace, name string) (*unikornv1.Project, error) {
	result := &unikornv1.Project{}

	if err := c.client.Get(ctx, client.ObjectKey{Namespace: namespace, Name: name}, result); err != nil {
		if kerrors.IsNotFound(err) {
			return nil, errors.HTTPNotFound().WithError(err)
		}

		return nil, errors.OAuth2ServerError("failed to get project").WithError(err)
	}

	return result, nil
}

func generate(organization *organization.Meta, request *generated.ProjectSpec) *unikornv1.Project {
	resource := &unikornv1.Project{
		ObjectMeta: metav1.ObjectMeta{
			Name:      request.Name,
			Namespace: organization.Namespace,
			Labels: map[string]string{
				constants.VersionLabel:      constants.Version,
				constants.OrganizationLabel: organization.Name,
			},
		},
	}

	return resource
}

// Create creates the implicit project indentified by the JTW claims.
func (c *Client) Create(ctx context.Context, organizationName string, request *generated.ProjectSpec) error {
	organization, err := organization.NewClient(c.client).GetOrCreateMetadata(ctx, organizationName)
	if err != nil {
		return err
	}

	if organization.Deleting {
		return errors.OAuth2InvalidRequest("organization is being deleted")
	}

	resource := generate(organization, request)

	if err := c.client.Create(ctx, resource); err != nil {
		// TODO: we can do a cached lookup to save the API traffic.
		if kerrors.IsAlreadyExists(err) {
			return errors.HTTPConflict()
		}

		return errors.OAuth2ServerError("failed to create project").WithError(err)
	}

	return nil
}

// Delete deletes the implicit project indentified by the JTW claims.
func (c *Client) Delete(ctx context.Context, organizationName, name string) error {
	organization, err := organization.NewClient(c.client).GetMetadata(ctx, organizationName)
	if err != nil {
		return err
	}

	if organization.Deleting {
		return errors.OAuth2InvalidRequest("organization is being deleted")
	}

	project := &unikornv1.Project{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: organization.Namespace,
		},
	}

	if err := c.client.Delete(ctx, project); err != nil {
		if kerrors.IsNotFound(err) {
			return errors.HTTPNotFound().WithError(err)
		}

		return errors.OAuth2ServerError("failed to delete project").WithError(err)
	}

	return nil
}
