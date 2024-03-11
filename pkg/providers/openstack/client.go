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
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
	"github.com/gophercloud/utils/openstack/clientconfig"
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

// CredentialProvider abstracts authentication methods.
type CredentialProvider interface {
	// Client returns a new provider client.
	Client() (*gophercloud.ProviderClient, error)
}

// ApplicationCredentialProvider allows use of an application credential.
type ApplicationCredentialProvider struct {
	endpoint string
	id       string
	secret   string
}

// Ensure the interface is implemented.
var _ CredentialProvider = &ApplicationCredentialProvider{}

// NewApplicationCredentialProvider creates a client that comsumes application
// credentials for authentication.
func NewApplicationCredentialProvider(endpoint, id, secret string) *ApplicationCredentialProvider {
	return &ApplicationCredentialProvider{
		endpoint: endpoint,
		id:       id,
		secret:   secret,
	}
}

// Client implements the Provider interface.
func (p *ApplicationCredentialProvider) Client() (*gophercloud.ProviderClient, error) {
	options := gophercloud.AuthOptions{
		IdentityEndpoint:            p.endpoint,
		ApplicationCredentialID:     p.id,
		ApplicationCredentialSecret: p.secret,
	}

	return authenticatedClient(options)
}

// PasswordProvider allows use of an application credential.
type PasswordProvider struct {
	endpoint  string
	userID    string
	password  string
	projectID string
}

// Ensure the interface is implemented.
var _ CredentialProvider = &PasswordProvider{}

// NewPasswordProvider creates a client that comsumes passwords
// for authentication.
func NewPasswordProvider(endpoint, userID, password, projectID string) *PasswordProvider {
	return &PasswordProvider{
		endpoint:  endpoint,
		userID:    userID,
		password:  password,
		projectID: projectID,
	}
}

// Client implements the Provider interface.
func (p *PasswordProvider) Client() (*gophercloud.ProviderClient, error) {
	options := gophercloud.AuthOptions{
		IdentityEndpoint: p.endpoint,
		UserID:           p.userID,
		Password:         p.password,
		TenantID:         p.projectID,
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
var _ CredentialProvider = &TokenProvider{}

// NewTokenProvider returns a new initialized provider.
func NewTokenProvider(endpoint, token string) *TokenProvider {
	return &TokenProvider{
		endpoint: endpoint,
		token:    token,
	}
}

// Client implements the Provider interface.
func (p *TokenProvider) Client() (*gophercloud.ProviderClient, error) {
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
var _ CredentialProvider = &CloudsProvider{}

// NewTokenProvider returns a new initialized provider.
func NewCloudsProvider(cloud string) *CloudsProvider {
	return &CloudsProvider{
		cloud: cloud,
	}
}

// Client implements the Provider interface.
func (p *CloudsProvider) Client() (*gophercloud.ProviderClient, error) {
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
var _ CredentialProvider = &UnauthenticatedProvider{}

// NewTokenProvider returns a new initialized provider.
func NewUnauthenticatedProvider(endpoint string) *UnauthenticatedProvider {
	return &UnauthenticatedProvider{
		endpoint: endpoint,
	}
}

// Client implements the Provider interface.
func (p *UnauthenticatedProvider) Client() (*gophercloud.ProviderClient, error) {
	client, err := openstack.NewClient(p.endpoint)
	if err != nil {
		return nil, err
	}

	return client, nil
}
