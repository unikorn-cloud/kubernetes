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

// ClusterManager A cluster manager.
type ClusterManager struct {
	// Metadata Required metadata for cluster managers.
	Metadata ClusterManagerMetadata `json:"metadata"`

	// Spec A cluster manager.
	Spec ClusterManagerSpec `json:"spec"`
}

// ClusterManagerMetadata Required metadata for cluster managers.
type ClusterManagerMetadata struct {
	// CreationTime The time the resource was created.
	CreationTime time.Time `json:"creationTime"`

	// DeletionTime The time the resource was deleted.
	DeletionTime *time.Time `json:"deletionTime,omitempty"`

	// Project Where the resource is related to a project, this is populated.
	Project string `json:"project"`

	// Status The current status of the resource. Intially the status will be "Unknown" until
	// the resource is reconciled by the relevant controller. It then will transition to
	// "Provisioning" and will be ready for use when it changes to "Provisioned". The status
	// will also transition to the "Provisioning" status during an update. The
	// status will change to "Deprovisioning" when a delete request is being processed.
	// It may also change to "Error" if an unexpected error occurred during any operation.
	// Errors may be transient.
	Status string `json:"status"`
}

// ClusterManagerSpec A cluster manager.
type ClusterManagerSpec struct {
	// Name The name of the resource.
	Name string `json:"name"`
}

// ClusterManagers A list of cluster managers.
type ClusterManagers = []ClusterManager

