// Package generated provides primitives to interact with the openapi HTTP API.
//
// Code generated by github.com/deepmap/oapi-codegen version v1.12.4 DO NOT EDIT.
package generated

import (
	"time"
)

const (
	Oauth2AuthenticationScopes = "oauth2Authentication.Scopes"
)

// Defines values for Oauth2ErrorError.
const (
	AccessDenied            Oauth2ErrorError = "access_denied"
	Conflict                Oauth2ErrorError = "conflict"
	Forbidden               Oauth2ErrorError = "forbidden"
	InvalidClient           Oauth2ErrorError = "invalid_client"
	InvalidGrant            Oauth2ErrorError = "invalid_grant"
	InvalidRequest          Oauth2ErrorError = "invalid_request"
	InvalidScope            Oauth2ErrorError = "invalid_scope"
	MethodNotAllowed        Oauth2ErrorError = "method_not_allowed"
	NotFound                Oauth2ErrorError = "not_found"
	ServerError             Oauth2ErrorError = "server_error"
	TemporarilyUnavailable  Oauth2ErrorError = "temporarily_unavailable"
	UnauthorizedClient      Oauth2ErrorError = "unauthorized_client"
	UnsupportedGrantType    Oauth2ErrorError = "unsupported_grant_type"
	UnsupportedMediaType    Oauth2ErrorError = "unsupported_media_type"
	UnsupportedResponseType Oauth2ErrorError = "unsupported_response_type"
)

// Application An application.
type Application struct {
	// Description Verbose description of what the application provides.
	Description string `json:"description"`

	// Documentation Documentation link for the application.
	Documentation string `json:"documentation"`

	// HumanReadableName Human readable application name.
	HumanReadableName string `json:"humanReadableName"`

	// Icon A base64 encoded SVG icon.  This should work in both light and dark themes.
	Icon []byte `json:"icon"`

	// License The license under which the application is released.
	License string `json:"license"`

	// Name Unique application name.
	Name string `json:"name"`

	// Tags A set of tags for filtering applications.
	Tags *ApplicationTags `json:"tags,omitempty"`

	// Versions A set of application versions.
	Versions ApplicationVersions `json:"versions"`
}

// ApplicationBundle A bundle of applications. This forms the basis of resource versions. Bundles marked
// as preview should not be selected by default, and end of life bundles should not be
// used to avoid unnecessary upgrades. If enabled, automatic upgrades will occur if
// a newer version of a bundle exists that is not in preview. When a bundle's end of
// life expires, resources will undergo a foreced upgrade, regardless of whether
// automatic upgrade is enabled for a resource or not.
type ApplicationBundle struct {
	// EndOfLife When the bundle is end-of-life.
	EndOfLife *time.Time `json:"endOfLife,omitempty"`

	// Name The resource name.
	Name string `json:"name"`

	// Preview Whether the bundle is in preview.
	Preview *bool `json:"preview,omitempty"`

	// Version The bundle version.
	Version string `json:"version"`
}

// ApplicationBundleAutoUpgrade When specified, enables auto upgrade of application bundles. All resources will be
// automatically upgraded if the currently selected bundle is end of life.
type ApplicationBundleAutoUpgrade struct {
	// DaysOfWeek Days of the week and time windows that permit operations to be performed in.
	DaysOfWeek *AutoUpgradeDaysOfWeek `json:"daysOfWeek,omitempty"`
}

// ApplicationBundles A list of application bundles.
type ApplicationBundles = []ApplicationBundle

// ApplicationDependencies A set of applications that will be installed before this application.
type ApplicationDependencies = []ApplicationDependency

// ApplicationDependency An application dependency.
type ApplicationDependency struct {
	// Name The application name.
	Name string `json:"name"`
}

// ApplicationRecommends A set of recommended application that may be installed after this application.
type ApplicationRecommends = []ApplicationDependency

// ApplicationTags A set of tags for filtering applications.
type ApplicationTags = []string

// ApplicationVersion An application version.
type ApplicationVersion struct {
	// Dependencies A set of applications that will be installed before this application.
	Dependencies *ApplicationDependencies `json:"dependencies,omitempty"`

	// Recommends A set of recommended application that may be installed after this application.
	Recommends *ApplicationRecommends `json:"recommends,omitempty"`

	// Version The application's Helm chart version.
	Version string `json:"version"`
}

// ApplicationVersions A set of application versions.
type ApplicationVersions = []ApplicationVersion

// Applications A list of appications.
type Applications = []Application

