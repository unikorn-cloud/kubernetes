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

package client

import (
	"context"
	"net/http"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"

	coreclient "github.com/unikorn-cloud/core/pkg/client"
	"github.com/unikorn-cloud/identity/pkg/middleware/authorization"
	"github.com/unikorn-cloud/identity/pkg/middleware/openapi/accesstoken"
	"github.com/unikorn-cloud/kubernetes/pkg/openapi"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Options = coreclient.HTTPOptions

// NewOptions must be used to create options for consistency.
func NewOptions() *Options {
	return coreclient.NewHTTPOptions("kubernetes")
}

// Client wraps up the raw OpenAPI client with things to make it useable e.g.
// authorization and TLS.
type Client struct {
	// client is a Kubenetes client.
	client client.Client
	// options allows setting of option from the CLI
	options *Options
	// clientOptions may be specified to inject client certificates etc.
	clientOptions *coreclient.HTTPClientOptions
}

// New creates a new client.
func New(client client.Client, options *Options, clientOptions *coreclient.HTTPClientOptions) *Client {
	return &Client{
		client:        client,
		options:       options,
		clientOptions: clientOptions,
	}
}

// HTTPClient returns a new http client that will transparently do oauth2 header
// injection and refresh token updates.
func (c *Client) HTTPClient(ctx context.Context) (*http.Client, error) {
	// Handle non-system CA certificates for the OIDC discovery protocol
	// and oauth2 token refresh. This will return nil if none is specified
	// and default to the system roots.
	tlsClientConfig, err := coreclient.TLSClientConfig(ctx, c.client, c.options, c.clientOptions)
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

// accessTokenInjector implements OAuth2 bearer token authorization.
func accessTokenInjector(ctx context.Context, req *http.Request) error {
	accessToken, err := accesstoken.FromContext(ctx)
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "bearer "+accessToken)
	otel.GetTextMapPropagator().Inject(ctx, propagation.HeaderCarrier(req.Header))
	authorization.InjectClientCert(ctx, req.Header)

	return nil
}

// Client returns a new OpenAPI client that can be used to access the API.
func (c *Client) Client(ctx context.Context) (*openapi.ClientWithResponses, error) {
	httpClient, err := c.HTTPClient(ctx)
	if err != nil {
		return nil, err
	}

	client, err := openapi.NewClientWithResponses(c.options.Host(), openapi.WithHTTPClient(httpClient), openapi.WithRequestEditorFn(accessTokenInjector))
	if err != nil {
		return nil, err
	}

	return client, nil
}
