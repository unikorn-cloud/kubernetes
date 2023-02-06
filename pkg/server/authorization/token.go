/*
Copyright 2022 EscherCloud.

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
	"crypto"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	jose "gopkg.in/go-jose/go-jose.v2"
	"gopkg.in/go-jose/go-jose.v2/jwt"

	"github.com/eschercloudai/unikorn/pkg/util/flags"
)

var (
	// ErrKeyFormat is raised when something is wrong with the
	// encryption keys.
	ErrKeyFormat = errors.New("key format error")

	// ErrTokenVerification is raised when token verification fails.
	ErrTokenVerification = errors.New("failed to verify token")
)

// JWTIssuer is in charge of API token issue and verification.
// It is expected that the keys come from a mounted kubernetes.io/tls
// secret, and that is managed by cert-manager.  As a result the keys
// will rotate every 60 days (by default), so you MUST ensure they are
// not cached in perpetuity.  Additionally, due to horizontal scale-out
// these secrets need to be shared between all replicas so that a token
// issued by one, can be verified by another.  As such if you ever do
// cache the certificate load, it will need to be coordinated between
// all instances.
type JWTIssuer struct {
	// tLSKeyPath identifies where to get the JWE/JWS private key from.
	tLSKeyPath string

	// tLSCertPath identifies where to get the JWE/JWS public key from.
	tLSCertPath string

	// duration allows the token lifetime to be capped.
	duration flags.DurationFlag
}

// NewJWTIssuer returns a new JWT issuer and validator.
func NewJWTIssuer() *JWTIssuer {
	return &JWTIssuer{
		duration: flags.DurationFlag{
			Duration: time.Hour,
		},
	}
}

// AddFlags registers flags with the provided flag set.
func (i *JWTIssuer) AddFlags(f *flag.FlagSet) {
	f.StringVar(&i.tLSKeyPath, "jose-tls-key", "", "TLS key used to sign JWS and decrypt JWE.")
	f.StringVar(&i.tLSCertPath, "jose-tls-cert", "", "TLS cert used to verify JWS and encrypt JWE.")
	f.Var(&i.duration, "token-expiry-duration", "JWT expiry duration")
}

// GetKeyPair returns the public and private key from the configuration data.
func (i *JWTIssuer) GetKeyPair() (any, crypto.PrivateKey, error) {
	// See JWTIssuer documentation for notes on caching.
	tlsCertificate, err := tls.LoadX509KeyPair(i.tLSCertPath, i.tLSKeyPath)
	if err != nil {
		return nil, nil, err
	}

	if len(tlsCertificate.Certificate) != 1 {
		return nil, nil, fmt.Errorf("%w: unexpected certificate chain", ErrKeyFormat)
	}

	certificate, err := x509.ParseCertificate(tlsCertificate.Certificate[0])
	if err != nil {
		return nil, nil, err
	}

	if certificate.PublicKeyAlgorithm != x509.ECDSA {
		return nil, nil, fmt.Errorf("%w: certifcate public key algorithm is not ECDSA", ErrKeyFormat)
	}

	return certificate.PublicKey, tlsCertificate.PrivateKey, nil
}

// Scope defines security context scopes for an API request.
type Scope string

const (
	// ScopeProject tells us the claims token is project scoped.
	ScopeProject Scope = "project"
)

// ScopeList defines a list of scopes.
type ScopeList struct {
	Scopes []Scope
}

// Ensure the correct interfaces are implemented.
var _ json.Marshaler = &ScopeList{}
var _ json.Unmarshaler = &ScopeList{}

// Includes tells you whether a scurity requirement is fulfilled.
func (l *ScopeList) Includes(scope Scope) bool {
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

	l.Scopes = make([]Scope, len(scopes))

	for i := range scopes {
		l.Scopes[i] = Scope(scopes[i])
	}

	return nil
}

// Claims is an application specific set of claims.
// TODO: this technically isn't conformant to oauth2 in that we don't specify
// the client_id claim, and there are probably others.
type Claims struct {
	jwt.Claims `json:",inline"`

	// Scope is the set of scopes for a JWT as defined by oauth2.
	// These also corrrespond to security requirements in the OpenAPI schema.
	Scope *ScopeList `json:"scope,omitempty"`

	// Token is the OpenStack Keystone token.
	Token string `json:"unikorn:token:keystone,omitempty"`
}

// Issue issues a new JWT token.
func (i *JWTIssuer) Issue(r *http.Request, subject string, keystoneToken string, scope *ScopeList, expiresAt time.Time) (string, error) {
	publicKey, privateKey, err := i.GetKeyPair()
	if err != nil {
		return "", fmt.Errorf("failed to get key pair: %w", err)
	}

	now := time.Now()

	// Override the default token expiration time if it exceeds our
	// security requirements.
	maxExpiresAt := now.Add(i.duration.Duration)

	if expiresAt.After(maxExpiresAt) {
		expiresAt = maxExpiresAt
	}

	nowRFC7519 := jwt.NumericDate(now.Unix())
	expiresAtRFC7519 := jwt.NumericDate(expiresAt.Unix())

	// The issuer and audience will be the same, and dyanmic based on the
	// HTTP 1.1 Host header.
	claims := Claims{
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
		Scope: scope,
		Token: keystoneToken,
	}

	signingKey := jose.SigningKey{
		Algorithm: jose.ES512,
		Key:       privateKey,
	}

	signer, err := jose.NewSigner(signingKey, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create signer: %w", err)
	}

	recipient := jose.Recipient{
		Algorithm: jose.ECDH_ES,
		Key:       publicKey,
	}

	encrypterOptions := &jose.EncrypterOptions{}
	encrypterOptions = encrypterOptions.WithType("JWT").WithContentType("JWT")

	encrypter, err := jose.NewEncrypter(jose.A256GCM, recipient, encrypterOptions)
	if err != nil {
		return "", fmt.Errorf("failed to create encrypter: %w", err)
	}

	token, err := jwt.SignedAndEncrypted(signer, encrypter).Claims(claims).CompactSerialize()
	if err != nil {
		return "", fmt.Errorf("failed to create token: %w", err)
	}

	return token, nil
}

// Verify checks the token parses and validates.
func (i *JWTIssuer) Verify(r *http.Request, tokenString string) (*Claims, error) {
	publicKey, privateKey, err := i.GetKeyPair()
	if err != nil {
		return nil, fmt.Errorf("failed to get key pair: %w", err)
	}

	// Parse and decrypt the JWE token with the private key.
	nestedToken, err := jwt.ParseSignedAndEncrypted(tokenString)
	if err != nil {
		return nil, fmt.Errorf("failed to parse encrypted token: %w", err)
	}

	token, err := nestedToken.Decrypt(privateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt token: %w", err)
	}

	// Parse and verify the claims with the public key.
	claims := &Claims{}

	if err := token.Claims(publicKey, claims); err != nil {
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