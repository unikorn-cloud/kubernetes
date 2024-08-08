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

package cluster

import (
	"context"
	"errors"
	"fmt"

	unikornv1core "github.com/unikorn-cloud/core/pkg/apis/unikorn/v1alpha1"
	coreclient "github.com/unikorn-cloud/core/pkg/client"
	"github.com/unikorn-cloud/core/pkg/constants"
	"github.com/unikorn-cloud/core/pkg/manager"
	"github.com/unikorn-cloud/core/pkg/provisioners"
	"github.com/unikorn-cloud/core/pkg/provisioners/concurrent"
	"github.com/unikorn-cloud/core/pkg/provisioners/conditional"
	"github.com/unikorn-cloud/core/pkg/provisioners/remotecluster"
	"github.com/unikorn-cloud/core/pkg/provisioners/serial"
	"github.com/unikorn-cloud/core/pkg/util"
	unikornv1 "github.com/unikorn-cloud/kubernetes/pkg/apis/unikorn/v1alpha1"
	"github.com/unikorn-cloud/kubernetes/pkg/provisioners/helmapplications/cilium"
	"github.com/unikorn-cloud/kubernetes/pkg/provisioners/helmapplications/clusterautoscaler"
	"github.com/unikorn-cloud/kubernetes/pkg/provisioners/helmapplications/clusterautoscaleropenstack"
	"github.com/unikorn-cloud/kubernetes/pkg/provisioners/helmapplications/clusteropenstack"
	"github.com/unikorn-cloud/kubernetes/pkg/provisioners/helmapplications/metricsserver"
	"github.com/unikorn-cloud/kubernetes/pkg/provisioners/helmapplications/nvidiagpuoperator"
	"github.com/unikorn-cloud/kubernetes/pkg/provisioners/helmapplications/openstackcloudprovider"
	"github.com/unikorn-cloud/kubernetes/pkg/provisioners/helmapplications/openstackplugincindercsi"
	"github.com/unikorn-cloud/kubernetes/pkg/provisioners/helmapplications/vcluster"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	ErrClusterManager = errors.New("cluster manager lookup failed")

	ErrAnnotation = errors.New("required annotation missing")
)

type ApplicationReferenceGetter struct {
	cluster *unikornv1.KubernetesCluster
}

func newApplicationReferenceGetter(cluster *unikornv1.KubernetesCluster) *ApplicationReferenceGetter {
	return &ApplicationReferenceGetter{
		cluster: cluster,
	}
}

func (a *ApplicationReferenceGetter) getApplication(ctx context.Context, name string) (*unikornv1core.ApplicationReference, error) {
	// TODO: we could cache this, it's from a cache anyway, so quite cheap...
	cli, err := coreclient.ProvisionerClientFromContext(ctx)
	if err != nil {
		return nil, err
	}

	key := client.ObjectKey{
		Name: *a.cluster.Spec.ApplicationBundle,
	}

	bundle := &unikornv1.KubernetesClusterApplicationBundle{}

	if err := cli.Get(ctx, key, bundle); err != nil {
		return nil, err
	}

	return bundle.Spec.GetApplication(name)
}

func (a *ApplicationReferenceGetter) clusterOpenstack(ctx context.Context) (*unikornv1core.ApplicationReference, error) {
	return a.getApplication(ctx, "cluster-openstack")
}

func (a *ApplicationReferenceGetter) cilium(ctx context.Context) (*unikornv1core.ApplicationReference, error) {
	return a.getApplication(ctx, "cilium")
}

func (a *ApplicationReferenceGetter) openstackCloudProvider(ctx context.Context) (*unikornv1core.ApplicationReference, error) {
	return a.getApplication(ctx, "openstack-cloud-provider")
}

func (a *ApplicationReferenceGetter) openstackPluginCinderCSI(ctx context.Context) (*unikornv1core.ApplicationReference, error) {
	return a.getApplication(ctx, "openstack-plugin-cinder-csi")
}

func (a *ApplicationReferenceGetter) metricsServer(ctx context.Context) (*unikornv1core.ApplicationReference, error) {
	return a.getApplication(ctx, "metrics-server")
}

func (a *ApplicationReferenceGetter) nvidiaGPUOperator(ctx context.Context) (*unikornv1core.ApplicationReference, error) {
	return a.getApplication(ctx, "nvidia-gpu-operator")
}

func (a *ApplicationReferenceGetter) clusterAutoscaler(ctx context.Context) (*unikornv1core.ApplicationReference, error) {
	return a.getApplication(ctx, "cluster-autoscaler")
}

