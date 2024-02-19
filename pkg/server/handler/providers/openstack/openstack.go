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

package openstack

import (
	goerrors "errors"
	"fmt"
	"net/http"
	"reflect"
	"slices"
	"sort"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/servergroups"
	"github.com/gophercloud/gophercloud/openstack/identity/v3/applicationcredentials"
	lru "github.com/hashicorp/golang-lru/v2"

	"github.com/unikorn-cloud/identity/pkg/oauth2"
	unikornv1 "github.com/unikorn-cloud/unikorn/pkg/apis/unikorn/v1alpha1"
	"github.com/unikorn-cloud/unikorn/pkg/providers/openstack"
	"github.com/unikorn-cloud/unikorn/pkg/server/errors"
	"github.com/unikorn-cloud/unikorn/pkg/server/generated"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	ErrResourceNotFound = goerrors.New("resource not found")
)

// covertError takes a generic gophercloud error and converts it into something
// more useful to our clients.
// / NOTE: at present we're only concerned with 401s because it reacts badly with
// the UI if we return a 500, when a 401 would cause a reauthentication and make
// the bad behaviour go away.
func covertError(err error) error {
	var err401 gophercloud.ErrDefault401

	if goerrors.As(err, &err401) {
		return errors.OAuth2AccessDenied("provider request denied").WithError(err)
	}

	var err403 gophercloud.ErrDefault403

	if goerrors.As(err, &err403) {
		return errors.HTTPForbidden("provider request forbidden, ensure you have the correct roles assigned to your user")
	}

	v := reflect.ValueOf(err)

	return errors.OAuth2ServerError("provider error unhandled: " + v.Type().Name()).WithError(err)
}

// Openstack provides an HTTP handler for Openstack resources.
type Openstack struct {
	client client.Client

	options *unikornv1.RegionOpenstackSpec

	// Cache clients as that's quite expensive.
	identityClientCache     *lru.Cache[string, *openstack.IdentityClient]
	computeClientCache      *lru.Cache[string, *openstack.ComputeClient]
	blockStorageClientCache *lru.Cache[string, *openstack.BlockStorageClient]
	networkClientCache      *lru.Cache[string, *openstack.NetworkClient]
	imageClientCache        *lru.Cache[string, *openstack.ImageClient]
}

// New returns a new initialized Openstack handler.
func New(client client.Client, options *unikornv1.RegionOpenstackSpec) (*Openstack, error) {
	identityClientCache, err := lru.New[string, *openstack.IdentityClient](1024)
	if err != nil {
		return nil, err
	}

	computeClientCache, err := lru.New[string, *openstack.ComputeClient](1024)
	if err != nil {
		return nil, err
	}

	blockStorageClientCache, err := lru.New[string, *openstack.BlockStorageClient](1024)
	if err != nil {
		return nil, err
	}

	networkClientCache, err := lru.New[string, *openstack.NetworkClient](1024)
	if err != nil {
		return nil, err
	}

	imageClientCache, err := lru.New[string, *openstack.ImageClient](1024)
	if err != nil {
		return nil, err
	}

	o := &Openstack{
		client:                  client,
		options:                 options,
		identityClientCache:     identityClientCache,
		computeClientCache:      computeClientCache,
		blockStorageClientCache: blockStorageClientCache,
		networkClientCache:      networkClientCache,
		imageClientCache:        imageClientCache,
	}

	return o, nil
}

func cacheKey(r *http.Request) (string, error) {
	claims, err := oauth2.ClaimsFromContext(r.Context())
	if err != nil {
		return "", errors.OAuth2ServerError("failed get cacheKe claims").WithError(err)
	}

	return claims.ID, nil
}

func getUser(r *http.Request) (string, error) {
	/*
		claims, err := oauth2.ClaimsFromContext(r.Context())
		if err != nil {
			return "", errors.OAuth2ServerError("failed get cacheKe claims").WithError(err)
		}

		if claims.UnikornClaims == nil {
			return "", errors.OAuth2ServerError("failed get cacheKe claim")
		}

		return claims.UnikornClaims.User, nil
	*/
	return "", nil
}

func (o *Openstack) providerClient() openstack.Provider {
	return openstack.NewApplicationCredentialProvider(o.client, o.options)
}

func (o *Openstack) IdentityClient(r *http.Request) (*openstack.IdentityClient, error) {
	cacheKey, err := cacheKey(r)
	if err != nil {
		return nil, err
	}

	if client, ok := o.identityClientCache.Get(cacheKey); ok {
		return client, nil
	}

	client, err := openstack.NewIdentityClient(r.Context(), o.providerClient())
	if err != nil {
		return nil, errors.OAuth2ServerError("failed get identity client").WithError(err)
	}

	o.identityClientCache.Add(cacheKey, client)

	return client, nil
}

