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
	"net/http"
	"slices"

	"github.com/unikorn-cloud/core/pkg/server/errors"
	"github.com/unikorn-cloud/core/pkg/server/util"
	identityclient "github.com/unikorn-cloud/identity/pkg/client"
	identityapi "github.com/unikorn-cloud/identity/pkg/openapi"
	"github.com/unikorn-cloud/identity/pkg/rbac"
	"github.com/unikorn-cloud/kubernetes/pkg/openapi"
	"github.com/unikorn-cloud/kubernetes/pkg/server/handler/cluster"
	"github.com/unikorn-cloud/kubernetes/pkg/server/handler/clustermanager"
	"github.com/unikorn-cloud/kubernetes/pkg/server/handler/region"
	regionclient "github.com/unikorn-cloud/region/pkg/client"
	regionapi "github.com/unikorn-cloud/region/pkg/openapi"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Handler struct {
	// client gives cached access to Kubernetes.
	client client.Client

	// namespace is where the controller is running.
	namespace string

	// options allows behaviour to be defined on the CLI.
	options *Options

	// issuer provides privilge escallation for the API so the end user doesn't
	// have to be granted unnecessary privilige.
	issuer *identityclient.TokenIssuer

	// region is a client to access regions.
	region *regionclient.Client
}

func New(client client.Client, namespace string, options *Options, issuer *identityclient.TokenIssuer, region *regionclient.Client) (*Handler, error) {
	h := &Handler{
		client:    client,
		namespace: namespace,
		options:   options,
		issuer:    issuer,
		region:    region,
	}

	return h, nil
}

func (h *Handler) regionClient(ctx context.Context) (*regionapi.ClientWithResponses, error) {
	token, err := h.issuer.Issue(ctx, "kubernetes-api")
	if err != nil {
		return nil, err
	}

	region, err := h.region.Client(ctx, token)
	if err != nil {
		return nil, err
	}

	return region, nil
}

/*
func (h *Handler) setCacheable(w http.ResponseWriter) {
	w.Header().Add("Cache-Control", fmt.Sprintf("max-age=%d", h.options.CacheMaxAge/time.Second))
	w.Header().Add("Cache-Control", "private")
}
*/

func (h *Handler) setUncacheable(w http.ResponseWriter) {
	w.Header().Add("Cache-Control", "no-cache")
}

func (h *Handler) GetApiV1OrganizationsOrganizationIDRegionsRegionIDFlavors(w http.ResponseWriter, r *http.Request, organizationID openapi.OrganizationIDParameter, regionID openapi.RegionIDParameter) {
	if err := rbac.AllowOrganizationScope(r.Context(), "kubernetes:flavors", identityapi.Read, organizationID); err != nil {
		errors.HandleError(w, r, err)
		return
	}

	client, err := h.regionClient(r.Context())
	if err != nil {
		errors.HandleError(w, r, err)
		return
	}

	result, err := region.Flavors(r.Context(), client, organizationID, regionID)
	if err != nil {
		errors.HandleError(w, r, errors.OAuth2ServerError("unable to read flavors").WithError(err))
		return
	}

	util.WriteJSONResponse(w, r, http.StatusOK, result)
}

func (h *Handler) GetApiV1OrganizationsOrganizationIDRegionsRegionIDImages(w http.ResponseWriter, r *http.Request, organizationID openapi.OrganizationIDParameter, regionID openapi.RegionIDParameter) {
	if err := rbac.AllowOrganizationScope(r.Context(), "kubernetes:images", identityapi.Read, organizationID); err != nil {
		errors.HandleError(w, r, err)
		return
	}

	client, err := h.regionClient(r.Context())
	if err != nil {
		errors.HandleError(w, r, err)
		return
	}

	result, err := region.Images(r.Context(), client, organizationID, regionID)
	if err != nil {
		errors.HandleError(w, r, errors.OAuth2ServerError("unable to read flavors").WithError(err))
		return
	}

	util.WriteJSONResponse(w, r, http.StatusOK, result)
}

