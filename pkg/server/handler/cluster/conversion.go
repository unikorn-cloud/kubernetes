/*
Copyright 2022-2024 EscherCloud.
Copyright 2024-2025 the Unikorn Authors.

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

	"github.com/Masterminds/semver/v3"

	unikornv1core "github.com/unikorn-cloud/core/pkg/apis/unikorn/v1alpha1"
	coreopenapi "github.com/unikorn-cloud/core/pkg/openapi"
	"github.com/unikorn-cloud/core/pkg/server/conversion"
	"github.com/unikorn-cloud/core/pkg/server/errors"
	"github.com/unikorn-cloud/identity/pkg/middleware/authorization"
	unikornv1 "github.com/unikorn-cloud/kubernetes/pkg/apis/unikorn/v1alpha1"
	"github.com/unikorn-cloud/kubernetes/pkg/openapi"
	"github.com/unikorn-cloud/kubernetes/pkg/server/handler/applicationbundle"
	"github.com/unikorn-cloud/kubernetes/pkg/server/handler/region"
	regionapi "github.com/unikorn-cloud/region/pkg/openapi"

	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/utils/ptr"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	// ErrResourceLookup is raised when we are looking for a referenced resource
	// but cannot find it.
	ErrResourceLookup = goerrors.New("could not find the requested resource")

	// ErrUnhandledCase is raised when an unhandled switch case is encountered.
	ErrUnhandledCase = goerrors.New("handled case")
)

// generator wraps up the myriad things we need to pass around as an object
// rather than a whole bunch of arguments.
type generator struct {
	// client allows Kubernetes access.
	client client.Client
	// options allows access to resource defaults.
	options *Options
	// region is a client to access regions.
	region regionapi.ClientWithResponsesInterface
	// namespace the resource is provisioned in.
	namespace string
	// organizationID is the unique organization identifier.
	organizationID string
	// projectID is the unique project identifier.
	projectID string
	// existing is the existing cluster used to preserve options
	// across updates.  This does two things, ensures we don't accidentally
	// pick up new defaults, and we preserve any modifications that were
	// made in a support capacity.
	existing *unikornv1.KubernetesCluster
}

func newGenerator(client client.Client, options *Options, region regionapi.ClientWithResponsesInterface, namespace, organizationID, projectID string) *generator {
	return &generator{
		client:         client,
		options:        options,
		region:         region,
		namespace:      namespace,
		organizationID: organizationID,
		projectID:      projectID,
	}
}

func (g *generator) withExisting(existing *unikornv1.KubernetesCluster) *generator {
	g.existing = existing

	return g
}

// convertMachine converts from a custom resource into the API definition.
func convertMachine(in *unikornv1core.MachineGeneric) *openapi.MachinePool {
	machine := &openapi.MachinePool{
		Replicas: in.Replicas,
		FlavorId: in.FlavorID,
	}

	if in.DiskSize != nil {
		machine.Disk = &openapi.Volume{
			Size: int(in.DiskSize.Value()) >> 30,
		}
	}

	return machine
}

// convertWorkloadPool converts from a custom resource into the API definition.
func convertWorkloadPool(in *unikornv1.KubernetesClusterWorkloadPoolsPoolSpec) openapi.KubernetesClusterWorkloadPool {
	workloadPool := openapi.KubernetesClusterWorkloadPool{
		Name:    in.Name,
		Machine: *convertMachine(&in.KubernetesWorkloadPoolSpec.MachineGeneric),
	}

	if in.KubernetesWorkloadPoolSpec.Labels != nil {
		workloadPool.Labels = &in.KubernetesWorkloadPoolSpec.Labels
	}

	if in.KubernetesWorkloadPoolSpec.Autoscaling != nil {
		workloadPool.Autoscaling = &openapi.KubernetesClusterAutoscaling{
			MinimumReplicas: *in.KubernetesWorkloadPoolSpec.Autoscaling.MinimumReplicas,
		}
	}

	return workloadPool
}

// convertWorkloadPools converts from a custom resource into the API definition.
func convertWorkloadPools(in *unikornv1.KubernetesCluster) []openapi.KubernetesClusterWorkloadPool {
	workloadPools := make([]openapi.KubernetesClusterWorkloadPool, len(in.Spec.WorkloadPools.Pools))

	for i := range in.Spec.WorkloadPools.Pools {
		workloadPools[i] = convertWorkloadPool(&in.Spec.WorkloadPools.Pools[i])
	}

	return workloadPools
}

// convert converts from a custom resource into the API definition.
func convert(in *unikornv1.KubernetesCluster) *openapi.KubernetesClusterRead {
	provisioningStatus := coreopenapi.ResourceProvisioningStatusUnknown

	if condition, err := in.StatusConditionRead(unikornv1core.ConditionAvailable); err == nil {
		provisioningStatus = conversion.ConvertStatusCondition(condition)
	}

	out := &openapi.KubernetesClusterRead{
		Metadata: conversion.ProjectScopedResourceReadMetadata(in, in.Spec.Tags, provisioningStatus),
		Spec: openapi.KubernetesClusterSpec{
			RegionId:         in.Spec.RegionID,
			ClusterManagerId: &in.Spec.ClusterManagerID,
			Version:          in.Spec.Version.Original(),
			WorkloadPools:    convertWorkloadPools(in),
		},
	}

	return out
}

// uconvertList converts from a custom resource list into the API definition.
func convertList(in *unikornv1.KubernetesClusterList) openapi.KubernetesClusters {
	out := make(openapi.KubernetesClusters, len(in.Items))

	for i := range in.Items {
		out[i] = *convert(&in.Items[i])
	}

	return out
}

// defaultApplicationBundle returns a default application bundle.
func (g *generator) defaultApplicationBundle(ctx context.Context) (*unikornv1.KubernetesClusterApplicationBundle, error) {
	applicationBundles, err := applicationbundle.NewClient(g.client).ListCluster(ctx)
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

// defaultControlPlaneFlavor returns a default control plane flavor.
func (g *generator) defaultControlPlaneFlavor(ctx context.Context, request *openapi.KubernetesClusterWrite) (*regionapi.Flavor, error) {
	flavors, err := region.Flavors(ctx, g.region, g.organizationID, request.Spec.RegionId)
	if err != nil {
		return nil, errors.OAuth2ServerError("failed to list flavors").WithError(err)
	}

	// No baremetal flavors, and no GPUs.  Would be very wasteful otherwise!
	flavors = slices.DeleteFunc(flavors, func(x regionapi.Flavor) bool {
		if x.Spec.Baremetal != nil && *x.Spec.Baremetal {
			return true
		}

		if x.Spec.Gpu != nil {
			return true
		}

		if x.Spec.Cpus > g.options.ControlPlaneCPUsMax {
			return true
		}

		if x.Spec.Memory > g.options.ControlPlaneMemoryMaxGiB {
			return true
		}

		return false
	})

	if len(flavors) == 0 {
		return nil, errors.OAuth2ServerError("unable to select a control plane flavor")
	}

	// Pick the most "epic" flavor possible, things tend to melt if you are too stingy.
	return &flavors[len(flavors)-1], nil
}

// defaultImage returns a default image for either control planes or workload pools
// based on the specified Kubernetes version.
func (g *generator) defaultImage(ctx context.Context, request *openapi.KubernetesClusterWrite, version string) (*regionapi.Image, error) {
	images, err := region.Images(ctx, g.region, g.organizationID, request.Spec.RegionId)
	if err != nil {
		return nil, errors.OAuth2ServerError("failed to list images").WithError(err)
	}

	// Only get the version asked for.
	images = slices.DeleteFunc(images, func(x regionapi.Image) bool {
		return (*x.Spec.SoftwareVersions)["kubernetes"] != version
	})

	if len(images) == 0 {
		return nil, errors.OAuth2ServerError("unable to select an image")
	}

	return &images[0], nil
}

// generateNetwork generates the network part of a cluster.
func (g *generator) generateNetwork() *unikornv1.KubernetesClusterNetworkSpec {
	// Grab some defaults (as these are in the right format already)
	// the override with anything coming in from the API, if set.
	nodeNetwork := g.options.NodeNetwork
	serviceNetwork := g.options.ServiceNetwork
	podNetwork := g.options.PodNetwork
	dnsNameservers := g.options.DNSNameservers

	network := &unikornv1.KubernetesClusterNetworkSpec{
		NetworkGeneric: unikornv1core.NetworkGeneric{
			NodeNetwork:    &unikornv1core.IPv4Prefix{IPNet: nodeNetwork},
			DNSNameservers: unikornv1core.IPv4AddressSliceFromIPSlice(dnsNameservers),
		},
		ServiceNetwork: &unikornv1core.IPv4Prefix{IPNet: serviceNetwork},
		PodNetwork:     &unikornv1core.IPv4Prefix{IPNet: podNetwork},
	}

	return network
}

// generateMachineGeneric generates a generic machine part of the cluster.
func (g *generator) generateMachineGeneric(ctx context.Context, request *openapi.KubernetesClusterWrite, m *openapi.MachinePool, imageID *string) (*unikornv1core.MachineGeneric, error) {
	machine := &unikornv1core.MachineGeneric{
		Replicas: m.Replicas,
		FlavorID: m.FlavorId,
		ImageID:  imageID,
	}

	if imageID == nil {
		image, err := g.defaultImage(ctx, request, request.Spec.Version)
		if err != nil {
			return nil, err
		}

		machine.ImageID = ptr.To(image.Metadata.Id)
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
func (g *generator) generateControlPlane(ctx context.Context, request *openapi.KubernetesClusterWrite) (*unikornv1core.MachineGeneric, error) {
	// Preserve anything that may have been generated previously so it doesn't change
	// randomly, or may have been set manually by an administrator.
	var imageID *string

	machineOptions := &openapi.MachinePool{
		Replicas: ptr.To(3),
	}

	if g.existing != nil {
		imageID = g.existing.Spec.ControlPlane.ImageID

		machineOptions.Replicas = g.existing.Spec.ControlPlane.Replicas
		machineOptions.FlavorId = g.existing.Spec.ControlPlane.FlavorID
	}

	// Add in any missing defaults.
	if machineOptions.FlavorId == nil {
		flavor, err := g.defaultControlPlaneFlavor(ctx, request)
		if err != nil {
			return nil, err
		}

		machineOptions.FlavorId = &flavor.Metadata.Id
	}

	machine, err := g.generateMachineGeneric(ctx, request, machineOptions, imageID)
	if err != nil {
		return nil, err
	}

	return machine, nil
}

// generateWorkloadPools generates the workload pools part of a cluster.
func (g *generator) generateWorkloadPools(ctx context.Context, request *openapi.KubernetesClusterWrite) (*unikornv1.KubernetesClusterWorkloadPoolsSpec, error) {
	workloadPools := &unikornv1.KubernetesClusterWorkloadPoolsSpec{}

	for i := range request.Spec.WorkloadPools {
		pool := &request.Spec.WorkloadPools[i]

		// Preserve anything we default to that may change across invocations.
		var imageID *string

		if g.existing != nil {
			if pool := g.existing.GetWorkloadPool(pool.Name); pool != nil {
				imageID = pool.ImageID
			}
		}

		machine, err := g.generateMachineGeneric(ctx, request, &pool.Machine, imageID)
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
			workloadPool.Autoscaling = &unikornv1.MachineGenericAutoscaling{
				MinimumReplicas: &pool.Autoscaling.MinimumReplicas,
				MaximumReplicas: pool.Machine.Replicas,
			}
		}

		workloadPools.Pools = append(workloadPools.Pools, workloadPool)
	}

	return workloadPools, nil
}

// generate generates the full cluster custom resource.
// TODO: there are a lot of parameters being passed about, we should make this
// a struct and pass them as a single blob.
func (g *generator) generate(ctx context.Context, request *openapi.KubernetesClusterWrite) (*unikornv1.KubernetesCluster, error) {
	kubernetesControlPlane, err := g.generateControlPlane(ctx, request)
	if err != nil {
		return nil, err
	}

	kubernetesWorkloadPools, err := g.generateWorkloadPools(ctx, request)
	if err != nil {
		return nil, err
	}

	// Handle anything defaulted so we preserve across calls.
	var applicationBundleName *string

	if g.existing != nil {
		applicationBundleName = g.existing.Spec.ApplicationBundle
	}

	if applicationBundleName == nil {
		applicationBundle, err := g.defaultApplicationBundle(ctx)
		if err != nil {
			return nil, err
		}

		applicationBundleName = &applicationBundle.Name
	}

	autoscaling := ptr.To(true)
	gpuOperator := ptr.To(true)

	if g.existing != nil {
		autoscaling = g.existing.Spec.Features.Autoscaling
		gpuOperator = g.existing.Spec.Features.GPUOperator
	}

	userinfo, err := authorization.UserinfoFromContext(ctx)
	if err != nil {
		return nil, err
	}

	version, err := semver.NewVersion(request.Spec.Version)
	if err != nil {
		return nil, err
	}

	cluster := &unikornv1.KubernetesCluster{
		ObjectMeta: conversion.NewObjectMetadata(&request.Metadata, g.namespace, userinfo.Sub).WithOrganization(g.organizationID).WithProject(g.projectID).Get(),
		Spec: unikornv1.KubernetesClusterSpec{
			Tags:             conversion.GenerateTagList(request.Metadata.Tags),
			RegionID:         request.Spec.RegionId,
			ClusterManagerID: *request.Spec.ClusterManagerId,
			Version: &unikornv1core.SemanticVersion{
				Version: *version,
			},
			ApplicationBundle:            applicationBundleName,
			ApplicationBundleAutoUpgrade: &unikornv1.ApplicationBundleAutoUpgradeSpec{},
			Network:                      g.generateNetwork(),
			ControlPlane:                 kubernetesControlPlane,
			WorkloadPools:                kubernetesWorkloadPools,
			Features: &unikornv1.KubernetesClusterFeaturesSpec{
				Autoscaling: autoscaling,
				GPUOperator: gpuOperator,
			},
		},
	}

	return cluster, nil
}
