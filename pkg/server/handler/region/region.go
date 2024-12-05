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
	"fmt"
	"net/http"
	"slices"

	regionapi "github.com/unikorn-cloud/region/pkg/openapi"
)

var (
	ErrStatusCode = errors.New("unexpected status code")
)

// Flavors returns all Kubernetes compatible flavors.
func Flavors(ctx context.Context, client regionapi.ClientWithResponsesInterface, organizationID, regionID string) ([]regionapi.Flavor, error) {
	resp, err := client.GetApiV1OrganizationsOrganizationIDRegionsRegionIDFlavorsWithResponse(ctx, organizationID, regionID)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("%w: expected 200, got %d", ErrStatusCode, resp.StatusCode())
	}

	flavors := *resp.JSON200

	flavors = slices.DeleteFunc(flavors, func(x regionapi.Flavor) bool {
		// kubeadm requires at least 2 VCPU and 2 GiB memory.
		return x.Spec.Cpus < 2 || x.Spec.Memory < 2
	})

	return flavors, nil
}

// Images returns all Kubernetes compatible images.
func Images(ctx context.Context, client regionapi.ClientWithResponsesInterface, organizationID, regionID string) ([]regionapi.Image, error) {
	resp, err := client.GetApiV1OrganizationsOrganizationIDRegionsRegionIDImagesWithResponse(ctx, organizationID, regionID)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("%w: expected 200, got %d", ErrStatusCode, resp.StatusCode())
	}

	images := *resp.JSON200

	images = slices.DeleteFunc(images, func(x regionapi.Image) bool {
		return x.Spec.SoftwareVersions == nil || (*x.Spec.SoftwareVersions)["kubernetes"] == ""
	})

	return images, nil
}