// AutoUpgradeDaysOfWeek Days of the week and time windows that permit operations to be performed in.
type AutoUpgradeDaysOfWeek struct {
	// Friday A time window that wraps into the next day if required.
	Friday *TimeWindow `json:"friday,omitempty"`

	// Monday A time window that wraps into the next day if required.
	Monday *TimeWindow `json:"monday,omitempty"`

	// Saturday A time window that wraps into the next day if required.
	Saturday *TimeWindow `json:"saturday,omitempty"`

	// Sunday A time window that wraps into the next day if required.
	Sunday *TimeWindow `json:"sunday,omitempty"`

	// Thursday A time window that wraps into the next day if required.
	Thursday *TimeWindow `json:"thursday,omitempty"`

	// Tuesday A time window that wraps into the next day if required.
	Tuesday *TimeWindow `json:"tuesday,omitempty"`

	// Wednesday A time window that wraps into the next day if required.
	Wednesday *TimeWindow `json:"wednesday,omitempty"`
}

// ControlPlane A control plane.
type ControlPlane struct {
	// ApplicationBundle A bundle of applications. This forms the basis of resource versions. Bundles marked
	// as preview should not be selected by default, and end of life bundles should not be
	// used to avoid unnecessary upgrades. If enabled, automatic upgrades will occur if
	// a newer version of a bundle exists that is not in preview. When a bundle's end of
	// life expires, resources will undergo a foreced upgrade, regardless of whether
	// automatic upgrade is enabled for a resource or not.
	ApplicationBundle *ApplicationBundle `json:"applicationBundle,omitempty"`

	// ApplicationBundleAutoUpgrade Whether to enable auto upgrade or not.
	ApplicationBundleAutoUpgrade *bool `json:"applicationBundleAutoUpgrade,omitempty"`

	// ApplicationBundleAutoUpgradeSchedule When specified, enables auto upgrade of application bundles. All resources will be
	// automatically upgraded if the currently selected bundle is end of life.
	ApplicationBundleAutoUpgradeSchedule *ApplicationBundleAutoUpgrade `json:"applicationBundleAutoUpgradeSchedule,omitempty"`

	// Metadata A resources's metadata
	Metadata *ResourceMetadata `json:"metadata,omitempty"`

	// Name The name of the resource.
	Name string `json:"name"`
}

// ControlPlanes A list of control planes.
type ControlPlanes = []ControlPlane

// Hour An hour of the day in UTC.
type Hour = int

// KubernetesCluster Kubernetes cluster creation parameters.
type KubernetesCluster struct {
	// Api Kubernetes API settings.
	Api *KubernetesClusterAPI `json:"api,omitempty"`

	// ApplicationBundle A bundle of applications. This forms the basis of resource versions. Bundles marked
	// as preview should not be selected by default, and end of life bundles should not be
	// used to avoid unnecessary upgrades. If enabled, automatic upgrades will occur if
	// a newer version of a bundle exists that is not in preview. When a bundle's end of
	// life expires, resources will undergo a foreced upgrade, regardless of whether
	// automatic upgrade is enabled for a resource or not.
	ApplicationBundle *ApplicationBundle `json:"applicationBundle,omitempty"`

	// ApplicationBundleAutoUpgrade When specified, enables auto upgrade of application bundles. All resources will be
	// automatically upgraded if the currently selected bundle is end of life.
	ApplicationBundleAutoUpgrade *ApplicationBundleAutoUpgrade `json:"applicationBundleAutoUpgrade,omitempty"`

	// ControlPlane A Kubernetes cluster machine.
	ControlPlane *OpenstackMachinePool `json:"controlPlane,omitempty"`

	// Metadata A resources's metadata
	Metadata *ResourceMetadata `json:"metadata,omitempty"`

	// Name Cluster name.
	Name string `json:"name"`

	// Network A kubernetes cluster network settings.
	Network *KubernetesClusterNetwork `json:"network,omitempty"`

	// Openstack Kubernetes cluster creation OpenStack parameters.
	Openstack *KubernetesClusterOpenStack `json:"openstack,omitempty"`

	// Region The region to provision the cluster in.
	Region string `json:"region"`

	// Version The Kuebernetes version.  This should be derived from image metadata.
	Version string `json:"version"`

	// WorkloadPools A list of Kubernetes cluster workload pools.
	WorkloadPools KubernetesClusterWorkloadPools `json:"workloadPools"`
}

// KubernetesClusterAPI Kubernetes API settings.
type KubernetesClusterAPI struct {
	// AllowedPrefixes Set of address prefixes to allow access to the Kubernetes API.
	AllowedPrefixes *[]string `json:"allowedPrefixes,omitempty"`

	// SubjectAlternativeNames Set of non-standard X.509 SANs to add to the API certificate.
	SubjectAlternativeNames *[]string `json:"subjectAlternativeNames,omitempty"`
}

