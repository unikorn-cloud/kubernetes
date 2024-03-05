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
	"fmt"
	"net"
	"slices"

	unikornv1core "github.com/unikorn-cloud/core/pkg/apis/unikorn/v1alpha1"
	"github.com/unikorn-cloud/core/pkg/constants"
	"github.com/unikorn-cloud/core/pkg/util"
	unikornv1 "github.com/unikorn-cloud/unikorn/pkg/apis/unikorn/v1alpha1"
	"github.com/unikorn-cloud/unikorn/pkg/server/errors"
	"github.com/unikorn-cloud/unikorn/pkg/server/generated"
	"github.com/unikorn-cloud/unikorn/pkg/server/handler/applicationbundle"
	"github.com/unikorn-cloud/unikorn/pkg/server/handler/common"
	"github.com/unikorn-cloud/unikorn/pkg/server/handler/controlplane"

	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// convertOpenstack converts from a custom resource into the API definition.
func convertOpenstack(in *unikornv1.KubernetesCluster) *generated.KubernetesClusterOpenStack {
	openstack := &generated.KubernetesClusterOpenStack{
		ComputeAvailabilityZone: in.Spec.Openstack.FailureDomain,
		VolumeAvailabilityZone:  in.Spec.Openstack.VolumeFailureDomain,
		ExternalNetworkID:       in.Spec.Openstack.ExternalNetworkID,
		SshKeyName:              in.Spec.Openstack.SSHKeyName,
	}

	return openstack
}

// convertNetwork converts from a custom resource into the API definition.
func convertNetwork(in *unikornv1.KubernetesCluster) *generated.KubernetesClusterNetwork {
	dnsNameservers := make([]string, len(in.Spec.Network.DNSNameservers))

	for i, address := range in.Spec.Network.DNSNameservers {
		dnsNameservers[i] = address.IP.String()
	}

	nodePrefix := in.Spec.Network.NodeNetwork.IPNet.String()
	servicePrefix := in.Spec.Network.ServiceNetwork.IPNet.String()
	podPrefix := in.Spec.Network.PodNetwork.IPNet.String()

	network := &generated.KubernetesClusterNetwork{
		NodePrefix:     &nodePrefix,
		ServicePrefix:  &servicePrefix,
		PodPrefix:      &podPrefix,
		DnsNameservers: &dnsNameservers,
	}

	return network
}

// convertAPI converts from a custom resource into the API definition.
func convertAPI(in *unikornv1.KubernetesCluster) *generated.KubernetesClusterAPI {
	if in.Spec.API == nil {
		return nil
	}

	api := &generated.KubernetesClusterAPI{}

	if len(in.Spec.API.SubjectAlternativeNames) > 0 {
		api.SubjectAlternativeNames = &in.Spec.API.SubjectAlternativeNames
	}

	if len(in.Spec.API.AllowedPrefixes) > 0 {
		allowedPrefixes := make([]string, len(in.Spec.API.AllowedPrefixes))

		for i, prefix := range in.Spec.API.AllowedPrefixes {
			allowedPrefixes[i] = prefix.IPNet.String()
		}

		api.AllowedPrefixes = &allowedPrefixes
	}

	return api
}

// convertMachine converts from a custom resource into the API definition.
func convertMachine(in *unikornv1.MachineGeneric) *generated.OpenstackMachinePool {
	machine := &generated.OpenstackMachinePool{
		Replicas:   in.Replicas,
		ImageName:  in.Image,
		FlavorName: in.Flavor,
	}

	if in.DiskSize != nil {
		machine.Disk = &generated.OpenstackVolume{
			Size:             int(in.DiskSize.Value()) >> 30,
			AvailabilityZone: in.VolumeFailureDomain,
		}
	}

	return machine
}

// convertWorkloadPool converts from a custom resource into the API definition.
func convertWorkloadPool(in *unikornv1.KubernetesClusterWorkloadPoolsPoolSpec) generated.KubernetesClusterWorkloadPool {
	workloadPool := generated.KubernetesClusterWorkloadPool{
		Name:    in.Name,
		Machine: *convertMachine(&in.KubernetesWorkloadPoolSpec.MachineGeneric),
	}

	if in.KubernetesWorkloadPoolSpec.Labels != nil {
		workloadPool.Labels = &in.KubernetesWorkloadPoolSpec.Labels
	}

	if in.KubernetesWorkloadPoolSpec.Autoscaling != nil {
		workloadPool.Autoscaling = &generated.KubernetesClusterAutoscaling{
			MinimumReplicas: *in.KubernetesWorkloadPoolSpec.Autoscaling.MinimumReplicas,
		}
	}

	return workloadPool
}

