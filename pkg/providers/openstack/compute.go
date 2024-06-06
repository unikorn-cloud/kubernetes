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

package openstack

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"time"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack"
	"github.com/gophercloud/gophercloud/v2/openstack/compute/v2/availabilityzones"
	"github.com/gophercloud/gophercloud/v2/openstack/compute/v2/flavors"
	"github.com/gophercloud/gophercloud/v2/openstack/compute/v2/keypairs"
	"github.com/gophercloud/gophercloud/v2/openstack/compute/v2/servergroups"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"

	"github.com/unikorn-cloud/core/pkg/util/cache"
	unikornv1 "github.com/unikorn-cloud/unikorn/pkg/apis/unikorn/v1alpha1"
	"github.com/unikorn-cloud/unikorn/pkg/constants"
)

var (
	// ErrExpression is raised at runtime when expression evaluation fails.
	ErrExpression = errors.New("expression must contain exactly one sub match that yields a number string")
)

// ComputeClient wraps the generic client because gophercloud is unsafe.
type ComputeClient struct {
	options *unikornv1.RegionOpenstackComputeSpec
	client  *gophercloud.ServiceClient

	flavorCache *cache.TimeoutCache[[]flavors.Flavor]
}

// NewComputeClient provides a simple one-liner to start computing.
func NewComputeClient(ctx context.Context, provider CredentialProvider, options *unikornv1.RegionOpenstackComputeSpec) (*ComputeClient, error) {
	providerClient, err := provider.Client(ctx)
	if err != nil {
		return nil, err
	}

	client, err := openstack.NewComputeV2(providerClient, gophercloud.EndpointOpts{})
	if err != nil {
		return nil, err
	}

	// Need at least 2.15 for soft-anti-affinity policy.
	// Need at least 2.64 for new server group interface.
	client.Microversion = "2.90"

	c := &ComputeClient{
		options:     options,
		client:      client,
		flavorCache: cache.New[[]flavors.Flavor](time.Hour),
	}

	return c, nil
}

// KeyPairs returns a list of key pairs.
func (c *ComputeClient) KeyPairs(ctx context.Context) ([]keypairs.KeyPair, error) {
	tracer := otel.GetTracerProvider().Tracer(constants.Application)

	_, span := tracer.Start(ctx, "/compute/v2/os-keypairs", trace.WithSpanKind(trace.SpanKindClient))
	defer span.End()

	page, err := keypairs.List(c.client, &keypairs.ListOpts{}).AllPages(ctx)
	if err != nil {
		return nil, err
	}

	return keypairs.ExtractKeyPairs(page)
}

// Flavors returns a list of flavors.
//
//nolint:cyclop
func (c *ComputeClient) Flavors(ctx context.Context) ([]flavors.Flavor, error) {
	if result, ok := c.flavorCache.Get(); ok {
		return result, nil
	}

	tracer := otel.GetTracerProvider().Tracer(constants.Application)

	_, span := tracer.Start(ctx, "/compute/v2/flavors", trace.WithSpanKind(trace.SpanKindClient))
	defer span.End()

	page, err := flavors.ListDetail(c.client, &flavors.ListOpts{SortKey: "name"}).AllPages(ctx)
	if err != nil {
		return nil, err
	}

	result, err := flavors.ExtractFlavors(page)
	if err != nil {
		return nil, err
	}

	result = slices.DeleteFunc(result, func(flavor flavors.Flavor) bool {
		// We are admin, so see all the things, throw out private flavors.
		// TODO: we _could_ allow if our project is in the allowed IDs.
		if !flavor.IsPublic {
			return true
		}

		// Kubeadm requires 2 VCPU, 2 "GB" of RAM (I'll pretend it's GiB) and no swap:
		// https://kubernetes.io/docs/setup/production-environment/tools/kubeadm/install-kubeadm/
		if flavor.VCPUs < 2 {
			return true
		}

		// In MB...
		if flavor.RAM < 2048 {
			return true
		}

		if flavor.Swap != 0 {
			return true
		}

		if c.options == nil {
			return false
		}

		for _, exclude := range c.options.FlavorExtraSpecsExclude {
			if _, ok := flavor.ExtraSpecs[exclude]; ok {
				return true
			}
		}

		return false
	})

	c.flavorCache.Set(result)

	return result, nil
}

