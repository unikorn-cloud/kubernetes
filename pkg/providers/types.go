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

package providers

import (
	"time"

	"k8s.io/apimachinery/pkg/api/resource"
)

// GPUVendor defines the GPU vendor.
type GPUVendor string

const (
	Nvidia GPUVendor = "nvidia"
	AMD    GPUVendor = "amd"
)

// Flavor represents a machine type.
type Flavor struct {
	// Name of the flavor.
	Name string
	// CPU count.
	CPUs int
	// Memory available.
	Memory *resource.Quantity
	// Disk available.
	Disk *resource.Quantity
	// GPU count.
	GPUs int
	// GPUVendor is who makes the GPU, used to determine the drivers etc.
	GPUVendor GPUVendor
	// BareMetal is a bare-metal flavor.
	BareMetal bool
}

// FlavorList allows us to attach sort functions and the like.
type FlavorList []Flavor

// Image represents an operating system image.
type Image struct {
	// Name of the image.
	Name string
	// Created is when the image was created.
	Created time.Time
	// Modified is when the image was modified.
	Modified time.Time
	// KubernetesVersion is only populated if the image contains a pre-installed
	// version of Kubernetes, this acts as a cache and improves provisioning performance.
	// This is pretty much the only source of truth about Kubernetes versions at
	// present, so should be populated.
	KubernetesVersion string
}

// ImageList allows us to attach sort functions and the like.
type ImageList []Image
