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

package server_test

import (
	"fmt"
	"net/http"
	"time"
)

// genericUnauthorized is returned by all APIs when the identity cannot be confirmed.
const genericUnauthorized = `{
  "error": {
    "code": 401,
    "message": "The request you have made requires authentication.",
    "title": "Unauthorized"
  }
}`

const genericForbidden = `{
  "error": {
    "code": 403,
    "message": "You are not authorized to perform the requested action.",
    "title": "Forbidden"
  }
}`

// v3AuthTokensSuccessResponse defines how we mock the OpenStack API.  Basically we'll
// multiplex all services through a single endpoint for simplicity.
// Important parts to pay attention to (in the context of gophercloud):
// * token.catalog.type is used to look for the service.
// * token.catalog.endpoints.interface is used to look a service endpoint, "public" is the default.
// * token.user.id is used by Unikorn for identity information in its access token.
func v3AuthTokensSuccessResponse(tc *TestContext) []byte {
	return []byte(`{
	"token": {
		"catalog": [
			{
				"name": "keystone",
				"type": "identity",
				"endpoints": [
					{
						"interface": "public",
						"region": "RegionOne",
						"region_id": "RegionOne",
						"url": "http://` + tc.OpenstackServerEndpoint() + `/identity"
					}
				]
			},
			{
				"name": "nova",
				"type": "compute",
				"endpoints": [
					{
						"interface": "public",
						"region": "RegionOne",
                                                "region_id": "RegionOne",
                                                "url": "http://` + tc.OpenstackServerEndpoint() + `/compute"
					}
				]
			},
			{
				"name": "glance",
				"type": "image",
				"endpoints": [
                                        {
                                                "interface": "public",
                                                "region": "RegionOne",
                                                "region_id": "RegionOne",
                                                "url": "http://` + tc.OpenstackServerEndpoint() + `/image"
                                        }
                                ]
                        },
			{
				"name": "neutron",
				"type": "network",
				"endpoints": [
					{
						"interface": "public",
                                                "region": "RegionOne",
                                                "region_id": "RegionOne",
                                                "url": "http://` + tc.OpenstackServerEndpoint() + `/network"
					}
				]
			},
			{
                                "name": "cinder",
                                "type": "volumev3",
				"endpoints": [
                                        {
                                                "interface": "public",
                                                "region": "RegionOne",
                                                "region_id": "RegionOne",
                                                "url": "http://` + tc.OpenstackServerEndpoint() + `/blockstorage"
					}
				]
			}
		],
		"domain": {
			"id": "default",
			"name": "Default"
		},
		"methods": [
			"password"
		],
		"expires_at": "` + time.Now().Add(time.Hour).Format(time.RFC3339) + `",
		"user": {
			"domain": {
				"id": "default",
				"name": "Default"
			},
			"id": "` + userID + `",
			"name": "foo"
		}
	}
}`)
}

// RegisterIdentityV3AuthTokensPostSuccessHandler is called when we want to login, or do a
// token exchange/rescoping.
func RegisterIdentityV3AuthTokensPostSuccessHandler(tc *TestContext) {
	tc.OpenstackRouter().Post("/identity/v3/auth/tokens", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Subject-Token", "ImAToken")
		w.WriteHeader(http.StatusCreated)
		if _, err := w.Write(v3AuthTokensSuccessResponse(tc)); err != nil {
			if debug {
				fmt.Println(err)
			}
		}
	})
}

// RegisterIdentityV3AuthTokensGetSuccessHandler is called by gophercloud to validate a
// token and to get the service catalog.
func RegisterIdentityV3AuthTokensGetSuccessHandler(tc *TestContext) {
	tc.OpenstackRouter().Get("/identity/v3/auth/tokens", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Subject-Token", "ImAToken")
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write(v3AuthTokensSuccessResponse(tc)); err != nil {
			if debug {
				fmt.Println(err)
			}
		}
	})
}

// RegisterIdentityV3AuthTokensPostUnauthorizedHandler is called when we want to login, or do a
// token exchange/rescoping.
func RegisterIdentityV3AuthTokensPostUnauthorizedHandler(tc *TestContext) {
	tc.OpenstackRouter().Post("/identity/v3/auth/tokens", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		if _, err := w.Write([]byte(genericUnauthorized)); err != nil {
			if debug {
				fmt.Println(err)
			}
		}
	})
}

// identityMetadata returns versioned endpoint information for the identity service.
// Gophercloud will check the links and preferentially select v3 over v2_0.
func identityMetadata(tc *TestContext) string {
	return `{
	"versions": {
		"values": [
			{
				"id": "v3.14",
				"status": "stable",
				"links": [
					{
						"rel": "self",
						"href": "http://` + tc.OpenstackServerEndpoint() + `/identity/v3"
					}
				],
				"media-types": [
					{
						"base": "application/json",
						"type": "application/vnd.openstack.identity-v3+json"
					}
				]
			}
		]
	}
}`
}

