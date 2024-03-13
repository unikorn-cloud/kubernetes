/*
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

package openstack

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"sync"

	"github.com/gophercloud/gophercloud/openstack/identity/v3/applicationcredentials"
	"github.com/gophercloud/gophercloud/openstack/identity/v3/projects"
	"github.com/gophercloud/gophercloud/openstack/identity/v3/roles"
	"github.com/gophercloud/utils/openstack/clientconfig"

	"github.com/unikorn-cloud/core/pkg/constants"
	unikornv1 "github.com/unikorn-cloud/unikorn/pkg/apis/unikorn/v1alpha1"
	"github.com/unikorn-cloud/unikorn/pkg/providers"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/util/rand"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
)

var (
	ErrKeyUndefined = errors.New("a required key was not defined")
)

type Provider struct {
	// client is Kubernetes client.
	client client.Client

	// region is the current region configuration.
	region *unikornv1.Region

	// secret is the current region secret.
	secret *corev1.Secret

	domainID string
	userID   string
	password string

	// DO NOT USE DIRECTLY, CALL AN ACCESSOR.
	_identity     *IdentityClient
	_compute      *ComputeClient
	_image        *ImageClient
	_network      *NetworkClient
	_blockStorage *BlockStorageClient

	lock sync.Mutex
}

var _ providers.Provider = &Provider{}

func New(client client.Client, region *unikornv1.Region) *Provider {
	return &Provider{
		client: client,
		region: region,
	}
}

// serviceClientRefresh updates clients if they need to e.g. in the event
// of a configuration update.
// NOTE: you MUST get the lock before calling this function.
//
//nolint:cyclop
func (p *Provider) serviceClientRefresh(ctx context.Context) error {
	refresh := false

	region := &unikornv1.Region{}

	if err := p.client.Get(ctx, client.ObjectKey{Name: p.region.Name}, region); err != nil {
		return err
	}

	// If anything changes with the configuration, referesh the clients as they may
	// do caching.
	if !reflect.DeepEqual(region.Spec.Openstack, p.region.Spec.Openstack) {
		refresh = true
	}

	secretkey := client.ObjectKey{
		Namespace: p.region.Spec.Openstack.ServiceAccountSecret.Namespace,
		Name:      p.region.Spec.Openstack.ServiceAccountSecret.Name,
	}

	secret := &corev1.Secret{}

	if err := p.client.Get(ctx, secretkey, secret); err != nil {
		return err
	}

	// If the secret hasn't beed read yet, or has changed e.g. credential rotation
	// then refresh the clients as they cache the API token.
	if p.secret == nil || !reflect.DeepEqual(secret.Data, p.secret.Data) {
		refresh = true
	}

	// Nothing to do, use what's there.
	if !refresh {
		return nil
	}

	// Create the core credential provider.
	domainID, ok := secret.Data["domain-id"]
	if !ok {
		return fmt.Errorf("%w: domain-id", ErrKeyUndefined)
	}

	userID, ok := secret.Data["user-id"]
	if !ok {
		return fmt.Errorf("%w: user-id", ErrKeyUndefined)
	}

	password, ok := secret.Data["password"]
	if !ok {
		return fmt.Errorf("%w: password", ErrKeyUndefined)
	}

	// Pass in an empty string to use the default project.
	providerClient := NewPasswordProvider(region.Spec.Openstack.Endpoint, string(userID), string(password), "")

	// Create the clients.
	identity, err := NewIdentityClient(ctx, providerClient)
	if err != nil {
		return err
	}

	compute, err := NewComputeClient(ctx, providerClient, region.Spec.Openstack.Compute)
	if err != nil {
		return err
	}

	image, err := NewImageClient(ctx, providerClient, region.Spec.Openstack.Image)
	if err != nil {
		return err
	}

	network, err := NewNetworkClient(ctx, providerClient)
	if err != nil {
		return err
	}

	blockStorage, err := NewBlockStorageClient(ctx, providerClient)
	if err != nil {
		return err
	}

	// Save the current configuration for checking next time.
	p.region = region
	p.secret = secret

	p.domainID = string(domainID)
	p.userID = string(userID)
	p.password = string(password)

	// Seve the clients
	p._identity = identity
	p._compute = compute
	p._image = image
	p._network = network
	p._blockStorage = blockStorage

	return nil
}

func (p *Provider) identity(ctx context.Context) (*IdentityClient, error) {
	p.lock.Lock()
	defer p.lock.Unlock()

	if err := p.serviceClientRefresh(ctx); err != nil {
		return nil, err
	}

	return p._identity, nil
}

func (p *Provider) compute(ctx context.Context) (*ComputeClient, error) {
	p.lock.Lock()
	defer p.lock.Unlock()

	if err := p.serviceClientRefresh(ctx); err != nil {
		return nil, err
	}

	return p._compute, nil
}

func (p *Provider) image(ctx context.Context) (*ImageClient, error) {
	p.lock.Lock()
	defer p.lock.Unlock()

	if err := p.serviceClientRefresh(ctx); err != nil {
		return nil, err
	}

	return p._image, nil
}

func (p *Provider) network(ctx context.Context) (*NetworkClient, error) {
	p.lock.Lock()
	defer p.lock.Unlock()

	if err := p.serviceClientRefresh(ctx); err != nil {
		return nil, err
	}

	return p._network, nil
}

//nolint:unused
func (p *Provider) blockStorage(ctx context.Context) (*BlockStorageClient, error) {
	p.lock.Lock()
	defer p.lock.Unlock()

	if err := p.serviceClientRefresh(ctx); err != nil {
		return nil, err
	}

	return p._blockStorage, nil
}

// Flavors list all available flavors.
func (p *Provider) Flavors(ctx context.Context) (providers.FlavorList, error) {
	computeService, err := p.compute(ctx)
	if err != nil {
		return nil, err
	}

	resources, err := computeService.Flavors(ctx)
	if err != nil {
		return nil, err
	}

	result := make(providers.FlavorList, 0, len(resources))

	for i := range resources {
		flavor := &resources[i]

		gpus, err := computeService.FlavorGPUs(flavor)
		if err != nil {
			return nil, err
		}

		// API memory is in MiB, disk is in GB
		result = append(result, providers.Flavor{
			Name:      flavor.Name,
			CPUs:      flavor.VCPUs,
			Memory:    resource.NewQuantity(int64(flavor.RAM)<<20, resource.BinarySI),
			Disk:      resource.NewScaledQuantity(int64(flavor.Disk), resource.Giga),
			GPUs:      gpus.GPUs,
			GPUVendor: providers.Nvidia,
		})
	}

	return result, nil
}

// Images lists all available images.
func (p *Provider) Images(ctx context.Context) (providers.ImageList, error) {
	imageService, err := p.image(ctx)
	if err != nil {
		return nil, err
	}

	resources, err := imageService.Images(ctx)
	if err != nil {
		return nil, err
	}

	result := make(providers.ImageList, 0, len(resources))

	for i := range resources {
		image := &resources[i]

		kuebernetesVersion, _ := image.Properties["k8s"].(string)

		result = append(result, providers.Image{
			Name:              image.Name,
			Created:           image.CreatedAt,
			Modified:          image.UpdatedAt,
			KubernetesVersion: kuebernetesVersion,
		})
	}

	return result, nil
}

const (
	// ProjectIDAnnotation records the project ID created for a cluster.
	ProjectIDAnnotation = "openstack.unikorn-cloud.org/project-id"

	// Projects are randomly named to avoid clashes, so we need to add some tags
	// in order to be able to reason about who they really belong to.  It is also
	// useful to have these in place so we can spot orphaned resources and garbage
	// collect them.
	OrganizationTag = "organization"
	ProjectTag      = "project"
	ClusterTag      = "cluster"
	ClusterUUIDTag  = "clusterUUID"
)

// projectTags defines how to tag projects.
func projectTags(cluster *unikornv1.KubernetesCluster) []string {
	tags := []string{
		OrganizationTag + "=" + cluster.Labels[constants.OrganizationLabel],
		ProjectTag + "=" + cluster.Labels[constants.ProjectLabel],
		ClusterTag + "=" + cluster.Name,
		ClusterUUIDTag + "=" + string(cluster.UID),
	}

	return tags
}

// provisionProject creates a project per-cluster.  Cluster API provider Openstack is
// somewhat broken in that networks can alias and cause all kinds of disasters, so it's
// safest to have one cluster in one project so it has its own namespace.
func (p *Provider) provisionProject(ctx context.Context, identityService *IdentityClient, cluster *unikornv1.KubernetesCluster) (*projects.Project, error) {
	name := "unikorn-" + rand.String(8)

	project, err := identityService.CreateProject(ctx, p.domainID, name, projectTags(cluster))
	if err != nil {
		return nil, err
	}

	if cluster.Annotations == nil {
		cluster.Annotations = map[string]string{}
	}

	// Annotate the cluster with the project ID so we know a) we have created it
	// and b) we can find it to make modifications e.g. add tags for garbage collection.
	cluster.Annotations[ProjectIDAnnotation] = project.ID

	return project, nil
}

// roleNameToID maps from something human readable to something Openstack will operate with
// because who doesn't like extra, slow, API calls...
func roleNameToID(roles []roles.Role, name string) (string, error) {
	for _, role := range roles {
		if role.Name == name {
			return role.ID, nil
		}
	}

	return "", fmt.Errorf("%w: role %s", ErrResourceNotFound, name)
}

// TODO: make this configurable, this is just a default.
func (p *Provider) getRequiredRoles() []string {
	defaultRoles := []string{
		"member",
		"load-balancer_member",
	}

	return defaultRoles
}

// provisionProjectRoles creates a binding between our service account and the project
// with the required roles to provision an application credential that will allow cluster
// creation, deletion and life-cycle management.
func (p *Provider) provisionProjectRoles(ctx context.Context, identityService *IdentityClient, project *projects.Project) error {
	allRoles, err := identityService.ListRoles(ctx)
	if err != nil {
		return err
	}

	for _, name := range p.getRequiredRoles() {
		roleID, err := roleNameToID(allRoles, name)
		if err != nil {
			return err
		}

		if err := identityService.CreateRoleAssignment(ctx, p.userID, project.ID, roleID); err != nil {
			return err
		}
	}

	return nil
}

func (p *Provider) provisionApplicationCredential(ctx context.Context, project *projects.Project) (*applicationcredentials.ApplicationCredential, error) {
	// Rescope to the project...
	providerClient := NewPasswordProvider(p.region.Spec.Openstack.Endpoint, p.userID, p.password, project.ID)

	projectScopedIdentity, err := NewIdentityClient(ctx, providerClient)
	if err != nil {
		return nil, err
	}

	// Application crdentials are scoped to the user, not the project, so the name needs
	// to be unique, so just use the project name.
	return projectScopedIdentity.CreateApplicationCredential(ctx, p.userID, project.Name, "IaaS lifecycle management", p.getRequiredRoles())
}

func (p *Provider) createClientConfig(cluster *unikornv1.KubernetesCluster, applicationCredential *applicationcredentials.ApplicationCredential) error {
	cloud := "cloud"

	clientConfig := &clientconfig.Clouds{
		Clouds: map[string]clientconfig.Cloud{
			cloud: {
				AuthType: clientconfig.AuthV3ApplicationCredential,
				AuthInfo: &clientconfig.AuthInfo{
					AuthURL:                     p.region.Spec.Openstack.Endpoint,
					ApplicationCredentialID:     applicationCredential.ID,
					ApplicationCredentialSecret: applicationCredential.Secret,
				},
			},
		},
	}

	clientConfigYAML, err := yaml.Marshal(clientConfig)
	if err != nil {
		return err
	}

	if cluster.Spec.Openstack == nil {
		cluster.Spec.Openstack = &unikornv1.KubernetesClusterOpenstackSpec{}
	}

	cluster.Spec.Openstack.Cloud = &cloud
	cluster.Spec.Openstack.CloudConfig = &clientConfigYAML

	return nil
}

// ConfigureCluster does any provider specific configuration for a cluster.
func (p *Provider) ConfigureCluster(ctx context.Context, cluster *unikornv1.KubernetesCluster) error {
	identityService, err := p.identity(ctx)
	if err != nil {
		return err
	}

	project, err := p.provisionProject(ctx, identityService, cluster)
	if err != nil {
		return err
	}

	if err := p.provisionProjectRoles(ctx, identityService, project); err != nil {
		return err
	}

	applicationCredential, err := p.provisionApplicationCredential(ctx, project)
	if err != nil {
		return err
	}

	if err := p.createClientConfig(cluster, applicationCredential); err != nil {
		return err
	}

	networkService, err := p.network(ctx)
	if err != nil {
		return err
	}

	if cluster.Spec.Openstack == nil {
		cluster.Spec.Openstack = &unikornv1.KubernetesClusterOpenstackSpec{}
	}

	if cluster.Spec.Openstack.ExternalNetworkID == nil {
		externalNetwork, err := networkService.defaultExternalNetwork(ctx)
		if err != nil {
			return err
		}

		cluster.Spec.Openstack.ExternalNetworkID = &externalNetwork.ID
	}

	return nil
}
