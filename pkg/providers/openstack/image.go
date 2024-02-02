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
	"crypto/ecdsa"
	"crypto/sha256"
	"encoding/base64"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
	"github.com/gophercloud/gophercloud/openstack/imageservice/v2/images"
	"github.com/spf13/pflag"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/exp/slices"

	"github.com/unikorn-cloud/core/pkg/constants"
	"github.com/unikorn-cloud/core/pkg/util"
	"github.com/unikorn-cloud/unikorn/pkg/flags"
)

// ImageOptions allows things like filtering to be configured.
type ImageOptions struct {
	Properties []string
	SigningKey flags.PublicKeyVar
}

func (o *ImageOptions) AddFlags(f *pflag.FlagSet) {
	f.StringSliceVar(&o.Properties, "openstack-image-properties", nil, "Properties an image must possess in order to be exposed")
	f.Var(&o.SigningKey, "openstack-image-signing-key", "Public key used to verify valid images for use with the platform")
}

// ImageClient wraps the generic client because gophercloud is unsafe.
type ImageClient struct {
	client  *gophercloud.ServiceClient
	options *ImageOptions
}

// NewImageClient provides a simple one-liner to start computing.
func NewImageClient(provider Provider, options *ImageOptions) (*ImageClient, error) {
	providerClient, err := provider.Client()
	if err != nil {
		return nil, err
	}

	client, err := openstack.NewImageServiceV2(providerClient, gophercloud.EndpointOpts{})
	if err != nil {
		return nil, err
	}

	c := &ImageClient{
		client:  client,
		options: options,
	}

	return c, nil
}

func (c *ImageClient) validateProperties(image *images.Image) bool {
	for _, r := range c.options.Properties {
		if !slices.Contains(util.Keys(image.Properties), r) {
			return false
		}
	}

	return true
}

// verifyImage asserts the image is trustworthy for use with our goodselves.
func (c *ImageClient) verifyImage(image *images.Image) bool {
	if image.Properties == nil {
		return false
	}

	if c.options.SigningKey.Key != nil {
		// These will be digitally signed by Baski when created, so we only trust
		// those images.
		signatureRaw, ok := image.Properties["digest"]
		if !ok {
			return false
		}

		signatureB64, ok := signatureRaw.(string)
		if !ok {
			return false
		}

		signature, err := base64.StdEncoding.DecodeString(signatureB64)
		if err != nil {
			return false
		}

		hash := sha256.Sum256([]byte(image.ID))

		return ecdsa.VerifyASN1(c.options.SigningKey.Key, hash[:], signature)
	}

	return true
}

// Images returns a list of images.
func (c *ImageClient) Images(ctx context.Context) ([]images.Image, error) {
	tracer := otel.GetTracerProvider().Tracer(constants.Application)

	_, span := tracer.Start(ctx, "/imageservice/v2/images", trace.WithSpanKind(trace.SpanKindClient))
	defer span.End()

	page, err := images.List(c.client, &images.ListOpts{}).AllPages()
	if err != nil {
		return nil, err
	}

	result, err := images.ExtractImages(page)
	if err != nil {
		return nil, err
	}

	// Filter out images that aren't compatible.
	filtered := []images.Image{}

	for i := range result {
		image := result[i]

		if image.Status != "active" {
			continue
		}

		if !c.validateProperties(&image) {
			continue
		}

		if !c.verifyImage(&image) {
			continue
		}

		filtered = append(filtered, image)
	}

	return filtered, nil
}
