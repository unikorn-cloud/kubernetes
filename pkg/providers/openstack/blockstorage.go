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

	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack"
	"github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/availabilityzones"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"

	"github.com/unikorn-cloud/unikorn/pkg/constants"
)

// BlockStorageClient wraps the generic client because gophercloud is unsafe.
type BlockStorageClient struct {
	client *gophercloud.ServiceClient
}

// NewBlockStorageClient provides a simple one-liner to start computing.
func NewBlockStorageClient(ctx context.Context, provider CredentialProvider) (*BlockStorageClient, error) {
	providerClient, err := provider.Client(ctx)
	if err != nil {
		return nil, err
	}

	client, err := openstack.NewBlockStorageV3(providerClient, gophercloud.EndpointOpts{})
	if err != nil {
		return nil, err
	}

	c := &BlockStorageClient{
		client: client,
	}

	return c, nil
}

// AvailabilityZones retrieves block storage availability zones.
func (c *BlockStorageClient) AvailabilityZones(ctx context.Context) ([]availabilityzones.AvailabilityZone, error) {
	url := c.client.ServiceURL("os-availability-zone")

	tracer := otel.GetTracerProvider().Tracer(constants.Application)

	_, span := tracer.Start(ctx, url, trace.WithSpanKind(trace.SpanKindClient))
	defer span.End()

	pages, err := availabilityzones.List(c.client).AllPages(ctx)
	if err != nil {
		return nil, err
	}

	result, err := availabilityzones.ExtractAvailabilityZones(pages)
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