// KubernetesClusterAutoscaling A Kubernetes cluster workload pool autoscaling configuration. Cluster autoscaling
// must also be enabled in the cluster features.
type KubernetesClusterAutoscaling struct {
	// MinimumReplicas The minimum number of replicas to allow. Must be less than the maximum.
	MinimumReplicas int `json:"minimumReplicas"`
}

// KubernetesClusterNetwork A kubernetes cluster network settings.
type KubernetesClusterNetwork struct {
	// DnsNameservers A list of DNS name server to use.
	DnsNameservers *[]string `json:"dnsNameservers,omitempty"`

	// NodePrefix Network prefix to provision nodes in. Must be a valid CIDR block.
	NodePrefix *string `json:"nodePrefix,omitempty"`

	// PodPrefix Network prefix to provision pods in. Must be a valid CIDR block.
	PodPrefix *string `json:"podPrefix,omitempty"`

	// ServicePrefix Network prefix to provision services in. Must be a valid CIDR block.
	ServicePrefix *string `json:"servicePrefix,omitempty"`
}

// KubernetesClusterOpenStack Kubernetes cluster creation OpenStack parameters.
type KubernetesClusterOpenStack struct {
	// ComputeAvailabilityZone Compute availability zone for control plane, and workload pool default.
	ComputeAvailabilityZone *string `json:"computeAvailabilityZone,omitempty"`

	// ExternalNetworkID OpenStack external network ID.
	ExternalNetworkID *string `json:"externalNetworkID,omitempty"`

	// SshKeyName OpenStack SSH Key to install on all machines.
	SshKeyName *string `json:"sshKeyName,omitempty"`

	// VolumeAvailabilityZone Volume availability zone for control plane, and workload pool default.
	VolumeAvailabilityZone *string `json:"volumeAvailabilityZone,omitempty"`
}

// KubernetesClusterWorkloadPool A Kuberntes cluster workload pool.
type KubernetesClusterWorkloadPool struct {
	// Autoscaling A Kubernetes cluster workload pool autoscaling configuration. Cluster autoscaling
	// must also be enabled in the cluster features.
	Autoscaling *KubernetesClusterAutoscaling `json:"autoscaling,omitempty"`

	// AvailabilityZone Workload pool availability zone. Overrides the cluster default.
	AvailabilityZone *string `json:"availabilityZone,omitempty"`

	// Labels Workload pool key value labels to apply on node creation.
	Labels *map[string]string `json:"labels,omitempty"`

	// Machine A Kubernetes cluster machine.
	Machine OpenstackMachinePool `json:"machine"`

	// Name Workload pool name.
	Name string `json:"name"`
}

// KubernetesClusterWorkloadPools A list of Kubernetes cluster workload pools.
type KubernetesClusterWorkloadPools = []KubernetesClusterWorkloadPool

// KubernetesClusters A list of Kubernetes clusters.
type KubernetesClusters = []KubernetesCluster

// KubernetesNameParameter A Kubernetes name. Must be a valid DNS containing only lower case characters, numbers or hyphens, start and end with a character or number, and be at most 63 characters in length.
type KubernetesNameParameter = string

// Oauth2Error Generic error message.
type Oauth2Error struct {
	// Error A terse error string expanding on the HTTP error code. Errors are based on the OAuth2 specification, but are expanded with proprietary status codes for APIs other than those specified by OAuth2.
	Error Oauth2ErrorError `json:"error"`

	// ErrorDescription Verbose message describing the error.
	ErrorDescription string `json:"error_description"`
}

// Oauth2ErrorError A terse error string expanding on the HTTP error code. Errors are based on the OAuth2 specification, but are expanded with proprietary status codes for APIs other than those specified by OAuth2.
type Oauth2ErrorError string

// OpenstackAvailabilityZone An OpenStack availability zone.
type OpenstackAvailabilityZone struct {
	// Name The availability zone name.
	Name string `json:"name"`
}

// OpenstackAvailabilityZones A list of OpenStack availability zones.
type OpenstackAvailabilityZones = []OpenstackAvailabilityZone

// OpenstackExternalNetwork An OpenStack external network.
type OpenstackExternalNetwork struct {
	// Id OpenStack external network ID.
	Id string `json:"id"`

	// Name Opestack external network name.
	Name string `json:"name"`
}

// OpenstackExternalNetworks A list of OpenStack external networks.
type OpenstackExternalNetworks = []OpenstackExternalNetwork

