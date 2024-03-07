/*
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

package providers

import (
	"context"

	unikornv1 "github.com/unikorn-cloud/unikorn/pkg/apis/unikorn/v1alpha1"
)

// Providers are expected to provide a provider agnostic manner.
// They are also expected to provide any caching or memoization required
// to provide high performance and a decent UX.
type Provider interface {
	// Flavors list all available flavors.
	Flavors(ctx context.Context) (FlavorList, error)
	// Images lists all available images.
	Images(ctx context.Context) (ImageList, error)
	// ConfigureCluster does any provider specific configuration for a cluster.
	ConfigureCluster(ctx context.Context, cluster *unikornv1.KubernetesCluster) error
}
