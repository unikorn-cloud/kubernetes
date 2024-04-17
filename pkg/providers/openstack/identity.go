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
	"context"
	"time"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack"
	"github.com/gophercloud/gophercloud/v2/openstack/identity/v3/applicationcredentials"
	"github.com/gophercloud/gophercloud/v2/openstack/identity/v3/projects"
	"github.com/gophercloud/gophercloud/v2/openstack/identity/v3/roles"
	"github.com/gophercloud/gophercloud/v2/openstack/identity/v3/tokens"
	"github.com/gophercloud/gophercloud/v2/openstack/identity/v3/users"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"

	"github.com/unikorn-cloud/core/pkg/util/cache"
	"github.com/unikorn-cloud/unikorn/pkg/constants"
)

// IdentityClient wraps up gophercloud identity management.
type IdentityClient struct {
	client *gophercloud.ServiceClient

	roleCache *cache.TimeoutCache[[]roles.Role]
}

// NewIdentityClient returns a new identity client.
func NewIdentityClient(ctx context.Context, provider CredentialProvider) (*IdentityClient, error) {
	providerClient, err := provider.Client(ctx)
	if err != nil {
		return nil, err
	}

	identity, err := openstack.NewIdentityV3(providerClient, gophercloud.EndpointOpts{})
	if err != nil {
		return nil, err
	}

	client := &IdentityClient{
		client:    identity,
		roleCache: cache.New[[]roles.Role](time.Hour),
	}

	return client, nil
}

// CreateTokenOptions abstracts away how schizophrenic Openstack is
// with its million options and million ways to fuck it up.
type CreateTokenOptions interface {
	// Options returns a valid set of authentication options.
	Options() *tokens.AuthOptions
}

// CreateTokenOptionsUnscopedPassword is typically used when logging on to a UI
// when you don't know anything other than username/password.
type CreateTokenOptionsUnscopedPassword struct {
	// domain a user belongs to.
	domain string

	// username of the user.
	username string

	// password of the user.
	password string
}

// Ensure the CreateTokenOptions interface is implemented.
var _ CreateTokenOptions = &CreateTokenOptionsUnscopedPassword{}

// NewCreateTokenOptionsUnscopedPassword returns a new instance of unscoped username/password options.
func NewCreateTokenOptionsUnscopedPassword(domain, username, password string) *CreateTokenOptionsUnscopedPassword {
	return &CreateTokenOptionsUnscopedPassword{
		domain:   domain,
		username: username,
		password: password,
	}
}

// Options implements the CreateTokenOptions interface.
func (o *CreateTokenOptionsUnscopedPassword) Options() *tokens.AuthOptions {
	return &tokens.AuthOptions{
		DomainName: o.domain,
		Username:   o.username,
		Password:   o.password,
	}
}

// CreateTokenOptionsScopedToken is typically used to upgrade from an unscoped
// password passed login to a project scoped one once you have determined
// a valid project.
type CreateTokenOptionsScopedToken struct {
	// token is the authentication token, it's already scoped to a user and
	// domain.
	token string

	// projectID is the project ID.  We expect an ID because the name/description
	// is returned to the user for context, however the ID being passed back in
	// defines both the domain and project name, so is simpler and less error
	// prone.
	projectID string
}

// Ensure the CreateTokenOptions interface is implemented.
var _ CreateTokenOptions = &CreateTokenOptionsScopedToken{}

// NewCreateTokenOptionsScopedToken returns a new instance of project scoped token options.
func NewCreateTokenOptionsScopedToken(token, projectID string) *CreateTokenOptionsScopedToken {
	return &CreateTokenOptionsScopedToken{
		token:     token,
		projectID: projectID,
	}
}

// Options implements the CreateTokenOptions interface.
func (o *CreateTokenOptionsScopedToken) Options() *tokens.AuthOptions {
	return &tokens.AuthOptions{
		TokenID: o.token,
		Scope: tokens.Scope{
			ProjectID: o.projectID,
		},
	}
}

// CreateToken issues a new token.
func (c *IdentityClient) CreateToken(ctx context.Context, options CreateTokenOptions) (*tokens.Token, *tokens.User, error) {
	tracer := otel.GetTracerProvider().Tracer(constants.Application)

	_, span := tracer.Start(ctx, "/identity/v3/auth/tokens", trace.WithSpanKind(trace.SpanKindClient))
	defer span.End()

	result := tokens.Create(ctx, c.client, options.Options())

	token, err := result.ExtractToken()
	if err != nil {
		return nil, nil, err
	}

	user, err := result.ExtractUser()
	if err != nil {
		return nil, nil, err
	}

	return token, user, nil
}

// CreateProject creates the named project.
func (c *IdentityClient) CreateProject(ctx context.Context, domainID, name string, tags []string) (*projects.Project, error) {
	tracer := otel.GetTracerProvider().Tracer(constants.Application)

	_, span := tracer.Start(ctx, "/identity/v3/auth/projects", trace.WithSpanKind(trace.SpanKindClient))
	defer span.End()

	// TODO: pass ID in from configuration.
	opts := &projects.CreateOpts{
		DomainID: domainID,
		Name:     name,
		Tags:     tags,
	}

	return projects.Create(ctx, c.client, opts).Extract()
}

func (c *IdentityClient) DeleteProject(ctx context.Context, projectID string) error {
	tracer := otel.GetTracerProvider().Tracer(constants.Application)

	_, span := tracer.Start(ctx, "/identity/v3/auth/projects/"+projectID, trace.WithSpanKind(trace.SpanKindClient))
	defer span.End()

	return projects.Delete(ctx, c.client, projectID).Err
}