func (h *Handler) GetApiV1OrganizationsOrganizationIDClustermanagers(w http.ResponseWriter, r *http.Request, organizationID openapi.OrganizationIDParameter) {
	result, err := clustermanager.NewClient(h.client).List(r.Context(), organizationID)
	if err != nil {
		errors.HandleError(w, r, err)
		return
	}

	ctx := r.Context()

	result = slices.DeleteFunc(result, func(resource openapi.ClusterManagerRead) bool {
		return rbac.AllowProjectScope(ctx, "kubernetes:clustermanagers", identityapi.Read, organizationID, resource.Metadata.ProjectId) != nil
	})

	h.setUncacheable(w)
	util.WriteJSONResponse(w, r, http.StatusOK, result)
}

func (h *Handler) PostApiV1OrganizationsOrganizationIDProjectsProjectIDClustermanagers(w http.ResponseWriter, r *http.Request, organizationID openapi.OrganizationIDParameter, projectID openapi.ProjectIDParameter) {
	if err := rbac.AllowProjectScope(r.Context(), "kubernetes:clustermanagers", identityapi.Create, organizationID, projectID); err != nil {
		errors.HandleError(w, r, err)
		return
	}

	request := &openapi.ClusterManagerWrite{}

	if err := util.ReadJSONBody(r, request); err != nil {
		errors.HandleError(w, r, err)
		return
	}

	result, err := clustermanager.NewClient(h.client).Create(r.Context(), organizationID, projectID, request)
	if err != nil {
		errors.HandleError(w, r, err)
		return
	}

	h.setUncacheable(w)
	util.WriteJSONResponse(w, r, http.StatusAccepted, result)
}

func (h *Handler) DeleteApiV1OrganizationsOrganizationIDProjectsProjectIDClustermanagersClusterManagerID(w http.ResponseWriter, r *http.Request, organizationID openapi.OrganizationIDParameter, projectID openapi.ProjectIDParameter, clusterManagerID openapi.ClusterManagerIDParameter) {
	if err := rbac.AllowProjectScope(r.Context(), "kubernetes:clustermanagers", identityapi.Delete, organizationID, projectID); err != nil {
		errors.HandleError(w, r, err)
		return
	}

	if err := clustermanager.NewClient(h.client).Delete(r.Context(), organizationID, projectID, clusterManagerID); err != nil {
		errors.HandleError(w, r, err)
		return
	}

	h.setUncacheable(w)
	w.WriteHeader(http.StatusAccepted)
}

func (h *Handler) PutApiV1OrganizationsOrganizationIDProjectsProjectIDClustermanagersClusterManagerID(w http.ResponseWriter, r *http.Request, organizationID openapi.OrganizationIDParameter, projectID openapi.ProjectIDParameter, clusterManagerID openapi.ClusterManagerIDParameter) {
	if err := rbac.AllowProjectScope(r.Context(), "kubernetes:clustermanagers", identityapi.Update, organizationID, projectID); err != nil {
		errors.HandleError(w, r, err)
		return
	}

	request := &openapi.ClusterManagerWrite{}

	if err := util.ReadJSONBody(r, request); err != nil {
		errors.HandleError(w, r, err)
		return
	}

	if err := clustermanager.NewClient(h.client).Update(r.Context(), organizationID, projectID, clusterManagerID, request); err != nil {
		errors.HandleError(w, r, err)
		return
	}

	h.setUncacheable(w)
	w.WriteHeader(http.StatusAccepted)
}

func (h *Handler) GetApiV1OrganizationsOrganizationIDClusters(w http.ResponseWriter, r *http.Request, organizationID openapi.OrganizationIDParameter) {
	region, err := h.regionClient(r.Context())
	if err != nil {
		errors.HandleError(w, r, err)
		return
	}

	result, err := cluster.NewClient(h.client, h.namespace, &h.options.Cluster, region).List(r.Context(), organizationID)
	if err != nil {
		errors.HandleError(w, r, err)
		return
	}

	ctx := r.Context()

	result = slices.DeleteFunc(result, func(resource openapi.KubernetesClusterRead) bool {
		return rbac.AllowProjectScope(ctx, "kubernetes:clusters", identityapi.Read, organizationID, resource.Metadata.ProjectId) != nil
	})

	h.setUncacheable(w)
	util.WriteJSONResponse(w, r, http.StatusOK, result)
}

