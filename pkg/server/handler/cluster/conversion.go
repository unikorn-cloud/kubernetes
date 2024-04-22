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
	goerrors "errors"
	"fmt"
	"slices"

	unikornv1core "github.com/unikorn-cloud/core/pkg/apis/unikorn/v1alpha1"
	coreconstants "github.com/unikorn-cloud/core/pkg/constants"
	"github.com/unikorn-cloud/core/pkg/server/errors"
	"github.com/unikorn-cloud/core/pkg/util"
	unikornv1 "github.com/unikorn-cloud/unikorn/pkg/apis/unikorn/v1alpha1"
	"github.com/unikorn-cloud/unikorn/pkg/constants"
	"github.com/unikorn-cloud/unikorn/pkg/providers"
	"github.com/unikorn-cloud/unikorn/pkg/server/generated"
	"github.com/unikorn-cloud/unikorn/pkg/server/handler/applicationbundle"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	ErrResourceLookup = goerrors.New("could not find the requested resource")
)

// convertMachine converts from a custom resource into the API definition.
func convertMachine(in *unikornv1.MachineGeneric) *generated.MachinePool {
	machine := &generated.MachinePool{
		Replicas:   in.Replicas,
		FlavorName: in.Flavor,
	}

	if in.DiskSize != nil {
		machine.Disk = &generated.Volume{
			Size: int(in.DiskSize.Value()) >> 30,
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
func convertMetadata(in *unikornv1.KubernetesCluster) (*generated.KubernetesClusterMetadata, error) {
	labels, err := in.ResourceLabels()
	if err != nil {
		return nil, err
	}

	// Validated to exist by ResourceLabels()
	project := labels[coreconstants.ProjectLabel]

	out := &generated.KubernetesClusterMetadata{
		Project:      project,
		Region:       in.Spec.Region,
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
func (c *Client) convert(in *unikornv1.KubernetesCluster) (*generated.KubernetesCluster, error) {
	metadata, err := convertMetadata(in)
	if err != nil {
		return nil, err
	}

	out := &generated.KubernetesCluster{
		Metadata: *metadata,
		Spec: generated.KubernetesClusterSpec{
			Name:          in.Name,
			Version:       string(*in.Spec.Version),
			WorkloadPools: convertWorkloadPools(in),
		},
	}

	return out, nil
}

// uconvertList converts from a custom resource list into the API definition.
func (c *Client) convertList(in *unikornv1.KubernetesClusterList) (generated.KubernetesClusters, error) {
	out := make(generated.KubernetesClusters, len(in.Items))

	for i := range in.Items {
		item, err := c.convert(&in.Items[i])
		if err != nil {
			return nil, err
		}

		out[i] = *item
	}

	return out, nil
}

// defaultApplicationBundle returns a default application bundle.
func (c *Client) defaultApplicationBundle(ctx context.Context) (*unikornv1.KubernetesClusterApplicationBundle, error) {
	applicationBundles, err := applicationbundle.NewClient(c.client).ListCluster(ctx)
	if err != nil {
		return nil, err
	}

	applicationBundles.Items = slices.DeleteFunc(applicationBundles.Items, func(bundle unikornv1.KubernetesClusterApplicationBundle) bool {
		if bundle.Spec.Preview != nil && *bundle.Spec.Preview {
			return true
		}

		if bundle.Spec.EndOfLife != nil {
			return true
		}

		return false
	})

	if len(applicationBundles.Items) == 0 {
		return nil, errors.OAuth2ServerError("unable to select an application bundle")
	}

	return &applicationBundles.Items[0], nil
}

// defaultControlPlaneFlavor returns a default control plane flavor.  This will be
// one that doesxn't have any GPUs.  The provider ensures the "nost cost-effective"
// comes first.
// TODO: we should allow this to be configured per region.
func (c *Client) defaultControlPlaneFlavor(ctx context.Context, provider providers.Provider) (*providers.Flavor, error) {
	flavors, err := provider.Flavors(ctx)
	if err != nil {
		return nil, err
	}

	flavors = slices.DeleteFunc(flavors, func(x providers.Flavor) bool { return x.GPUs != 0 })

	if len(flavors) == 0 {
		return nil, errors.OAuth2ServerError("unable to select a control plane flavor")
	}

	return &flavors[0], nil
}

// defaultImage returns a default image for either control planes or workload pools
// based on the specified Kubernetes version.
func (c *Client) defaultImage(ctx context.Context, provider providers.Provider, version string) (*providers.Image, error) {
	// Images will be
	images, err := provider.Images(ctx)
	if err != nil {
		return nil, err
	}

	images = slices.DeleteFunc(images, func(x providers.Image) bool {
		return x.KubernetesVersion != version
	})

	if len(images) == 0 {
		return nil, errors.OAuth2ServerError("unable to select an image")
	}

	return &images[0], nil
}

// generateNetwork generates the network part of a cluster.
func (c *Client) generateNetwork() *unikornv1.KubernetesClusterNetworkSpec {
	// Grab some defaults (as these are in the right format already)
	// the override with anything coming in from the API, if set.
	nodeNetwork := c.options.NodeNetwork
	serviceNetwork := c.options.ServiceNetwork
	podNetwork := c.options.PodNetwork
	dnsNameservers := c.options.DNSNameservers

	network := &unikornv1.KubernetesClusterNetworkSpec{
		NodeNetwork:    &unikornv1.IPv4Prefix{IPNet: nodeNetwork},
		ServiceNetwork: &unikornv1.IPv4Prefix{IPNet: serviceNetwork},
		PodNetwork:     &unikornv1.IPv4Prefix{IPNet: podNetwork},
		DNSNameservers: unikornv1.IPv4AddressSliceFromIPSlice(dnsNameservers),
	}

	return network
}

// generateMachineGeneric generates a generic machine part of the cluster.
func (c *Client) generateMachineGeneric(ctx context.Context, provider providers.Provider, options *generated.KubernetesClusterSpec, m *generated.MachinePool) (*unikornv1.MachineGeneric, error) {
	if m.Replicas == nil {
		m.Replicas = util.ToPointer(3)
	}

	image, err := c.defaultImage(ctx, provider, options.Version)
	if err != nil {
		return nil, err
	}

	machine := &unikornv1.MachineGeneric{
		Replicas: m.Replicas,
		Image:    util.ToPointer(image.Name),
		Flavor:   m.FlavorName,
	}

	if m.Disk != nil {
		size, err := resource.ParseQuantity(fmt.Sprintf("%dGi", m.Disk.Size))
		if err != nil {
			return nil, errors.OAuth2InvalidRequest("failed to parse disk size").WithError(err)
		}

		machine.DiskSize = &size
	}

	return machine, nil
}

// generateControlPlane generates the control plane part of a cluster.
func (c *Client) generateControlPlane(ctx context.Context, provider providers.Provider, options *generated.KubernetesClusterSpec) (*unikornv1.KubernetesClusterControlPlaneSpec, error) {
	// Add in any missing defaults.
	resource, err := c.defaultControlPlaneFlavor(ctx, provider)
	if err != nil {
		return nil, err
	}

	machineOptions := &generated.MachinePool{
		FlavorName: &resource.Name,
	}

	machine, err := c.generateMachineGeneric(ctx, provider, options, machineOptions)
	if err != nil {
		return nil, err
	}

	spec := &unikornv1.KubernetesClusterControlPlaneSpec{
		MachineGeneric: *machine,
	}

	return spec, nil
}

// generateWorkloadPools generates the workload pools part of a cluster.
func (c *Client) generateWorkloadPools(ctx context.Context, provider providers.Provider, options *generated.KubernetesClusterSpec) (*unikornv1.KubernetesClusterWorkloadPoolsSpec, error) {
	workloadPools := &unikornv1.KubernetesClusterWorkloadPoolsSpec{}

	for i := range options.WorkloadPools {
		pool := &options.WorkloadPools[i]

		machine, err := c.generateMachineGeneric(ctx, provider, options, &pool.Machine)
		if err != nil {
			return nil, err
		}

		workloadPool := unikornv1.KubernetesClusterWorkloadPoolsPoolSpec{
			KubernetesWorkloadPoolSpec: unikornv1.KubernetesWorkloadPoolSpec{
				Name:           pool.Name,
				MachineGeneric: *machine,
			},
		}

		if pool.Labels != nil {
			workloadPool.Labels = *pool.Labels
		}

		// With autoscaling, we automatically fill in the required metadata from
		// the flavor used in validation, this prevents having to surface this
		// complexity to the client via the API.
		if pool.Autoscaling != nil {
			flavor, err := lookupFlavor(ctx, provider, *machine.Flavor)
			if err != nil {
				return nil, err
			}

			workloadPool.Autoscaling = &unikornv1.MachineGenericAutoscaling{
				MinimumReplicas: &pool.Autoscaling.MinimumReplicas,
				MaximumReplicas: pool.Machine.Replicas,
				Scheduler: &unikornv1.MachineGenericAutoscalingScheduler{
					CPU:    &flavor.CPUs,
					Memory: flavor.Memory,
				},
			}

			if flavor.GPUs > 0 {
				// TODO: this is needed for scale from zero, at no point do the docs
				// mention AMD...
				t := "nvidia.com/gpu"

				workloadPool.Autoscaling.Scheduler.GPU = &unikornv1.MachineGenericAutoscalingSchedulerGPU{
					Type:  &t,
					Count: &flavor.GPUs,
				}
			}
		}

		workloadPools.Pools = append(workloadPools.Pools, workloadPool)
	}

	return workloadPools, nil
}

// lookupFlavor resolves the flavor from its name.
// NOTE: It looks like garbage performance, but the provider should be memoized...
func lookupFlavor(ctx context.Context, provider providers.Provider, name string) (*providers.Flavor, error) {
	flavors, err := provider.Flavors(ctx)
	if err != nil {
		return nil, err
	}

	index := slices.IndexFunc(flavors, func(flavor providers.Flavor) bool {
		return flavor.Name == name
	})

	if index < 0 {
		return nil, fmt.Errorf("%w: flavor %s", ErrResourceLookup, name)
	}

	return &flavors[index], nil
}

// installNvidiaOperator installs the nvidia operator if any workload pool flavor
// has a GPU in it.
func installNvidiaOperator(ctx context.Context, provider providers.Provider, cluster *unikornv1.KubernetesCluster) error {
	for _, pool := range cluster.Spec.WorkloadPools.Pools {
		flavor, err := lookupFlavor(ctx, provider, *pool.MachineGeneric.Flavor)
		if err != nil {
			return err
		}

		if flavor.GPUs > 0 {
			cluster.Spec.Features.NvidiaOperator = util.ToPointer(true)

			return nil
		}
	}

	return nil
}

// installClusterAutoscaler installs the cluster autoscaler if any workload pool has
// autoscaling enabled.
// TODO: probably push this down into the cluster manager.
func installClusterAutoscaler(cluster *unikornv1.KubernetesCluster) {
	for _, pool := range cluster.Spec.WorkloadPools.Pools {
		if pool.Autoscaling != nil {
			cluster.Spec.Features.Autoscaling = util.ToPointer(true)

			return
		}
	}
}

// generate generates the full cluster custom resource.
func (c *Client) generate(ctx context.Context, provider providers.Provider, namespace *corev1.Namespace, organization, project string, options *generated.KubernetesClusterSpec) (*unikornv1.KubernetesCluster, error) {
	kubernetesControlPlane, err := c.generateControlPlane(ctx, provider, options)
	if err != nil {
		return nil, err
	}

	kubernetesWorkloadPools, err := c.generateWorkloadPools(ctx, provider, options)
	if err != nil {
		return nil, err
	}

	applicationBundle, err := c.defaultApplicationBundle(ctx)
	if err != nil {
		return nil, err
	}

	version := options.Version
	if version[0] != 'v' {
		version = "v" + version
	}

	cluster := &unikornv1.KubernetesCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      options.Name,
			Namespace: namespace.Name,
			Labels: map[string]string{
				coreconstants.VersionLabel:      constants.Version,
				coreconstants.OrganizationLabel: organization,
				coreconstants.ProjectLabel:      project,
			},
		},
		Spec: unikornv1.KubernetesClusterSpec{
			Region:                       options.Region,
			ClusterManager:               *options.ClusterManager,
			Version:                      util.ToPointer(unikornv1.SemanticVersion(version)),
			ApplicationBundle:            &applicationBundle.Name,
			ApplicationBundleAutoUpgrade: &unikornv1.ApplicationBundleAutoUpgradeSpec{},
			Network:                      c.generateNetwork(),
			ControlPlane:                 kubernetesControlPlane,
			WorkloadPools:                kubernetesWorkloadPools,
			Features:                     &unikornv1.KubernetesClusterFeaturesSpec{},
		},
	}

	installClusterAutoscaler(cluster)

	if err := installNvidiaOperator(ctx, provider, cluster); err != nil {
		return nil, err
	}

	return cluster, nil
}
