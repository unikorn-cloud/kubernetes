/*
Copyright 2025 the Unikorn Authors.

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

package checker

import (
	"context"
	"errors"
	"fmt"

	unikornv1 "github.com/unikorn-cloud/core/pkg/apis/unikorn/v1alpha1"
	"github.com/unikorn-cloud/core/pkg/cd"
	"github.com/unikorn-cloud/core/pkg/cd/argocd"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	ErrTypeConversion = errors.New("type conversion error")
)

// HealthChecker lists all resources of the specified type and does a health check on it.
// The type itself is constrained to a manageable resource so we can get the label selector
// to pass to the CD layer to get all applications for the resource, then once we have
// checked the status of those applications we can set the condition generically, again
// as provided by the manageable resource interface.
type HealthChecker[T unikornv1.ManagableResourceInterface, L client.ObjectList] struct {
	// client allows access to Kubernetes resources.
	client client.Client
	// t is storage for the manageable resource.
	t T
	// l is storage for the manageable resource list.
	l L
}

// New creates a new checker.  All types can be inferred, the template parameters
// are purely for type constraints.
func New[T unikornv1.ManagableResourceInterface, L client.ObjectList](client client.Client, t T, l L) *HealthChecker[T, L] {
	return &HealthChecker[T, L]{
		client: client,
		t:      t,
		l:      l,
	}
}

// resourceIdentifierFromResource takes our manageable resource type and returns
// a CD resource ID to identify its applications.
func resourceIdentifierFromResource(r unikornv1.ManagableResourceInterface) (*cd.ResourceIdentifier, error) {
	labels, err := r.ResourceLabels()
	if err != nil {
		return nil, err
	}

	id := &cd.ResourceIdentifier{
		Labels: make([]cd.ResourceIdentifierLabel, 0, len(labels)),
	}

	for k, v := range labels {
		id.Labels = append(id.Labels, cd.ResourceIdentifierLabel{
			Name:  k,
			Value: v,
		})
	}

	return id, nil
}

// convertHealthStatus translates from the CD interface to the Kubernetes API.
func convertHealthStatus(status cd.HealthStatus) (corev1.ConditionStatus, unikornv1.ConditionReason, string) {
	switch status {
	case cd.HealthStatusUnknown:
		return corev1.ConditionUnknown, unikornv1.ConditionReasonUnknown, "unable to poll application status"
	case cd.HealthStatusHealthy:
		return corev1.ConditionTrue, unikornv1.ConditionReasonHealthy, "resource applications healthy"
	case cd.HealthStatusDegraded:
		return corev1.ConditionFalse, unikornv1.ConditionReasonDegraded, "one or more resource applications are degraded"
	}

	// NOTE: the linter will warn about non-exhaustive switches.
	return corev1.ConditionUnknown, unikornv1.ConditionReasonUnknown, "unreachable code reached"
}

// check does the actual check for a resource and updates its status.
func (c *HealthChecker[T, L]) check(ctx context.Context, r unikornv1.ManagableResourceInterface) error {
	// Grab the overall health status.
	id, err := resourceIdentifierFromResource(r)
	if err != nil {
		return err
	}

	// TODO: we only support argo now, but will need an abstraction down the line.
	// There is precedent in the main controllers.
	healthStatus, err := argocd.New(c.client, argocd.Options{}).GetHealthStatus(ctx, id)
	if err != nil {
		return err
	}

	updated, ok := r.DeepCopyObject().(unikornv1.ManagableResourceInterface)
	if !ok {
		return fmt.Errorf("%w: unable to deep copy manageable resource", ErrTypeConversion)
	}

	// And finally set the status condition.
	status, reason, message := convertHealthStatus(healthStatus)

	updated.StatusConditionWrite(unikornv1.ConditionHealthy, status, reason, message)

	if err := c.client.Status().Patch(ctx, updated, client.MergeFrom(r)); err != nil {
		return err
	}

	return nil
}

// Check does the actual check as described for the health checker type.
func (c *HealthChecker[T, L]) Check(ctx context.Context) error {
	// NOTE: This looks expensive, but it's all cached by controller-runtime.
	if err := c.client.List(ctx, c.l, &client.ListOptions{}); err != nil {
		return err
	}

	// Now for the fun bit... there is no generic way to iterate over list
	// items, so we need to destructure...
	o, err := runtime.DefaultUnstructuredConverter.ToUnstructured(c.l)
	if err != nil {
		return err
	}

	// ... then the unstructured stuff will handle any types with an items
	// field in it.
	l := &unstructured.UnstructuredList{}
	l.SetUnstructuredContent(o)

	callback := func(o runtime.Object) error {
		// As we iterate over the objects we can convert back into a
		// useful interface type.
		u, ok := o.(*unstructured.Unstructured)
		if !ok {
			return fmt.Errorf("%w: failed to convert from runtime.Object to ManageableResourceInterface", ErrTypeConversion)
		}

		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(u.Object, c.t); err != nil {
			return err
		}

		return c.check(ctx, c.t)
	}

	if err := l.EachListItem(callback); err != nil {
		return err
	}

	return nil
}
