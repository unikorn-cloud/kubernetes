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
	"errors"
	"fmt"
	"net"
	"strings"
	"time"

	unikornv1core "github.com/unikorn-cloud/core/pkg/apis/unikorn/v1alpha1"
	"github.com/unikorn-cloud/core/pkg/constants"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
)

var (
	// ErrStatusunikornv1core.ConditionLookup is raised when a condition is not found in
	// the resource status.
	ErrStatusConditionLookup = errors.New("status condition not found")

	// ErrMissingLabel is raised when an expected label is not present on
	// a resource.
	ErrMissingLabel = errors.New("expected label is missing")

	// ErrApplicationLookup is raised when the named application is not
	// present in an application bundle bundle.
	ErrApplicationLookup = errors.New("failed to lookup an application")
)

// IPv4AddressSliceFromIPSlice is a simple converter from Go types
// to API types.
func IPv4AddressSliceFromIPSlice(in []net.IP) []IPv4Address {
	out := make([]IPv4Address, len(in))

	for i, ip := range in {
		out[i] = IPv4Address{IP: ip}
	}

	return out
}

// Paused implements the ReconcilePauser interface.
func (c *Organization) Paused() bool {
	return c.Spec.Pause
}

// Paused implements the ReconcilePauser interface.
func (c *Project) Paused() bool {
	return c.Spec.Pause
}

// Paused implements the ReconcilePauser interface.
func (c *ControlPlane) Paused() bool {
	return c.Spec.Pause
}

// Paused implements the ReconcilePauser interface.
func (c *KubernetesCluster) Paused() bool {
	return c.Spec.Pause
}

// StatusConditionRead scans the status conditions for an existing condition whose type
// matches.
func (c *Organization) StatusConditionRead(t unikornv1core.ConditionType) (*unikornv1core.Condition, error) {
	return unikornv1core.GetCondition(c.Status.Conditions, t)
}

// StatusConditionWrite either adds or updates a condition in the control plane status.
// If the condition, status and message match an existing condition the update is
// ignored.
func (c *Organization) StatusConditionWrite(t unikornv1core.ConditionType, status corev1.ConditionStatus, reason unikornv1core.ConditionReason, message string) {
	unikornv1core.UpdateCondition(&c.Status.Conditions, t, status, reason, message)
}

// ResourceLabels generates a set of labels to uniquely identify the resource
// if it were to be placed in a single global namespace.
func (c *Organization) ResourceLabels() (labels.Set, error) {
	labels := labels.Set{
		constants.KindLabel:         constants.KindLabelValueOrganization,
		constants.OrganizationLabel: c.Name,
	}

	return labels, nil
}

// StatusConditionRead scans the status conditions for an existing condition whose type
// matches.
func (c *Project) StatusConditionRead(t unikornv1core.ConditionType) (*unikornv1core.Condition, error) {
	return unikornv1core.GetCondition(c.Status.Conditions, t)
}

// StatusConditionWrite either adds or updates a condition in the control plane status.
// If the condition, status and message match an existing condition the update is
// ignored.
func (c *Project) StatusConditionWrite(t unikornv1core.ConditionType, status corev1.ConditionStatus, reason unikornv1core.ConditionReason, message string) {
	unikornv1core.UpdateCondition(&c.Status.Conditions, t, status, reason, message)
}

// ResourceLabels generates a set of labels to uniquely identify the resource
// if it were to be placed in a single global namespace.
func (c *Project) ResourceLabels() (labels.Set, error) {
	organization, ok := c.Labels[constants.OrganizationLabel]
	if !ok {
		return nil, ErrMissingLabel
	}

	labels := labels.Set{
		constants.KindLabel:         constants.KindLabelValueProject,
		constants.OrganizationLabel: organization,
		constants.ProjectLabel:      c.Name,
	}

	return labels, nil
}

// StatusConditionRead scans the status conditions for an existing condition whose type
// matches.
func (c *ControlPlane) StatusConditionRead(t unikornv1core.ConditionType) (*unikornv1core.Condition, error) {
	return unikornv1core.GetCondition(c.Status.Conditions, t)
}

