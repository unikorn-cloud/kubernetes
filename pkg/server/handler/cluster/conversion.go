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
	"fmt"
	"net"

	unikornv1 "github.com/spjmurray/unikorn/pkg/apis/unikorn/v1alpha1"
	"github.com/spjmurray/unikorn/pkg/constants"
	"github.com/spjmurray/unikorn/pkg/server/errors"
	"github.com/spjmurray/unikorn/pkg/server/generated"
	"github.com/spjmurray/unikorn/pkg/server/handler/applicationbundle"
	"github.com/spjmurray/unikorn/pkg/server/handler/common"
	"github.com/spjmurray/unikorn/pkg/server/handler/controlplane"

	coreunikornv1 "github.com/spjmurray/unikorn-core/pkg/apis/unikorn/v1alpha1"
	coreconstants "github.com/spjmurray/unikorn-core/pkg/constants"

	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// convertOpenstack converts from a custom resource into the API definition.
func convertOpenstack(in *unikornv1.KubernetesCluster) generated.KubernetesClusterOpenStack {
	openstack := generated.KubernetesClusterOpenStack{
		ComputeAvailabilityZone: *in.Spec.Openstack.FailureDomain,
		VolumeAvailabilityZone:  *in.Spec.Openstack.VolumeFailureDomain,
		ExternalNetworkID:       *in.Spec.Openstack.ExternalNetworkID,
		SshKeyName:              in.Spec.Openstack.SSHKeyName,
	}

	return openstack
}

