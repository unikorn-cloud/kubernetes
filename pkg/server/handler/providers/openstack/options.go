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
	"github.com/spf13/pflag"

	"github.com/unikorn-cloud/unikorn/pkg/providers/openstack"
)

type Options struct {
	// Endpoint is the Keystone/discovery endpoint.
	Endpoint string

	// ServiceAccountSecret is the secret that we get our credentials from.
	// By design, this should be read each time to facilitate credential
	// rotation.
	ServiceAccountSecret string

	// ApplicationCredentialRoles sets the roles an application credential
	// is granted on creation.
	ApplicationCredentialRoles []string

	ComputeOptions openstack.ComputeOptions
	ImageOptions   openstack.ImageOptions
}

func (o *Options) AddFlags(f *pflag.FlagSet) {
	f.StringVar(&o.Endpoint, "openstack-endpoint", "https://keystone.openstack.org:5000", "Openstack discovery endpoint")
	f.StringVar(&o.ServiceAccountSecret, "openstack-serviceaccount-secret", "", "Secret containing a 'seviceaccountid' key")
	f.StringSliceVar(&o.ApplicationCredentialRoles, "openstack-identity-application-credential-roles", nil, "A role to be added to application credentials on creation.  May be specified more than once.")

	o.ComputeOptions.AddFlags(f)
	o.ImageOptions.AddFlags(f)
}