// RegisterIdentityHandler allows gophercloud to derive the correct base path to
// use for identity operations.
func RegisterIdentityHandler(tc *TestContext) {
	tc.OpenstackRouter().Get("/identity/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte(identityMetadata(tc))); err != nil {
			if debug {
				fmt.Println(err)
			}
		}
	})
}

const userID = "5e6bb9d8-03a1-4d26-919c-6884ff574a31"

// userInfo returns user information based on the token.
// Only email is considered by Unikorn server at present.
func userInfo() []byte {
	return []byte(fmt.Sprintf(`{
	"user": {
		"domain_id": "default",
		"enabled": true,
		"id": "%s",
		"name": "foo",
		"email": "foo@bar.com"
	}
}`, userID))
}

// RegisterIdentityV3User allows Unikorn to lookup a user in oder to issue
// an access token.
func RegisterIdentityV3User(tc *TestContext) {
	tc.OpenstackRouter().Get("/identity/v3/users/{user_id}", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write(userInfo()); err != nil {
			if debug {
				fmt.Println(err)
			}
		}
	})
}

// applicationCredentials is what you get when you list credentials.
// Start with one as that flexes more code.
func applicationCredentials() []byte {
	return []byte(`{
	"links": {},
	"application_credentials": [
		{
			"id": "75f56f78-18e0-4f60-83c4-7109cafe3fd1",
			"name": "foo-foo"
		}
	]
}`)
}

// applicationCredentialCreate is what you get when you create an application
// credential.  Please note this is the ONLY time it will return the secret.
func applicationCredentialCreate() []byte {
	return []byte(`{
	"application_credential": {
		"id": "69a5f849-5112-44b7-9424-64ee0f30c23d",
		"name": "foo-foo",
		"secret": "shhhh"
	}
}`)
}

func RegisterIdentityV3UserApplicationCredentials(tc *TestContext) {
	tc.OpenstackRouter().Get("/identity/v3/users/{user_id}/application_credentials", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write(applicationCredentials()); err != nil {
			if debug {
				fmt.Println(err)
			}
		}
	})
	tc.OpenstackRouter().Post("/identity/v3/users/{user_id}/application_credentials", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		if _, err := w.Write(applicationCredentialCreate()); err != nil {
			if debug {
				fmt.Println(err)
			}
		}
	})
	tc.OpenstackRouter().Delete("/identity/v3/users/{user_id}/application_credentials/{credential_id}", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
}

func RegisterIdentityV3UserApplicationCredentialsForbidden(tc *TestContext) {
	tc.OpenstackRouter().Post("/identity/v3/users/{user_id}/application_credentials", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		if _, err := w.Write([]byte(genericForbidden)); err != nil {
			if debug {
				fmt.Println(err)
			}
		}
	})
}

const projectID = "63051c2c-4d9e-40c0-bf57-93907a61b738"
const projectName = "foo"

func projects() []byte {
	return []byte(fmt.Sprintf(`{
	"projects": [
		{
			"id": "%s",
			"name": "%s"
		}
	],
	"links": {}
}`, projectID, projectName))
}

func RegisterIdentityV3AuthProjects(tc *TestContext) {
	tc.OpenstackRouter().Get("/identity/v3/auth/projects", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write(projects()); err != nil {
			if debug {
				fmt.Println(err)
			}
		}
	})
}

func RegisterIdentityV3AuthProjectsUnauthorized(tc *TestContext) {
	tc.OpenstackRouter().Get("/identity/v3/auth/projects", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		if _, err := w.Write([]byte(genericUnauthorized)); err != nil {
			if debug {
				fmt.Println(err)
			}
		}
	})
}

// RegisterIdentityHandlers adds all the basic handlers required for token
// acquisition.
func RegisterIdentityHandlers(tc *TestContext) {
	RegisterIdentityHandler(tc)
	RegisterIdentityV3AuthTokensPostSuccessHandler(tc)
	RegisterIdentityV3AuthTokensGetSuccessHandler(tc)
	RegisterIdentityV3User(tc)
	RegisterIdentityV3UserApplicationCredentials(tc)
	RegisterIdentityV3AuthProjects(tc)
}

const imageID = "aa21abae-5743-442c-bb69-39a6411558a7"
const imageName = "ubuntu-22.04-lts"
const imageK8sVersion = "1.28.0"
const imageGpuVersion = "525.85.05"
const imageTimestamp = "2019-01-01T00:00:00Z"