// OpenstackFlavor An OpenStack flavor.
type OpenstackFlavor struct {
	// Cpus The number of CPUs.
	Cpus int `json:"cpus"`

	// Disk The amount of ephemeral disk in GB.
	Disk int `json:"disk"`

	// Gpus The number of GPUs, if not set there are none.
	Gpus *int `json:"gpus,omitempty"`

	// Id The unique flavor ID.
	Id string `json:"id"`

	// Memory The amount of memory in GiB.
	Memory int `json:"memory"`

	// Name The flavor name.
	Name string `json:"name"`
}

// OpenstackFlavors A list of OpenStack flavors.
type OpenstackFlavors = []OpenstackFlavor

// OpenstackImage And OpenStack image.
type OpenstackImage struct {
	// Created Time when the image was created. Images with a newer creation time should
	// be favoured over older images as they will contain updates and fewer vulnerabilities.
	Created time.Time `json:"created"`

	// Id The unique image ID.
	Id string `json:"id"`

	// Modified Time when the image was last modified.
	Modified time.Time `json:"modified"`

	// Name The image name.
	Name string `json:"name"`

	// Versions Image version metadata.
	Versions OpenstackImageVersions `json:"versions"`
}

// OpenstackImageVersions Image version metadata.
type OpenstackImageVersions struct {
	// Kubernetes The kubernetes semantic version.  This should be used directly when specifying
	// Kubernetes control planes and workload pools in a cluster specification.
	Kubernetes string `json:"kubernetes"`

	// NvidiaDriver The nvidia driver version.
	NvidiaDriver string `json:"nvidiaDriver"`
}

// OpenstackImages A list of OpenStack images that are compatible with this platform.
type OpenstackImages = []OpenstackImage

// OpenstackKeyPair An OpenStack SSH key pair.
type OpenstackKeyPair struct {
	// Name The key pair name.
	Name string `json:"name"`
}

// OpenstackKeyPairs A list of OpenStack key pairs.
type OpenstackKeyPairs = []OpenstackKeyPair

// OpenstackMachinePool A Kubernetes cluster machine.
type OpenstackMachinePool struct {
	// Disk An OpenStack volume.
	Disk *OpenstackVolume `json:"disk,omitempty"`

	// FlavorName OpenStack flavor name.
	FlavorName *string `json:"flavorName,omitempty"`

	// ImageName OpenStack image name.
	ImageName *string `json:"imageName,omitempty"`

	// Replicas Number of machines for a statically sized pool or the maximum for an auto-scaled pool.
	Replicas *int `json:"replicas,omitempty"`
}

// OpenstackVolume An OpenStack volume.
type OpenstackVolume struct {
	// AvailabilityZone Volume availability zone. Overrides the cluster default.
	AvailabilityZone *string `json:"availabilityZone,omitempty"`

	// Size Disk size in GiB.
	Size int `json:"size"`
}

// Project A project.
type Project struct {
	// Metadata A resources's metadata
	Metadata *ResourceMetadata `json:"metadata,omitempty"`
	Name     string            `json:"name"`
}

// Projects A list of projects.
type Projects = []Project

// Region A region.
type Region struct {
	// Name The region name.
	Name string `json:"name"`
}

// Regions A list of regions.
type Regions = []Region

// ResourceMetadata A resources's metadata
type ResourceMetadata struct {
	// Controlplane Where the resource is scoped to a control plane, this is populated.
	Controlplane *string `json:"controlplane,omitempty"`

	// CreationTime The time the resource was created.
	CreationTime time.Time `json:"creationTime"`

	// DeletionTime The time the resource was deleted.
	DeletionTime *time.Time `json:"deletionTime,omitempty"`

	// Project Where the resource is related to a project, this is populated.
	Project *string `json:"project,omitempty"`

	// Region Where the resource is scoped to a region, this is populated,
	Region *string `json:"region,omitempty"`

	// Status The current status of the resource. Intially the status will be "Unknown" until
	// the resource is reconciled by the relevant controller. It then will transition to
	// "Provisioning" and will be ready for use when it changes to "Provisioned". The status
	// will also transition to the "Provisioning" status during an update. The
	// status will change to "Deprovisioning" when a delete request is being processed.
	// It may also change to "Error" if an unexpected error occurred during any operation.
	// Errors may be transient.
	Status string `json:"status"`
}

// TimeWindow A time window that wraps into the next day if required.
type TimeWindow struct {
	// End An hour of the day in UTC.
	End Hour `json:"end"`

	// Start An hour of the day in UTC.
	Start Hour `json:"start"`
}