// convertWorkloadPools converts from a custom resource into the API definition.
func convertWorkloadPools(in *unikornv1.KubernetesCluster) []generated.KubernetesClusterWorkloadPool {
	workloadPools := make([]generated.KubernetesClusterWorkloadPool, len(in.Spec.WorkloadPools.Pools))

	for i := range in.Spec.WorkloadPools.Pools {
		workloadPools[i] = convertWorkloadPool(&in.Spec.WorkloadPools.Pools[i])
	}

	return workloadPools
}

// convertMetadata converts from a custom resource into the API definition.
func convertMetadata(in *unikornv1.KubernetesCluster) (*generated.ResourceMetadata, error) {
	labels, err := in.ResourceLabels()
	if err != nil {
		return nil, err
	}

	// Validated to exist by ResourceLabels()
	project := labels[constants.ProjectLabel]
	controlplane := labels[constants.ControlPlaneLabel]

	out := &generated.ResourceMetadata{
		Project:      &project,
		Controlplane: &controlplane,
		Region:       &in.Spec.Region,
		CreationTime: in.CreationTimestamp.Time,
		Status:       "Unknown",
	}

	if in.DeletionTimestamp != nil {
		out.DeletionTime = &in.DeletionTimestamp.Time
	}

	condition, err := in.StatusConditionRead(unikornv1core.ConditionAvailable)
	if err == nil {
		out.Status = string(condition.Reason)
	}

	return out, nil
}

// convert converts from a custom resource into the API definition.
func (c *Client) convert(ctx context.Context, in *unikornv1.KubernetesCluster) (*generated.KubernetesCluster, error) {
	metadata, err := convertMetadata(in)
	if err != nil {
		return nil, err
	}

	bundle, err := applicationbundle.NewClient(c.client).GetKubernetesCluster(ctx, *in.Spec.ApplicationBundle)
	if err != nil {
		return nil, err
	}

	out := &generated.KubernetesCluster{
		Metadata:                     metadata,
		Name:                         in.Name,
		Version:                      string(*in.Spec.ControlPlane.Version),
		ApplicationBundle:            bundle,
		ApplicationBundleAutoUpgrade: common.ConvertApplicationBundleAutoUpgrade(in.Spec.ApplicationBundleAutoUpgrade),
		Openstack:                    convertOpenstack(in),
		Network:                      convertNetwork(in),
		Api:                          convertAPI(in),
		ControlPlane:                 convertMachine(&in.Spec.ControlPlane.MachineGeneric),
		WorkloadPools:                convertWorkloadPools(in),
	}

	return out, nil
}

// uconvertList converts from a custom resource list into the API definition.
func (c *Client) convertList(ctx context.Context, in *unikornv1.KubernetesClusterList) ([]*generated.KubernetesCluster, error) {
	out := make([]*generated.KubernetesCluster, len(in.Items))

	for i := range in.Items {
		item, err := c.convert(ctx, &in.Items[i])
		if err != nil {
			return nil, err
		}

		out[i] = item
	}

	return out, nil
}

// defaultApplicationBundle returns a default application bundle.
func (c *Client) defaultApplicationBundle(ctx context.Context) (*generated.ApplicationBundle, error) {
	applicationBundles, err := applicationbundle.NewClient(c.client).ListCluster(ctx)
	if err != nil {
		return nil, err
	}

	applicationBundles = slices.DeleteFunc(applicationBundles, func(bundle *generated.ApplicationBundle) bool {
		if bundle.Preview != nil && *bundle.Preview {
			return true
		}

		if bundle.EndOfLife != nil {
			return true
		}

		return false
	})

	if len(applicationBundles) == 0 {
		return nil, errors.OAuth2ServerError("unable to select an application bundle")
	}

	return applicationBundles[0], nil
}

// defaultExternalNetwork returns a default network.
func (c *Client) defaultExternalNetwork(ctx context.Context) (*generated.OpenstackExternalNetwork, error) {
	externalNetworks, err := c.openstack.ListExternalNetworks(ctx)
	if err != nil {
		return nil, err
	}

	if len(externalNetworks) == 0 {
		return nil, errors.OAuth2ServerError("unable to select an external network")
	}

	return &externalNetworks[0], nil
}

