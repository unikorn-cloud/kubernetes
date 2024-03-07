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

	unikornv1 "github.com/unikorn-cloud/unikorn/pkg/apis/unikorn/v1alpha1"
	"github.com/unikorn-cloud/unikorn/pkg/providers"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"

	"sigs.k8s.io/controller-runtime/pkg/client"
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

	// Save the current configuration for checking next time.
	p.region = region
	p.secret = secret

	// Create the core credential provider.
	username, ok := secret.Data["username"]
	if !ok {
		return fmt.Errorf("%w: username", ErrKeyUndefined)
	}

	password, ok := secret.Data["password"]
	if !ok {
		return fmt.Errorf("%w: password", ErrKeyUndefined)
	}

	providerClient := NewPasswordProvider(region.Spec.Openstack.Endpoint, string(username), string(password))

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

	p._identity = identity
	p._compute = compute
	p._image = image
	p._network = network
	p._blockStorage = blockStorage

	return nil
}

//nolint:unused
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

// ConfigureCluster does any provider specific configuration for a cluster.
func (p *Provider) ConfigureCluster(ctx context.Context, cluster *unikornv1.KubernetesCluster) error {
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
