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

package middleware

import (
	"net/http"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"

	"github.com/unikorn-cloud/unikorn/pkg/server/authorization"
	"github.com/unikorn-cloud/unikorn/pkg/server/authorization/jose"
	"github.com/unikorn-cloud/unikorn/pkg/server/authorization/oauth2"
	"github.com/unikorn-cloud/unikorn/pkg/server/errors"
)

// authorizationContext is passed through the middleware to propagate
// information back to the top level handler.
type authorizationContext struct {
	// err allows us to return a verbose error, unwrapped by whatever
	// the openapi validaiton is doing.
	err error

	// claims contains all claims defined in the token.
	claims *oauth2.Claims
}

// Authorizer provides OpenAPI based authorization middleware.
type Authorizer struct {
	// issuer allows creation and validation of JWT bearer tokens.
	issuer *jose.JWTIssuer
}

// NewAuthorizer returns a new authorizer with required parameters.
func NewAuthorizer(issuer *jose.JWTIssuer) *Authorizer {
	return &Authorizer{
		issuer: issuer,
	}
}

// authorizeOAuth2 checks APIs that require and oauth2 bearer token.
func (a *Authorizer) authorizeOAuth2(ctx *authorizationContext, r *http.Request, scopes []string) error {
	authorizationScheme, token, err := authorization.GetHTTPAuthenticationScheme(r)
	if err != nil {
		return err
	}

	if !strings.EqualFold(authorizationScheme, "bearer") {
		return errors.OAuth2InvalidRequest("authorization scheme not allowed").WithValues("scheme", authorizationScheme)
	}

	// Check the token is from us, for us, and in date.
	claims, err := oauth2.Verify(a.issuer, r, token)
	if err != nil {
		return errors.OAuth2AccessDenied("token validation failed").WithError(err)
	}

	// Check the token is authorized to do what the schema says.
	for _, scope := range scopes {
		if !claims.Scope.Includes(oauth2.APIScope(scope)) {
			return errors.OAuth2InvalidScope("token missing required scope").WithValues("scope", scope)
		}
	}

	// Set the claims in the context for use by the handlers.
	ctx.claims = claims

	return nil
}

// authorizeScheme requires the individual scheme to match.
func (a *Authorizer) authorizeScheme(ctx *authorizationContext, r *http.Request, scheme *openapi3.SecurityScheme, scopes []string) error {
	if scheme.Type == "oauth2" {
		return a.authorizeOAuth2(ctx, r, scopes)
	}

	return errors.OAuth2InvalidRequest("authorization scheme unsupported").WithValues("scheme", scheme.Type)
}