// StatusConditionWrite either adds or updates a condition in the control plane status.
// If the condition, status and message match an existing condition the update is
// ignored.
func (c *ControlPlane) StatusConditionWrite(t unikornv1core.ConditionType, status corev1.ConditionStatus, reason unikornv1core.ConditionReason, message string) {
	unikornv1core.UpdateCondition(&c.Status.Conditions, t, status, reason, message)
}

// ResourceLabels generates a set of labels to uniquely identify the resource
// if it were to be placed in a single global namespace.
func (c *ControlPlane) ResourceLabels() (labels.Set, error) {
	organization, ok := c.Labels[constants.OrganizationLabel]
	if !ok {
		return nil, ErrMissingLabel
	}

	project, ok := c.Labels[constants.ProjectLabel]
	if !ok {
		return nil, ErrMissingLabel
	}

	labels := labels.Set{
		constants.KindLabel:         constants.KindLabelValueControlPlane,
		constants.OrganizationLabel: organization,
		constants.ProjectLabel:      project,
		constants.ControlPlaneLabel: c.Name,
	}

	return labels, nil
}

func (c ControlPlane) Entropy() []byte {
	return []byte(c.UID)
}

func (c ControlPlane) UpgradeSpec() *ApplicationBundleAutoUpgradeSpec {
	return c.Spec.ApplicationBundleAutoUpgrade
}

// StatusConditionRead scans the status conditions for an existing condition whose type
// matches.
func (c *KubernetesCluster) StatusConditionRead(t unikornv1core.ConditionType) (*unikornv1core.Condition, error) {
	return unikornv1core.GetCondition(c.Status.Conditions, t)
}

// StatusConditionWrite either adds or updates a condition in the cluster status.
// If the condition, status and message match an existing condition the update is
// ignored.
func (c *KubernetesCluster) StatusConditionWrite(t unikornv1core.ConditionType, status corev1.ConditionStatus, reason unikornv1core.ConditionReason, message string) {
	unikornv1core.UpdateCondition(&c.Status.Conditions, t, status, reason, message)
}

// ResourceLabels generates a set of labels to uniquely identify the resource
// if it were to be placed in a single global namespace.
func (c *KubernetesCluster) ResourceLabels() (labels.Set, error) {
	organization, ok := c.Labels[constants.OrganizationLabel]
	if !ok {
		return nil, ErrMissingLabel
	}

	project, ok := c.Labels[constants.ProjectLabel]
	if !ok {
		return nil, ErrMissingLabel
	}

	controlPlane, ok := c.Labels[constants.ControlPlaneLabel]
	if !ok {
		return nil, ErrMissingLabel
	}

	labels := labels.Set{
		constants.KindLabel:              constants.KindLabelValueKubernetesCluster,
		constants.OrganizationLabel:      organization,
		constants.ProjectLabel:           project,
		constants.ControlPlaneLabel:      controlPlane,
		constants.KubernetesClusterLabel: c.Name,
	}

	return labels, nil
}

func (c KubernetesCluster) Entropy() []byte {
	return []byte(c.UID)
}

func (c KubernetesCluster) UpgradeSpec() *ApplicationBundleAutoUpgradeSpec {
	return c.Spec.ApplicationBundleAutoUpgrade
}

// AutoscalingEnabled indicates whether cluster autoscaling is enabled for the cluster.
func (c *KubernetesCluster) AutoscalingEnabled() bool {
	return c.Spec.Features != nil && c.Spec.Features.Autoscaling != nil && *c.Spec.Features.Autoscaling
}

// IngressEnabled indicates whether an ingress controller is required.
func (c *KubernetesCluster) IngressEnabled() bool {
	return c.Spec.Features != nil && c.Spec.Features.Ingress != nil && *c.Spec.Features.Ingress
}

// CertManagerEnabled indicates whether cert-manager is required.
func (c *KubernetesCluster) CertManagerEnabled() bool {
	return c.Spec.Features != nil && c.Spec.Features.CertManager != nil && *c.Spec.Features.CertManager
}

// KubernetesDashboardEnabled indicates whether the Kubernetes dashboard is required.
func (c *KubernetesCluster) KubernetesDashboardEnabled() bool {
	return c.Spec.Features != nil && c.Spec.Features.KubernetesDashboard != nil && *c.Spec.Features.KubernetesDashboard
}

