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
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"

	"github.com/spf13/pflag"

	"github.com/unikorn-cloud/unikorn/pkg/providers/openstack"
)

var (
	// ErrPEMDecode is raised when the PEM decode failed for some reason.
	ErrPEMDecode = errors.New("PEM decode error")

	// ErrPEMType is raised when the encounter the wrong PEM type, e.g. PKCS#1.
	ErrPEMType = errors.New("PEM type unsupported")

	// ErrKeyType is raised when we encounter an unsupported key type.
	ErrKeyType = errors.New("key type unsupported")
)

// PublicKeyVar contains a public key.
type PublicKeyVar struct {
	key *ecdsa.PublicKey
}

// Set accepts a base64 encoded PEM public key and tries to decode it.
func (v *PublicKeyVar) Set(s string) error {
	if s == "" {
		return nil
	}

	pemString, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return err
	}

	pemBlock, _ := pem.Decode(pemString)
	if pemBlock == nil {
		return ErrPEMDecode
	}

	if pemBlock.Type != "PUBLIC KEY" {
		return fmt.Errorf("%w: %s", ErrPEMType, pemBlock.Type)
	}

	key, err := x509.ParsePKIXPublicKey(pemBlock.Bytes)
	if err != nil {
		return err
	}

	ecKey, ok := key.(*ecdsa.PublicKey)
	if !ok {
		return ErrKeyType
	}

	v.key = ecKey

	return nil
}

func (v *PublicKeyVar) String() string {
	return ""
}

func (v *PublicKeyVar) Type() string {
	return "publickey"
}

type Options struct {
	ComputeOptions    openstack.ComputeOptions
	Key               PublicKeyVar
	ServerGroupPolicy string
	Properties        []string
	// applicationCredentialRoles sets the roles an application credential
	// is granted on creation.
	ApplicationCredentialRoles []string
}

func (o *Options) AddFlags(f *pflag.FlagSet) {
	o.ComputeOptions.AddFlags(f)
	f.Var(&o.Key, "image-signing-key", "Key used to verify valid images for use with the platform")
	f.StringSliceVar(&o.Properties, "image-properties", nil, "Properties used to filter the list of images")
	f.StringVar(&o.ServerGroupPolicy, "server-group-policy", "soft-anti-affinity", "Scheduling policy to use for server groups")
	f.StringSliceVar(&o.ApplicationCredentialRoles, "application-credential-roles", nil, "A role to be added to application credentials on creation.  May be specified more than once.")
}
