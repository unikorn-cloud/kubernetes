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
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
	"github.com/gophercloud/gophercloud/openstack/imageservice/v2/images"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"

	"github.com/unikorn-cloud/core/pkg/util"
	"github.com/unikorn-cloud/core/pkg/util/cache"
	unikornv1 "github.com/unikorn-cloud/unikorn/pkg/apis/unikorn/v1alpha1"
	"github.com/unikorn-cloud/unikorn/pkg/constants"
)

var (
	// ErrPEMDecode is raised when the PEM decode failed for some reason.
	ErrPEMDecode = errors.New("PEM decode error")

	// ErrPEMType is raised when the encounter the wrong PEM type, e.g. PKCS#1.
	ErrPEMType = errors.New("PEM type unsupported")

	// ErrKeyType is raised when we encounter an unsupported key type.
	ErrKeyType = errors.New("key type unsupported")
)

// ImageClient wraps the generic client because gophercloud is unsafe.
type ImageClient struct {
	client     *gophercloud.ServiceClient
	options    *unikornv1.RegionOpenstackImageSpec
	imageCache *cache.TimeoutCache[[]images.Image]
}

// NewImageClient provides a simple one-liner to start computing.
func NewImageClient(ctx context.Context, provider CredentialProvider, options *unikornv1.RegionOpenstackImageSpec) (*ImageClient, error) {
	providerClient, err := provider.Client()
	if err != nil {
		return nil, err
	}

	client, err := openstack.NewImageServiceV2(providerClient, gophercloud.EndpointOpts{})
	if err != nil {
		return nil, err
	}

	c := &ImageClient{
		client:     client,
		options:    options,
		imageCache: cache.New[[]images.Image](time.Hour),
	}

	return c, nil
}

func (c *ImageClient) validateProperties(image *images.Image) bool {
	if c.options == nil {
		return true
	}

	for _, r := range c.options.PropertiesInclude {
		if !slices.Contains(util.Keys(image.Properties), r) {
			return false
		}
	}

	return true
}

func (c *ImageClient) decodeSigningKey() (*ecdsa.PublicKey, error) {
	pemBlock, _ := pem.Decode(c.options.SigningKey)
	if pemBlock == nil {
		return nil, ErrPEMDecode
	}

	if pemBlock.Type != "PUBLIC KEY" {
		return nil, fmt.Errorf("%w: %s", ErrPEMType, pemBlock.Type)
	}

	key, err := x509.ParsePKIXPublicKey(pemBlock.Bytes)
	if err != nil {
		return nil, err
	}

	ecKey, ok := key.(*ecdsa.PublicKey)
	if !ok {
		return nil, ErrKeyType
	}

	return ecKey, nil
}

// verifyImage asserts the image is trustworthy for use with our goodselves.
func (c *ImageClient) verifyImage(image *images.Image) bool {
	if c.options == nil || c.options.SigningKey == nil {
		return true
	}

	if image.Properties == nil {
		return false
	}

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

	signingKey, err := c.decodeSigningKey()
	if err != nil {
		return false
	}

	return ecdsa.VerifyASN1(signingKey, hash[:], signature)
}

func (c *ImageClient) imageValid(image *images.Image) bool {
	if image.Status != "active" {
		return false
	}

	if !c.validateProperties(image) {
		return false
	}

	if !c.verifyImage(image) {
		return false
	}

	return true
}

// Images returns a list of images.
func (c *ImageClient) Images(ctx context.Context) ([]images.Image, error) {
	if result, ok := c.imageCache.Get(); ok {
		return result, nil
	}

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
	result = slices.DeleteFunc(result, func(image images.Image) bool {
		return !c.imageValid(&image)
	})

	// Sort by age, the newest should have the fewest CVEs!
	slices.SortStableFunc(result, func(a, b images.Image) int {
		return a.CreatedAt.Compare(b.CreatedAt)
	})

	c.imageCache.Set(result)

	return result, nil
}