func (o *Openstack) ComputeClient(r *http.Request) (*openstack.ComputeClient, error) {
	cacheKe, err := cacheKey(r)
	if err != nil {
		return nil, err
	}

	if client, ok := o.computeClientCache.Get(cacheKe); ok {
		return client, nil
	}

	client, err := openstack.NewComputeClient(r.Context(), o.options.Compute, o.providerClient())
	if err != nil {
		return nil, errors.OAuth2ServerError("failed get compute client").WithError(err)
	}

	o.computeClientCache.Add(cacheKe, client)

	return client, nil
}

func (o *Openstack) BlockStorageClient(r *http.Request) (*openstack.BlockStorageClient, error) {
	cacheKe, err := cacheKey(r)
	if err != nil {
		return nil, err
	}

	if client, ok := o.blockStorageClientCache.Get(cacheKe); ok {
		return client, nil
	}

	client, err := openstack.NewBlockStorageClient(r.Context(), o.providerClient())
	if err != nil {
		return nil, errors.OAuth2ServerError("failed get block storage client").WithError(err)
	}

	o.blockStorageClientCache.Add(cacheKe, client)

	return client, nil
}

func (o *Openstack) NetworkClient(r *http.Request) (*openstack.NetworkClient, error) {
	cacheKe, err := cacheKey(r)
	if err != nil {
		return nil, err
	}

	if client, ok := o.networkClientCache.Get(cacheKe); ok {
		return client, nil
	}

	client, err := openstack.NewNetworkClient(r.Context(), o.providerClient())
	if err != nil {
		return nil, errors.OAuth2ServerError("failed get network client").WithError(err)
	}

	o.networkClientCache.Add(cacheKe, client)

	return client, nil
}

func (o *Openstack) ImageClient(r *http.Request) (*openstack.ImageClient, error) {
	cacheKe, err := cacheKey(r)
	if err != nil {
		return nil, err
	}

	if client, ok := o.imageClientCache.Get(cacheKe); ok {
		return client, nil
	}

	client, err := openstack.NewImageClient(r.Context(), o.providerClient(), o.options.Image)
	if err != nil {
		return nil, errors.OAuth2ServerError("failed get image client").WithError(err)
	}

	o.imageClientCache.Add(cacheKe, client)

	return client, nil
}

func (o *Openstack) ListAvailabilityZonesCompute(r *http.Request) (generated.OpenstackAvailabilityZones, error) {
	client, err := o.ComputeClient(r)
	if err != nil {
		return nil, errors.OAuth2ServerError("failed get compute client").WithError(err)
	}

	result, err := client.AvailabilityZones(r.Context())
	if err != nil {
		return nil, covertError(err)
	}

	azs := make(generated.OpenstackAvailabilityZones, len(result))

	for i, az := range result {
		azs[i].Name = az.ZoneName
	}

	return azs, nil
}

func (o *Openstack) ListAvailabilityZonesBlockStorage(r *http.Request) (generated.OpenstackAvailabilityZones, error) {
	client, err := o.BlockStorageClient(r)
	if err != nil {
		return nil, errors.OAuth2ServerError("failed get block storage client").WithError(err)
	}

	result, err := client.AvailabilityZones(r.Context())
	if err != nil {
		return nil, covertError(err)
	}

	azs := make(generated.OpenstackAvailabilityZones, len(result))

	for i, az := range result {
		azs[i].Name = az.ZoneName
	}

	return azs, nil
}

func (o *Openstack) ListExternalNetworks(r *http.Request) (interface{}, error) {
	client, err := o.NetworkClient(r)
	if err != nil {
		return nil, errors.OAuth2ServerError("failed get network client").WithError(err)
	}

	result, err := client.ExternalNetworks(r.Context())
	if err != nil {
		return nil, covertError(err)
	}

	externalNetworks := make(generated.OpenstackExternalNetworks, len(result))

	for i, externalNetwork := range result {
		externalNetworks[i].Id = externalNetwork.ID
		externalNetworks[i].Name = externalNetwork.Name
	}

	return externalNetworks, nil
}