func (a *ApplicationReferenceGetter) clusterAutoscalerOpenstack(ctx context.Context) (*unikornv1core.ApplicationReference, error) {
	return a.getApplication(ctx, "cluster-autoscaler-openstack")
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

func (p *Provisioner) Object() unikornv1core.ManagableResourceInterface {
	return &p.cluster
}

// getClusterManager gets the control plane object that owns this cluster.
func (p *Provisioner) getClusterManager(ctx context.Context) (*unikornv1.ClusterManager, error) {
	cli, err := coreclient.ProvisionerClientFromContext(ctx)
	if err != nil {
		return nil, err
	}

	key := client.ObjectKey{
		Namespace: p.cluster.Namespace,
		Name:      p.cluster.Spec.ClusterManagerID,
	}

	var clusterManager unikornv1.ClusterManager

	if err := cli.Get(ctx, key, &clusterManager); err != nil {
		return nil, fmt.Errorf("%w: %s", ErrClusterManager, err.Error())
	}

	return &clusterManager, nil
}

func (p *Provisioner) getProvisioner(ctx context.Context) (provisioners.Provisioner, error) {
	apps := newApplicationReferenceGetter(&p.cluster)

	clusterManager, err := p.getClusterManager(ctx)
	if err != nil {
		return nil, err
	}

	remoteClusterManager := remotecluster.New(vcluster.NewRemoteCluster(p.cluster.Namespace, clusterManager.Name, clusterManager), false)

	clusterManagerPrefix, err := util.GetNATPrefix(ctx)
	if err != nil {
		return nil, err
	}

	remoteCluster := remotecluster.New(clusteropenstack.NewRemoteCluster(&p.cluster), true)

	clusterProvisioner := clusteropenstack.New(apps.clusterOpenstack, clusterManagerPrefix).InNamespace(p.cluster.Name)

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

	addonsProvisioner := concurrent.New("cluster add-ons",
		openstackplugincindercsi.New(apps.openstackPluginCinderCSI),
		metricsserver.New(apps.metricsServer),
		conditional.New("nvidia-gpu-operator", p.cluster.NvidiaOperatorEnabled, nvidiagpuoperator.New(apps.nvidiaGPUOperator)),
	)

	// Create the cluster and the boostrap components in parallel, the cluster will
	// come up but never reach healthy until the CNI and cloud controller manager
	// are added.  Follow that up by the autoscaler as some addons may require worker
	// nodes to schedule onto.
	provisioner := remoteClusterManager.ProvisionOn(
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

// managerReady gates cluster creation on the manager being up and ready.
// Due to https://github.com/argoproj/argo-cd/issues/18041 Argo will break
// quite spectacularly if you try to install an application when the requisite
// CRDs are not present yet.  As a result we need to provision the implicit
// manager serially, and that takes a long time.  So long the request times
// out, so we essentially have to defer cluster creation until we know the
// manager is working and Argo isn't going to fail.
func (p *Provisioner) managerReady(ctx context.Context) error {
	clusterManager, err := p.getClusterManager(ctx)
	if err != nil {
		return err
	}

	condition, err := clusterManager.StatusConditionRead(unikornv1core.ConditionAvailable)
	if err != nil {
		return err
	}

	if condition.Reason != unikornv1core.ConditionReasonProvisioned {
		return provisioners.ErrYield
	}

	return nil
}

// Provision implements the Provision interface.
func (p *Provisioner) Provision(ctx context.Context) error {
	if err := p.managerReady(ctx); err != nil {
		return err
	}

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
		if errors.Is(err, ErrClusterManager) {
			return nil
		}

		return err
	}

	if err := provisioner.Deprovision(ctx); err != nil {
		return err
	}

	// This event is used to trigger cleanup operations in the provider.
	recorder := manager.FromContext(ctx).GetEventRecorderFor("kubernetescluster")

	if len(p.cluster.Annotations) == 0 {
		return fmt.Errorf("%w: no annotations present", ErrAnnotation)
	}

	identityID, ok := p.cluster.Annotations[constants.CloudIdentityAnnotation]
	if !ok {
		return fmt.Errorf("%w: identity annotation missing", ErrAnnotation)
	}

	annotations := map[string]string{
		constants.CloudIdentityAnnotation: identityID,
	}

	recorder.AnnotatedEventf(&p.cluster, annotations, "Normal", constants.IdentityCleanupReadyEventReason, "kubetnetes cluster has been deleted successfully")

	return nil
}
