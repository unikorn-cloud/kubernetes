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
	"net/http"

	"github.com/spf13/pflag"

	unikornv1core "github.com/unikorn-cloud/core/pkg/apis/unikorn/v1alpha1"
	coreclient "github.com/unikorn-cloud/core/pkg/client"
	coreconstants "github.com/unikorn-cloud/core/pkg/constants"
	coreerrors "github.com/unikorn-cloud/core/pkg/errors"
	"github.com/unikorn-cloud/core/pkg/manager"
	coreapi "github.com/unikorn-cloud/core/pkg/openapi"
	"github.com/unikorn-cloud/core/pkg/provisioners"
	"github.com/unikorn-cloud/core/pkg/provisioners/concurrent"
	"github.com/unikorn-cloud/core/pkg/provisioners/conditional"
	"github.com/unikorn-cloud/core/pkg/provisioners/remotecluster"
	"github.com/unikorn-cloud/core/pkg/provisioners/serial"
	"github.com/unikorn-cloud/core/pkg/util"
	identityclient "github.com/unikorn-cloud/identity/pkg/client"
	unikornv1 "github.com/unikorn-cloud/kubernetes/pkg/apis/unikorn/v1alpha1"
	"github.com/unikorn-cloud/kubernetes/pkg/constants"
	kubernetesprovisioners "github.com/unikorn-cloud/kubernetes/pkg/provisioners"
	"github.com/unikorn-cloud/kubernetes/pkg/provisioners/helmapplications/cilium"
	"github.com/unikorn-cloud/kubernetes/pkg/provisioners/helmapplications/clusterautoscaler"
	"github.com/unikorn-cloud/kubernetes/pkg/provisioners/helmapplications/clusterautoscaleropenstack"
	"github.com/unikorn-cloud/kubernetes/pkg/provisioners/helmapplications/clusteropenstack"
	"github.com/unikorn-cloud/kubernetes/pkg/provisioners/helmapplications/metricsserver"
	"github.com/unikorn-cloud/kubernetes/pkg/provisioners/helmapplications/nvidiagpuoperator"
	"github.com/unikorn-cloud/kubernetes/pkg/provisioners/helmapplications/openstackcloudprovider"
	"github.com/unikorn-cloud/kubernetes/pkg/provisioners/helmapplications/openstackplugincindercsi"
	"github.com/unikorn-cloud/kubernetes/pkg/provisioners/helmapplications/vcluster"
	regionclient "github.com/unikorn-cloud/region/pkg/client"
	regionapi "github.com/unikorn-cloud/region/pkg/openapi"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

