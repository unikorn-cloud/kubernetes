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

package authorization

import (
	"net/http"
	"time"

	"github.com/spjmurray/unikorn/pkg/providers/openstack"
	"github.com/spjmurray/unikorn/pkg/server/authorization/jose"
	"github.com/spjmurray/unikorn/pkg/server/authorization/keystone"
	"github.com/spjmurray/unikorn/pkg/server/authorization/oauth2"
	"github.com/spjmurray/unikorn/pkg/server/errors"
	"github.com/spjmurray/unikorn/pkg/server/generated"
)

// Authenticator provides Keystone authentication functionality.
type Authenticator struct {
	// issuer allows creation and validation of JWT bearer tokens.
	issuer *jose.JWTIssuer

	// OAuth2 is the oauth2 deletgating authenticator.
	OAuth2 *oauth2.Authenticator

	// Keystone provides OpenStack authentication.
	Keystone *keystone.Authenticator
}

// NewAuthenticator returns a new authenticator with required fields populated.
// You must call AddFlags after this.
func NewAuthenticator(issuer *jose.JWTIssuer, oauth2 *oauth2.Authenticator, keystone *keystone.Authenticator) *Authenticator {
	return &Authenticator{
		issuer:   issuer,
		OAuth2:   oauth2,
		Keystone: keystone,
	}
}

// Token performs token based authentication against Keystone with a scope, and returns a new token.
// Used to upgrade from unscoped, or to refresh a token.
func (a *Authenticator) Token(r *http.Request, scope *generated.TokenScope) (*generated.Token, error) {
	tokenClaims, err := oauth2.ClaimsFromContext(r.Context())
	if err != nil {
		return nil, errors.OAuth2ServerError("failed get claims").WithError(err)
	}

	identity, err := openstack.NewIdentityClient(openstack.NewUnauthenticatedProvider(a.Keystone.Endpoint()))
	if err != nil {
		return nil, errors.OAuth2ServerError("unable to initialize identity").WithError(err)
	}

	if tokenClaims.UnikornClaims == nil {
		return nil, errors.OAuth2ServerError("unable to get unikorn claims")
	}

	keystoneToken, user, err := identity.CreateToken(r.Context(), openstack.NewCreateTokenOptionsScopedToken(tokenClaims.UnikornClaims.Token, scope.Project.Id))
	if err != nil {
		return nil, errors.OAuth2AccessDenied("authentication failed").WithError(err)
	}

	uClaims := &oauth2.UnikornClaims{
		Token:   keystoneToken.ID,
		User:    user.ID,
		Project: scope.Project.Id,
	}

	// Add some scope to the claims to allow the token to do more.
	oAuth2Scope := &oauth2.ScopeList{
		Scopes: []oauth2.APIScope{
			oauth2.ScopeProject,
		},
	}

	accessToken, err := oauth2.Issue(a.issuer, r, tokenClaims.Subject, uClaims, oAuth2Scope, keystoneToken.ExpiresAt)
	if err != nil {
		return nil, errors.OAuth2ServerError("unable to create access token").WithError(err)
	}

	result := &generated.Token{
		TokenType:   "Bearer",
		AccessToken: accessToken,
		ExpiresIn:   int(time.Until(keystoneToken.ExpiresAt).Seconds()),
	}

	return result, nil
}

func (a *Authenticator) JWKS() (interface{}, error) {
	result, err := a.issuer.JWKS()
	if err != nil {
		return nil, errors.OAuth2ServerError("unable to generate json web key set").WithError(err)
	}

	return result, nil
}
