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

	"github.com/unikorn-cloud/core/pkg/constants"
	"github.com/unikorn-cloud/core/pkg/util/retry"
	unikornv1 "github.com/unikorn-cloud/unikorn/pkg/apis/unikorn/v1alpha1"
	"github.com/unikorn-cloud/unikorn/pkg/server/errors"

	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
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

// NameFromContext translates an Openstack project ID to one we an use.
func NameFromContext(ctx context.Context) (string, error) {
	/*
		claims, err := oauth2.ClaimsFromContext(ctx)
		if err != nil {
			return "", err
		}

		return fmt.Sprintf("unikorn-server-%s", claims.UnikornClaims.Project), nil
	*/
	return "", nil
}

// Meta describes the project.
type Meta struct {
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

// active returns true if the project is usable.
func active(p *unikornv1.Project) error {
	// No namespace created yet, you cannot provision any child resources.
	if p.Status.Namespace == "" {
		return ErrNamespaceUnset
	}

	return nil
}

// GetMetadata retrieves the project metadata.
// Clients should consult at least the Active status before doing anything
// with the project.
func (c *Client) GetMetadata(ctx context.Context) (*Meta, error) {
	name, err := NameFromContext(ctx)
	if err != nil {
		return nil, err
	}

	result, err := c.get(ctx, name)
	if err != nil {
		return nil, err
	}

	metadata := &Meta{
		Name:      name,
		Namespace: result.Status.Namespace,
		Deleting:  result.DeletionTimestamp != nil,
	}

	return metadata, nil
}

func (c *Client) GetOrCreateMetadata(ctx context.Context) (*Meta, error) {
	name, err := NameFromContext(ctx)
	if err != nil {
		return nil, err
	}

	result, err := c.get(ctx, name)
	if err != nil {
		if !errors.IsHTTPNotFound(err) {
			return nil, err
		}

		log := log.FromContext(ctx)

		log.Info("creating implicit project")

		// GetMetadata should be called by descendents of the project
		// e.g. control planes, and by transitive closure, clusters.
		// Rather than delegate creation to each and every client,
		// implicitly create it.
		if err := c.Create(ctx); err != nil {
			return nil, err
		}
	}

	waitCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Allow a grace period for the project to become active to avoid client
	// errors and retries.  The namespace creation should be ostensibly instant
	// and likewise show up due to non-blocking yields.
	callback := func() error {
		result, err = c.get(waitCtx, name)
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
		Name:      name,
		Namespace: result.Status.Namespace,
		Deleting:  result.DeletionTimestamp != nil,
	}

	return metadata, nil
}

// get returns the implicit project identified by the JWT claims.
func (c *Client) get(ctx context.Context, name string) (*unikornv1.Project, error) {
	result := &unikornv1.Project{}

	if err := c.client.Get(ctx, client.ObjectKey{Name: name}, result); err != nil {
		if kerrors.IsNotFound(err) {
			return nil, errors.HTTPNotFound().WithError(err)
		}

		return nil, errors.OAuth2ServerError("failed to get project").WithError(err)
	}

	return result, nil
}

// Create creates the implicit project indentified by the JTW claims.
func (c *Client) Create(ctx context.Context) error {
	name, err := NameFromContext(ctx)
	if err != nil {
		return err
	}

	// TODO: common with CLI tools.
	project := &unikornv1.Project{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			Labels: map[string]string{
				constants.VersionLabel: constants.Version,
			},
		},
	}

	if err := c.client.Create(ctx, project); err != nil {
		// TODO: we can do a cached lookup to save the API traffic.
		if kerrors.IsAlreadyExists(err) {
			return errors.HTTPConflict()
		}

		return errors.OAuth2ServerError("failed to create project").WithError(err)
	}

	return nil
}

// Delete deletes the implicit project indentified by the JTW claims.
func (c *Client) Delete(ctx context.Context) error {
	name, err := NameFromContext(ctx)
	if err != nil {
		return err
	}

	project := &unikornv1.Project{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
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