// defaultControlPlaneFlavor returns a default control plane flavor.  This will be
// one that doesxn't have any GPUs.  The provider ensures the "nost cost-effective"
// comes first.
// TODO: we should allow this to be configured per region.
func (c *Client) defaultControlPlaneFlavor(ctx context.Context) (*generated.OpenstackFlavor, error) {
	flavors, err := c.openstack.ListFlavors(ctx)
	if err != nil {
		return nil, err
	}

	flavors = slices.DeleteFunc(flavors, func(x generated.OpenstackFlavor) bool { return x.Gpus != nil })

	if len(flavors) == 0 {
		return nil, errors.OAuth2ServerError("unable to select a control plane flavor")
	}

	return &flavors[0], nil
}

// defaultImage returns a default image for either control planes or workload pools
// based on the specified Kubernetes version.
func (c *Client) defaultImage(ctx context.Context, version string) (*generated.OpenstackImage, error) {
	// Images will be
	images, err := c.openstack.ListImages(ctx)
	if err != nil {
		return nil, err
	}

	images = slices.DeleteFunc(images, func(x generated.OpenstackImage) bool { return x.Versions.Kubernetes == version })

	if len(images) == 0 {
		return nil, errors.OAuth2ServerError("unable to select an image")
	}

	return &images[0], nil
}

// generateOpenstack generates the Openstack configuration part of a cluster.
func (c *Client) generateOpenstack(ctx context.Context, options *generated.KubernetesCluster) (*unikornv1.KubernetesClusterOpenstackSpec, error) {
	// Default missing configuration.
	os := options.Openstack
	if os == nil {
		os = &generated.KubernetesClusterOpenStack{}
	}

	if os.ExternalNetworkID == nil {
		resource, err := c.defaultExternalNetwork(ctx)
		if err != nil {
			return nil, err
		}

		os.ExternalNetworkID = &resource.Id
	}

	openstack := &unikornv1.KubernetesClusterOpenstackSpec{
		FailureDomain:       os.ComputeAvailabilityZone,
		VolumeFailureDomain: os.VolumeAvailabilityZone,
		ExternalNetworkID:   os.ExternalNetworkID,
	}

	if os.SshKeyName != nil {
		openstack.SSHKeyName = os.SshKeyName
	}

	return openstack, nil
}

// generateNetwork generates the network part of a cluster.
//
//nolint:cyclop
func (c *Client) generateNetwork(options *generated.KubernetesCluster) (*unikornv1.KubernetesClusterNetworkSpec, error) {
	// Grab some defaults (as these are in the right format already)
	// the override with anything coming in from the API, if set.
	nodeNetwork := c.options.NodeNetwork
	serviceNetwork := c.options.ServiceNetwork
	podNetwork := c.options.PodNetwork
	dnsNameservers := c.options.DNSNameservers

	//nolint:nestif
	if options.Network != nil {
		if options.Network.NodePrefix != nil {
			_, t, err := net.ParseCIDR(*options.Network.NodePrefix)
			if err != nil {
				return nil, errors.OAuth2InvalidRequest("failed to parse node prefix").WithError(err)
			}

			nodeNetwork = *t
		}

		if options.Network.ServicePrefix != nil {
			_, t, err := net.ParseCIDR(*options.Network.ServicePrefix)
			if err != nil {
				return nil, errors.OAuth2InvalidRequest("failed to parse service prefix").WithError(err)
			}

			serviceNetwork = *t
		}

		if options.Network.PodPrefix != nil {
			_, t, err := net.ParseCIDR(*options.Network.PodPrefix)
			if err != nil {
				return nil, errors.OAuth2InvalidRequest("failed to parse pod prefix").WithError(err)
			}

			podNetwork = *t
		}

		if options.Network.DnsNameservers != nil {
			dnsNameservers := make([]net.IP, len(*options.Network.DnsNameservers))

			for i, server := range *options.Network.DnsNameservers {
				ip := net.ParseIP(server)
				if ip == nil {
					return nil, errors.OAuth2InvalidRequest("failed to parse dns server IP")
				}

				dnsNameservers[i] = ip
			}
		}
	}

	network := &unikornv1.KubernetesClusterNetworkSpec{
		NodeNetwork:    &unikornv1.IPv4Prefix{IPNet: nodeNetwork},
		ServiceNetwork: &unikornv1.IPv4Prefix{IPNet: serviceNetwork},
		PodNetwork:     &unikornv1.IPv4Prefix{IPNet: podNetwork},
		DNSNameservers: unikornv1.IPv4AddressSliceFromIPSlice(dnsNameservers),
	}

	return network, nil
}

