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
	"net"
	"net/http"
	"slices"

	"github.com/spf13/pflag"

	coreclient "github.com/unikorn-cloud/core/pkg/client"
	"github.com/unikorn-cloud/core/pkg/constants"
	coreapi "github.com/unikorn-cloud/core/pkg/openapi"
	"github.com/unikorn-cloud/core/pkg/server/conversion"
	"github.com/unikorn-cloud/core/pkg/server/errors"
	coreapiutils "github.com/unikorn-cloud/core/pkg/util/api"
	identityapi "github.com/unikorn-cloud/identity/pkg/openapi"
	unikornv1 "github.com/unikorn-cloud/kubernetes/pkg/apis/unikorn/v1alpha1"
	"github.com/unikorn-cloud/kubernetes/pkg/openapi"
	"github.com/unikorn-cloud/kubernetes/pkg/provisioners/helmapplications/clusteropenstack"
	"github.com/unikorn-cloud/kubernetes/pkg/provisioners/helmapplications/vcluster"
	"github.com/unikorn-cloud/kubernetes/pkg/server/handler/clustermanager"
	"github.com/unikorn-cloud/kubernetes/pkg/server/handler/common"
	"github.com/unikorn-cloud/kubernetes/pkg/server/handler/identity"
	"github.com/unikorn-cloud/kubernetes/pkg/server/handler/region"
	regionapi "github.com/unikorn-cloud/region/pkg/openapi"

	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/utils/ptr"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	ErrConsistency = goerrors.New("consistency error")
)

type Options struct {
	ControlPlaneCPUsMax      int
	ControlPlaneMemoryMaxGiB int
	NodeNetwork              net.IPNet
	ServiceNetwork           net.IPNet
	PodNetwork               net.IPNet
	DNSNameservers           []net.IP
}

func (o *Options) AddFlags(f *pflag.FlagSet) {
	_, nodeNetwork, _ := net.ParseCIDR("192.168.0.0/24")
	_, serviceNetwork, _ := net.ParseCIDR("172.16.0.0/12")
	_, podNetwork, _ := net.ParseCIDR("10.0.0.0/8")

	dnsNameservers := []net.IP{net.ParseIP("8.8.8.8")}

	f.IntVar(&o.ControlPlaneCPUsMax, "control-plane-cpus-max", 8, "Default maximum CPUs for control plane flavor selection")
	f.IntVar(&o.ControlPlaneMemoryMaxGiB, "control-plane-memory-max-gib", 16, "Default maximum memory for control plane flavor selection")
	f.IPNetVar(&o.NodeNetwork, "default-node-network", *nodeNetwork, "Default node network to use when creating a cluster")
	f.IPNetVar(&o.ServiceNetwork, "default-service-network", *serviceNetwork, "Default service network to use when creating a cluster")
	f.IPNetVar(&o.PodNetwork, "default-pod-network", *podNetwork, "Default pod network to use when creating a cluster")
	f.IPSliceVar(&o.DNSNameservers, "default-dns-nameservers", dnsNameservers, "Default DNS nameserver to use when creating a cluster")
}

// Client wraps up cluster related management handling.
type Client struct {
	// client allows Kubernetes API access.
	client client.Client

	// namespace the controller runs in.
	namespace string

	// options control various defaults and the like.
	options *Options

	// identity is a client to access the identity service.
	identity *identity.Client

	// region is a client to access regions.
	region *region.Client
}

// NewClient returns a new client with required parameters.
func NewClient(client client.Client, namespace string, options *Options, identity *identity.Client, region *region.Client) *Client {
	return &Client{
		client:    client,
		namespace: namespace,
		options:   options,
		identity:  identity,
		region:    region,
	}
}

// List returns all clusters owned by the implicit control plane.
func (c *Client) List(ctx context.Context, organizationID string) (openapi.KubernetesClusters, error) {
	result := &unikornv1.KubernetesClusterList{}

	requirement, err := labels.NewRequirement(constants.OrganizationLabel, selection.Equals, []string{organizationID})
	if err != nil {
		return nil, errors.OAuth2ServerError("failed to build label selector").WithError(err)
	}

	selector := labels.NewSelector()
	selector = selector.Add(*requirement)

	options := &client.ListOptions{
		LabelSelector: selector,
	}

	if err := c.client.List(ctx, result, options); err != nil {
		return nil, errors.OAuth2ServerError("failed to list clusters").WithError(err)
	}

	slices.SortStableFunc(result.Items, unikornv1.CompareKubernetesCluster)

	return convertList(result), nil
}

