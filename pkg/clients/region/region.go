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
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"net/http"

	"github.com/spf13/pflag"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"

	"github.com/unikorn-cloud/core/pkg/authorization/accesstoken"
	"github.com/unikorn-cloud/region/pkg/openapi"
)

var (
	// ErrFormatError is returned when a file doesn't meet the specification.
	ErrFormatError = errors.New("file incorrectly formatted")
)

type Options struct {
	// host is the region host name.
	host string
	// ca allows a non-public CA to be used for TLS verification.
	ca []byte
}

func (o *Options) AddFlags(f *pflag.FlagSet) {
	f.StringVar(&o.host, "region-host", "", "Region endpoint URL.")
	f.BytesBase64Var(&o.ca, "region-ca", nil, "Region endpoint CA certificate.")
}

func tlsClientConfig(o *Options) (*tls.Config, error) {
	if o.ca == nil {
		//nolint:nilnil
		return nil, nil
	}

	block, _ := pem.Decode(o.ca)
	if block == nil {
		return nil, fmt.Errorf("%w: CA file contains no PEM data", ErrFormatError)
	}

	if block.Type != "CERTIFICATE" && block.Type != "RSA CERTIFICATE" {
		return nil, fmt.Errorf("%w: CA file has wrong PEM type: %s", ErrFormatError, block.Type)
	}

	certificate, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, err
	}

	pool := x509.NewCertPool()
	pool.AddCert(certificate)

	config := &tls.Config{
		RootCAs:    pool,
		MinVersion: tls.VersionTLS13,
	}

	return config, nil
}

// getClient returns a new http client that will transparently do oauth2 header
// injection and refresh token updates.
func getClient(o *Options) (*http.Client, error) {
	// Handle non-system CA certificates for the OIDC discovery protocol
	// and oauth2 token refresh. This will return nil if none is specified
	// and default to the system roots.
	tlsClientConfig, err := tlsClientConfig(o)
	if err != nil {
		return nil, err
	}

	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: tlsClientConfig,
		},
	}

	return client, nil
}

func accessTokenInjector(ctx context.Context, req *http.Request) error {
	req.Header.Set("Authorization", "bearer "+accesstoken.FromContext(ctx))

	otel.GetTextMapPropagator().Inject(ctx, propagation.HeaderCarrier(req.Header))

	return nil
}

func New(o *Options) (*openapi.ClientWithResponses, error) {
	client, err := getClient(o)
	if err != nil {
		return nil, err
	}

	region, err := openapi.NewClientWithResponses(o.host, openapi.WithHTTPClient(client), openapi.WithRequestEditorFn(accessTokenInjector))
	if err != nil {
		return nil, err
	}

	return region, nil
}
