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
	"context"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"

	unikornv1 "github.com/unikorn-cloud/unikorn/pkg/apis/unikorn/v1alpha1"

	unikornv1core "github.com/unikorn-cloud/core/pkg/apis/unikorn/v1alpha1"
	"github.com/unikorn-cloud/core/pkg/constants"
	"github.com/unikorn-cloud/core/pkg/util"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// mustCreateProjectFixture creates a project, and its randomly named namespace
// just as if unikorn-project-manager had picked up the create request, preformed it
// and also updated the status.
//
//nolint:unparam
func mustCreateProjectFixture(t *testing.T, tc *TestContext, projectID string) *unikornv1.Project {
	t.Helper()

	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "project-",
			Labels: map[string]string{
				constants.ProjectLabel: projectNameFromID(projectID),
			},
		},
	}

	assert.NoError(t, tc.KubernetesClient().Create(context.TODO(), namespace))

	project := &unikornv1.Project{
		ObjectMeta: metav1.ObjectMeta{
			Name: projectNameFromID(projectID),
		},
		Status: unikornv1.ProjectStatus{
			Namespace: namespace.Name,
			Conditions: []unikornv1core.Condition{
				{
					Type:   unikornv1core.ConditionAvailable,
					Status: corev1.ConditionTrue,
					Reason: unikornv1core.ConditionReasonProvisioned,
				},
			},
		},
	}

	assert.NoError(t, tc.KubernetesClient().Create(context.TODO(), project))

	return project
}

// mustCreateControlPlaneFixture creates a control plane , and its randomly named namespace
// just as if unikorn-controlplane-manager had picked up the create request, preformed it
// and also updated the status.
//
//nolint:unparam
func mustCreateControlPlaneFixture(t *testing.T, tc *TestContext, namespace, name string) *unikornv1.ControlPlane {
	t.Helper()

	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "controlplane-",
			Labels: map[string]string{
				constants.ProjectLabel:      projectNameFromID(projectID),
				constants.ControlPlaneLabel: name,
			},
		},
	}

	assert.NoError(t, tc.KubernetesClient().Create(context.TODO(), ns))

	bundleVersion := controlPlaneApplicationBundleName

	controlPlane := &unikornv1.ControlPlane{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
		},
		Spec: unikornv1.ControlPlaneSpec{
			ApplicationBundle: &bundleVersion,
		},
		Status: unikornv1.ControlPlaneStatus{
			Namespace: ns.Name,
			Conditions: []unikornv1core.Condition{
				{
					Type:   unikornv1core.ConditionAvailable,
					Status: corev1.ConditionTrue,
					Reason: unikornv1core.ConditionReasonProvisioned,
				},
			},
		},
	}

	assert.NoError(t, tc.KubernetesClient().Create(context.TODO(), controlPlane))

	return controlPlane
}

const (
	clusterComputeFailureDomain = "danger_nova"
	clusterStorageFailureDomain = "ceph"
	clusterSSHKeyName           = "chubb"
	clusterExternalNetworkID    = "e0c16797-e5db-4ee4-8305-16a72ded2b7e"
	clusterNodeNetwork          = "1.0.0.0/24"
	clusterServiceNetwork       = "2.0.0.0/24"
	clusterPodNetwork           = "3.0.0.0/24"
	clusterDNSNameserver        = "8.8.8.8"
	clusterControlPlaneReplicas = 3
	clusterWorkloadPoolName     = "you-got-to-work-hard"
	clusterWorkloadPoolReplicas = 10
)