// generateAPI generates the Kubernetes API part of the cluster.
func generateAPI(options *generated.KubernetesCluster) (*unikornv1.KubernetesClusterAPISpec, error) {
	if options.Api == nil {
		//nolint:nilnil
		return nil, nil
	}

	api := &unikornv1.KubernetesClusterAPISpec{}

	if options.Api.SubjectAlternativeNames != nil {
		api.SubjectAlternativeNames = *options.Api.SubjectAlternativeNames
	}

	if options.Api.AllowedPrefixes != nil {
		prefixes := make([]unikornv1.IPv4Prefix, len(*options.Api.AllowedPrefixes))

		for i, prefix := range *options.Api.AllowedPrefixes {
			_, network, err := net.ParseCIDR(prefix)
			if err != nil {
				return nil, errors.OAuth2InvalidRequest("failed to parse api allowed prefix").WithError(err)
			}

			prefixes[i] = unikornv1.IPv4Prefix{IPNet: *network}
		}

		api.AllowedPrefixes = prefixes
	}

	return api, nil
}

// generateMachineGeneric generates a generic machine part of the cluster.
func (c *Client) generateMachineGeneric(ctx context.Context, options *generated.KubernetesCluster, m *generated.OpenstackMachinePool) (*unikornv1.MachineGeneric, *generated.OpenstackFlavor, error) {
	if m.Replicas == nil {
		m.Replicas = util.ToPointer(3)
	}

	if m.ImageName == nil {
		resource, err := c.defaultImage(ctx, options.Version)
		if err != nil {
			return nil, nil, err
		}

		m.ImageName = &resource.Name
	}

	// Lookup the flavor so we can assess whether to install GPU controllers.
	flavor, err := c.openstack.GetFlavor(ctx, *m.FlavorName)
	if err != nil {
		if errors.IsHTTPNotFound(err) {
			return nil, nil, errors.OAuth2InvalidRequest("invalid flavor").WithError(err)
		}

		return nil, nil, err
	}

	machine := &unikornv1.MachineGeneric{
		Replicas: m.Replicas,
		Image:    m.ImageName,
		Flavor:   m.FlavorName,
	}

	if m.Disk != nil {
		size, err := resource.ParseQuantity(fmt.Sprintf("%dGi", m.Disk.Size))
		if err != nil {
			return nil, nil, errors.OAuth2InvalidRequest("failed to parse disk size").WithError(err)
		}

		machine.DiskSize = &size

		if m.Disk.AvailabilityZone != nil {
			machine.VolumeFailureDomain = m.Disk.AvailabilityZone
		}
	}

	return machine, flavor, nil
}

// generateControlPlane generates the control plane part of a cluster.
func (c *Client) generateControlPlane(ctx context.Context, options *generated.KubernetesCluster) (*unikornv1.KubernetesClusterControlPlaneSpec, error) {
	// Add in any missing defaults.
	controlPlaneOptions := options.ControlPlane

	if controlPlaneOptions == nil {
		controlPlaneOptions = &generated.OpenstackMachinePool{}
	}

	if controlPlaneOptions.FlavorName == nil {
		resource, err := c.defaultControlPlaneFlavor(ctx)
		if err != nil {
			return nil, err
		}

		controlPlaneOptions.FlavorName = &resource.Name
	}

	machine, _, err := c.generateMachineGeneric(ctx, options, controlPlaneOptions)
	if err != nil {
		return nil, err
	}

	controlPlane := &unikornv1.KubernetesClusterControlPlaneSpec{
		MachineGeneric: *machine,
	}

	return controlPlane, nil
}

