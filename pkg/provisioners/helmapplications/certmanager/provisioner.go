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

package certmanager

import (
	"github.com/unikorn-cloud/core/pkg/provisioners/application"
)

const (
	// applicationName is the unique name of the application.
	applicationName = "cert-manager"
)

// New returns a new initialized provisioner object.
func New(getApplication application.GetterFunc) *application.Provisioner {
	// Cert manager doesn't need any special handling, ensure it's installed in the specified
	// remote and in the cert-manager namespace.
	return application.New(getApplication).InNamespace(applicationName)
}
