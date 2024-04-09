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

	"github.com/unikorn-cloud/core/pkg/authorization/rbac"
	"github.com/unikorn-cloud/core/pkg/authorization/userinfo"
	"github.com/unikorn-cloud/core/pkg/constants"

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

func (s *Scoper) listProjectNames() ([]string, error) {
	organization, err := s.permissions.LookupOrganization(s.organization)
	if err != nil {
		return nil, err
	}

	result := make([]string, len(organization.Projects))

	for i, project := range organization.Projects {
		result[i] = project.Name
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
	if !s.permissions.IsSuperAdmin {
		projects, err := s.listProjectNames()
		if err != nil {
			return nil, err
		}

		if len(projects) == 0 {
			log.Info("request has nothing in scope")

			return nil, ErrNoScope
		}

		log.Info("scoping request to projects", "projects", projects)

		projectReq, err := labels.NewRequirement(constants.ProjectLabel, selection.In, projects)
		if err != nil {
			return nil, err
		}

		selector = selector.Add(*projectReq)
	}

	return selector, nil
}