var (
	ErrClusterManager = errors.New("cluster manager lookup failed")

	ErrAnnotation = errors.New("required annotation missing")

	ErrResourceDependency = errors.New("resource deplendedncy error")
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

// Options allows access to CLI options in the provisioner.
type Options struct {
	// identityOptions allow the identity host and CA to be set.
	identityOptions *identityclient.Options
	// regionOptions allows the region host and CA to be set.
	regionOptions *regionclient.Options
	// clientOptions give access to client certificate information as
	// we need to talk to identity to get a token, and then to region
	// to ensure cloud identities and networks are provisioned, as well
	// as deptovisioning them.
	clientOptions coreclient.HTTPClientOptions
}

func (o *Options) AddFlags(f *pflag.FlagSet) {
	if o.identityOptions == nil {
		o.identityOptions = identityclient.NewOptions()
	}

	if o.regionOptions == nil {
		o.regionOptions = regionclient.NewOptions()
	}

	o.identityOptions.AddFlags(f)
	o.regionOptions.AddFlags(f)
	o.clientOptions.AddFlags(f)
}

// Provisioner encapsulates control plane provisioning.
type Provisioner struct {
	provisioners.Metadata

	// cluster is the Kubernetes cluster we're provisioning.
	cluster unikornv1.KubernetesCluster

	// options are documented for the type.
	options *Options
}

// New returns a new initialized provisioner object.
func New(options manager.ControllerOptions) provisioners.ManagerProvisioner {
	o, _ := options.(*Options)

	return &Provisioner{
		options: o,
	}
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

func (p *Provisioner) getProvisioner(ctx context.Context, options *kubernetesprovisioners.ClusterOpenstackOptions) (provisioners.Provisioner, error) {
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

	clusterProvisioner := clusteropenstack.New(apps.clusterOpenstack, options, clusterManagerPrefix).InNamespace(p.cluster.Name)

	// These applications are required to get the cluster up and running, they must
	// tolerate control plane taints, be scheduled onto control plane nodes and allow
	// scale from zero.
	bootstrapProvisioner := concurrent.New("cluster bootstrap",
		cilium.New(apps.cilium),
		openstackcloudprovider.New(apps.openstackCloudProvider, options),
	)

	clusterAutoscalerProvisioner := conditional.New("cluster-autoscaler",
		p.cluster.AutoscalingEnabled,
		concurrent.New("cluster-autoscaler",
			clusterautoscaler.New(apps.clusterAutoscaler).InNamespace(p.cluster.Name),
			clusterautoscaleropenstack.New(apps.clusterAutoscalerOpenstack).InNamespace(p.cluster.Name),
		),
	)

	addonsProvisioner := concurrent.New("cluster add-ons",
		openstackplugincindercsi.New(apps.openstackPluginCinderCSI, options),
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
	log := log.FromContext(ctx)

	clusterManager, err := p.getClusterManager(ctx)
	if err != nil {
		return err
	}

	condition, err := clusterManager.StatusConditionRead(unikornv1core.ConditionAvailable)
	if err != nil {
		return err
	}

	if condition.Reason != unikornv1core.ConditionReasonProvisioned {
		log.Info("waiting for cluster manager to become ready")

		return provisioners.ErrYield
	}

	return nil
}

// getRegionClient returns an authenticated context with a client credentials access token
// and a client.  The context must be used by subseqent API calls in order to extract
// the access token.
func (p *Provisioner) getRegionClient(ctx context.Context, traceName string) (context.Context, regionapi.ClientWithResponsesInterface, error) {
	cli, err := coreclient.ProvisionerClientFromContext(ctx)
	if err != nil {
		return nil, nil, err
	}

	tokenIssuer := identityclient.NewTokenIssuer(cli, p.options.identityOptions, &p.options.clientOptions, constants.Application, constants.Version)

	ctx, err = tokenIssuer.Context(ctx, traceName)
	if err != nil {
		return nil, nil, err
	}

	getter := regionclient.New(cli, p.options.regionOptions, &p.options.clientOptions)

	client, err := getter.Client(ctx)
	if err != nil {
		return nil, nil, err
	}

	return ctx, client, nil
}

func (p *Provisioner) getIdentity(ctx context.Context, client regionapi.ClientWithResponsesInterface) (*regionapi.IdentityRead, error) {
	log := log.FromContext(ctx)

	response, err := client.GetApiV1OrganizationsOrganizationIDProjectsProjectIDIdentitiesIdentityIDWithResponse(ctx, p.cluster.Labels[coreconstants.OrganizationLabel], p.cluster.Labels[coreconstants.ProjectLabel], p.cluster.Annotations[coreconstants.IdentityAnnotation])
	if err != nil {
		return nil, err
	}

	if response.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("%w: identity GET expected 200 got %d", coreerrors.ErrAPIStatus, response.StatusCode())
	}

	resource := response.JSON200

	//nolint:exhaustive
	switch resource.Metadata.ProvisioningStatus {
	case coreapi.ResourceProvisioningStatusProvisioned:
		return resource, nil
	case coreapi.ResourceProvisioningStatusUnknown, coreapi.ResourceProvisioningStatusProvisioning:
		log.Info("waiting for identity to become ready")

		return nil, provisioners.ErrYield
	}

	return nil, fmt.Errorf("%w: unhandled status %s", ErrResourceDependency, resource.Metadata.ProvisioningStatus)
}

func (p *Provisioner) deleteIdentity(ctx context.Context, client regionapi.ClientWithResponsesInterface) error {
	response, err := client.DeleteApiV1OrganizationsOrganizationIDProjectsProjectIDIdentitiesIdentityIDWithResponse(ctx, p.cluster.Labels[coreconstants.OrganizationLabel], p.cluster.Labels[coreconstants.ProjectLabel], p.cluster.Annotations[coreconstants.IdentityAnnotation])
	if err != nil {
		return err
	}

	statusCode := response.StatusCode()

	// An accepted status means the API has recoded the deletion event and
	// we can delete the cluster, a not found means it's been deleted already
	// and again can proceed.  The goal here is not to leak resources.
	if statusCode != http.StatusAccepted && statusCode != http.StatusNotFound {
		return fmt.Errorf("%w: identity DELETE expected 202,404 got %d", ErrResourceDependency, statusCode)
	}

	return nil
}

func (p *Provisioner) getPhysicalNetwork(ctx context.Context, client regionapi.ClientWithResponsesInterface) (*regionapi.PhysicalNetworkRead, error) {
	log := log.FromContext(ctx)

	physicalNetworkID, ok := p.cluster.Annotations[coreconstants.PhysicalNetworkAnnotation]
	if !ok {
		//nolint: nilnil
		return nil, nil
	}

	response, err := client.GetApiV1OrganizationsOrganizationIDProjectsProjectIDIdentitiesIdentityIDPhysicalnetworksPhysicalNetworkIDWithResponse(ctx, p.cluster.Labels[coreconstants.OrganizationLabel], p.cluster.Labels[coreconstants.ProjectLabel], p.cluster.Annotations[coreconstants.IdentityAnnotation], physicalNetworkID)
	if err != nil {
		return nil, err
	}

	if response.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("%w: physical network GET expected 200 got %d", coreerrors.ErrAPIStatus, response.StatusCode())
	}

	resource := response.JSON200

	//nolint:exhaustive
	switch resource.Metadata.ProvisioningStatus {
	case coreapi.ResourceProvisioningStatusProvisioned:
		return resource, nil
	case coreapi.ResourceProvisioningStatusUnknown, coreapi.ResourceProvisioningStatusProvisioning:
		log.Info("waiting for physical network to become ready")

		return nil, provisioners.ErrYield
	}

	return nil, fmt.Errorf("%w: unhandled status %s", ErrResourceDependency, resource.Metadata.ProvisioningStatus)
}

func (p *Provisioner) getExternalNetwork(ctx context.Context, client regionapi.ClientWithResponsesInterface) (*regionapi.ExternalNetwork, error) {
	response, err := client.GetApiV1OrganizationsOrganizationIDRegionsRegionIDExternalnetworksWithResponse(ctx, p.cluster.Labels[coreconstants.OrganizationLabel], p.cluster.Spec.RegionID)
	if err != nil {
		return nil, err
	}

	if response.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("%w: external networks GET expected 200 got %d", coreerrors.ErrAPIStatus, response.StatusCode())
	}

	externalNetworks := *response.JSON200

	if len(externalNetworks) == 0 {
		return nil, fmt.Errorf("%w: no external networks available", ErrResourceDependency)
	}

	// NOTE: this relies on the region API enforcing ordering constraints so the selection
	// process is deterministic.
	return &externalNetworks[0], nil
}