// convertFlavor traslates from Openstack's mess into our API types.
func convertFlavor(client *openstack.ComputeClient, flavor *openstack.Flavor) (*generated.OpenstackFlavor, error) {
	f := &generated.OpenstackFlavor{
		Id:     flavor.ID,
		Name:   flavor.Name,
		Cpus:   flavor.VCPUs,
		Memory: flavor.RAM >> 10, // Convert MiB to GiB
		Disk:   flavor.Disk,
	}

	gpu, err := client.FlavorGPUs(flavor)
	if err != nil {
		return nil, errors.OAuth2ServerError("unable to get GPU flavor metadata").WithError(err)
	}

	if gpu != nil {
		f.Gpus = &gpu.GPUs
	}

	return f, nil
}

type flavorSortWrapper struct {
	f generated.OpenstackFlavors
}

func (w flavorSortWrapper) Len() int {
	return len(w.f)
}

func (w flavorSortWrapper) Less(i, j int) bool {
	// Sort by GPUs, we want these to have precedence, we are selling GPUs
	// after all.
	if w.f[i].Gpus != nil {
		if w.f[j].Gpus == nil {
			return true
		}

		// Those with the smallest number of GPUs go first, we want to
		// prevent over provisioning.
		if *w.f[i].Gpus < *w.f[j].Gpus {
			return true
		}
	}

	if w.f[j].Gpus != nil && w.f[i].Gpus == nil {
		return false
	}

	// If the GPUs are the same, sort by CPUs.
	if w.f[i].Cpus < w.f[j].Cpus {
		return true
	}

	return false
}

func (w flavorSortWrapper) Swap(i, j int) {
	w.f[i], w.f[j] = w.f[j], w.f[i]
}

func (o *Openstack) ListFlavors(r *http.Request) (generated.OpenstackFlavors, error) {
	client, err := o.ComputeClient(r)
	if err != nil {
		return nil, errors.OAuth2ServerError("failed get compute client").WithError(err)
	}

	result, err := client.Flavors(r.Context())
	if err != nil {
		return nil, covertError(err)
	}

	flavors := make(generated.OpenstackFlavors, len(result))

	for i := range result {
		flavor, err := convertFlavor(client, &result[i])
		if err != nil {
			return nil, err
		}

		flavors[i] = *flavor
	}

	w := flavorSortWrapper{
		f: flavors,
	}

	sort.Stable(w)

	return w.f, nil
}

// GetFlavor does a list and find, while inefficient, it does do image filtering.
func (o *Openstack) GetFlavor(r *http.Request, name string) (*generated.OpenstackFlavor, error) {
	flavors, err := o.ListFlavors(r)
	if err != nil {
		return nil, err
	}

	for i := range flavors {
		if flavors[i].Name == name {
			return &flavors[i], nil
		}
	}

	return nil, errors.HTTPNotFound().WithError(fmt.Errorf("%w: flavor %s", ErrResourceNotFound, name))
}

// imageSortWrapper sorts images by age.
type imageSortWrapper struct {
	images []generated.OpenstackImage
}

func (w imageSortWrapper) Len() int {
	return len(w.images)
}

func (w imageSortWrapper) Less(i, j int) bool {
	return w.images[i].Created.Before(w.images[j].Created)
}

func (w imageSortWrapper) Swap(i, j int) {
	w.images[i], w.images[j] = w.images[j], w.images[i]
}

func (o *Openstack) ListImages(r *http.Request) (generated.OpenstackImages, error) {
	client, err := o.ImageClient(r)
	if err != nil {
		return nil, errors.OAuth2ServerError("failed get image client").WithError(err)
	}

	result, err := client.Images(r.Context())
	if err != nil {
		return nil, covertError(err)
	}

	images := make(generated.OpenstackImages, len(result))

	for i, image := range result {
		kubernetesVersion, _ := image.Properties["k8s"].(string)
		nvidiaDriverVersion, _ := image.Properties["gpu"].(string)

		images[i].Id = image.ID
		images[i].Name = image.Name
		images[i].Created = image.CreatedAt
		images[i].Modified = image.UpdatedAt
		images[i].Versions.Kubernetes = "v" + kubernetesVersion
		images[i].Versions.NvidiaDriver = nvidiaDriverVersion
	}

	w := imageSortWrapper{
		images: images,
	}

	sort.Stable(w)

	return w.images, nil
}

// GetImage does a list and find, while inefficient, it does do image filtering.
func (o *Openstack) GetImage(r *http.Request, name string) (*generated.OpenstackImage, error) {
	images, err := o.ListImages(r)
	if err != nil {
		return nil, err
	}

	for i := range images {
		if images[i].Name == name {
			return &images[i], nil
		}
	}

	return nil, errors.HTTPNotFound().WithError(fmt.Errorf("%w: image %s", ErrResourceNotFound, name))
}

