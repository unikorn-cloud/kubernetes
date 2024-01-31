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

package nvidiagpuoperator

import (
	"context"

	"github.com/unikorn-cloud/core/pkg/provisioners/application"
	"github.com/unikorn-cloud/core/pkg/provisioners/util"
)

const (
	// defaultNamespace is where to install the component.
	// NOTE: this requires the namespace to exist first, so pick an existing one.
	defaultNamespace = "kube-system"
)

// New returns a new initialized provisioner object.
func New(getApplication application.GetterFunc) *application.Provisioner {
	p := &Provisioner{}

	return application.New(getApplication).WithGenerator(p).InNamespace(defaultNamespace)
}

type Provisioner struct{}

// Ensure the Provisioner interface is implemented.
var _ application.ValuesGenerator = &Provisioner{}

// Generate implements the application.Generator interface.
func (p *Provisioner) Values(ctx context.Context, version *string) (interface{}, error) {
	// We limit images to those with the driver pre-installed as it's far quicker for UX.
	// Also the default affinity is broken and prevents scale to zero, also tolerations
	// don't allow execution using our default taints.
	// TODO: This includes the node-feature-discovery as a subchart, and doesn't expose
	// node selectors/tolerations, however, it does scale to zero.
	values := map[string]interface{}{
		"driver": map[string]interface{}{
			"enabled": false,
		},
		"operator": map[string]interface{}{
			"affinity": map[string]interface{}{
				"nodeAffinity": map[string]interface{}{
					"preferredDuringSchedulingIgnoredDuringExecution": nil,
					"requiredDuringSchedulingIgnoredDuringExecution": map[string]interface{}{
						"nodeSelectorTerms": []interface{}{
							map[string]interface{}{
								"matchExpressions": []interface{}{
									map[string]interface{}{
										"key":      "node-role.kubernetes.io/control-plane",
										"operator": "Exists",
									},
								},
							},
						},
					},
				},
			},
			"tolerations": util.ControlPlaneTolerations(),
		},
	}

	return values, nil
}
