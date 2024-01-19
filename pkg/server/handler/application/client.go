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

package application

import (
	"context"
	"slices"
	"strings"

	"github.com/spjmurray/unikorn/pkg/server/errors"
	"github.com/spjmurray/unikorn/pkg/server/generated"

	coreunikornv1 "github.com/spjmurray/unikorn-core/pkg/apis/unikorn/v1alpha1"

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

func convert(in *coreunikornv1.HelmApplication) *generated.Application {
	versions := make(generated.ApplicationVersions, 0, len(in.Spec.Versions))

	for _, version := range in.Spec.Versions {
		v := generated.ApplicationVersion{
			Version: *version.Version,
		}

		if len(version.Dependencies) != 0 {
			deps := make(generated.ApplicationDependencies, 0, len(version.Dependencies))

			for _, dependency := range version.Dependencies {
				deps = append(deps, generated.ApplicationDependency{
					Name: *dependency.Name,
				})
			}

			v.Dependencies = &deps
		}

		if len(version.Recommends) != 0 {
			recommends := make(generated.ApplicationRecommends, 0, len(version.Recommends))

			for _, recommend := range version.Recommends {
				recommends = append(recommends, generated.ApplicationDependency{
					Name: *recommend.Name,
				})
			}

			v.Recommends = &recommends
		}

		versions = append(versions, v)
	}

	out := &generated.Application{
		Name:              in.Name,
		HumanReadableName: *in.Spec.Name,
		Description:       strings.ReplaceAll(*in.Spec.Description, "\n", " "),
		Documentation:     *in.Spec.Documentation,
		License:           *in.Spec.License,
		Icon:              in.Spec.Icon,
		Versions:          versions,
		Tags:              &in.Spec.Tags,
	}

	return out
}

func convertList(in []coreunikornv1.HelmApplication) []*generated.Application {
	out := make([]*generated.Application, len(in))

	for i := range in {
		out[i] = convert(&in[i])
	}

	return out
}

func (c *Client) List(ctx context.Context) ([]*generated.Application, error) {
	result := &coreunikornv1.HelmApplicationList{}

	if err := c.client.List(ctx, result); err != nil {
		return nil, errors.OAuth2ServerError("failed to list applications").WithError(err)
	}

	exported := result.Exported()

	slices.SortStableFunc(exported.Items, coreunikornv1.CompareHelmApplication)

	return convertList(exported.Items), nil
}
