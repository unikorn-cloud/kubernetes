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

package kubernetesdashboard

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"

	coreclient "github.com/unikorn-cloud/core/pkg/client"
	"github.com/unikorn-cloud/core/pkg/constants"
	"github.com/unikorn-cloud/core/pkg/provisioners/application"
	"github.com/unikorn-cloud/core/pkg/provisioners/util"

	corev1 "k8s.io/api/core/v1"
)

var (
	ErrIngress           = errors.New("ingress not as expected")
	ErrIngressIPNotFound = errors.New("unable to find remote ingress IP address")
)

type Provisioner struct{}

// Ensure the Provisioner interface is implemented.
var _ application.ValuesGenerator = &Provisioner{}

// New returns a new initialized provisioner object.
func New(getApplication application.GetterFunc) *application.Provisioner {
	return application.New(getApplication).WithGenerator(&Provisioner{}).InNamespace("kube-system")
}

func (p *Provisioner) remoteIngressIP(ctx context.Context) (net.IP, error) {
	var services corev1.ServiceList

	if err := coreclient.DynamicClientFromContext(ctx).List(ctx, &services); err != nil {
		return nil, err
	}

	// TODO: we provision this (for now) as a second wave from the ingress controller
	// so we expect this to work.  If it doesn't it'll error.  Consider making this a
	// parallel task and just yielding.
	for _, service := range services.Items {
		if _, ok := service.Annotations[constants.IngressEndpointAnnotation]; !ok {
			continue
		}

		if service.Spec.Type != corev1.ServiceTypeLoadBalancer {
			return nil, fmt.Errorf("%w: incorrect service type", ErrIngress)
		}

		if len(service.Status.LoadBalancer.Ingress) != 1 {
			return nil, fmt.Errorf("%w: not provisioned yet", ErrIngress)
		}

		ip := net.ParseIP(service.Status.LoadBalancer.Ingress[0].IP)
		if ip == nil {
			return nil, fmt.Errorf("%w: not provisioned yet", ErrIngress)
		}

		return ip, nil
	}

	return nil, ErrIngressIPNotFound
}

// Generate implements the application.Generator interface.
func (p *Provisioner) Values(ctx context.Context, version *string) (interface{}, error) {
	// Now, we _should_ combine cert-manager's HTTP-01 acme challenge with external-dns
	// however, in lieu of a DDNS server, we are using IP wildcard DNS via nip.io.  Now
	// sadly to use _that_, you need to know the IP address of the ingress.  So two
	// things need to happen: first, the ingress controller gets installed first (we will do the
	// ordering here, the responsibility of opting you into that add-on is delegated to the
	// client e.g. UI).  Second we need to wait for the external IP address to get allocated.
	// At present, we just look for a LoadBalancer Service.  In furture we may need to label
	// it to discriminate.
	// TODO: read above!
	ip, err := p.remoteIngressIP(ctx)
	if err != nil {
		return nil, err
	}

	host := fmt.Sprintf("dashboard-%s.nip.io", strings.ReplaceAll(ip.String(), ".", "-"))

	values := map[string]interface{}{
		"ingress": map[string]interface{}{
			"enabled":   true,
			"className": "nginx",
			"annotations": map[string]interface{}{
				// TODO: We will need to make this production before allowing
				// customers to go wild, and we will also need some form of
				// payment when the traffic gets big enough.  For now you can
				// install the staging roots in your browser.
				"cert-manager.io/cluster-issuer": "letsencrypt-staging",
			},
			"tls": []interface{}{
				map[string]interface{}{
					"secretName": "kubernetes-dashboard-tls",
					"hosts": []interface{}{
						host,
					},
				},
			},
			"hosts": []interface{}{
				host,
			},
		},
		"tolerations":  util.ControlPlaneTolerations(),
		"nodeSelector": util.ControlPlaneNodeSelector(),
	}

	return values, nil
}