// Note the first entry should be filtered out due to lack of a digest,
// then we should be presented with the third image first then the second
// as they are time ordered.
func images() []byte {
	return []byte(fmt.Sprintf(`{
	"first": "/images/v2/images",
	"images": [
		{
			"id": "6876460a-64be-40d1-8520-a3dad947cfba",
			"name": "foo"
		},
		{
			"id": "6daa3bee-63b8-48a3-a082-52ad680dd3c0",
			"name": "ubuntu-24.04-lts",
			"status": "active",
			"created_at": "2020-01-01T00:00:00Z",
			"updated_at": "2020-01-01T00:00:00Z",
			"k8s": "1.28.0",
			"gpu": "525.85.05",
			"digest": "MGYCMQD9kCkukyFePyvNbKe8/DLC4BZAyNJb6e5EvEqf1guR63qBr7E55/GKTVFoWBPS/v0CMQD9AK4aLdRhzWNoAC/IPT7lKQ6k20A/l/CN3cH9x8Qq9y7kfzPUOP1C15nJZsinpzk="
		},
		{
			"id": "%s",
			"name": "%s",
			"status": "active",
                        "created_at": "%s",
			"updated_at": "%s",
                        "k8s": "%s",
                        "gpu": "%s",
                        "digest": "MGYCMQDTPrcsaQJvsbc+hAFSuU6keI5Cf+jjGWPHs3qRkPegMAtjfABvrZNFl3ZMWkR76ygCMQCyLm2+xhAr92DgKs7IEOcG3rbax5Ye/C2MfKPGSiUFQYBD4kMT9XQZ+GMz/jpLUYw="
		}
	]
}`, imageID, imageName, imageTimestamp, imageTimestamp, imageK8sVersion, imageGpuVersion))
}

func RegisterImageV2Images(tc *TestContext) {
	tc.OpenstackRouter().Get("/image/v2/images", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write(images()); err != nil {
			if debug {
				fmt.Println(err)
			}
		}
	})
}

func RegisterImageV2ImagesUnauthorized(tc *TestContext) {
	tc.OpenstackRouter().Get("/image/v2/images", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		if _, err := w.Write([]byte(genericUnauthorized)); err != nil {
			if debug {
				fmt.Println(err)
			}
		}
	})
}

const flavorID = "f547e5e4-5d9e-4434-bb78-d43cabcce79c"
const flavorName = "blueberry"
const flavorCpus = 2
const flavorMemory = 8 << 10
const flavorDisk = 20

const flavorName2 = "strawberry"
const flavorName3 = "raspberry"

// flavorsDetail returns a load of different flavors, we expect these to
// be sorted by the provider so things with a GPU come first, and then CPU
// only flavors.  Those buckets are then sorted by the number of GPUs and
// CPUs respectively, low to high.  Flavors with <1 CPU and <2GiB RAM should
// not be returned.
// NOTE: Extra specs are available in microversion 2.61 onward.
func flavorsDetail() []byte {
	return []byte(fmt.Sprintf(`{
	"first": "/flavors/detail",
	"flavors": [
		{
			"id": "6b1cede8-a814-4cf6-94c1-16ca5f51b1ec",
			"name": "%s",
			"vcpus": 2,
			"ram": 8192,
			"disk": 20
		},
		{	"id": "77bf7ac4-429e-45db-b101-5736ef1b8d3c",
                        "name": "%s",
                        "vcpus": 8,
                        "ram": 32768,
                        "disk": 20,
                        "extra_specs": {
                                "pci_passthrough:alias": "a100:2"
                        }
		},
		{
			"id": "2b037c3a-24c5-49b3-820a-c72c232d26a0",
			"name": "filtered1",
			"vcpus": 1,
			"ram": 4096,
			"disk": 20
		},
		{
                        "id": "2b037c3a-24c5-49b3-820a-c72c232d26a0",
                        "name": "filtered2",
                        "vcpus": 2,
                        "ram": 512,
                        "disk": 20
                },
		{
                        "id": "2b037c3a-24c5-49b3-820a-c72c232d26a0",
                        "name": "filtered3",
                        "vcpus": 2,
                        "ram": 4096,
                        "disk": 20,
			"swap": 1024
                },
		{
			"id": "d7f75b0f-888c-4fed-a807-2cbbee7d5afe",
			"name": "g.baremetal",
			"vcpus": 64,
			"ram": 524228,
			"disk": 12000,
			"extra_specs": {
				"resources:CUSTOM_BAREMETAL": "1"
			}
		},
		{
			"id": "%s",
			"name": "%s",
			"vcpus": %d,
			"ram": %d,
			"disk": %d,
			"extra_specs": {
				"resources:VGPU": "1",
				"trait:CUSTOM_A100D_3_40C": "required"
			}
		}
	]
}`, flavorName3, flavorName2, flavorID, flavorName, flavorCpus, flavorMemory, flavorDisk))
}

