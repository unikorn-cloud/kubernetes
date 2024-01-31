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

package applicationbundle

import (
	"context"
	"slices"

	unikornv1 "github.com/unikorn-cloud/unikorn/pkg/apis/unikorn/v1alpha1"
	"github.com/unikorn-cloud/unikorn/pkg/server/errors"
	"github.com/unikorn-cloud/unikorn/pkg/server/generated"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Client wraps up application bundle related management handling.
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

func convertControlPlane(in *unikornv1.ControlPlaneApplicationBundle) *generated.ApplicationBundle {
	out := &generated.ApplicationBundle{
		Name:    in.Name,
		Version: *in.Spec.Version,
		Preview: in.Spec.Preview,
	}

	if in.Spec.EndOfLife != nil {
		out.EndOfLife = &in.Spec.EndOfLife.Time
	}

	return out
}

func convertKubernetesCluster(in *unikornv1.KubernetesClusterApplicationBundle) *generated.ApplicationBundle {
	out := &generated.ApplicationBundle{
		Name:    in.Name,
		Version: *in.Spec.Version,
		Preview: in.Spec.Preview,
	}

	if in.Spec.EndOfLife != nil {
		out.EndOfLife = &in.Spec.EndOfLife.Time
	}

	return out
}

func convertControlPlaneList(in []unikornv1.ControlPlaneApplicationBundle) []*generated.ApplicationBundle {
	out := make([]*generated.ApplicationBundle, len(in))

	for i := range in {
		out[i] = convertControlPlane(&in[i])
	}

	return out
}

func convertKubernetesClusterList(in []unikornv1.KubernetesClusterApplicationBundle) []*generated.ApplicationBundle {
	out := make([]*generated.ApplicationBundle, len(in))

	for i := range in {
		out[i] = convertKubernetesCluster(&in[i])
	}

	return out
}

func (c *Client) GetControlPlane(ctx context.Context, name string) (*generated.ApplicationBundle, error) {
	result := &unikornv1.ControlPlaneApplicationBundle{}

	if err := c.client.Get(ctx, client.ObjectKey{Name: name}, result); err != nil {
		return nil, errors.HTTPNotFound().WithError(err)
	}

	return convertControlPlane(result), nil
}

func (c *Client) GetKubernetesCluster(ctx context.Context, name string) (*generated.ApplicationBundle, error) {
	result := &unikornv1.KubernetesClusterApplicationBundle{}

	if err := c.client.Get(ctx, client.ObjectKey{Name: name}, result); err != nil {
		return nil, errors.HTTPNotFound().WithError(err)
	}

	return convertKubernetesCluster(result), nil
}

func (c *Client) ListControlPlane(ctx context.Context) ([]*generated.ApplicationBundle, error) {
	result := &unikornv1.ControlPlaneApplicationBundleList{}

	if err := c.client.List(ctx, result); err != nil {
		return nil, errors.OAuth2ServerError("failed to list application bundles").WithError(err)
	}

	slices.SortStableFunc(result.Items, unikornv1.CompareControlPlaneApplicationBundle)

	return convertControlPlaneList(result.Items), nil
}

func (c *Client) ListCluster(ctx context.Context) ([]*generated.ApplicationBundle, error) {
	result := &unikornv1.KubernetesClusterApplicationBundleList{}

	if err := c.client.List(ctx, result); err != nil {
		return nil, errors.OAuth2ServerError("failed to list application bundles").WithError(err)
	}

	slices.SortStableFunc(result.Items, unikornv1.CompareKubernetesClusterApplicationBundle)

	return convertKubernetesClusterList(result.Items), nil
}
