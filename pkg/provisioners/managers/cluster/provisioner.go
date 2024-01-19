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

package cluster

import (
	"context"

	unikornv1 "github.com/spjmurray/unikorn/pkg/apis/unikorn/v1alpha1"
	"github.com/spjmurray/unikorn/pkg/provisioners/helmapplications/certmanager"
	"github.com/spjmurray/unikorn/pkg/provisioners/helmapplications/certmanagerissuers"
	"github.com/spjmurray/unikorn/pkg/provisioners/helmapplications/cilium"
	"github.com/spjmurray/unikorn/pkg/provisioners/helmapplications/clusterautoscaler"
	"github.com/spjmurray/unikorn/pkg/provisioners/helmapplications/clusterautoscaleropenstack"
	"github.com/spjmurray/unikorn/pkg/provisioners/helmapplications/clusteropenstack"
	"github.com/spjmurray/unikorn/pkg/provisioners/helmapplications/ingressnginx"
	"github.com/spjmurray/unikorn/pkg/provisioners/helmapplications/kubernetesdashboard"
	"github.com/spjmurray/unikorn/pkg/provisioners/helmapplications/longhorn"
	"github.com/spjmurray/unikorn/pkg/provisioners/helmapplications/metricsserver"
	"github.com/spjmurray/unikorn/pkg/provisioners/helmapplications/nvidiagpuoperator"
	"github.com/spjmurray/unikorn/pkg/provisioners/helmapplications/openstackcloudprovider"
	"github.com/spjmurray/unikorn/pkg/provisioners/helmapplications/openstackplugincindercsi"
	"github.com/spjmurray/unikorn/pkg/provisioners/helmapplications/prometheus"
	"github.com/spjmurray/unikorn/pkg/provisioners/helmapplications/vcluster"

	coreunikornv1 "github.com/spjmurray/unikorn-core/pkg/apis/unikorn/v1alpha1"
	coreclient "github.com/spjmurray/unikorn-core/pkg/client"
	"github.com/spjmurray/unikorn-core/pkg/constants"
	"github.com/spjmurray/unikorn-core/pkg/provisioners"
	"github.com/spjmurray/unikorn-core/pkg/provisioners/concurrent"
	"github.com/spjmurray/unikorn-core/pkg/provisioners/conditional"
	"github.com/spjmurray/unikorn-core/pkg/provisioners/remotecluster"
	"github.com/spjmurray/unikorn-core/pkg/provisioners/serial"
	provisionersutil "github.com/spjmurray/unikorn-core/pkg/provisioners/util"
	"github.com/spjmurray/unikorn-core/pkg/util"

	"k8s.io/apimachinery/pkg/labels"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ApplicationReferenceGetter struct {
	cluster *unikornv1.KubernetesCluster
}

func newApplicationReferenceGetter(cluster *unikornv1.KubernetesCluster) *ApplicationReferenceGetter {
	return &ApplicationReferenceGetter{
		cluster: cluster,
	}
}

func (a *ApplicationReferenceGetter) getApplication(ctx context.Context, name string) (*coreunikornv1.ApplicationReference, error) {
	// TODO: we could cache this, it's from a cache anyway, so quite cheap...
	cli := coreclient.StaticClientFromContext(ctx)

	key := client.ObjectKey{
		Name: *a.cluster.Spec.ApplicationBundle,
	}

	bundle := &unikornv1.KubernetesClusterApplicationBundle{}

	if err := cli.Get(ctx, key, bundle); err != nil {
		return nil, err
	}

	return bundle.Spec.GetApplication(name)
}

func (a *ApplicationReferenceGetter) certManager(ctx context.Context) (*coreunikornv1.ApplicationReference, error) {
	return a.getApplication(ctx, "cert-manager")
}

func (a *ApplicationReferenceGetter) certManagerIssuers(ctx context.Context) (*coreunikornv1.ApplicationReference, error) {
	return a.getApplication(ctx, "cert-manager-issuers")
}

func (a *ApplicationReferenceGetter) clusterOpenstack(ctx context.Context) (*coreunikornv1.ApplicationReference, error) {
	return a.getApplication(ctx, "cluster-openstack")
}

func (a *ApplicationReferenceGetter) cilium(ctx context.Context) (*coreunikornv1.ApplicationReference, error) {
	return a.getApplication(ctx, "cilium")
}

func (a *ApplicationReferenceGetter) openstackCloudProvider(ctx context.Context) (*coreunikornv1.ApplicationReference, error) {
	return a.getApplication(ctx, "openstack-cloud-provider")
}

func (a *ApplicationReferenceGetter) openstackPluginCinderCSI(ctx context.Context) (*coreunikornv1.ApplicationReference, error) {
	return a.getApplication(ctx, "openstack-plugin-cinder-csi")
}

func (a *ApplicationReferenceGetter) metricsServer(ctx context.Context) (*coreunikornv1.ApplicationReference, error) {
	return a.getApplication(ctx, "metrics-server")
}

func (a *ApplicationReferenceGetter) nvidiaGPUOperator(ctx context.Context) (*coreunikornv1.ApplicationReference, error) {
	return a.getApplication(ctx, "nvidia-gpu-operator")
}

func (a *ApplicationReferenceGetter) clusterAutoscaler(ctx context.Context) (*coreunikornv1.ApplicationReference, error) {
	return a.getApplication(ctx, "cluster-autoscaler")
}

func (a *ApplicationReferenceGetter) clusterAutoscalerOpenstack(ctx context.Context) (*coreunikornv1.ApplicationReference, error) {
	return a.getApplication(ctx, "cluster-autoscaler-openstack")
}

func (a *ApplicationReferenceGetter) ingressNginx(ctx context.Context) (*coreunikornv1.ApplicationReference, error) {
	return a.getApplication(ctx, "ingress-nginx")
}

func (a *ApplicationReferenceGetter) kubernetesDashboard(ctx context.Context) (*coreunikornv1.ApplicationReference, error) {
	return a.getApplication(ctx, "kubernetes-dashboard")
}

func (a *ApplicationReferenceGetter) longhorn(ctx context.Context) (*coreunikornv1.ApplicationReference, error) {
	return a.getApplication(ctx, "longhorn")
}

func (a *ApplicationReferenceGetter) prometheus(ctx context.Context) (*coreunikornv1.ApplicationReference, error) {
	return a.getApplication(ctx, "prometheus")
}

// Provisioner encapsulates control plane provisioning.
type Provisioner struct {
	provisioners.Metadata

	// cluster is the Kubernetes cluster we're provisioning.
	cluster unikornv1.KubernetesCluster
}

// New returns a new initialized provisioner object.
func New() provisioners.ManagerProvisioner {
	return &Provisioner{}
}

// Ensure the ManagerProvisioner interface is implemented.
var _ provisioners.ManagerProvisioner = &Provisioner{}

func (p *Provisioner) Object() coreunikornv1.ManagableResourceInterface {
	return &p.cluster
}

// getControlPlane gets the control plane object that owns this cluster.
func (p *Provisioner) getControlPlane(ctx context.Context) (*unikornv1.ControlPlane, error) {
	// TODO: error checking.
	projectLabels := labels.Set{
		constants.KindLabel:    constants.KindLabelValueProject,
		constants.ProjectLabel: p.cluster.Labels[constants.ProjectLabel],
	}

	projectNamespace, err := provisionersutil.GetResourceNamespace(ctx, projectLabels)
	if err != nil {
		return nil, err
	}

	var controlPlane unikornv1.ControlPlane

	key := client.ObjectKey{
		Namespace: projectNamespace.Name,
		Name:      p.cluster.Labels[constants.ControlPlaneLabel],
	}

	if err := coreclient.StaticClientFromContext(ctx).Get(ctx, key, &controlPlane); err != nil {
		return nil, err
	}

	return &controlPlane, nil
}

func (p *Provisioner) getProvisioner(ctx context.Context) (provisioners.Provisioner, error) {
	apps := newApplicationReferenceGetter(&p.cluster)

	controlPlane, err := p.getControlPlane(ctx)
	if err != nil {
		return nil, err
	}

	remoteControlPlane := remotecluster.New(vcluster.NewRemoteCluster(p.cluster.Namespace, controlPlane), false)

	controlPlanePrefix, err := util.GetNATPrefix(ctx)
	if err != nil {
		return nil, err
	}

	remoteCluster := remotecluster.New(clusteropenstack.NewRemoteCluster(&p.cluster), true)

	clusterProvisioner := clusteropenstack.New(apps.clusterOpenstack, controlPlanePrefix).InNamespace(p.cluster.Name)

	// These applications are required to get the cluster up and running, they must
	// tolerate control plane taints, be scheduled onto control plane nodes and allow
	// scale from zero.
	bootstrapProvisioner := concurrent.New("cluster bootstrap",
		cilium.New(apps.cilium),
		openstackcloudprovider.New(apps.openstackCloudProvider),
	)

	clusterAutoscalerProvisioner := conditional.New("cluster-autoscaler",
		p.cluster.AutoscalingEnabled,
		concurrent.New("cluster-autoscaler",
			clusterautoscaler.New(apps.clusterAutoscaler).InNamespace(p.cluster.Name),
			clusterautoscaleropenstack.New(apps.clusterAutoscalerOpenstack).InNamespace(p.cluster.Name),
		),
	)

	certManagerProvisioner := serial.New("cert-manager",
		certmanager.New(apps.certManager),
		certmanagerissuers.New(apps.certManagerIssuers),
	)

	addonsProvisioner := serial.New("cluster add-ons",
		concurrent.New("cluster add-ons wave 1",
			openstackplugincindercsi.New(apps.openstackPluginCinderCSI),
			metricsserver.New(apps.metricsServer),
			conditional.New("nvidia-gpu-operator", p.cluster.NvidiaOperatorEnabled, nvidiagpuoperator.New(apps.nvidiaGPUOperator)),
			conditional.New("ingress-nginx", p.cluster.IngressEnabled, ingressnginx.New(apps.ingressNginx)),
			conditional.New("cert-manager", p.cluster.CertManagerEnabled, certManagerProvisioner),
			conditional.New("longhorn", p.cluster.FileStorageEnabled, longhorn.New(apps.longhorn)),
			conditional.New("prometheus", p.cluster.PrometheusEnabled, prometheus.New(apps.prometheus)),
		),
		concurrent.New("cluster add-ons wave 2",
			// TODO: this hack where it needs the remote is pretty ugly.
			conditional.New("kubernetes-dashboard", p.cluster.KubernetesDashboardEnabled, kubernetesdashboard.New(apps.kubernetesDashboard)),
		),
	)

	// Create the cluster and the boostrap components in parallel, the cluster will
	// come up but never reach healthy until the CNI and cloud controller manager
	// are added.  Follow that up by the autoscaler as some addons may require worker
	// nodes to schedule onto.
	provisioner := remoteControlPlane.ProvisionOn(
		serial.New("kubernetes cluster",
			concurrent.New("kubernetes cluster",
				clusterProvisioner,
				remoteCluster.ProvisionOn(bootstrapProvisioner, remotecluster.BackgroundDeletion),
			),
			clusterAutoscalerProvisioner,
			remoteCluster.ProvisionOn(addonsProvisioner, remotecluster.BackgroundDeletion),
		),
	)

	return provisioner, nil
}

// Provision implements the Provision interface.
func (p *Provisioner) Provision(ctx context.Context) error {
	provisioner, err := p.getProvisioner(ctx)
	if err != nil {
		return err
	}

	if err := provisioner.Provision(ctx); err != nil {
		return err
	}

	return nil
}

// Deprovision implements the Provision interface.
func (p *Provisioner) Deprovision(ctx context.Context) error {
	provisioner, err := p.getProvisioner(ctx)
	if err != nil {
		return err
	}

	if err := provisioner.Deprovision(ctx); err != nil {
		return err
	}

	return nil
}