func RegisterComputeV2FlavorsDetail(tc *TestContext) {
	tc.OpenstackRouter().Get("/compute/flavors/detail", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write(flavorsDetail()); err != nil {
			if debug {
				fmt.Println(err)
			}
		}
	})
}

func RegisterComputeV2FlavorsDetailUnauthorized(tc *TestContext) {
	tc.OpenstackRouter().Get("/compute/flavors/detail", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		if _, err := w.Write([]byte(genericUnauthorized)); err != nil {
			if debug {
				fmt.Println(err)
			}
		}
	})
}

const serverGroupsEmpty = `{
	"first": "/os-server-groups",
	"server_groups": []
}`

const serverGroup = `{
	"server_group": {
		"id": "51ec3d7e-c52b-4b47-aa82-c99bc374ea23",
		"name": "foo",
		"policies": [
			"soft-anti-affinity"
		]
	}
}`

func RegisterComputeV2ServerGroups(tc *TestContext) {
	tc.OpenstackRouter().Get("/compute/os-server-groups", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte(serverGroupsEmpty)); err != nil {
			if debug {
				fmt.Println(err)
			}
		}
	})
	tc.OpenstackRouter().Post("/compute/os-server-groups", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte(serverGroup)); err != nil {
			if debug {
				fmt.Println(err)
			}
		}
	})
}

const keyPairName = "chubb"

func keyPairs() []byte {
	return []byte(fmt.Sprintf(`{
	"keypairs": [
		{
			"keypair": {
				"name": "%s"
			}
		}
	]
}`, keyPairName))
}

func RegisterComputeV2Keypairs(tc *TestContext) {
	tc.OpenstackRouter().Get("/compute/os-keypairs", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write(keyPairs()); err != nil {
			if debug {
				fmt.Println(err)
			}
		}
	})
}

func RegisterComputeV2KeypairsUnauthorized(tc *TestContext) {
	tc.OpenstackRouter().Get("/compute/os-keypairs", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		if _, err := w.Write([]byte(genericUnauthorized)); err != nil {
			if debug {
				fmt.Println(err)
			}
		}
	})
}

const computeAvailabilityZoneName = "danger_nova"

func computeAvailabilityZones() []byte {
	return []byte(fmt.Sprintf(`{
	"availabilityZoneInfo": [
		{
			"zoneName": "%s",
			"zoneState": {
                                "available": true
                        }
		}
	]
}`, computeAvailabilityZoneName))
}

func RegisterComputeV2AvailabilityZone(tc *TestContext) {
	tc.OpenstackRouter().Get("/compute/os-availability-zone", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write(computeAvailabilityZones()); err != nil {
			if debug {
				fmt.Println(err)
			}
		}
	})
}

func RegisterComputeV2AvailabilityZoneUnauthorized(tc *TestContext) {
	tc.OpenstackRouter().Get("/compute/os-availability-zone", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		if _, err := w.Write([]byte(genericUnauthorized)); err != nil {
			if debug {
				fmt.Println(err)
			}
		}
	})
}

const blockStorageAvailabilityZone = "ceph"

func blockStorageAvailabilityZones() []byte {
	return []byte(fmt.Sprintf(`{
        "availabilityZoneInfo": [
		{
			"zoneName": "%s",
			"zoneState": {
				"available": true
			}
		}
	]
}`, blockStorageAvailabilityZone))
}

func RegisterBlockStorageV3AvailabilityZone(tc *TestContext) {
	tc.OpenstackRouter().Get("/blockstorage/os-availability-zone", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write(blockStorageAvailabilityZones()); err != nil {
			if debug {
				fmt.Println(err)
			}
		}
	})
}

func RegisterBlockStorageV3AvailabilityZoneUnauthorized(tc *TestContext) {
	tc.OpenstackRouter().Get("/blockstorage/os-availability-zone", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		if _, err := w.Write([]byte(genericUnauthorized)); err != nil {
			if debug {
				fmt.Println(err)
			}
		}
	})
}

const externalNetworkID = "605eddb9-39e1-4309-972f-c62ced50f40f"

func externalNetworks() []byte {
	return []byte(fmt.Sprintf(`{
	"networks": [
		{
			"id": "%s"
		}
	]
}`, externalNetworkID))
}

func RegisterNetworkV2Networks(tc *TestContext) {
	tc.OpenstackRouter().Get("/network/v2.0/networks", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write(externalNetworks()); err != nil {
			if debug {
				fmt.Println(err)
			}
		}
	})
}

func RegisterNetworkV2NetworksUnauthorized(tc *TestContext) {
	tc.OpenstackRouter().Get("/network/v2.0/networks", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		if _, err := w.Write([]byte(genericUnauthorized)); err != nil {
			if debug {
				fmt.Println(err)
			}
		}
	})
}