func (p *Provisioner) identityOptions(ctx context.Context, client regionapi.ClientWithResponsesInterface) (*kubernetesprovisioners.ClusterOpenstackOptions, error) {
	identity, err := p.getIdentity(ctx, client)
	if err != nil {
		return nil, err
	}

	externalNetwork, err := p.getExternalNetwork(ctx, client)
	if err != nil {
		return nil, err
	}

	options := &kubernetesprovisioners.ClusterOpenstackOptions{
		CloudConfig:       *identity.Spec.Openstack.CloudConfig,
		Cloud:             *identity.Spec.Openstack.Cloud,
		ServerGroupID:     identity.Spec.Openstack.ServerGroupId,
		ExternalNetworkID: &externalNetwork.Id,
	}

	physicalNetwork, err := p.getPhysicalNetwork(ctx, client)
	if err != nil {
		return nil, err
	}

	if physicalNetwork != nil {
		options.ProviderNetwork = &kubernetesprovisioners.ClusterOpenstackProviderOptions{
			NetworkID: physicalNetwork.Spec.Openstack.NetworkId,
			SubnetID:  physicalNetwork.Spec.Openstack.SubnetId,
		}
	}

	return options, nil
}

// Provision implements the Provision interface.
func (p *Provisioner) Provision(ctx context.Context) error {
	// The cluster manager is provisioned asynchronously as it takes a good minute
	// to provision, so we have to poll for readiness.
	if err := p.managerReady(ctx); err != nil {
		return err
	}

	// Likewise identity creation is provisioned asynchronously as it too takes a
	// long time, epspectially if a physical network is being provisioned and that
	// needs to go out and talk to swiches.
	clientContext, client, err := p.getRegionClient(ctx, "provision")
	if err != nil {
		return err
	}

	options, err := p.identityOptions(clientContext, client)
	if err != nil {
		return err
	}

	provisioner, err := p.getProvisioner(ctx, options)
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
	provisioner, err := p.getProvisioner(ctx, nil)
	if err != nil {
		if errors.Is(err, ErrClusterManager) {
			return nil
		}

		return err
	}

	if err := provisioner.Deprovision(ctx); err != nil {
		return err
	}

	// Clean up the identity when everything has cleanly deprovisioned.
	// An accepted status means the API has recoded the deletion event and
	// we can delete the cluster, a not found means it's been deleted already
	// and again can proceed.  The goal here is not to leak resources.
	clientContext, client, err := p.getRegionClient(ctx, "deprovision")
	if err != nil {
		return err
	}

	if err := p.deleteIdentity(clientContext, client); err != nil {
		return err
	}

	return nil
}
