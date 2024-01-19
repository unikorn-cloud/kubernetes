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

package oauth2

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-jose/go-jose/v3/jwt"
	"github.com/google/uuid"

	"github.com/spjmurray/unikorn/pkg/server/authorization/jose"
)

var (
	// ErrKeyFormat is raised when something is wrong with the
	// encryption keys.
	ErrKeyFormat = errors.New("key format error")

	// ErrTokenVerification is raised when token verification fails.
	ErrTokenVerification = errors.New("failed to verify token")

	// ErrContextError is raised when a required value cannot be retrieved
	// from a context.
	ErrContextError = errors.New("value missing from context")
)

// APIScope defines security context scopes for an API request.
type APIScope string

const (
	// ScopeProject tells us the claims token is project scoped.
	ScopeProject APIScope = "project"
)

// ScopeList defines a list of scopes.
type ScopeList struct {
	Scopes []APIScope
}

// Ensure the correct interfaces are implemented.
var _ json.Marshaler = &ScopeList{}
var _ json.Unmarshaler = &ScopeList{}

// Includes tells you whether a scurity requirement is fulfilled.
func (l *ScopeList) Includes(scope APIScope) bool {
	if l == nil {
		return false
	}

	for _, s := range l.Scopes {
		if s == scope {
			return true
		}
	}

	return false
}

// MarshalJSON implements json.Marshaller.
func (l *ScopeList) MarshalJSON() ([]byte, error) {
	scopes := make([]string, len(l.Scopes))

	for i := range l.Scopes {
		scopes[i] = string(l.Scopes[i])
	}

	data, err := json.Marshal(strings.Join(scopes, " "))
	if err != nil {
		return nil, err
	}

	return data, nil
}

// UnmarshalJSON implments json.Unmarshaller.
func (l *ScopeList) UnmarshalJSON(value []byte) error {
	var list string

	if err := json.Unmarshal(value, &list); err != nil {
		return err
	}

	scopes := strings.Split(list, " ")

	l.Scopes = make([]APIScope, len(scopes))

	for i := range scopes {
		l.Scopes[i] = APIScope(scopes[i])
	}

	return nil
}

// UnikornClaims are JWT claims we add to the underlying specification.
// TODO: we should bind the access token to a specific client IP (and
// validate), or something to that effect in order to mitigate impersonation
// from another source.
type UnikornClaims struct {
	// Token is the OpenStack Keystone token.
	Token string `json:"token,omitempty"`

	// Project is the Openstack/Unikorn project ID the token is scoped to.
	// This is a unique identifier for the region.
	Project string `json:"projectId,omitempty"`

	// User is the Openstack user ID, the user name is stored in the "sub" claim.
	// This effectively caches the unique user ID so we don't have to translate
	// between names in the scope of the token, and what Openstack APIs expect.
	User string `json:"userId,omitempty"`
}

// Claims is an application specific set of claims.
// TODO: this technically isn't conformant to oauth2 in that we don't specify
// the client_id claim, and there are probably others.
type Claims struct {
	jwt.Claims `json:",inline"`

	// Scope is the set of scopes for a JWT as defined by oauth2.
	// These also correspond to security requirements in the OpenAPI schema.
	Scope *ScopeList `json:"scope,omitempty"`

	// UnikornClaims are claims required by this application.
	UnikornClaims *UnikornClaims `json:"unikorn,omitempty"`
}

// contextKey defines a new context key type unique to this package.
type contextKey int

const (
	// claimsKey is used to store claims in a context.
	claimsKey contextKey = iota
)

// NewContextWithClaims injects the given claims into a new context.
func NewContextWithClaims(ctx context.Context, claims *Claims) context.Context {
	return context.WithValue(ctx, claimsKey, claims)
}

// ClaimsFromContext extracts the claims from a context.
func ClaimsFromContext(ctx context.Context) (*Claims, error) {
	value := ctx.Value(claimsKey)
	if value == nil {
		return nil, fmt.Errorf("%w: unable to find claims", ErrContextError)
	}

	claims, ok := value.(*Claims)
	if !ok {
		return nil, fmt.Errorf("%w: unable to assert claims", ErrContextError)
	}

	return claims, nil
}

// Issue issues a new JWT token.
func Issue(i *jose.JWTIssuer, r *http.Request, subject string, uclaims *UnikornClaims, scope *ScopeList, expiresAt time.Time) (string, error) {
	now := time.Now()

	nowRFC7519 := jwt.NumericDate(now.Unix())
	expiresAtRFC7519 := jwt.NumericDate(expiresAt.Unix())

	// The issuer and audience will be the same, and dyanmic based on the
	// HTTP 1.1 Host header.
	claims := &Claims{
		Claims: jwt.Claims{
			ID:      uuid.New().String(),
			Subject: subject,
			Audience: jwt.Audience{
				r.Host,
			},
			Issuer:    r.Host,
			IssuedAt:  &nowRFC7519,
			NotBefore: &nowRFC7519,
			Expiry:    &expiresAtRFC7519,
		},
		Scope:         scope,
		UnikornClaims: uclaims,
	}

	token, err := i.EncodeJWEToken(claims)
	if err != nil {
		return "", err
	}

	return token, nil
}

// Verify checks the token parses and validates.
func Verify(i *jose.JWTIssuer, r *http.Request, tokenString string) (*Claims, error) {
	// Parse and verify the claims with the public key.
	claims := &Claims{}

	if err := i.DecodeJWEToken(tokenString, claims); err != nil {
		return nil, fmt.Errorf("failed to decrypt claims: %w", err)
	}

	// Verify the claims.
	expected := jwt.Expected{
		Audience: jwt.Audience{
			r.Host,
		},
		Issuer: r.Host,
		Time:   time.Now(),
	}

	if err := claims.Claims.Validate(expected); err != nil {
		return nil, fmt.Errorf("failed to validate claims: %w", err)
	}

	return claims, nil
}
