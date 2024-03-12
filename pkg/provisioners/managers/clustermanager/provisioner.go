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

package clustermanager

import (
	"context"

	"github.com/prometheus/client_golang/prometheus"

	unikornv1core "github.com/unikorn-cloud/core/pkg/apis/unikorn/v1alpha1"
	coreclient "github.com/unikorn-cloud/core/pkg/client"
	"github.com/unikorn-cloud/core/pkg/provisioners"
	"github.com/unikorn-cloud/core/pkg/provisioners/concurrent"
	"github.com/unikorn-cloud/core/pkg/provisioners/remotecluster"
	"github.com/unikorn-cloud/core/pkg/provisioners/serial"
	unikornv1 "github.com/unikorn-cloud/unikorn/pkg/apis/unikorn/v1alpha1"
	"github.com/unikorn-cloud/unikorn/pkg/provisioners/helmapplications/certmanager"
	"github.com/unikorn-cloud/unikorn/pkg/provisioners/helmapplications/clusterapi"
	"github.com/unikorn-cloud/unikorn/pkg/provisioners/helmapplications/vcluster"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

var (
	// On home broadband it'll take about 150s to pull down images, plus any
	// readniness gates we put in the way.  If images are cached then 45s.
	//nolint:gochecknoglobals
	durationMetric = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name: "unikorn_clustermanager_provision_duration",
		Help: "Time taken for clustermanager to provision",
		Buckets: []float64{
			1, 5, 10, 15, 20, 30, 45, 60, 90, 120, 180, 240, 300,
		},
	})
)

//nolint:gochecknoinits
func init() {
	metrics.Registry.MustRegister(durationMetric)
}

type ApplicationReferenceGetter struct {
	clusterManager *unikornv1.ClusterManager
}

func newApplicationReferenceGetter(clusterManager *unikornv1.ClusterManager) *ApplicationReferenceGetter {
	return &ApplicationReferenceGetter{
		clusterManager: clusterManager,
	}
}

func (a *ApplicationReferenceGetter) getApplication(ctx context.Context, name string) (*unikornv1core.ApplicationReference, error) {
	cli := coreclient.StaticClientFromContext(ctx)

	key := client.ObjectKey{
		Name: *a.clusterManager.Spec.ApplicationBundle,
	}

	bundle := &unikornv1.ClusterManagerApplicationBundle{}

	if err := cli.Get(ctx, key, bundle); err != nil {
		return nil, err
	}

	return bundle.Spec.GetApplication(name)
}

func (a *ApplicationReferenceGetter) vCluster(ctx context.Context) (*unikornv1core.ApplicationReference, error) {
	return a.getApplication(ctx, "vcluster")
}

func (a *ApplicationReferenceGetter) certManager(ctx context.Context) (*unikornv1core.ApplicationReference, error) {
	return a.getApplication(ctx, "cert-manager")
}

func (a *ApplicationReferenceGetter) clusterAPI(ctx context.Context) (*unikornv1core.ApplicationReference, error) {
	return a.getApplication(ctx, "cluster-api")
}

// Provisioner encapsulates control plane provisioning.
type Provisioner struct {
	provisioners.Metadata

	// clusterManager is the control plane CR this deployment relates to
	clusterManager unikornv1.ClusterManager
}

// New returns a new initialized provisioner object.
func New() provisioners.ManagerProvisioner {
	return &Provisioner{}
}

// Ensure the ManagerProvisioner interface is implemented.
var _ provisioners.ManagerProvisioner = &Provisioner{}

func (p *Provisioner) Object() unikornv1core.ManagableResourceInterface {
	return &p.clusterManager
}

// getClusterManagerProvisioner returns a provisoner that encodes control plane
// provisioning steps.
func (p *Provisioner) getClusterManagerProvisioner() provisioners.Provisioner {
	apps := newApplicationReferenceGetter(&p.clusterManager)

	remoteClusterManager := remotecluster.New(vcluster.NewRemoteCluster(p.clusterManager.Namespace, p.clusterManager.Name, &p.clusterManager), true)

	clusterAPIProvisioner := concurrent.New("cluster-api",
		certmanager.New(apps.certManager),
		clusterapi.New(apps.clusterAPI),
	)

	// Set up deletion semantics.
	clusterAPIProvisioner.BackgroundDeletion()

	// Provision the vitual cluster, setup the remote cluster then
	// install cert manager and cluster API into it.
	return serial.New("control plane",
		vcluster.New(apps.vCluster, p.clusterManager.Name).InNamespace(p.clusterManager.Namespace),
		remoteClusterManager.ProvisionOn(clusterAPIProvisioner),
	)
}

// Provision implements the Provision interface.
func (p *Provisioner) Provision(ctx context.Context) error {
	log := log.FromContext(ctx)

	log.Info("provisioning control plane")

	timer := prometheus.NewTimer(durationMetric)
	defer timer.ObserveDuration()

	if err := p.getClusterManagerProvisioner().Provision(ctx); err != nil {
		return err
	}

	log.Info("control plane provisioned")

	return nil
}

// Deprovision implements the Provision interface.
func (p *Provisioner) Deprovision(ctx context.Context) error {
	// Remove the control plane.
	if err := p.getClusterManagerProvisioner().Deprovision(ctx); err != nil {
		return err
	}

	return nil
}