// get returns the cluster.
func (c *Client) get(ctx context.Context, namespace, clusterID string) (*unikornv1.KubernetesCluster, error) {
	result := &unikornv1.KubernetesCluster{}

	if err := c.client.Get(ctx, client.ObjectKey{Namespace: namespace, Name: clusterID}, result); err != nil {
		if kerrors.IsNotFound(err) {
			return nil, errors.HTTPNotFound().WithError(err)
		}

		return nil, errors.OAuth2ServerError("unable to get cluster").WithError(err)
	}

	return result, nil
}

// GetKubeconfig returns the kubernetes configuation associated with a cluster.
func (c *Client) GetKubeconfig(ctx context.Context, organizationID, projectID, clusterID string) ([]byte, error) {
	project, err := common.New(c.client).ProjectNamespace(ctx, organizationID, projectID)
	if err != nil {
		return nil, err
	}

	cluster, err := c.get(ctx, project.Name, clusterID)
	if err != nil {
		return nil, err
	}

	// TODO: propagate the client like we do in the controllers, then code sharing
	// becomes a lot easier!
	clusterContext := &coreclient.ClusterContext{
		Client: c.client,
	}

	ctx = coreclient.NewContextWithCluster(ctx, clusterContext)

	vc := vcluster.NewControllerRuntimeClient()

	vclusterConfig, err := vc.RESTConfig(ctx, project.Name, cluster.Spec.ClusterManagerID, false)
	if err != nil {
		return nil, errors.OAuth2ServerError("failed to get control plane rest config").WithError(err)
	}

	vclusterClient, err := client.New(vclusterConfig, client.Options{})
	if err != nil {
		return nil, errors.OAuth2ServerError("failed to get control plane client").WithError(err)
	}

	objectKey := client.ObjectKey{
		Namespace: clusterID,
		Name:      clusteropenstack.KubeconfigSecretName(cluster),
	}

	secret := &corev1.Secret{}

	if err := vclusterClient.Get(ctx, objectKey, secret); err != nil {
		if kerrors.IsNotFound(err) {
			return nil, errors.HTTPNotFound().WithError(err)
		}

		return nil, errors.OAuth2ServerError("unable to get cluster configuration").WithError(err)
	}

	return secret.Data["value"], nil
}

func (c *Client) generateAllocations(ctx context.Context, organizationID string, resource *unikornv1.KubernetesCluster) (*identityapi.AllocationWrite, error) {
	flavors, err := c.region.Flavors(ctx, organizationID, resource.Spec.RegionID)
	if err != nil {
		return nil, err
	}

	var serversCommitted int

	var serversReserved int

	var gpusCommitted int

	var gpusReserved int

	// NOTE: the control plane is "free".
	for _, pool := range resource.Spec.WorkloadPools.Pools {
		serversMinimum := pool.Replicas
		serversMaximum := pool.Replicas

		if pool.Autoscaling != nil {
			serversMinimum = pool.Autoscaling.MinimumReplicas
		}

		reserved := serversMaximum - serversMinimum

		serversCommitted += serversMinimum
		serversReserved += reserved

		flavorByID := func(f regionapi.Flavor) bool {
			return f.Metadata.Id == pool.FlavorID
		}

		index := slices.IndexFunc(flavors, flavorByID)
		if index < 0 {
			return nil, fmt.Errorf("%w: flavorID does not exist", ErrConsistency)
		}

		flavor := flavors[index]

		if flavor.Spec.Gpu != nil {
			gpusCommitted += serversMinimum * flavor.Spec.Gpu.PhysicalCount
			gpusReserved += reserved * flavor.Spec.Gpu.PhysicalCount
		}
	}

	request := &identityapi.AllocationWrite{
		Metadata: coreapi.ResourceWriteMetadata{
			Name: constants.UndefinedName,
		},
		Spec: identityapi.AllocationSpec{
			Kind: "kubernetescluster",
			Id:   resource.Name,
			Allocations: identityapi.ResourceAllocationList{
				{
					Kind:      "clusters",
					Committed: 1,
					Reserved:  0,
				},
				{
					Kind:      "servers",
					Committed: serversCommitted,
					Reserved:  serversReserved,
				},
				{
					Kind:      "gpus",
					Committed: gpusCommitted,
					Reserved:  gpusReserved,
				},
			},
		},
	}

	return request, nil
}

