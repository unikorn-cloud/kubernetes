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
	"os"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
	"github.com/gophercloud/utils/openstack/clientconfig"

	corev1 "k8s.io/api/core/v1"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	ErrKeyUndefined = errors.New("a required key was not defined")
)

// authenticatedClient returns a provider client used to initialize service clients.
func authenticatedClient(options gophercloud.AuthOptions) (*gophercloud.ProviderClient, error) {
	// TODO: the JWT token issuer will cap the expiry at that of the
	// keystone token, so we shouldn't get an unauthorized error.  Just
	// as well as we cannot disambiguate from what gophercloud returns.
	client, err := openstack.AuthenticatedClient(options)
	if err != nil {
		return nil, err
	}

	return client, nil
}

// Provider abstracts authentication methods.
type Provider interface {
	// Client returns a new provider client.
	Client(context.Context) (*gophercloud.ProviderClient, error)
}

// ApplicationCredentialProvider allows use of an application credential.
type ApplicationCredentialProvider struct {
	client client.Client

	// endpoint is the Keystone endpoint to hit to get access to tokens
	// and the service catalog.
	endpoint string

	secretName string
}

// Ensure the interface is implemented.
var _ Provider = &ApplicationCredentialProvider{}

// NewApplicationCredentialProvider creates a client that comsumes application
// credentials for authentication.
// NOTE: The intent here, by passing around the client and secret name is to
// ride through credential rotation, however, gophercloud caches this information.
// However, given ACs should be rotated leaving the old one in place while the
// new one comes on line, and the limited lifespan of OIDC access tokens (that
// cached clients are keyed to) means it should work well enough.
func NewApplicationCredentialProvider(client client.Client, endpoint, secretName string) *ApplicationCredentialProvider {
	return &ApplicationCredentialProvider{
		client:     client,
		endpoint:   endpoint,
		secretName: secretName,
	}
}

// Client implements the Provider interface.
func (p *ApplicationCredentialProvider) Client(ctx context.Context) (*gophercloud.ProviderClient, error) {
	key := client.ObjectKey{
		Namespace: os.Getenv("KUBERNETES_NAMESPACE"),
		Name:      p.secretName,
	}

	var secret corev1.Secret

	if err := p.client.Get(ctx, key, &secret); err != nil {
		return nil, err
	}

	acID, ok := secret.Data["applicationcredentialid"]
	if !ok {
		return nil, fmt.Errorf("%w: applicationcredentialid", ErrKeyUndefined)
	}

	acSecret, ok := secret.Data["secret"]
	if !ok {
		return nil, fmt.Errorf("%w: secret", ErrKeyUndefined)
	}

	options := gophercloud.AuthOptions{
		IdentityEndpoint:            p.endpoint,
		ApplicationCredentialID:     string(acID),
		ApplicationCredentialSecret: string(acSecret),
	}

	return authenticatedClient(options)
}

// TokenProvider creates a client from an endpoint and token.
type TokenProvider struct {
	// endpoint is the Keystone endpoint to hit to get access to tokens
	// and the service catalog.
	endpoint string

	// token is an Openstack authorization token.
	token string
}

// Ensure the interface is implemented.
var _ Provider = &TokenProvider{}

// NewTokenProvider returns a new initialized provider.
func NewTokenProvider(endpoint, token string) *TokenProvider {
	return &TokenProvider{
		endpoint: endpoint,
		token:    token,
	}
}

// Client implements the Provider interface.
func (p *TokenProvider) Client(_ context.Context) (*gophercloud.ProviderClient, error) {
	options := gophercloud.AuthOptions{
		IdentityEndpoint: p.endpoint,
		TokenID:          p.token,
	}

	return authenticatedClient(options)
}

// CloudsProvider cretes a client from clouds.yaml.
type CloudsProvider struct {
	// cloud is the key to lookup in clouds.yaml.
	cloud string
}

// Ensure the interface is implemented.
var _ Provider = &CloudsProvider{}

// NewTokenProvider returns a new initialized provider.
func NewCloudsProvider(cloud string) *CloudsProvider {
	return &CloudsProvider{
		cloud: cloud,
	}
}

// Client implements the Provider interface.
func (p *CloudsProvider) Client(_ context.Context) (*gophercloud.ProviderClient, error) {
	clientOpts := &clientconfig.ClientOpts{
		Cloud: p.cloud,
	}

	options, err := clientconfig.AuthOptions(clientOpts)
	if err != nil {
		return nil, err
	}

	return authenticatedClient(*options)
}

// UnauthenticatedProvider is used for token issue.
type UnauthenticatedProvider struct {
	// endpoint is the Keystone endpoint to hit to get access to tokens
	// and the service catalog.
	endpoint string
}

// Ensure the interface is implemented.
var _ Provider = &UnauthenticatedProvider{}

// NewTokenProvider returns a new initialized provider.
func NewUnauthenticatedProvider(endpoint string) *UnauthenticatedProvider {
	return &UnauthenticatedProvider{
		endpoint: endpoint,
	}
}

// Client implements the Provider interface.
func (p *UnauthenticatedProvider) Client(_ context.Context) (*gophercloud.ProviderClient, error) {
	client, err := openstack.NewClient(p.endpoint)
	if err != nil {
		return nil, err
	}

	return client, nil
}
