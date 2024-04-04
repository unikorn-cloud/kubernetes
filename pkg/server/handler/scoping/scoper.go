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

package scoping

import (
	"context"
	"errors"
	"slices"

	"github.com/unikorn-cloud/core/pkg/authorization/rbac"
	"github.com/unikorn-cloud/core/pkg/authorization/userinfo"
	"github.com/unikorn-cloud/core/pkg/constants"
	unikornv1 "github.com/unikorn-cloud/unikorn/pkg/apis/unikorn/v1alpha1"

	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

var (
	ErrNoScope = errors.New("nothing in scope")
)

type Scoper struct {
	client       client.Client
	permissions  *rbac.Permissions
	organization string
}

func New(ctx context.Context, client client.Client, organization string) *Scoper {
	userinfo := userinfo.FromContext(ctx)

	return &Scoper{
		client:       client,
		permissions:  userinfo.RBAC,
		organization: organization,
	}
}

func (s *Scoper) MustApplyScope() bool {
	// Super admin sees all.
	if s.permissions.IsSuperAdmin {
		return false
	}

	// NOTE: RBAC should have determined this exists by now.
	organization, _ := s.permissions.LookupOrganization(s.organization)

	// Organization admin sees all.
	for _, group := range organization.Groups {
		if slices.Contains(group.Roles, "admin") {
			return false
		}
	}

	return true
}

func (s *Scoper) isGroupMember(groupIDs []string) bool {
	// NOTE: RBAC should have determined this exists by now.
	organization, _ := s.permissions.LookupOrganization(s.organization)

	for _, group := range organization.Groups {
		if slices.Contains(groupIDs, group.ID) {
			return true
		}
	}

	return false
}

func (s *Scoper) ListProjects(ctx context.Context) (*unikornv1.ProjectList, error) {
	selector := labels.NewSelector()

	orgRequirement, err := labels.NewRequirement(constants.OrganizationLabel, selection.Equals, []string{s.organization})
	if err != nil {
		return nil, err
	}

	selector = selector.Add(*orgRequirement)

	options := &client.ListOptions{
		LabelSelector: selector,
	}

	result := &unikornv1.ProjectList{}

	if err := s.client.List(ctx, result, options); err != nil {
		return nil, err
	}

	if s.MustApplyScope() {
		result.Items = slices.DeleteFunc(result.Items, func(project unikornv1.Project) bool {
			return !s.isGroupMember(project.Spec.GroupIDs)
		})
	}

	return result, nil
}

func (s *Scoper) listProjectNames(ctx context.Context) ([]string, error) {
	projects, err := s.ListProjects(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]string, len(projects.Items))

	for i := range projects.Items {
		result[i] = projects.Items[i].Name
	}

	return result, nil
}

func (s *Scoper) GetSelector(ctx context.Context) (labels.Selector, error) {
	log := log.FromContext(ctx)

	selector := labels.NewSelector()

	// Scope to the organization.
	orgReq, err := labels.NewRequirement(constants.OrganizationLabel, selection.Equals, []string{s.organization})
	if err != nil {
		return nil, err
	}

	selector = selector.Add(*orgReq)

	// Scope to projects that you are a member of, if necessary.
	if s.MustApplyScope() {
		projects, err := s.listProjectNames(ctx)
		if err != nil {
			return nil, err
		}

		if len(projects) == 0 {
			log.Info("request has nothing in scope")

			return nil, ErrNoScope
		}

		log.Info("scoping request to projects", "projects", projects)

		if err != nil {
			return nil, err
		}

		projectReq, err := labels.NewRequirement(constants.ProjectLabel, selection.In, projects)
		if err != nil {
			return nil, err
		}

		selector = selector.Add(*projectReq)
	}

	return selector, nil
}