func (c *Client) createAllocation(ctx context.Context, organizationID, projectID string, resource *unikornv1.KubernetesCluster) (*identityapi.AllocationRead, error) {
	allocations, err := c.generateAllocations(ctx, organizationID, resource)
	if err != nil {
		return nil, err
	}

	client, err := c.identity.Client(ctx)
	if err != nil {
		return nil, err
	}

	resp, err := client.PostApiV1OrganizationsOrganizationIDProjectsProjectIDAllocationsWithResponse(ctx, organizationID, projectID, *allocations)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode() != http.StatusCreated {
		return nil, coreapiutils.ExtractError(resp.StatusCode(), resp)
	}

	return resp.JSON201, nil
}

func (c *Client) updateAllocation(ctx context.Context, organizationID, projectID string, resource *unikornv1.KubernetesCluster) error {
	allocations, err := c.generateAllocations(ctx, organizationID, resource)
	if err != nil {
		return err
	}

	client, err := c.identity.Client(ctx)
	if err != nil {
		return err
	}

	resp, err := client.PutApiV1OrganizationsOrganizationIDProjectsProjectIDAllocationsAllocationIDWithResponse(ctx, organizationID, projectID, resource.Annotations[constants.AllocationAnnotation], *allocations)
	if err != nil {
		return err
	}

	if resp.StatusCode() != http.StatusOK {
		return coreapiutils.ExtractError(resp.StatusCode(), resp)
	}

	return nil
}

func (c *Client) deleteAllocation(ctx context.Context, organizationID, projectID, allocationID string) error {
	client, err := c.identity.Client(ctx)
	if err != nil {
		return err
	}

	resp, err := client.DeleteApiV1OrganizationsOrganizationIDProjectsProjectIDAllocationsAllocationIDWithResponse(ctx, organizationID, projectID, allocationID)
	if err != nil {
		return err
	}

	if resp.StatusCode() != http.StatusAccepted {
		return coreapiutils.ExtractError(resp.StatusCode(), resp)
	}

	return nil
}

func (c *Client) createIdentity(ctx context.Context, organizationID, projectID, regionID, clusterID string) (*regionapi.IdentityRead, error) {
	tags := coreapi.TagList{
		coreapi.Tag{
			Name:  constants.KubernetesClusterLabel,
			Value: clusterID,
		},
	}

	request := regionapi.PostApiV1OrganizationsOrganizationIDProjectsProjectIDIdentitiesJSONRequestBody{
		Metadata: coreapi.ResourceWriteMetadata{
			Name:        "kubernetes-cluster-" + clusterID,
			Description: ptr.To("Identity for Kubernetes cluster " + clusterID),
			Tags:        &tags,
		},
		Spec: regionapi.IdentityWriteSpec{
			RegionId: regionID,
		},
	}

	client, err := c.region.Client(ctx)
	if err != nil {
		return nil, errors.OAuth2ServerError("unable to create region client").WithError(err)
	}

	resp, err := client.PostApiV1OrganizationsOrganizationIDProjectsProjectIDIdentitiesWithResponse(ctx, organizationID, projectID, request)
	if err != nil {
		return nil, errors.OAuth2ServerError("unable to create identity").WithError(err)
	}

	if resp.StatusCode() != http.StatusCreated {
		return nil, errors.OAuth2ServerError("unable to create identity").WithError(coreapiutils.ExtractError(resp.StatusCode(), resp))
	}

	return resp.JSON201, nil
}