// FileStorageEnabled indicates whether a POSIX file storage CSI is required.
func (c *KubernetesCluster) FileStorageEnabled() bool {
	return c.Spec.Features != nil && c.Spec.Features.FileStorage != nil && *c.Spec.Features.FileStorage
}

// PrometheusEnabled indicates whether the Prometheus Operator is required.
func (c *KubernetesCluster) PrometheusEnabled() bool {
	return c.Spec.Features != nil && c.Spec.Features.Prometheus != nil && *c.Spec.Features.Prometheus
}

// NvidiaOperatorEnabled indicates whether to install the Nvidia GPU operator.
func (c *KubernetesCluster) NvidiaOperatorEnabled() bool {
	return c.Spec.Features != nil && c.Spec.Features.NvidiaOperator != nil && *c.Spec.Features.NvidiaOperator
}

func CompareProject(a, b Project) int {
	return strings.Compare(a.Name, b.Name)
}

func CompareControlPlane(a, b ControlPlane) int {
	return strings.Compare(a.Name, b.Name)
}

func CompareKubernetesCluster(a, b KubernetesCluster) int {
	return strings.Compare(a.Name, b.Name)
}

func CompareControlPlaneApplicationBundle(a, b ControlPlaneApplicationBundle) int {
	// TODO: while this works now, it won't unless we parse and compare as
	// a semantic version.
	return strings.Compare(*a.Spec.Version, *b.Spec.Version)
}

func CompareKubernetesClusterApplicationBundle(a, b KubernetesClusterApplicationBundle) int {
	// TODO: while this works now, it won't unless we parse and compare as
	// a semantic version.
	return strings.Compare(*a.Spec.Version, *b.Spec.Version)
}

// Get retrieves the named bundle.
func (l ControlPlaneApplicationBundleList) Get(name string) *ControlPlaneApplicationBundle {
	for i := range l.Items {
		if l.Items[i].Name == name {
			return &l.Items[i]
		}
	}

	return nil
}

func (l KubernetesClusterApplicationBundleList) Get(name string) *KubernetesClusterApplicationBundle {
	for i := range l.Items {
		if l.Items[i].Name == name {
			return &l.Items[i]
		}
	}

	return nil
}

// Upgradable returns a new list of bundles that are "stable" e.g. not end of life and
// not a preview.
func (l ControlPlaneApplicationBundleList) Upgradable() *ControlPlaneApplicationBundleList {
	result := &ControlPlaneApplicationBundleList{}

	for _, bundle := range l.Items {
		if bundle.Spec.Preview != nil && *bundle.Spec.Preview {
			continue
		}

		if bundle.Spec.EndOfLife != nil {
			continue
		}

		result.Items = append(result.Items, bundle)
	}

	return result
}

func (l KubernetesClusterApplicationBundleList) Upgradable() *KubernetesClusterApplicationBundleList {
	result := &KubernetesClusterApplicationBundleList{}

	for _, bundle := range l.Items {
		if bundle.Spec.Preview != nil && *bundle.Spec.Preview {
			continue
		}

		if bundle.Spec.EndOfLife != nil {
			continue
		}

		result.Items = append(result.Items, bundle)
	}

	return result
}

func (s ApplicationBundleSpec) GetApplication(name string) (*unikornv1core.ApplicationReference, error) {
	for i := range s.Applications {
		if *s.Applications[i].Name == name {
			return s.Applications[i].Reference, nil
		}
	}

	return nil, fmt.Errorf("%w: %s", ErrApplicationLookup, name)
}

// Weekdays returns the days of the week that are set in the spec.
func (s ApplicationBundleAutoUpgradeWeekDaySpec) Weekdays() []time.Weekday {
	var result []time.Weekday

	if s.Sunday != nil {
		result = append(result, time.Sunday)
	}

	if s.Monday != nil {
		result = append(result, time.Monday)
	}

	if s.Tuesday != nil {
		result = append(result, time.Tuesday)
	}

	if s.Wednesday != nil {
		result = append(result, time.Wednesday)
	}

	if s.Thursday != nil {
		result = append(result, time.Thursday)
	}

	if s.Friday != nil {
		result = append(result, time.Friday)
	}

	if s.Saturday != nil {
		result = append(result, time.Saturday)
	}

	return result
}