// convertNetwork converts from a custom resource into the API definition.
func convertNetwork(in *unikornv1.KubernetesCluster) generated.KubernetesClusterNetwork {
	dnsNameservers := make([]string, len(in.Spec.Network.DNSNameservers))

	for i, address := range in.Spec.Network.DNSNameservers {
		dnsNameservers[i] = address.IP.String()
	}

	network := generated.KubernetesClusterNetwork{
		NodePrefix:     in.Spec.Network.NodeNetwork.IPNet.String(),
		ServicePrefix:  in.Spec.Network.ServiceNetwork.IPNet.String(),
		PodPrefix:      in.Spec.Network.PodNetwork.IPNet.String(),
		DnsNameservers: dnsNameservers,
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
func convertMachine(in *unikornv1.MachineGeneric) generated.OpenstackMachinePool {
	machine := generated.OpenstackMachinePool{
		Replicas:   *in.Replicas,
		Version:    string(*in.Version),
		ImageName:  *in.Image,
		FlavorName: *in.Flavor,
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
		Machine: convertMachine(&in.KubernetesWorkloadPoolSpec.MachineGeneric),
	}

	if in.KubernetesWorkloadPoolSpec.Labels != nil {
		workloadPool.Labels = &in.KubernetesWorkloadPoolSpec.Labels
	}

	if in.KubernetesWorkloadPoolSpec.Autoscaling != nil {
		workloadPool.Autoscaling = &generated.KubernetesClusterAutoscaling{
			MinimumReplicas: *in.KubernetesWorkloadPoolSpec.Autoscaling.MinimumReplicas,
			MaximumReplicas: *in.KubernetesWorkloadPoolSpec.Autoscaling.MaximumReplicas,
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

// convertFeatures converts from a custom resource into the API definition.
func convertFeatures(in *unikornv1.KubernetesCluster) *generated.KubernetesClusterFeatures {
	if in.Spec.Features == nil {
		return nil
	}

	features := &generated.KubernetesClusterFeatures{
		Autoscaling:         in.Spec.Features.Autoscaling,
		Ingress:             in.Spec.Features.Ingress,
		CertManager:         in.Spec.Features.CertManager,
		KubernetesDashboard: in.Spec.Features.KubernetesDashboard,
		FileStorage:         in.Spec.Features.FileStorage,
		Prometheus:          in.Spec.Features.Prometheus,
		NvidiaOperator:      in.Spec.Features.NvidiaOperator,
	}

	return features
}

// convertStatus converts from a custom resource into the API definition.
func convertStatus(in *unikornv1.KubernetesCluster) *generated.KubernetesResourceStatus {
	out := &generated.KubernetesResourceStatus{
		Name:         in.Name,
		CreationTime: in.CreationTimestamp.Time,
		Status:       "Unknown",
	}

	if in.DeletionTimestamp != nil {
		out.DeletionTime = &in.DeletionTimestamp.Time
	}

	condition, err := in.StatusConditionRead(coreunikornv1.ConditionAvailable)
	if err == nil {
		out.Status = string(condition.Reason)
	}

	return out
}

// convert converts from a custom resource into the API definition.
func (c *Client) convert(ctx context.Context, in *unikornv1.KubernetesCluster) (*generated.KubernetesCluster, error) {
	bundle, err := applicationbundle.NewClient(c.client).GetKubernetesCluster(ctx, *in.Spec.ApplicationBundle)
	if err != nil {
		return nil, err
	}

	out := &generated.KubernetesCluster{
		Name:                         in.Name,
		ApplicationBundle:            *bundle,
		ApplicationBundleAutoUpgrade: common.ConvertApplicationBundleAutoUpgrade(in.Spec.ApplicationBundleAutoUpgrade),
		Openstack:                    convertOpenstack(in),
		Network:                      convertNetwork(in),
		Api:                          convertAPI(in),
		ControlPlane:                 convertMachine(&in.Spec.ControlPlane.MachineGeneric),
		WorkloadPools:                convertWorkloadPools(in),
		Features:                     convertFeatures(in),
		Status:                       convertStatus(in),
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

// createOpenstack creates the Openstack configuration part of a cluster.
func createOpenstack(options *generated.KubernetesCluster) *unikornv1.KubernetesClusterOpenstackSpec {
	openstack := &unikornv1.KubernetesClusterOpenstackSpec{
		FailureDomain:       &options.Openstack.ComputeAvailabilityZone,
		VolumeFailureDomain: &options.Openstack.VolumeAvailabilityZone,
		ExternalNetworkID:   &options.Openstack.ExternalNetworkID,
	}

	if options.Openstack.SshKeyName != nil {
		openstack.SSHKeyName = options.Openstack.SshKeyName
	}

	return openstack
}

// createNetwork creates the network part of a cluster.
func createNetwork(options *generated.KubernetesCluster) (*unikornv1.KubernetesClusterNetworkSpec, error) {
	_, nodeNet, err := net.ParseCIDR(options.Network.NodePrefix)
	if err != nil {
		return nil, errors.OAuth2InvalidRequest("failed to parse node prefix").WithError(err)
	}

	_, serviceNet, err := net.ParseCIDR(options.Network.ServicePrefix)
	if err != nil {
		return nil, errors.OAuth2InvalidRequest("failed to parse service prefix").WithError(err)
	}

	_, podNet, err := net.ParseCIDR(options.Network.PodPrefix)
	if err != nil {
		return nil, errors.OAuth2InvalidRequest("failed to parse pod prefix").WithError(err)
	}

	dnsNameservers := make([]net.IP, len(options.Network.DnsNameservers))

	for i, server := range options.Network.DnsNameservers {
		ip := net.ParseIP(server)
		if ip == nil {
			return nil, errors.OAuth2InvalidRequest("failed to parse dns server IP")
		}

		dnsNameservers[i] = ip
	}

	network := &unikornv1.KubernetesClusterNetworkSpec{
		NodeNetwork:    &unikornv1.IPv4Prefix{IPNet: *nodeNet},
		ServiceNetwork: &unikornv1.IPv4Prefix{IPNet: *serviceNet},
		PodNetwork:     &unikornv1.IPv4Prefix{IPNet: *podNet},
		DNSNameservers: unikornv1.IPv4AddressSliceFromIPSlice(dnsNameservers),
	}

	return network, nil
}

// createAPI creates the Kubernetes API part of the cluster.
func createAPI(options *generated.KubernetesCluster) (*unikornv1.KubernetesClusterAPISpec, error) {
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

// createMachineGeneric creates a generic machine part of the cluster.
func (c *Client) createMachineGeneric(m *generated.OpenstackMachinePool) (*unikornv1.MachineGeneric, *generated.OpenstackFlavor, error) {
	// Check the image passed in is valid.
	image, err := c.openstack.GetImage(c.request, m.ImageName)
	if err != nil {
		if errors.IsHTTPNotFound(err) {
			return nil, nil, errors.OAuth2InvalidRequest("invalid image").WithError(err)
		}

		return nil, nil, err
	}

	// TODO: we can derive the version from the image, but its useful to have that
	// in the GET data.
	if m.Version != image.Versions.Kubernetes {
		return nil, nil, errors.OAuth2InvalidRequest("invalid version for image").WithError(err)
	}

	// Check the flavor is valid
	flavor, err := c.openstack.GetFlavor(c.request, m.FlavorName)
	if err != nil {
		if errors.IsHTTPNotFound(err) {
			return nil, nil, errors.OAuth2InvalidRequest("invalid flavor").WithError(err)
		}

		return nil, nil, err
	}

	version := unikornv1.SemanticVersion(m.Version)

	machine := &unikornv1.MachineGeneric{
		Version:  &version,
		Replicas: &m.Replicas,
		Image:    &m.ImageName,
		Flavor:   &m.FlavorName,
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

// createControlPlane creates the control plane part of a cluster.
func (c *Client) createControlPlane(options *generated.KubernetesCluster) (*unikornv1.KubernetesClusterControlPlaneSpec, error) {
	machine, _, err := c.createMachineGeneric(&options.ControlPlane)
	if err != nil {
		return nil, err
	}

	controlPlane := &unikornv1.KubernetesClusterControlPlaneSpec{
		MachineGeneric: *machine,
	}

	return controlPlane, nil
}

// createWorkloadPools creates the workload pools part of a cluster.
func (c *Client) createWorkloadPools(clusterContext *createClusterContext, options *generated.KubernetesCluster) (*unikornv1.KubernetesClusterWorkloadPoolsSpec, error) {
	workloadPools := &unikornv1.KubernetesClusterWorkloadPoolsSpec{}

	for i := range options.WorkloadPools {
		pool := &options.WorkloadPools[i]

		machine, flavor, err := c.createMachineGeneric(&pool.Machine)
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
				MaximumReplicas: &pool.Autoscaling.MaximumReplicas,
				Scheduler: &unikornv1.MachineGenericAutoscalingScheduler{
					CPU:    &flavor.Cpus,
					Memory: &memory,
				},
			}

			if flavor.Gpus != nil {
				t := constants.NvidiaGPUType

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

// createFeatures creates the features part of a cluster.
func createFeatures(options *generated.KubernetesCluster) *unikornv1.KubernetesClusterFeaturesSpec {
	if options.Features == nil {
		return nil
	}

	features := &unikornv1.KubernetesClusterFeaturesSpec{
		Autoscaling:         options.Features.Autoscaling,
		Ingress:             options.Features.Ingress,
		CertManager:         options.Features.CertManager,
		KubernetesDashboard: options.Features.KubernetesDashboard,
		FileStorage:         options.Features.FileStorage,
		Prometheus:          options.Features.Prometheus,
		NvidiaOperator:      options.Features.NvidiaOperator,
	}

	return features
}

type createClusterContext struct {
	hasGPUWorkloadPool bool
}

func installNvidiaOperator(features *unikornv1.KubernetesClusterFeaturesSpec) bool {
	return features.NvidiaOperator == nil || *features.NvidiaOperator
}

// createCluster creates the full cluster custom resource.
func (c *Client) createCluster(controlPlane *controlplane.Meta, options *generated.KubernetesCluster) (*unikornv1.KubernetesCluster, error) {
	var clusterContext createClusterContext

	network, err := createNetwork(options)
	if err != nil {
		return nil, err
	}

	api, err := createAPI(options)
	if err != nil {
		return nil, err
	}

	kubernetesControlPlane, err := c.createControlPlane(options)
	if err != nil {
		return nil, err
	}

	kubernetesWorkloadPools, err := c.createWorkloadPools(&clusterContext, options)
	if err != nil {
		return nil, err
	}

	cluster := &unikornv1.KubernetesCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      options.Name,
			Namespace: controlPlane.Namespace,
			Labels: map[string]string{
				coreconstants.VersionLabel:      coreconstants.Version,
				coreconstants.ProjectLabel:      controlPlane.Project.Name,
				coreconstants.ControlPlaneLabel: controlPlane.Name,
			},
		},
		Spec: unikornv1.KubernetesClusterSpec{
			ApplicationBundle:            &options.ApplicationBundle.Name,
			ApplicationBundleAutoUpgrade: common.CreateApplicationBundleAutoUpgrade(options.ApplicationBundleAutoUpgrade),
			Openstack:                    createOpenstack(options),
			Network:                      network,
			API:                          api,
			ControlPlane:                 kubernetesControlPlane,
			WorkloadPools:                kubernetesWorkloadPools,
			Features:                     createFeatures(options),
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