func (c *Client) createPhysicalNetworkOpenstack(ctx context.Context, organizationID, projectID string, cluster *unikornv1.KubernetesCluster, identity *regionapi.IdentityRead) (*regionapi.NetworkRead, error) {
	tags := coreapi.TagList{
		coreapi.Tag{
			Name:  constants.KubernetesClusterLabel,
			Value: cluster.Name,
		},
	}

	dnsNameservers := make([]string, len(cluster.Spec.Network.DNSNameservers))

	for i, ip := range cluster.Spec.Network.DNSNameservers {
		dnsNameservers[i] = ip.String()
	}

	request := regionapi.NetworkWrite{
		Metadata: coreapi.ResourceWriteMetadata{
			Name:        "kubernetes-cluster-" + cluster.Name,
			Description: ptr.To("Physical network for cluster " + cluster.Name),
			Tags:        &tags,
		},
		Spec: &regionapi.NetworkWriteSpec{
			Prefix:         cluster.Spec.Network.NodeNetwork.String(),
			DnsNameservers: dnsNameservers,
		},
	}

	client, err := c.region.Client(ctx)
	if err != nil {
		return nil, errors.OAuth2ServerError("unable to create region client").WithError(err)
	}

	resp, err := client.PostApiV1OrganizationsOrganizationIDProjectsProjectIDIdentitiesIdentityIDNetworksWithResponse(ctx, organizationID, projectID, identity.Metadata.Id, request)
	if err != nil {
		return nil, errors.OAuth2ServerError("unable to physical network").WithError(err)
	}

	if resp.StatusCode() != http.StatusCreated {
		return nil, errors.OAuth2ServerError("unable to create physical network").WithError(coreapiutils.ExtractError(resp.StatusCode(), resp))
	}

	return resp.JSON201, nil
}

func (c *Client) applyCloudSpecificConfiguration(ctx context.Context, organizationID, projectID, regionID string, allocation *identityapi.AllocationRead, identity *regionapi.IdentityRead, cluster *unikornv1.KubernetesCluster) error {
	// Save the identity ID for later cleanup.
	if cluster.Annotations == nil {
		cluster.Annotations = map[string]string{}
	}

	cluster.Annotations[constants.AllocationAnnotation] = allocation.Metadata.Id
	cluster.Annotations[constants.IdentityAnnotation] = identity.Metadata.Id

	// Apply any region specific configuration based on feature flags.
	region, err := c.region.Get(ctx, organizationID, regionID)
	if err != nil {
		return err
	}

	// Provision a vlan physical network for bare-metal nodes to attach to.
	// For now, do this for everything, given you may start with a VM only cluster
	// and suddely want some baremetal nodes.  CAPO won't allow you to change
	// networks, so play it safe.  Please note that the cluster controller will
	// automatically discover the physical network, so we don't need an annotation.
	if region.Spec.Features.PhysicalNetworks {
		physicalNetwork, err := c.createPhysicalNetworkOpenstack(ctx, organizationID, projectID, cluster, identity)
		if err != nil {
			return errors.OAuth2ServerError("failed to create physical network").WithError(err)
		}

		cluster.Annotations[constants.PhysicalNetworkAnnotation] = physicalNetwork.Metadata.Id
	}

	return nil
}

func preserveAnnotations(requested, current *unikornv1.KubernetesCluster) error {
	identity, ok := current.Annotations[constants.IdentityAnnotation]
	if !ok {
		return fmt.Errorf("%w: identity annotation missing", ErrConsistency)
	}

	allocation, ok := current.Annotations[constants.AllocationAnnotation]
	if !ok {
		return fmt.Errorf("%w: allocation annotation missing", ErrConsistency)
	}

	if requested.Annotations == nil {
		requested.Annotations = map[string]string{}
	}

	requested.Annotations[constants.IdentityAnnotation] = identity
	requested.Annotations[constants.AllocationAnnotation] = allocation

	if network, ok := current.Annotations[constants.PhysicalNetworkAnnotation]; ok {
		requested.Annotations[constants.PhysicalNetworkAnnotation] = network
	}

	return nil
}

type appBundleLister interface {
	ListCluster(ctx context.Context) (*unikornv1.KubernetesClusterApplicationBundleList, error)
}

type appBundleListerPlus interface {
	appBundleLister
	ListClusterManager(ctx context.Context) (*unikornv1.ClusterManagerApplicationBundleList, error)
}