// generateWorkloadPools generates the workload pools part of a cluster.
func (c *Client) generateWorkloadPools(ctx context.Context, clusterContext *generateContext, options *generated.KubernetesCluster) (*unikornv1.KubernetesClusterWorkloadPoolsSpec, error) {
	workloadPools := &unikornv1.KubernetesClusterWorkloadPoolsSpec{}

	for i := range options.WorkloadPools {
		pool := &options.WorkloadPools[i]

		machine, flavor, err := c.generateMachineGeneric(ctx, options, &pool.Machine)
		if err != nil {
			return nil, err
		}

		if flavor.Gpus != nil {
			clusterContext.hasGPUWorkloadPool = true
		}

		workloadPool := unikornv1.KubernetesClusterWorkloadPoolsPoolSpec{
			KubernetesWorkloadPoolSpec: unikornv1.KubernetesWorkloadPoolSpec{
				Name:           pool.Name,
				MachineGeneric: *machine,
				FailureDomain:  pool.AvailabilityZone,
			},
		}

		if pool.Labels != nil {
			workloadPool.Labels = *pool.Labels
		}

		// With autoscaling, we automatically fill in the required metadata from
		// the flavor used in validation, this prevents having to surface this
		// complexity to the client via the API.
		if pool.Autoscaling != nil {
			memory, err := resource.ParseQuantity(fmt.Sprintf("%dGi", flavor.Memory))
			if err != nil {
				return nil, errors.OAuth2InvalidRequest("failed to parse workload pool memory hint").WithError(err)
			}

			workloadPool.Autoscaling = &unikornv1.MachineGenericAutoscaling{
				MinimumReplicas: &pool.Autoscaling.MinimumReplicas,
				MaximumReplicas: pool.Machine.Replicas,
				Scheduler: &unikornv1.MachineGenericAutoscalingScheduler{
					CPU:    &flavor.Cpus,
					Memory: &memory,
				},
			}

			if flavor.Gpus != nil {
				t := "nvidia.com/gpu"

				workloadPool.Autoscaling.Scheduler.GPU = &unikornv1.MachineGenericAutoscalingSchedulerGPU{
					Type:  &t,
					Count: flavor.Gpus,
				}
			}
		}

		workloadPools.Pools = append(workloadPools.Pools, workloadPool)
	}

	return workloadPools, nil
}

type generateContext struct {
	hasGPUWorkloadPool bool
}

func installNvidiaOperator(features *unikornv1.KubernetesClusterFeaturesSpec) bool {
	return features.NvidiaOperator == nil || *features.NvidiaOperator
}

// generate generates the full cluster custom resource.
func (c *Client) generate(ctx context.Context, controlPlane *controlplane.Meta, options *generated.KubernetesCluster) (*unikornv1.KubernetesCluster, error) {
	var clusterContext generateContext

	openstack, err := c.generateOpenstack(ctx, options)
	if err != nil {
		return nil, err
	}

	network, err := c.generateNetwork(options)
	if err != nil {
		return nil, err
	}

	api, err := generateAPI(options)
	if err != nil {
		return nil, err
	}

	kubernetesControlPlane, err := c.generateControlPlane(ctx, options)
	if err != nil {
		return nil, err
	}

	kubernetesWorkloadPools, err := c.generateWorkloadPools(ctx, &clusterContext, options)
	if err != nil {
		return nil, err
	}

	applicationBundle := options.ApplicationBundle

	if applicationBundle == nil {
		resource, err := c.defaultApplicationBundle(ctx)
		if err != nil {
			return nil, err
		}

		applicationBundle = resource
	}

	cluster := &unikornv1.KubernetesCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      options.Name,
			Namespace: controlPlane.Namespace,
			Labels: map[string]string{
				constants.VersionLabel:      constants.Version,
				constants.OrganizationLabel: controlPlane.Project.Organization.Name,
				constants.ProjectLabel:      controlPlane.Project.Name,
				constants.ControlPlaneLabel: controlPlane.Name,
			},
		},
		Spec: unikornv1.KubernetesClusterSpec{
			Region:                       options.Region,
			ApplicationBundle:            &applicationBundle.Name,
			ApplicationBundleAutoUpgrade: common.CreateApplicationBundleAutoUpgrade(options.ApplicationBundleAutoUpgrade),
			Openstack:                    openstack,
			Network:                      network,
			API:                          api,
			ControlPlane:                 kubernetesControlPlane,
			WorkloadPools:                kubernetesWorkloadPools,
		},
	}

	// Automatically install the nvidia operator if a workload pool has GPUs, check if it's
	// been explicitly set to false
	if cluster.Spec.Features == nil {
		cluster.Spec.Features = &unikornv1.KubernetesClusterFeaturesSpec{}
	}

	if installNvidiaOperator(cluster.Spec.Features) {
		cluster.Spec.Features.NvidiaOperator = &clusterContext.hasGPUWorkloadPool
	}

	return cluster, nil
}
