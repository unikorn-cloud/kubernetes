/*
Copyright 2022-2024 EscherCloud.

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

package keystone

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/gophercloud/gophercloud/openstack/identity/v3/tokens"
	"github.com/gophercloud/gophercloud/openstack/identity/v3/users"
	"github.com/spf13/pflag"

	"github.com/unikorn-cloud/unikorn/pkg/providers/openstack"
)

var (
	ErrTokenExchange = errors.New("keystone token exchange failed")
)

type Options struct {
	// Endpoint is the Keystone Endpoint.
	Endpoint string

	// Domain is the default domain users live under.
	Domain string

	// keystoneFederationTokenEndpoint is where we can exchange an OpenID
	// id_token for a Keystone API token.
	keystoneFederationTokenEndpoint string
}

// AddFlags to the specified flagset.
func (o *Options) AddFlags(f *pflag.FlagSet) {
	f.StringVar(&o.Endpoint, "keystone-endpoint", "https://nl1.eschercloud.com:5000", "Keystone endpoint to use for authn/authz.")
	f.StringVar(&o.keystoneFederationTokenEndpoint, "keystone-federation-token-endpoint", "https://nl1.eschercloud.com:5000/v3/OS-FEDERATION/identity_providers/onelogindev/protocols/openid/auth", "Where we can exchange an OpenID identity for an API token.")
	f.StringVar(&o.Domain, "keystone-user-domain-name", "Default", "Keystone user domain name for password authentication.")
}

// Authenticator provides Keystone authentication functionality.
type Authenticator struct {
	options *Options
}

// New returns a new authenticator with required fields populated.
// You must call AddFlags after this.
func New(options *Options) *Authenticator {
	return &Authenticator{
		options: options,
	}
}

// Endpoint returns the endpoint host.
func (a *Authenticator) Endpoint() string {
	return a.options.Endpoint
}

// Domain returns the default user domain.
// TODO: It stands to reason that the user should supply this in future.
func (a *Authenticator) Domain() string {
	return a.options.Domain
}

// OIDCTokenExchangeResult is what's returned by Keystone when we give it an OIDC
// token to exchnage for an OpenStack token.
//
//nolint:tagliatelle
type OIDCTokenExchangeResult struct {
	Token *struct {
		// Methods will list the authentication methods e.g. openid.
		Methods []string `json:"methods"`
		// User contains metadata about the mapped user.
		User *struct {
			// Domain contains metadata about the user's domain.
			Domain *struct {
				// ID is the globally unique domain ID.
				ID string `json:"id"`
				// Name is a human readable domain name within the scope
				// of its parent domain.
				Name string `json:"name"`
			} `json:"domain"`
			// ID is the globally unique user ID.
			ID string `json:"id"`
			// Name is the human readable user name that the mapping extracts
			// from the OIDC ID token.  This is not guaranteed to be an email
			// address.
			Name string `json:"name"`
			// Federation contains metadata from the federation engine.
			Federation *struct {
				// Groups lists the groups mapped from the claims in the
				// OIDC ID token.
				Groups []struct {
					// ID is the globally unique group ID.
					ID string `json:"id"`
				} `json:"groups"`
				// IdentityProvider contains metadata about the IdP.
				IdentityProvider *struct {
					// ID is the globally unique ID of the IdP.
					ID string `json:"id"`
				} `json:"identity_provider"`
				// Protocol contains metadata about the protocol.
				Protocol *struct {
					// ID is the globally unique ID of the protocol.
					ID string `json:"id"`
				} `json:"protocol"`
			} `json:"OS-FEDERATION"`
		} `json:"user"`
		// AuditIDs provide tracing for the user.
		AuditIDs []string `json:"audit_ids"`
		// ExpiresAt indicates the time the token will cease to work.
		ExpiresAt time.Time `json:"expires_at"`
		// IssuedAt indicates the time the token was issued.
		IssuedAt time.Time `json:"issued_at"`
	} `json:"token"`
}

// OIDCTokenExchange sends the OIDC ID token to keystone, which will then
// map that to a shadow user and group, and return an unscoped API token.
func (a *Authenticator) OIDCTokenExchange(ctx context.Context, token string) (string, *OIDCTokenExchangeResult, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, a.options.keystoneFederationTokenEndpoint, nil)
	if err != nil {
		return "", nil, err
	}

	req.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{}

	resp, err := client.Do(req)
	if err != nil {
		return "", nil, err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return "", nil, fmt.Errorf("%w: unexpected status code", ErrTokenExchange)
	}

	subjectToken := resp.Header.Get("X-Subject-Token")

	if subjectToken == "" {
		// TODO: error extraction...
		return "", nil, fmt.Errorf("%w: subject token not in header", ErrTokenExchange)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", nil, err
	}

	tokenMeta := &OIDCTokenExchangeResult{}

	if err := json.Unmarshal(body, tokenMeta); err != nil {
		return "", nil, err
	}

	return subjectToken, tokenMeta, nil
}

// Basic does basic authentication, please rethink your life before using this.
func (a *Authenticator) Basic(ctx context.Context, username, password string) (*tokens.Token, *tokens.User, error) {
	identity, err := openstack.NewIdentityClient(openstack.NewUnauthenticatedProvider(a.options.Endpoint))
	if err != nil {
		return nil, nil, err
	}

	token, user, err := identity.CreateToken(ctx, openstack.NewCreateTokenOptionsUnscopedPassword(a.options.Domain, username, password))
	if err != nil {
		return nil, nil, err
	}

	return token, user, nil
}

// GetUser returns user details.
func (a *Authenticator) GetUser(ctx context.Context, token, userID string) (*users.User, error) {
	identity, err := openstack.NewIdentityClient(openstack.NewTokenProvider(a.options.Endpoint, token))
	if err != nil {
		return nil, err
	}

	return identity.GetUser(ctx, userID)
}
