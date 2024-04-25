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

//nolint:revive,stylecheck
package handler

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"time"

	"github.com/unikorn-cloud/core/pkg/authorization/constants"
	"github.com/unikorn-cloud/core/pkg/authorization/rbac"
	"github.com/unikorn-cloud/core/pkg/authorization/userinfo"
	"github.com/unikorn-cloud/core/pkg/server/errors"
	"github.com/unikorn-cloud/core/pkg/server/middleware/openapi/oidc"
	coreutil "github.com/unikorn-cloud/core/pkg/util"
	"github.com/unikorn-cloud/unikorn/pkg/server/generated"
	"github.com/unikorn-cloud/unikorn/pkg/server/handler/application"
	"github.com/unikorn-cloud/unikorn/pkg/server/handler/cluster"
	"github.com/unikorn-cloud/unikorn/pkg/server/handler/clustermanager"
	"github.com/unikorn-cloud/unikorn/pkg/server/handler/region"
	"github.com/unikorn-cloud/unikorn/pkg/server/util"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Handler struct {
	// client gives cached access to Kubernetes.
	client client.Client

	// options allows behaviour to be defined on the CLI.
	options *Options

	// authorizerOptions allows access to the identity service for RBAC callbacks.
	authorizerOptions *oidc.Options
}

func New(client client.Client, options *Options, authorizerOptions *oidc.Options) (*Handler, error) {
	h := &Handler{
		client:            client,
		options:           options,
		authorizerOptions: authorizerOptions,
	}

	return h, nil
}

func (h *Handler) setCacheable(w http.ResponseWriter) {
	w.Header().Add("Cache-Control", fmt.Sprintf("max-age=%d", h.options.CacheMaxAge/time.Second))
	w.Header().Add("Cache-Control", "private")
}

func (h *Handler) setUncacheable(w http.ResponseWriter) {
	w.Header().Add("Cache-Control", "no-cache")
}

//nolint:unparam
func (h *Handler) checkRBAC(ctx context.Context, organization, scope string, permission constants.Permission) error {
	aclGetter := rbac.NewIdentityACLGetter(h.authorizerOptions.Issuer, organization).WithCA(h.authorizerOptions.IssuerCA)

	authorizer, err := userinfo.NewAuthorizer(ctx, aclGetter)
	if err != nil {
		return errors.HTTPForbidden("operation is not allowed by rbac").WithError(err)
	}

	if err := authorizer.Allow(ctx, scope, permission); err != nil {
		return errors.HTTPForbidden("operation is not allowed by rbac").WithError(err)
	}

	return nil
}

func (h *Handler) GetApiV1OrganizationsOrganizationClustermanagers(w http.ResponseWriter, r *http.Request, organization generated.OrganizationParameter) {
	if err := h.checkRBAC(r.Context(), organization, "infrastructure", constants.Read); err != nil {
		errors.HandleError(w, r, err)
		return
	}

	result, err := clustermanager.NewClient(h.client).List(r.Context(), organization)
	if err != nil {
		errors.HandleError(w, r, err)
		return
	}

	h.setUncacheable(w)
	util.WriteJSONResponse(w, r, http.StatusOK, result)
}

func (h *Handler) PostApiV1OrganizationsOrganizationProjectsProjectClustermanagers(w http.ResponseWriter, r *http.Request, organization generated.OrganizationParameter, project generated.ProjectParameter) {
	if err := h.checkRBAC(r.Context(), organization, "infrastructure", constants.Create); err != nil {
		errors.HandleError(w, r, err)
		return
	}

	request := &generated.ClusterManagerSpec{}

	if err := util.ReadJSONBody(r, request); err != nil {
		errors.HandleError(w, r, err)
		return
	}

	if err := clustermanager.NewClient(h.client).Create(r.Context(), organization, project, request); err != nil {
		errors.HandleError(w, r, err)
		return
	}

	h.setUncacheable(w)
	w.WriteHeader(http.StatusAccepted)
}