// mustCreateKubernetesClusterFixture creates a basic cluster resource in Kubernetes.
//
//nolint:unparam
func mustCreateKubernetesClusterFixture(t *testing.T, tc *TestContext, namespace, name string) {
	t.Helper()

	controlPlaneKubernetesVersion := unikornv1.SemanticVersion("v" + imageK8sVersion)
	workloadPoolKubernetesVersion := unikornv1.SemanticVersion("v" + imageK8sVersion)

	_, nodenetwork, err := net.ParseCIDR(clusterNodeNetwork)
	assert.NoError(t, err)
	_, serviceNetwork, err := net.ParseCIDR(clusterServiceNetwork)
	assert.NoError(t, err)
	_, podNetwork, err := net.ParseCIDR(clusterPodNetwork)
	assert.NoError(t, err)

	dnsNameserver := net.ParseIP(clusterDNSNameserver)

	cluster := &unikornv1.KubernetesCluster{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
		},
		Spec: unikornv1.KubernetesClusterSpec{
			ApplicationBundle: util.ToPointer(kubernetesClusterApplicationBundleName),
			Openstack: &unikornv1.KubernetesClusterOpenstackSpec{
				FailureDomain:       util.ToPointer(clusterComputeFailureDomain),
				VolumeFailureDomain: util.ToPointer(clusterStorageFailureDomain),
				SSHKeyName:          util.ToPointer(clusterSSHKeyName),
				ExternalNetworkID:   util.ToPointer(clusterExternalNetworkID),
			},
			Network: &unikornv1.KubernetesClusterNetworkSpec{
				NodeNetwork:    &unikornv1.IPv4Prefix{IPNet: *nodenetwork},
				ServiceNetwork: &unikornv1.IPv4Prefix{IPNet: *serviceNetwork},
				PodNetwork:     &unikornv1.IPv4Prefix{IPNet: *podNetwork},
				DNSNameservers: []unikornv1.IPv4Address{
					{IP: dnsNameserver},
				},
			},
			ControlPlane: &unikornv1.KubernetesClusterControlPlaneSpec{
				MachineGeneric: unikornv1.MachineGeneric{
					Replicas: util.ToPointer(clusterControlPlaneReplicas),
					Version:  &controlPlaneKubernetesVersion,
					Image:    util.ToPointer(imageName),
					Flavor:   util.ToPointer(flavorName),
				},
			},
			WorkloadPools: &unikornv1.KubernetesClusterWorkloadPoolsSpec{
				Pools: []unikornv1.KubernetesClusterWorkloadPoolsPoolSpec{
					{
						KubernetesWorkloadPoolSpec: unikornv1.KubernetesWorkloadPoolSpec{
							Name: clusterWorkloadPoolName,
							MachineGeneric: unikornv1.MachineGeneric{
								Replicas: util.ToPointer(clusterWorkloadPoolReplicas),
								Version:  &workloadPoolKubernetesVersion,
								Image:    util.ToPointer(imageName),
								Flavor:   util.ToPointer(flavorName),
							},
						},
					},
				},
			},
		},
		Status: unikornv1.KubernetesClusterStatus{
			Conditions: []unikornv1core.Condition{
				{
					Type:   unikornv1core.ConditionAvailable,
					Status: corev1.ConditionTrue,
					Reason: unikornv1core.ConditionReasonProvisioned,
				},
			},
		},
	}

	assert.NoError(t, tc.KubernetesClient().Create(context.TODO(), cluster))
}

const (
	controlPlaneApplicationBundleName    = "control-plane-1.0.0"
	controlPlaneApplicationBundleVersion = "1.0.0"
)

// mustCreateControlPlaneApplicationBundleFixture creates a basic application bundle
// for a control plane.
func mustCreateControlPlaneApplicationBundleFixture(t *testing.T, tc *TestContext) {
	t.Helper()

	bundle := &unikornv1.ControlPlaneApplicationBundle{
		ObjectMeta: metav1.ObjectMeta{
			Name: controlPlaneApplicationBundleName,
		},
		Spec: unikornv1.ApplicationBundleSpec{
			Version: util.ToPointer(controlPlaneApplicationBundleVersion),
		},
	}

	assert.NoError(t, tc.KubernetesClient().Create(context.TODO(), bundle))
}

const (
	kubernetesClusterApplicationBundleName    = "kubernetes-cluster-1.0.0"
	kubernetesClusterApplicationBundleVersion = "2.0.0"
)

// mustCreateKubernetesClusterApplicationBundleFixture creates a basic application bundle
// for a Kubernetes cluster.
func mustCreateKubernetesClusterApplicationBundleFixture(t *testing.T, tc *TestContext) {
	t.Helper()

	bundle := &unikornv1.KubernetesClusterApplicationBundle{
		ObjectMeta: metav1.ObjectMeta{
			Name: kubernetesClusterApplicationBundleName,
		},
		Spec: unikornv1.ApplicationBundleSpec{
			Version: util.ToPointer(kubernetesClusterApplicationBundleVersion),
		},
	}

	assert.NoError(t, tc.KubernetesClient().Create(context.TODO(), bundle))
}

const (
	applicationName              = "my-app-1.0.0"
	applicationHumanReadableName = "My Application"
	applicationDescription       = "Blah blah blah."
	applicationDocumentation     = "https://docs.my-app.io"
	applicationLicense           = "Apache License-2.0"
	applicationIcon              = "<svg />"
	applicationVersion           = "1.0.0"
)

func mustCreateHelmApplicationFixture(t *testing.T, tc *TestContext) {
	t.Helper()

	app := &unikornv1core.HelmApplication{
		ObjectMeta: metav1.ObjectMeta{
			Name: applicationName,
		},
		Spec: unikornv1core.HelmApplicationSpec{
			Name:          util.ToPointer(applicationHumanReadableName),
			Description:   util.ToPointer(applicationDescription),
			Documentation: util.ToPointer(applicationDocumentation),
			License:       util.ToPointer(applicationLicense),
			Icon:          []byte(applicationIcon),
			Exported:      util.ToPointer(true),
			Versions: []unikornv1core.HelmApplicationVersion{
				{
					Version: util.ToPointer(applicationVersion),
				},
			},
		},
	}

	assert.NoError(t, tc.KubernetesClient().Create(context.TODO(), app))
}