// GPUMeta describes GPUs.
type GPUMeta struct {
	// GPUs is the number of GPUs, this may be the total number
	// or physical GPUs, or a single virtual GPU.  This value
	// is what will be reported for Kubernetes scheduling.
	GPUs int
}

// extraSpecToGPUs evaluates the falvor extra spec and tries to derive
// the number of GPUs, returns -1 if none are found.
func (c *ComputeClient) extraSpecToGPUs(name, value string) (int, error) {
	if c.options == nil {
		return -1, nil
	}

	for _, desc := range c.options.GPUDescriptors {
		if desc.Property != name {
			continue
		}

		re, err := regexp.Compile(desc.Expression)
		if err != nil {
			return -1, err
		}

		matches := re.FindStringSubmatch(value)
		if matches == nil {
			continue
		}

		if len(matches) != 2 {
			return -1, ErrExpression
		}

		i, err := strconv.Atoi(matches[1])
		if err != nil {
			return -1, fmt.Errorf("%w: %s", ErrExpression, err.Error())
		}

		return i, nil
	}

	return -1, nil
}

// FlavorGPUs returns metadata about GPUs, e.g. the number of GPUs.  Sadly there is absolutely
// no way of assiging metadata to flavors without having to add those same values to your host
// aggregates, so we have to have knowledge of flavors built in somewhere.
func (c *ComputeClient) FlavorGPUs(flavor *flavors.Flavor) (GPUMeta, error) {
	var result GPUMeta

	for name, value := range flavor.ExtraSpecs {
		gpus, err := c.extraSpecToGPUs(name, value)
		if err != nil {
			return result, err
		}

		if gpus == -1 {
			continue
		}

		result.GPUs = gpus

		break
	}

	return result, nil
}

// AvailabilityZones returns a list of availability zones.
func (c *ComputeClient) AvailabilityZones(ctx context.Context) ([]availabilityzones.AvailabilityZone, error) {
	tracer := otel.GetTracerProvider().Tracer(constants.Application)

	_, span := tracer.Start(ctx, "/compute/v2/os-availability-zones", trace.WithSpanKind(trace.SpanKindClient))
	defer span.End()

	page, err := availabilityzones.List(c.client).AllPages(ctx)
	if err != nil {
		return nil, err
	}

	result, err := availabilityzones.ExtractAvailabilityZones(page)
	if err != nil {
		return nil, err
	}

	filtered := []availabilityzones.AvailabilityZone{}

	for _, az := range result {
		if !az.ZoneState.Available {
			continue
		}

		filtered = append(filtered, az)
	}

	return filtered, nil
}

// ListServerGroups returns all server groups in the project.
func (c *ComputeClient) ListServerGroups(ctx context.Context) ([]servergroups.ServerGroup, error) {
	tracer := otel.GetTracerProvider().Tracer(constants.Application)

	_, span := tracer.Start(ctx, "/compute/v2/os-server-groups", trace.WithSpanKind(trace.SpanKindClient))
	defer span.End()

	page, err := servergroups.List(c.client, &servergroups.ListOpts{}).AllPages(ctx)
	if err != nil {
		return nil, err
	}

	return servergroups.ExtractServerGroups(page)
}

// CreateServerGroup creates the named server group with the given policy and returns
// the result.
func (c *ComputeClient) CreateServerGroup(ctx context.Context, name string) (*servergroups.ServerGroup, error) {
	tracer := otel.GetTracerProvider().Tracer(constants.Application)

	_, span := tracer.Start(ctx, "/compute/v2/os-server-groups", trace.WithSpanKind(trace.SpanKindClient))
	defer span.End()

	opts := &servergroups.CreateOpts{
		Name:   name,
		Policy: "soft-anti-affinity",
	}

	if c.options != nil && c.options.ServerGroupPolicy != nil {
		opts.Policy = *c.options.ServerGroupPolicy
	}

	return servergroups.Create(ctx, c.client, opts).Extract()
}
