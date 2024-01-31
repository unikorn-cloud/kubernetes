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

package v1alpha1

import (
	"k8s.io/apimachinery/pkg/runtime/schema"

	"sigs.k8s.io/controller-runtime/pkg/scheme"
)

const (
	// GroupName is the Kubernetes API group our resources belong to.
	GroupName = "unikorn-cloud.org"
	// GroupVersion is the version of our custom resources.
	GroupVersion = "v1alpha1"
	// Group is group/version of our resources.
	Group = GroupName + "/" + GroupVersion

	// ProjectKind is the API kind for a project.
	ProjectKind = "Projects"
	// ProjectResource is the API endpoint for a project.
	ProjectResource = "projects"
	// ControlPlaneKind is the API kind of a control plane.
	ControlPlaneKind = "ControlPlane"
	// ControlPlaneResource is the API endpoint for control plane resources.
	ControlPlaneResource = "controlplanes"
	// KubernetesClusterKind is the API kind for a cluster.
	// NOTE: This is deliberately explicit to avoid a clash with CAPI Cluster
	// objects (yes it's namespaced by group, but this makes it easier), and
	// to provide future expansion...
	KubernetesClusterKind = "KubernetesCluster"
	// KubernetesClusterResource is the API endpoint for a cluster resource.
	KubernetesClusterResource = "kubernetesclusters"
	// ControlPlaneApplicationBundleKind is the API kind for a bundle of applications.
	ControlPlaneApplicationBundleKind = "ControlPlaneApplicationBundle"
	// ControlPlaneApplicationBundleResource is the API endpoint for bundles of applications.
	ControlPlaneApplicationBundleResource = "controlplaneapplicationbundles"
	// KubernetesClusterApplicationBundleKind is the API kind for a bundle of applications.
	KubernetesClusterApplicationBundleKind = "KubernetesClusterApplicationBundle"
	// KubernetesClusterApplicationBundleResource is the API endpoint for bundles of applications.
	KubernetesClusterApplicationBundleResource = "kubernetesclusterapplicationbundles"
)

var (
	// SchemeGroupVersion defines the GV of our resources.
	//nolint:gochecknoglobals
	SchemeGroupVersion = schema.GroupVersion{Group: GroupName, Version: GroupVersion}

	// SchemeBuilder creates a mapping between GVK and type.
	//nolint:gochecknoglobals
	SchemeBuilder = &scheme.Builder{GroupVersion: SchemeGroupVersion}

	// AddToScheme adds our GVK to resource mappings to an existing scheme.
	//nolint:gochecknoglobals
	AddToScheme = SchemeBuilder.AddToScheme
)

//nolint:gochecknoinits
func init() {
	SchemeBuilder.Register(&Project{}, &ProjectList{})
	SchemeBuilder.Register(&ControlPlane{}, &ControlPlaneList{})
	SchemeBuilder.Register(&KubernetesCluster{}, &KubernetesClusterList{})
	SchemeBuilder.Register(&ControlPlaneApplicationBundle{}, &ControlPlaneApplicationBundleList{})
	SchemeBuilder.Register(&KubernetesClusterApplicationBundle{}, &KubernetesClusterApplicationBundleList{})
}

// Resource maps a resource type to a group resource.
func Resource(resource string) schema.GroupResource {
	return SchemeGroupVersion.WithResource(resource).GroupResource()
}