func (h *Handler) DeleteApiV1OrganizationsOrganizationProjectsProjectClustermanagersClusterManager(w http.ResponseWriter, r *http.Request, organization generated.OrganizationParameter, project generated.ProjectParameter, clusterManager generated.ClusterManagerParameter) {
	if err := h.checkRBAC(r.Context(), organization, "infrastructure", constants.Delete); err != nil {
		errors.HandleError(w, r, err)
		return
	}

	if err := clustermanager.NewClient(h.client).Delete(r.Context(), organization, project, clusterManager); err != nil {
		errors.HandleError(w, r, err)
		return
	}

	h.setUncacheable(w)
	w.WriteHeader(http.StatusAccepted)
}

func (h *Handler) PutApiV1OrganizationsOrganizationProjectsProjectClustermanagersClusterManager(w http.ResponseWriter, r *http.Request, organization generated.OrganizationParameter, project generated.ProjectParameter, controlPlaneName generated.ClusterManagerParameter) {
	if err := h.checkRBAC(r.Context(), organization, "infrastructure", constants.Update); err != nil {
		errors.HandleError(w, r, err)
		return
	}

	request := &generated.ClusterManagerSpec{}

	if err := util.ReadJSONBody(r, request); err != nil {
		errors.HandleError(w, r, err)
		return
	}

	if err := clustermanager.NewClient(h.client).Update(r.Context(), organization, project, controlPlaneName, request); err != nil {
		errors.HandleError(w, r, err)
		return
	}

	h.setUncacheable(w)
	w.WriteHeader(http.StatusAccepted)
}

func (h *Handler) GetApiV1OrganizationsOrganizationClusters(w http.ResponseWriter, r *http.Request, organization generated.OrganizationParameter) {
	if err := h.checkRBAC(r.Context(), organization, "infrastructure", constants.Read); err != nil {
		errors.HandleError(w, r, err)
		return
	}

	result, err := cluster.NewClient(h.client, &h.options.Cluster).List(r.Context(), organization)
	if err != nil {
		errors.HandleError(w, r, err)
		return
	}

	h.setUncacheable(w)
	util.WriteJSONResponse(w, r, http.StatusOK, result)
}

func (h *Handler) PostApiV1OrganizationsOrganizationProjectsProjectClusters(w http.ResponseWriter, r *http.Request, organization generated.OrganizationParameter, project generated.ProjectParameter) {
	if err := h.checkRBAC(r.Context(), organization, "infrastructure", constants.Create); err != nil {
		errors.HandleError(w, r, err)
		return
	}

	request := &generated.KubernetesClusterSpec{}

	if err := util.ReadJSONBody(r, request); err != nil {
		errors.HandleError(w, r, err)
		return
	}

	if err := cluster.NewClient(h.client, &h.options.Cluster).Create(r.Context(), organization, project, request); err != nil {
		errors.HandleError(w, r, err)
		return
	}

	h.setUncacheable(w)
	w.WriteHeader(http.StatusAccepted)
}

func (h *Handler) DeleteApiV1OrganizationsOrganizationProjectsProjectClustersCluster(w http.ResponseWriter, r *http.Request, organization generated.OrganizationParameter, project generated.ProjectParameter, clusterName generated.ClusterParameter) {
	if err := h.checkRBAC(r.Context(), organization, "infrastructure", constants.Delete); err != nil {
		errors.HandleError(w, r, err)
		return
	}

	if err := cluster.NewClient(h.client, &h.options.Cluster).Delete(r.Context(), organization, project, clusterName); err != nil {
		errors.HandleError(w, r, err)
		return
	}

	h.setUncacheable(w)
	w.WriteHeader(http.StatusAccepted)
}