// ListAvailableProjects lists projects that an authenticated (but unscoped) user can
// scope to.
func (c *IdentityClient) ListAvailableProjects(ctx context.Context) ([]projects.Project, error) {
	tracer := otel.GetTracerProvider().Tracer(constants.Application)

	_, span := tracer.Start(ctx, "/identity/v3/auth/projects", trace.WithSpanKind(trace.SpanKindClient))
	defer span.End()

	page, err := projects.ListAvailable(c.client).AllPages(ctx)
	if err != nil {
		return nil, err
	}

	items, err := projects.ExtractProjects(page)
	if err != nil {
		return nil, err
	}

	return items, nil
}

// ListRoles grabs a set of roles that are on the provider.
func (c *IdentityClient) ListRoles(ctx context.Context) ([]roles.Role, error) {
	if result, ok := c.roleCache.Get(); ok {
		return result, nil
	}

	tracer := otel.GetTracerProvider().Tracer(constants.Application)

	_, span := tracer.Start(ctx, "/identity/v3/auth/roles", trace.WithSpanKind(trace.SpanKindClient))
	defer span.End()

	page, err := roles.List(c.client, &roles.ListOpts{}).AllPages(ctx)
	if err != nil {
		return nil, err
	}

	items, err := roles.ExtractRoles(page)
	if err != nil {
		return nil, err
	}

	c.roleCache.Set(items)

	return items, nil
}

// CreateRoleAssignment creates a role between a user and a project.
func (c *IdentityClient) CreateRoleAssignment(ctx context.Context, userID, projectID, roleID string) error {
	tracer := otel.GetTracerProvider().Tracer(constants.Application)

	_, span := tracer.Start(ctx, "/identity/v3/auth/role_assignments", trace.WithSpanKind(trace.SpanKindClient))
	defer span.End()

	opts := roles.AssignOpts{
		UserID:    userID,
		ProjectID: projectID,
	}

	err := roles.Assign(ctx, c.client, roleID, opts).ExtractErr()
	if err != nil {
		return err
	}

	return nil
}

// ListApplicationCredentials lists application credentials for the scoped user.
func (c *IdentityClient) ListApplicationCredentials(ctx context.Context, userID string) ([]applicationcredentials.ApplicationCredential, error) {
	tracer := otel.GetTracerProvider().Tracer(constants.Application)

	_, span := tracer.Start(ctx, "/identity/v3/users/"+userID+"/application_credentials", trace.WithSpanKind(trace.SpanKindClient))
	defer span.End()

	page, err := applicationcredentials.List(c.client, userID, nil).AllPages(ctx)
	if err != nil {
		return nil, err
	}

	items, err := applicationcredentials.ExtractApplicationCredentials(page)
	if err != nil {
		return nil, err
	}

	return items, nil
}

// CreateApplicationCredential creates an application credential for the user.
func (c *IdentityClient) CreateApplicationCredential(ctx context.Context, userID, name, description string, roles []string) (*applicationcredentials.ApplicationCredential, error) {
	tracer := otel.GetTracerProvider().Tracer(constants.Application)

	_, span := tracer.Start(ctx, "/identity/v3/users/"+userID+"/application_credentials", trace.WithSpanKind(trace.SpanKindClient))
	defer span.End()

	applicationRoles := make([]applicationcredentials.Role, len(roles))

	for i, role := range roles {
		applicationRoles[i].Name = role
	}

	opts := &applicationcredentials.CreateOpts{
		Name:        name,
		Description: description,
		Roles:       applicationRoles,
	}

	result, err := applicationcredentials.Create(ctx, c.client, userID, opts).Extract()
	if err != nil {
		return nil, err
	}

	return result, err
}

// DeleteApplicationCredential deletes an application credential for the user.
func (c *IdentityClient) DeleteApplicationCredential(ctx context.Context, userID, id string) error {
	tracer := otel.GetTracerProvider().Tracer(constants.Application)

	_, span := tracer.Start(ctx, "/identity/v3/users/"+userID+"/application_credentials/"+id, trace.WithSpanKind(trace.SpanKindClient))
	defer span.End()

	return applicationcredentials.Delete(ctx, c.client, userID, id).ExtractErr()
}

// CreateUser creates a new user.
func (c *IdentityClient) CreateUser(ctx context.Context, domainID, name, password string) (*users.User, error) {
	tracer := otel.GetTracerProvider().Tracer(constants.Application)

	_, span := tracer.Start(ctx, "/identity/v3/users", trace.WithSpanKind(trace.SpanKindClient))
	defer span.End()

	opts := &users.CreateOpts{
		Name:     name,
		DomainID: domainID,
		Password: password,
	}

	return users.Create(ctx, c.client, opts).Extract()
}

// GetUser returns user details.
func (c *IdentityClient) GetUser(ctx context.Context, userID string) (*users.User, error) {
	tracer := otel.GetTracerProvider().Tracer(constants.Application)

	_, span := tracer.Start(ctx, "/identity/v3/users/"+userID, trace.WithSpanKind(trace.SpanKindClient))
	defer span.End()

	return users.Get(ctx, c.client, userID).Extract()
}

// DeleteUser removes an existing user.
func (c *IdentityClient) DeleteUser(ctx context.Context, userID string) error {
	tracer := otel.GetTracerProvider().Tracer(constants.Application)

	_, span := tracer.Start(ctx, "/identity/v3/users/"+userID, trace.WithSpanKind(trace.SpanKindClient))
	defer span.End()

	return users.Delete(ctx, c.client, userID).Err
}
