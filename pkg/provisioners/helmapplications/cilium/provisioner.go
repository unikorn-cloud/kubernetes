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

package cilium

import (
	"context"

	"github.com/unikorn-cloud/core/pkg/provisioners/application"
	"github.com/unikorn-cloud/core/pkg/provisioners/util"
	unikornv1 "github.com/unikorn-cloud/kubernetes/pkg/apis/unikorn/v1alpha1"
)

// New returns a new initialized provisioner object.
func New(getApplication application.GetterFunc) *application.Provisioner {
	provisioner := &Provisioner{}

	return application.New(getApplication).WithGenerator(provisioner).InNamespace("kube-system")
}

type Provisioner struct{}

// Ensure the Provisioner interface is implemented.
var _ application.ValuesGenerator = &Provisioner{}

func (p *Provisioner) Values(ctx context.Context, _ *string) (interface{}, error) {
	//nolint:forcetypeassert
	cluster := application.FromContext(ctx).(*unikornv1.KubernetesCluster)

	// Scale to zero support.
	operatorValues := map[string]interface{}{
		"nodeSelector": util.ControlPlaneNodeSelector(),
	}

	// If the cluster CP has one node, then this will fail to deploy
	// as cilium has 2 as the default, we need to work some magic here.
	if *cluster.Spec.ControlPlane.Replicas == 1 {
		operatorValues["replicas"] = cluster.Spec.ControlPlane.Replicas
	}

	values := map[string]interface{}{
		"operator": operatorValues,
		"hubble": map[string]interface{}{
			"relay": map[string]interface{}{
				"nodeSelector": util.ControlPlaneNodeSelector(),
				"tolerations":  util.ControlPlaneTolerations(),
			},
			"ui": map[string]interface{}{
				"nodeSelector": util.ControlPlaneNodeSelector(),
				"tolerations":  util.ControlPlaneTolerations(),
			},
		},
	}

	return values, nil
}
