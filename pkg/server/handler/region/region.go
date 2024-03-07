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

package region

import (
	"context"
	"errors"

	unikornv1 "github.com/unikorn-cloud/unikorn/pkg/apis/unikorn/v1alpha1"
	"github.com/unikorn-cloud/unikorn/pkg/providers"
	"github.com/unikorn-cloud/unikorn/pkg/providers/openstack"
	"github.com/unikorn-cloud/unikorn/pkg/server/generated"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	// ErrRegionNotFound is raised when a region doesn't exist.
	ErrRegionNotFound = errors.New("region doesn't exist")

	// ErrRegionProviderUnimplmented is raised when you haven't written
	// it yet!
	ErrRegionProviderUnimplmented = errors.New("region provider unimplmented")
)

type Client struct {
	client client.Client
	// region string
}

func NewClient(client client.Client) *Client {
	return &Client{
		client: client,
	}
}

// list is a canonical lister function that allows filtering to be applied
// in one place e.g. health, ownership, etc.
func (c *Client) list(ctx context.Context) (*unikornv1.RegionList, error) {
	var regions unikornv1.RegionList

	if err := c.client.List(ctx, &regions); err != nil {
		return nil, err
	}

	return &regions, nil
}

func findRegion(regions *unikornv1.RegionList, region string) (*unikornv1.Region, error) {
	for i := range regions.Items {
		if regions.Items[i].Name == region {
			return &regions.Items[i], nil
		}
	}

	return nil, ErrRegionNotFound
}

//nolint:gochecknoglobals
var cache = map[string]providers.Provider{}

func (c Client) newProvider(region *unikornv1.Region) (providers.Provider, error) {
	//nolint:gocritic
	switch region.Spec.Provider {
	case unikornv1.ProviderOpenstack:
		return openstack.New(c.client, region), nil
	}

	return nil, ErrRegionProviderUnimplmented
}

func (c *Client) Provider(ctx context.Context, regionName string) (providers.Provider, error) {
	regions, err := c.list(ctx)
	if err != nil {
		return nil, err
	}

	region, err := findRegion(regions, regionName)
	if err != nil {
		return nil, err
	}

	if provider, ok := cache[region.Name]; ok {
		return provider, nil
	}

	provider, err := c.newProvider(region)
	if err != nil {
		return nil, err
	}

	cache[region.Name] = provider

	return provider, nil
}

func convert(in *unikornv1.Region) *generated.Region {
	out := &generated.Region{
		Name: in.Name,
	}

	return out
}

func convertList(in *unikornv1.RegionList) generated.Regions {
	out := make(generated.Regions, 0, len(in.Items))

	for i := range in.Items {
		out = append(out, *convert(&in.Items[i]))
	}

	return out
}

func (c *Client) List(ctx context.Context) (generated.Regions, error) {
	regions, err := c.list(ctx)
	if err != nil {
		return nil, err
	}

	return convertList(regions), nil
}