func (o *Openstack) ListKeyPairs(r *http.Request) (generated.OpenstackKeyPairs, error) {
	client, err := o.ComputeClient(r)
	if err != nil {
		return nil, errors.OAuth2ServerError("failed get compute client").WithError(err)
	}

	result, err := client.KeyPairs(r.Context())
	if err != nil {
		return nil, covertError(err)
	}

	keyPairs := generated.OpenstackKeyPairs{}

	for _, keyPair := range result {
		// Undocumented (what a shocker), but absence means SSH as that's
		// all that used to be supported.  Obviously it could be something else
		// being odd that means we have to parse the public key...
		if keyPair.Type != "" && keyPair.Type != "ssh" {
			continue
		}

		k := generated.OpenstackKeyPair{
			Name: keyPair.Name,
		}

		keyPairs = append(keyPairs, k)
	}

	return keyPairs, nil
}

// findApplicationCredential, in the spirit of making the platform usable, allows
// a client to use names, rather than IDs for lookups.
func findApplicationCredential(in []applicationcredentials.ApplicationCredential, name string) (*applicationcredentials.ApplicationCredential, error) {
	for i, c := range in {
		if c.Name == name {
			return &in[i], nil
		}
	}

	return nil, errors.HTTPNotFound().WithError(fmt.Errorf("%w: application credential %s", ErrResourceNotFound, name))
}

func (o *Openstack) GetApplicationCredential(r *http.Request, name string) (*applicationcredentials.ApplicationCredential, error) {
	user, err := getUser(r)
	if err != nil {
		return nil, err
	}

	client, err := o.IdentityClient(r)
	if err != nil {
		return nil, errors.OAuth2ServerError("failed get identity client").WithError(err)
	}

	result, err := client.ListApplicationCredentials(r.Context(), user)
	if err != nil {
		return nil, covertError(err)
	}

	match, err := findApplicationCredential(result, name)
	if err != nil {
		return nil, err
	}

	return match, nil
}

func (o *Openstack) CreateApplicationCredential(r *http.Request, name string) (*applicationcredentials.ApplicationCredential, error) {
	user, err := getUser(r)
	if err != nil {
		return nil, err
	}

	client, err := o.IdentityClient(r)
	if err != nil {
		return nil, errors.OAuth2ServerError("failed get identity client").WithError(err)
	}

	description := "Automatically generated by platform service [DO NOT DELETE]."

	roles := []string{
		"member",
		"load-balancer_member",
	}

	if o.options.Identity != nil && o.options.Identity.ClusterRoles != nil {
		roles = o.options.Identity.ClusterRoles
	}

	result, err := client.CreateApplicationCredential(r.Context(), user, name, description, roles)
	if err != nil {
		return nil, covertError(err)
	}

	return result, nil
}

func (o *Openstack) DeleteApplicationCredential(r *http.Request, name string) error {
	user, err := getUser(r)
	if err != nil {
		return err
	}

	client, err := o.IdentityClient(r)
	if err != nil {
		return errors.OAuth2ServerError("failed get identity client").WithError(err)
	}

	result, err := client.ListApplicationCredentials(r.Context(), user)
	if err != nil {
		return covertError(err)
	}

	match, err := findApplicationCredential(result, name)
	if err != nil {
		return err
	}

	if err := client.DeleteApplicationCredential(r.Context(), user, match.ID); err != nil {
		return errors.OAuth2ServerError("failed delete application credentials").WithError(err)
	}

	return nil
}

func (o *Openstack) GetServerGroup(r *http.Request, name string) (*servergroups.ServerGroup, error) {
	client, err := o.ComputeClient(r)
	if err != nil {
		return nil, errors.OAuth2ServerError("failed get compute client").WithError(err)
	}

	result, err := client.ListServerGroups(r.Context())
	if err != nil {
		return nil, covertError(err)
	}

	filtered := slices.DeleteFunc(result, func(group servergroups.ServerGroup) bool {
		return group.Name != name
	})

	switch len(filtered) {
	case 0:
		return nil, errors.HTTPNotFound().WithError(fmt.Errorf("%w: server group %s", ErrResourceNotFound, name))
	case 1:
		return &filtered[0], nil
	default:
		return nil, errors.OAuth2ServerError("multiple server groups matched name")
	}
}

func (o *Openstack) CreateServerGroup(r *http.Request, name string) (*servergroups.ServerGroup, error) {
	client, err := o.ComputeClient(r)
	if err != nil {
		return nil, errors.OAuth2ServerError("failed get compute client").WithError(err)
	}

	result, err := client.CreateServerGroup(r.Context(), name)
	if err != nil {
		return nil, covertError(err)
	}

	return result, nil
}
