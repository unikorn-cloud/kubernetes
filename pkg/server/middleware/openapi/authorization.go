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

package openapi

import (
	"crypto/tls"
	"crypto/x509"
	"net/http"
	"slices"
	"strings"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/spf13/pflag"

	"github.com/unikorn-cloud/identity/pkg/oauth2"
	"github.com/unikorn-cloud/unikorn/pkg/server/authorization"
	"github.com/unikorn-cloud/unikorn/pkg/server/errors"
)

// authorizationContext is passed through the middleware to propagate
// information back to the top level handler.
type authorizationContext struct {
	// err allows us to return a verbose error, unwrapped by whatever
	// the openapi validaiton is doing.
	err error

	// claims contains all claims defined in the token.
	claims oauth2.Claims
}

type Options struct {
	// issuer is used to perform OIDC discovery and verify access tokens
	// using the JWKS endpoint.
	issuer string

	// issuerCA is the root CA of the identity endpoint.
	issuerCA []byte
}

func (o *Options) AddFlags(f *pflag.FlagSet) {
	f.StringVar(&o.issuer, "oidc-issuer", "", "OIDC issuer URL to use for token validation.")
	f.BytesBase64Var(&o.issuerCA, "oidc-issuer-ca", nil, "base64 OIDC endpoint CA certificate.")
}

// Authorizer provides OpenAPI based authorization middleware.
type Authorizer struct {
	options *Options
}

// NewAuthorizer returns a new authorizer with required parameters.
func NewAuthorizer(options *Options) *Authorizer {
	return &Authorizer{
		options: options,
	}
}

// authorizeOAuth2 checks APIs that require and oauth2 bearer token.
func (a *Authorizer) authorizeOAuth2(authContext *authorizationContext, r *http.Request, scopes []string) error {
	authorizationScheme, rawToken, err := authorization.GetHTTPAuthenticationScheme(r)
	if err != nil {
		return err
	}

	if !strings.EqualFold(authorizationScheme, "bearer") {
		return errors.OAuth2InvalidRequest("authorization scheme not allowed").WithValues("scheme", authorizationScheme)
	}

	// Handle non-public CA certiifcates used in development.
	ctx := r.Context()

	if a.options.issuerCA != nil {
		certPool := x509.NewCertPool()

		if ok := certPool.AppendCertsFromPEM(a.options.issuerCA); !ok {
			return errors.OAuth2InvalidRequest("failed to parse oidc issuer CA cert")
		}

		client := &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					RootCAs:    certPool,
					MinVersion: tls.VersionTLS13,
				},
			},
		}

		ctx = oidc.ClientContext(ctx, client)
	}

	// Note: although we are talking about ID tokens, the identity service uses
	// the same data structures and algorithms for access tokens.  The raitonale here
	// is to avoid sending sensitive information in the token, and avoid using JWE as
	// we'd need to get access to the private key in order to decrypt it, which brings
	// it's own challenges.
	provider, err := oidc.NewProvider(ctx, a.options.issuer)
	if err != nil {
		return errors.OAuth2ServerError("oidc service discovery failed").WithError(err)
	}

	config := &oidc.Config{
		SkipClientIDCheck: true,
	}

	verifier := provider.Verifier(config)

	token, err := verifier.Verify(ctx, rawToken)
	if err != nil {
		return errors.OAuth2AccessDenied("access token validation failed").WithError(err)
	}

	var claims oauth2.Claims

	if err := token.Claims(&claims); err != nil {
		return errors.OAuth2ServerError("access token claims extraction failed").WithError(err)
	}

	// Check the token is authorized to do what the schema says.
	for _, scope := range scopes {
		if !slices.Contains(claims.Scope, scope) {
			return errors.OAuth2InvalidScope("token missing required scope").WithValues("scope", scope)
		}
	}

	// Set the claims in the context for use by the handlers.
	authContext.claims = claims

	return nil
}

// authorizeScheme requires the individual scheme to match.
func (a *Authorizer) authorizeScheme(ctx *authorizationContext, r *http.Request, scheme *openapi3.SecurityScheme, scopes []string) error {
	if scheme.Type == "oauth2" {
		return a.authorizeOAuth2(ctx, r, scopes)
	}

	return errors.OAuth2InvalidRequest("authorization scheme unsupported").WithValues("scheme", scheme.Type)
}