func (h *Handler) PostApiV1OrganizationsOrganizationIDProjectsProjectIDClusters(w http.ResponseWriter, r *http.Request, organizationID openapi.OrganizationIDParameter, projectID openapi.ProjectIDParameter) {
	if err := rbac.AllowProjectScope(r.Context(), "kubernetes:clusters", identityapi.Create, organizationID, projectID); err != nil {
		errors.HandleError(w, r, err)
		return
	}

	request := &openapi.KubernetesClusterWrite{}

	if err := util.ReadJSONBody(r, request); err != nil {
		errors.HandleError(w, r, err)
		return
	}

	region, err := h.regionClient(r.Context())
	if err != nil {
		errors.HandleError(w, r, err)
		return
	}

	result, err := cluster.NewClient(h.client, h.namespace, &h.options.Cluster, region).Create(r.Context(), organizationID, projectID, request)
	if err != nil {
		errors.HandleError(w, r, err)
		return
	}

	h.setUncacheable(w)
	util.WriteJSONResponse(w, r, http.StatusAccepted, result)
}

func (h *Handler) DeleteApiV1OrganizationsOrganizationIDProjectsProjectIDClustersClusterID(w http.ResponseWriter, r *http.Request, organizationID openapi.OrganizationIDParameter, projectID openapi.ProjectIDParameter, clusterID openapi.ClusterIDParameter) {
	if err := rbac.AllowProjectScope(r.Context(), "kubernetes:clusters", identityapi.Delete, organizationID, projectID); err != nil {
		errors.HandleError(w, r, err)
		return
	}

	region, err := h.regionClient(r.Context())
	if err != nil {
		errors.HandleError(w, r, err)
		return
	}

	if err := cluster.NewClient(h.client, h.namespace, &h.options.Cluster, region).Delete(r.Context(), organizationID, projectID, clusterID); err != nil {
		errors.HandleError(w, r, err)
		return
	}

	h.setUncacheable(w)
	w.WriteHeader(http.StatusAccepted)
}

func (h *Handler) PutApiV1OrganizationsOrganizationIDProjectsProjectIDClustersClusterID(w http.ResponseWriter, r *http.Request, organizationID openapi.OrganizationIDParameter, projectID openapi.ProjectIDParameter, clusterID openapi.ClusterIDParameter) {
	if err := rbac.AllowProjectScope(r.Context(), "kubernetes:clusters", identityapi.Update, organizationID, projectID); err != nil {
		errors.HandleError(w, r, err)
		return
	}

	request := &openapi.KubernetesClusterWrite{}

	if err := util.ReadJSONBody(r, request); err != nil {
		errors.HandleError(w, r, err)
		return
	}

	region, err := h.regionClient(r.Context())
	if err != nil {
		errors.HandleError(w, r, err)
		return
	}

	if err := cluster.NewClient(h.client, h.namespace, &h.options.Cluster, region).Update(r.Context(), organizationID, projectID, clusterID, request); err != nil {
		errors.HandleError(w, r, err)
		return
	}

	h.setUncacheable(w)
	w.WriteHeader(http.StatusAccepted)
}

func (h *Handler) GetApiV1OrganizationsOrganizationIDProjectsProjectIDClustersClusterIDKubeconfig(w http.ResponseWriter, r *http.Request, organizationID openapi.OrganizationIDParameter, projectID openapi.ProjectIDParameter, clusterID openapi.ClusterIDParameter) {
	if err := rbac.AllowProjectScope(r.Context(), "kubernetes:clusters", identityapi.Read, organizationID, projectID); err != nil {
		errors.HandleError(w, r, err)
		return
	}

	region, err := h.regionClient(r.Context())
	if err != nil {
		errors.HandleError(w, r, err)
		return
	}

	result, err := cluster.NewClient(h.client, h.namespace, &h.options.Cluster, region).GetKubeconfig(r.Context(), organizationID, projectID, clusterID)
	if err != nil {
		errors.HandleError(w, r, err)
		return
	}

	h.setUncacheable(w)
	util.WriteOctetStreamResponse(w, r, http.StatusOK, result)
}
