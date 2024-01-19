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

package jose

import (
	"crypto"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"errors"
	"fmt"

	jose "github.com/go-jose/go-jose/v3"
	"github.com/go-jose/go-jose/v3/jwt"
	"github.com/spf13/pflag"
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

const (
	tlsKeyPathDefault  = "/var/lib/secrets/unikorn.spjmurray.co.uk/jose/tls.key"
	tlsCertPathDefault = "/var/lib/secrets/unikorn.spjmurray.co.uk/jose/tls.crt"
)

type Options struct {
	// TLSKeyPath identifies where to get the JWE/JWS private key from.
	TLSKeyPath string

	// TLSCertPath identifies where to get the JWE/JWS public key from.
	TLSCertPath string
}

// AddFlags registers flags with the provided flag set.
func (o *Options) AddFlags(f *pflag.FlagSet) {
	f.StringVar(&o.TLSKeyPath, "jose-tls-key", tlsKeyPathDefault, "TLS key used to sign JWS and decrypt JWE.")
	f.StringVar(&o.TLSCertPath, "jose-tls-cert", tlsCertPathDefault, "TLS cert used to verify JWS and encrypt JWE.")
}

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
	options *Options
}

// NewJWTIssuer returns a new JWT issuer and validator.
func NewJWTIssuer(options *Options) *JWTIssuer {
	return &JWTIssuer{
		options: options,
	}
}

// GetKeyPair returns the public key, private key and key id from the configuration data.
// The key id is inspired by X.509 subject key identifiers, so a hash over the subject public
// key info.
func (i *JWTIssuer) GetKeyPair() (any, crypto.PrivateKey, string, error) {
	// See JWTIssuer documentation for notes on caching.
	tlsCertificate, err := tls.LoadX509KeyPair(i.options.TLSCertPath, i.options.TLSKeyPath)
	if err != nil {
		return nil, nil, "", err
	}

	if len(tlsCertificate.Certificate) != 1 {
		return nil, nil, "", fmt.Errorf("%w: unexpected certificate chain", ErrKeyFormat)
	}

	certificate, err := x509.ParseCertificate(tlsCertificate.Certificate[0])
	if err != nil {
		return nil, nil, "", err
	}

	if certificate.PublicKeyAlgorithm != x509.ECDSA {
		return nil, nil, "", fmt.Errorf("%w: certifcate public key algorithm is not ECDSA", ErrKeyFormat)
	}

	kid := sha256.Sum256(certificate.RawSubjectPublicKeyInfo)

	return certificate.PublicKey, tlsCertificate.PrivateKey, base64.RawURLEncoding.EncodeToString(kid[:]), nil
}

func (i *JWTIssuer) EncodeJWT(claims interface{}) (string, error) {
	_, privateKey, _, err := i.GetKeyPair()
	if err != nil {
		return "", fmt.Errorf("failed to get key pair: %w", err)
	}

	signingKey := jose.SigningKey{
		Algorithm: jose.ES512,
		Key:       privateKey,
	}

	signer, err := jose.NewSigner(signingKey, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create signer: %w", err)
	}

	return jwt.Signed(signer).Claims(claims).CompactSerialize()
}

func (i *JWTIssuer) EncodeJWEToken(claims interface{}) (string, error) {
	publicKey, privateKey, kid, err := i.GetKeyPair()
	if err != nil {
		return "", fmt.Errorf("failed to get key pair: %w", err)
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
		KeyID:     kid,
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

func (i *JWTIssuer) DecodeJWEToken(tokenString string, claims interface{}) error {
	publicKey, privateKey, _, err := i.GetKeyPair()
	if err != nil {
		return fmt.Errorf("failed to get key pair: %w", err)
	}

	// Parse and decrypt the JWE token with the private key.
	nestedToken, err := jwt.ParseSignedAndEncrypted(tokenString)
	if err != nil {
		return fmt.Errorf("failed to parse encrypted token: %w", err)
	}

	token, err := nestedToken.Decrypt(privateKey)
	if err != nil {
		return fmt.Errorf("failed to decrypt token: %w", err)
	}

	// Parse and verify the claims with the public key.
	if err := token.Claims(publicKey, claims); err != nil {
		return fmt.Errorf("failed to decrypt claims: %w", err)
	}

	return nil
}

func (i *JWTIssuer) JWKS() (*jose.JSONWebKeySet, error) {
	pub, _, kid, err := i.GetKeyPair()
	if err != nil {
		return nil, err
	}

	jwks := &jose.JSONWebKeySet{
		Keys: []jose.JSONWebKey{
			{
				Key:   pub,
				KeyID: kid,
				Use:   "sig",
			},
		},
	}

	return jwks, nil
}