// Flavor A flavor.
type Flavor struct {
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

// Flavors A list of flavors.
type Flavors = []Flavor

// GroupIDs A list of group IDs.
type GroupIDs = []string

// Image An image.
type Image struct {
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
	Versions ImageVersions `json:"versions"`
}

// ImageVersions Image version metadata.
type ImageVersions struct {
	// Kubernetes The kubernetes semantic version.  This should be used directly when specifying
	// Kubernetes cluster managers and workload pools in a cluster specification.
	Kubernetes string `json:"kubernetes"`

	// NvidiaDriver The nvidia driver version.
	NvidiaDriver string `json:"nvidiaDriver"`
}

// Images A list of images that are compatible with this platform.
type Images = []Image

// KubernetesCluster Kubernetes cluster read.
type KubernetesCluster struct {
	// Metadata Required metadata for clusters.
	Metadata KubernetesClusterMetadata `json:"metadata"`

	// Spec Kubernetes cluster creation parameters.
	Spec KubernetesClusterSpec `json:"spec"`
}

// KubernetesClusterAutoscaling A Kubernetes cluster workload pool autoscaling configuration. Cluster autoscaling
// must also be enabled in the cluster features.
type KubernetesClusterAutoscaling struct {
	// MinimumReplicas The minimum number of replicas to allow. Must be less than the maximum.
	MinimumReplicas int `json:"minimumReplicas"`
}

// KubernetesClusterMetadata Required metadata for clusters.
type KubernetesClusterMetadata struct {
	// Clustermanager Where the resource is scoped to a cluster manager, this is populated.
	Clustermanager string `json:"clustermanager"`

	// CreationTime The time the resource was created.
	CreationTime time.Time `json:"creationTime"`

	// DeletionTime The time the resource was deleted.
	DeletionTime *time.Time `json:"deletionTime,omitempty"`

	// Project Where the resource is related to a project, this is populated.
	Project string `json:"project"`

	// Region Where the resource is scoped to a region, this is populated,
	Region string `json:"region"`

	// Status The current status of the resource. Intially the status will be "Unknown" until
	// the resource is reconciled by the relevant controller. It then will transition to
	// "Provisioning" and will be ready for use when it changes to "Provisioned". The status
	// will also transition to the "Provisioning" status during an update. The
	// status will change to "Deprovisioning" when a delete request is being processed.
	// It may also change to "Error" if an unexpected error occurred during any operation.
	// Errors may be transient.
	Status string `json:"status"`
}

// KubernetesClusterSpec Kubernetes cluster creation parameters.
type KubernetesClusterSpec struct {
	// ClusterManager The name of the cluster manager to use, if one is not specified
	// the system will create one for you.
	ClusterManager *string `json:"clusterManager,omitempty"`

	// Name Cluster name.
	Name string `json:"name"`

	// Region The region to provision the cluster in.
	Region string `json:"region"`

	// Version The Kuebernetes version.  This should be derived from image metadata.
	Version string `json:"version"`

	// WorkloadPools A list of Kubernetes cluster workload pools.
	WorkloadPools KubernetesClusterWorkloadPools `json:"workloadPools"`
}

// KubernetesClusterWorkloadPool A Kuberntes cluster workload pool.
type KubernetesClusterWorkloadPool struct {
	// Autoscaling A Kubernetes cluster workload pool autoscaling configuration. Cluster autoscaling
	// must also be enabled in the cluster features.
	Autoscaling *KubernetesClusterAutoscaling `json:"autoscaling,omitempty"`

	// Labels Workload pool key value labels to apply on node creation.
	Labels *map[string]string `json:"labels,omitempty"`

	// Machine A Kubernetes cluster machine.
	Machine MachinePool `json:"machine"`

	// Name Workload pool name.
	Name string `json:"name"`
}

// KubernetesClusterWorkloadPools A list of Kubernetes cluster workload pools.
type KubernetesClusterWorkloadPools = []KubernetesClusterWorkloadPool

// KubernetesClusters A list of Kubernetes clusters.
type KubernetesClusters = []KubernetesCluster

// KubernetesNameParameter A Kubernetes name. Must be a valid DNS containing only lower case characters, numbers or hyphens, start and end with a character or number, and be at most 63 characters in length.
type KubernetesNameParameter = string

// MachinePool A Kubernetes cluster machine.
type MachinePool struct {
	// Disk A volume.
	Disk *Volume `json:"disk,omitempty"`

	// FlavorName Flavor name.
	FlavorName *string `json:"flavorName,omitempty"`

	// Replicas Number of machines for a statically sized pool or the maximum for an auto-scaled pool.
	Replicas *int `json:"replicas,omitempty"`
}

// Oauth2Error Generic error message.
type Oauth2Error struct {
	// Error A terse error string expanding on the HTTP error code. Errors are based on the OAuth2 specification, but are expanded with proprietary status codes for APIs other than those specified by OAuth2.
	Error Oauth2ErrorError `json:"error"`

	// ErrorDescription Verbose message describing the error.
	ErrorDescription string `json:"error_description"`
}

// Oauth2ErrorError A terse error string expanding on the HTTP error code. Errors are based on the OAuth2 specification, but are expanded with proprietary status codes for APIs other than those specified by OAuth2.
type Oauth2ErrorError string

// Project A project.
type Project struct {
	// Metadata Required metadata for projects.
	Metadata ProjectMetadata `json:"metadata"`

	// Spec A project.
	Spec ProjectSpec `json:"spec"`
}

// ProjectMetadata Required metadata for projects.
type ProjectMetadata struct {
	// CreationTime The time the resource was created.
	CreationTime time.Time `json:"creationTime"`

	// DeletionTime The time the resource was deleted.
	DeletionTime *time.Time `json:"deletionTime,omitempty"`

	// Status The current status of the resource. Intially the status will be "Unknown" until
	// the resource is reconciled by the relevant controller. It then will transition to
	// "Provisioning" and will be ready for use when it changes to "Provisioned". The status
	// will also transition to the "Provisioning" status during an update. The
	// status will change to "Deprovisioning" when a delete request is being processed.
	// It may also change to "Error" if an unexpected error occurred during any operation.
	// Errors may be transient.
	Status string `json:"status"`
}

// ProjectSpec A project.
type ProjectSpec struct {
	// GroupIDs A list of group IDs.
	GroupIDs *GroupIDs `json:"groupIDs,omitempty"`
	Name     string    `json:"name"`
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

// Volume A volume.
type Volume struct {
	// Size Disk size in GiB.
	Size int `json:"size"`
}

// ClusterManagerNameParameter A Kubernetes name. Must be a valid DNS containing only lower case characters, numbers or hyphens, start and end with a character or number, and be at most 63 characters in length.
type ClusterManagerNameParameter = KubernetesNameParameter

// ClusterNameParameter A Kubernetes name. Must be a valid DNS containing only lower case characters, numbers or hyphens, start and end with a character or number, and be at most 63 characters in length.
type ClusterNameParameter = KubernetesNameParameter

// OrganizationNameParameter A Kubernetes name. Must be a valid DNS containing only lower case characters, numbers or hyphens, start and end with a character or number, and be at most 63 characters in length.
type OrganizationNameParameter = KubernetesNameParameter

// ProjectNameParameter A Kubernetes name. Must be a valid DNS containing only lower case characters, numbers or hyphens, start and end with a character or number, and be at most 63 characters in length.
type ProjectNameParameter = KubernetesNameParameter

// RegionNameParameter A Kubernetes name. Must be a valid DNS containing only lower case characters, numbers or hyphens, start and end with a character or number, and be at most 63 characters in length.
type RegionNameParameter = KubernetesNameParameter

// ApplicationResponse A list of appications.
type ApplicationResponse = Applications

// BadRequestResponse Generic error message.
type BadRequestResponse = Oauth2Error

// ClusterManagersResponse A list of cluster managers.
type ClusterManagersResponse = ClusterManagers

// ConflictResponse Generic error message.
type ConflictResponse = Oauth2Error

// FlavorsResponse A list of flavors.
type FlavorsResponse = Flavors

// ForbiddenResponse Generic error message.
type ForbiddenResponse = Oauth2Error

// ImagesResponse A list of images that are compatible with this platform.
type ImagesResponse = Images

// InternalServerErrorResponse Generic error message.
type InternalServerErrorResponse = Oauth2Error

// KubernetesClustersResponse A list of Kubernetes clusters.
type KubernetesClustersResponse = KubernetesClusters

// NotFoundResponse Generic error message.
type NotFoundResponse = Oauth2Error

// ProjectsResponse A list of projects.
type ProjectsResponse = Projects

// RegionsResponse A list of regions.
type RegionsResponse = Regions

// UnauthorizedResponse Generic error message.
type UnauthorizedResponse = Oauth2Error

// CreateControlPlaneRequest A cluster manager.
type CreateControlPlaneRequest = ClusterManagerSpec

// CreateKubernetesClusterRequest Kubernetes cluster creation parameters.
type CreateKubernetesClusterRequest = KubernetesClusterSpec

// CreateProjectRequest A project.
type CreateProjectRequest = ProjectSpec

// PostApiV1OrganizationsOrganizationNameProjectsJSONRequestBody defines body for PostApiV1OrganizationsOrganizationNameProjects for application/json ContentType.
type PostApiV1OrganizationsOrganizationNameProjectsJSONRequestBody = ProjectSpec

// PostApiV1OrganizationsOrganizationNameProjectsProjectNameClustermanagersJSONRequestBody defines body for PostApiV1OrganizationsOrganizationNameProjectsProjectNameClustermanagers for application/json ContentType.
type PostApiV1OrganizationsOrganizationNameProjectsProjectNameClustermanagersJSONRequestBody = ClusterManagerSpec

// PutApiV1OrganizationsOrganizationNameProjectsProjectNameClustermanagersClusterManagerNameJSONRequestBody defines body for PutApiV1OrganizationsOrganizationNameProjectsProjectNameClustermanagersClusterManagerName for application/json ContentType.
type PutApiV1OrganizationsOrganizationNameProjectsProjectNameClustermanagersClusterManagerNameJSONRequestBody = ClusterManagerSpec

// PostApiV1OrganizationsOrganizationNameProjectsProjectNameClustersJSONRequestBody defines body for PostApiV1OrganizationsOrganizationNameProjectsProjectNameClusters for application/json ContentType.
type PostApiV1OrganizationsOrganizationNameProjectsProjectNameClustersJSONRequestBody = KubernetesClusterSpec

// PutApiV1OrganizationsOrganizationNameProjectsProjectNameClustersClusterNameJSONRequestBody defines body for PutApiV1OrganizationsOrganizationNameProjectsProjectNameClustersClusterName for application/json ContentType.
type PutApiV1OrganizationsOrganizationNameProjectsProjectNameClustersClusterNameJSONRequestBody = KubernetesClusterSpec