// Create creates the implicit cluster identified by the JTW claims.
func (c *Client) Create(ctx context.Context, appclient appBundleListerPlus, organizationID, projectID string, request *openapi.KubernetesClusterWrite) (*openapi.KubernetesClusterRead, error) {
	namespace, err := common.New(c.client).ProjectNamespace(ctx, organizationID, projectID)
	if err != nil {
		return nil, err
	}

	// Implicitly create the controller manager.
	if request.Spec.ClusterManagerId == nil {
		clusterManager, err := clustermanager.NewClient(c.client).CreateImplicit(ctx, appclient, organizationID, projectID)
		if err != nil {
			return nil, err
		}

		request.Spec.ClusterManagerId = ptr.To(clusterManager.Name)
	}

	cluster, err := newGenerator(c.client, c.options, c.region, namespace.Name, organizationID, projectID).generate(ctx, appclient, request)
	if err != nil {
		return nil, err
	}

	allocation, err := c.createAllocation(ctx, organizationID, projectID, cluster)
	if err != nil {
		return nil, errors.OAuth2ServerError("failed to create quota allocation").WithError(err)
	}

	identity, err := c.createIdentity(ctx, organizationID, projectID, request.Spec.RegionId, cluster.Name)
	if err != nil {
		return nil, err
	}

	if err := c.applyCloudSpecificConfiguration(ctx, organizationID, projectID, request.Spec.RegionId, allocation, identity, cluster); err != nil {
		return nil, err
	}

	if err := c.client.Create(ctx, cluster); err != nil {
		return nil, errors.OAuth2ServerError("failed to create cluster").WithError(err)
	}

	return convert(cluster), nil
}

// Delete deletes the implicit cluster identified by the JTW claims.
func (c *Client) Delete(ctx context.Context, organizationID, projectID, clusterID string) error {
	namespace, err := common.New(c.client).ProjectNamespace(ctx, organizationID, projectID)
	if err != nil {
		return err
	}

	cluster, err := c.get(ctx, namespace.Name, clusterID)
	if err != nil {
		if kerrors.IsNotFound(err) {
			return errors.HTTPNotFound().WithError(err)
		}

		return errors.OAuth2ServerError("failed to get cluster")
	}

	if err := c.client.Delete(ctx, cluster); err != nil {
		if kerrors.IsNotFound(err) {
			return errors.HTTPNotFound().WithError(err)
		}

		return errors.OAuth2ServerError("failed to delete cluster").WithError(err)
	}

	if err := c.deleteAllocation(ctx, organizationID, projectID, cluster.Annotations[constants.AllocationAnnotation]); err != nil {
		return errors.OAuth2ServerError("failed to delete quota allocation").WithError(err)
	}

	return nil
}

// Update implements read/modify/write for the cluster.
func (c *Client) Update(ctx context.Context, appclient appBundleLister, organizationID, projectID, clusterID string, request *openapi.KubernetesClusterWrite) error {
	namespace, err := common.New(c.client).ProjectNamespace(ctx, organizationID, projectID)
	if err != nil {
		return err
	}

	if namespace.DeletionTimestamp != nil {
		return errors.OAuth2InvalidRequest("control plane is being deleted")
	}

	current, err := c.get(ctx, namespace.Name, clusterID)
	if err != nil {
		return err
	}

	required, err := newGenerator(c.client, c.options, c.region, namespace.Name, organizationID, projectID).withExisting(current).generate(ctx, appclient, request)
	if err != nil {
		return err
	}

	if err := conversion.UpdateObjectMetadata(required, current, []string{constants.IdentityAnnotation}, []string{constants.PhysicalNetworkAnnotation}); err != nil {
		return errors.OAuth2ServerError("failed to merge metadata").WithError(err)
	}

	if err := preserveAnnotations(required, current); err != nil {
		return errors.OAuth2ServerError("failed to merge annotations").WithError(err)
	}

	// Preserve networking options as if they change it'll be fairly catastrophic.
	required.Spec.Network = current.Spec.Network

	// Experience has taught me that modifying caches by accident is a bad thing
	// so be extra safe and deep copy the existing resource.
	updated := current.DeepCopy()
	updated.Labels = required.Labels
	updated.Annotations = required.Annotations
	updated.Spec = required.Spec

	if err := c.updateAllocation(ctx, organizationID, projectID, updated); err != nil {
		return errors.OAuth2ServerError("failed to update quota allocation").WithError(err)
	}

	if err := c.client.Patch(ctx, updated, client.MergeFrom(current)); err != nil {
		return errors.OAuth2ServerError("failed to patch cluster").WithError(err)
	}

	return nil
}