func (h *Handler) PutApiV1OrganizationsOrganizationProjectsProjectClustersCluster(w http.ResponseWriter, r *http.Request, organization generated.OrganizationParameter, project generated.ProjectParameter, clusterName generated.ClusterParameter) {
	if err := h.checkRBAC(r.Context(), organization, "infrastructure", constants.Update); err != nil {
		errors.HandleError(w, r, err)
		return
	}

	request := &generated.KubernetesClusterSpec{}

	if err := util.ReadJSONBody(r, request); err != nil {
		errors.HandleError(w, r, err)
		return
	}

	if err := cluster.NewClient(h.client, &h.options.Cluster).Update(r.Context(), organization, project, clusterName, request); err != nil {
		errors.HandleError(w, r, err)
		return
	}

	h.setUncacheable(w)
	w.WriteHeader(http.StatusAccepted)
}

func (h *Handler) GetApiV1OrganizationsOrganizationProjectsProjectClustersClusterKubeconfig(w http.ResponseWriter, r *http.Request, organization generated.OrganizationParameter, project generated.ProjectParameter, clusterName generated.ClusterParameter) {
	if err := h.checkRBAC(r.Context(), organization, "infrastructure", constants.Read); err != nil {
		errors.HandleError(w, r, err)
		return
	}

	result, err := cluster.NewClient(h.client, &h.options.Cluster).GetKubeconfig(r.Context(), organization, project, clusterName)
	if err != nil {
		errors.HandleError(w, r, err)
		return
	}

	h.setUncacheable(w)
	util.WriteOctetStreamResponse(w, r, http.StatusOK, result)
}

func (h *Handler) GetApiV1Applications(w http.ResponseWriter, r *http.Request) {
	result, err := application.NewClient(h.client).List(r.Context())
	if err != nil {
		errors.HandleError(w, r, err)
		return
	}

	h.setUncacheable(w)
	util.WriteJSONResponse(w, r, http.StatusOK, result)
}

func (h *Handler) GetApiV1Regions(w http.ResponseWriter, r *http.Request) {
	result, err := region.NewClient(h.client).List(r.Context())
	if err != nil {
		errors.HandleError(w, r, err)
		return
	}

	h.setUncacheable(w)
	util.WriteJSONResponse(w, r, http.StatusOK, result)
}

func (h *Handler) GetApiV1RegionsRegionNameFlavors(w http.ResponseWriter, r *http.Request, regionName generated.RegionNameParameter) {
	provider, err := region.NewClient(h.client).Provider(r.Context(), regionName)
	if err != nil {
		errors.HandleError(w, r, err)
		return
	}

	result, err := provider.Flavors(r.Context())
	if err != nil {
		errors.HandleError(w, r, err)
		return
	}

	// Apply ordering guarantees.
	sort.Stable(result)

	out := make(generated.Flavors, 0, len(result))

	for _, r := range result {
		t := generated.Flavor{
			Name:   r.Name,
			Cpus:   r.CPUs,
			Memory: int(r.Memory.Value()) >> 30,
			Disk:   int(r.Disk.Value()) / 1000000000,
		}

		if r.GPUs != 0 {
			t.Gpus = coreutil.ToPointer(r.GPUs)
		}

		out = append(out, t)
	}

	h.setCacheable(w)
	util.WriteJSONResponse(w, r, http.StatusOK, out)
}

func (h *Handler) GetApiV1RegionsRegionNameImages(w http.ResponseWriter, r *http.Request, regionName generated.RegionNameParameter) {
	provider, err := region.NewClient(h.client).Provider(r.Context(), regionName)
	if err != nil {
		errors.HandleError(w, r, err)
		return
	}

	result, err := provider.Images(r.Context())
	if err != nil {
		errors.HandleError(w, r, err)
		return
	}

	// Apply ordering guarantees.
	sort.Stable(result)

	out := make(generated.Images, 0, len(result))

	for _, r := range result {
		out = append(out, generated.Image{
			Name:     r.Name,
			Created:  r.Created,
			Modified: r.Modified,
			Versions: generated.ImageVersions{
				Kubernetes: r.KubernetesVersion,
			},
		})
	}

	h.setCacheable(w)
	util.WriteJSONResponse(w, r, http.StatusOK, out)
}