// ClusterNameParameter A Kubernetes name. Must be a valid DNS containing only lower case characters, numbers or hyphens, start and end with a character or number, and be at most 63 characters in length.
type ClusterNameParameter = KubernetesNameParameter

// ControlPlaneNameParameter A Kubernetes name. Must be a valid DNS containing only lower case characters, numbers or hyphens, start and end with a character or number, and be at most 63 characters in length.
type ControlPlaneNameParameter = KubernetesNameParameter

// ProjectNameParameter A Kubernetes name. Must be a valid DNS containing only lower case characters, numbers or hyphens, start and end with a character or number, and be at most 63 characters in length.
type ProjectNameParameter = KubernetesNameParameter

// RegionNameParameter A Kubernetes name. Must be a valid DNS containing only lower case characters, numbers or hyphens, start and end with a character or number, and be at most 63 characters in length.
type RegionNameParameter = KubernetesNameParameter

// ApplicationBundleResponse A list of application bundles.
type ApplicationBundleResponse = ApplicationBundles

// ApplicationResponse A list of appications.
type ApplicationResponse = Applications

// BadRequestResponse Generic error message.
type BadRequestResponse = Oauth2Error

// ConflictResponse Generic error message.
type ConflictResponse = Oauth2Error

// ControlPlanesResponse A list of control planes.
type ControlPlanesResponse = ControlPlanes

// ForbiddenResponse Generic error message.
type ForbiddenResponse = Oauth2Error

// InternalServerErrorResponse Generic error message.
type InternalServerErrorResponse = Oauth2Error

// KubernetesClustersResponse A list of Kubernetes clusters.
type KubernetesClustersResponse = KubernetesClusters

// NotFoundResponse Generic error message.
type NotFoundResponse = Oauth2Error

// OpenstackBlockStorageAvailabilityZonesResponse A list of OpenStack availability zones.
type OpenstackBlockStorageAvailabilityZonesResponse = OpenstackAvailabilityZones

// OpenstackComputeAvailabilityZonesResponse A list of OpenStack availability zones.
type OpenstackComputeAvailabilityZonesResponse = OpenstackAvailabilityZones

// OpenstackExternalNetworksResponse A list of OpenStack external networks.
type OpenstackExternalNetworksResponse = OpenstackExternalNetworks

// OpenstackFlavorsResponse A list of OpenStack flavors.
type OpenstackFlavorsResponse = OpenstackFlavors

// OpenstackImagesResponse A list of OpenStack images that are compatible with this platform.
type OpenstackImagesResponse = OpenstackImages

// OpenstackKeyPairsResponse A list of OpenStack key pairs.
type OpenstackKeyPairsResponse = OpenstackKeyPairs

// ProjectsResponse A list of projects.
type ProjectsResponse = Projects

// RegionsResponse A list of regions.
type RegionsResponse = Regions

// UnauthorizedResponse Generic error message.
type UnauthorizedResponse = Oauth2Error

// CreateControlPlaneRequest A control plane.
type CreateControlPlaneRequest = ControlPlane

// CreateKubernetesClusterRequest Kubernetes cluster creation parameters.
type CreateKubernetesClusterRequest = KubernetesCluster

// CreateProjectRequest A project.
type CreateProjectRequest = Project

// PostApiV1ProjectsJSONRequestBody defines body for PostApiV1Projects for application/json ContentType.
type PostApiV1ProjectsJSONRequestBody = Project

// PostApiV1ProjectsProjectNameControlplanesJSONRequestBody defines body for PostApiV1ProjectsProjectNameControlplanes for application/json ContentType.
type PostApiV1ProjectsProjectNameControlplanesJSONRequestBody = ControlPlane

// PutApiV1ProjectsProjectNameControlplanesControlPlaneNameJSONRequestBody defines body for PutApiV1ProjectsProjectNameControlplanesControlPlaneName for application/json ContentType.
type PutApiV1ProjectsProjectNameControlplanesControlPlaneNameJSONRequestBody = ControlPlane

// PostApiV1ProjectsProjectNameControlplanesControlPlaneNameClustersJSONRequestBody defines body for PostApiV1ProjectsProjectNameControlplanesControlPlaneNameClusters for application/json ContentType.
type PostApiV1ProjectsProjectNameControlplanesControlPlaneNameClustersJSONRequestBody = KubernetesCluster

// PutApiV1ProjectsProjectNameControlplanesControlPlaneNameClustersClusterNameJSONRequestBody defines body for PutApiV1ProjectsProjectNameControlplanesControlPlaneNameClustersClusterName for application/json ContentType.
type PutApiV1ProjectsProjectNameControlplanesControlPlaneNameClustersClusterNameJSONRequestBody = KubernetesCluster
