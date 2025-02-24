/*
Copyright 2022-2024 EscherCloud.
Copyright 2024-2025 the Unikorn Authors.

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
	"github.com/unikorn-cloud/kubernetes/pkg/server/handler/virtualcluster"
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

	// identity is a client to access the identity service.
	identity *identityclient.Client

	// region is a client to access regions.
	region *regionclient.Client
}

func New(client client.Client, namespace string, options *Options, issuer *identityclient.TokenIssuer, identity *identityclient.Client, region *regionclient.Client) (*Handler, error) {
	h := &Handler{
		client:    client,
		namespace: namespace,
		options:   options,
		issuer:    issuer,
		identity:  identity,
		region:    region,
	}

	return h, nil
}

func (h *Handler) identityClient(ctx context.Context) (*identityapi.ClientWithResponses, error) {
	token, err := h.issuer.Issue(ctx, "kubernetes-api")
	if err != nil {
		return nil, err
	}

	identity, err := h.identity.Client(ctx, token)
	if err != nil {
		return nil, err
	}

	return identity, nil
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

func (h *Handler) GetApiV1OrganizationsOrganizationIDRegions(w http.ResponseWriter, r *http.Request, organizationID openapi.OrganizationIDParameter, params openapi.GetApiV1OrganizationsOrganizationIDRegionsParams) {
	ctx := r.Context()

	if err := rbac.AllowOrganizationScope(ctx, "kubernetes:regions", identityapi.Read, organizationID); err != nil {
		errors.HandleError(w, r, err)
		return
	}

	client, err := h.regionClient(ctx)
	if err != nil {
		errors.HandleError(w, r, err)
		return
	}

	result, err := region.Regions(ctx, client, organizationID, params)
	if err != nil {
		errors.HandleError(w, r, errors.OAuth2ServerError("unable to read regions").WithError(err))
		return
	}

	util.WriteJSONResponse(w, r, http.StatusOK, result)
}

func (h *Handler) GetApiV1OrganizationsOrganizationIDRegionsRegionIDFlavors(w http.ResponseWriter, r *http.Request, organizationID openapi.OrganizationIDParameter, regionID openapi.RegionIDParameter) {
	ctx := r.Context()

	if err := rbac.AllowOrganizationScope(ctx, "kubernetes:flavors", identityapi.Read, organizationID); err != nil {
		errors.HandleError(w, r, err)
		return
	}

	client, err := h.regionClient(ctx)
	if err != nil {
		errors.HandleError(w, r, err)
		return
	}

	result, err := region.Flavors(ctx, client, organizationID, regionID)
	if err != nil {
		errors.HandleError(w, r, errors.OAuth2ServerError("unable to read flavors").WithError(err))
		return
	}

	util.WriteJSONResponse(w, r, http.StatusOK, result)
}

func (h *Handler) GetApiV1OrganizationsOrganizationIDRegionsRegionIDImages(w http.ResponseWriter, r *http.Request, organizationID openapi.OrganizationIDParameter, regionID openapi.RegionIDParameter) {
	ctx := r.Context()

	if err := rbac.AllowOrganizationScope(ctx, "kubernetes:images", identityapi.Read, organizationID); err != nil {
		errors.HandleError(w, r, err)
		return
	}

	client, err := h.regionClient(ctx)
	if err != nil {
		errors.HandleError(w, r, err)
		return
	}

	result, err := region.Images(ctx, client, organizationID, regionID)
	if err != nil {
		errors.HandleError(w, r, errors.OAuth2ServerError("unable to read flavors").WithError(err))
		return
	}

	util.WriteJSONResponse(w, r, http.StatusOK, result)
}

func (h *Handler) GetApiV1OrganizationsOrganizationIDClustermanagers(w http.ResponseWriter, r *http.Request, organizationID openapi.OrganizationIDParameter) {
	ctx := r.Context()

	result, err := clustermanager.NewClient(h.client).List(ctx, organizationID)
	if err != nil {
		errors.HandleError(w, r, err)
		return
	}

	result = slices.DeleteFunc(result, func(resource openapi.ClusterManagerRead) bool {
		return rbac.AllowProjectScope(ctx, "kubernetes:clustermanagers", identityapi.Read, organizationID, resource.Metadata.ProjectId) != nil
	})

	h.setUncacheable(w)
	util.WriteJSONResponse(w, r, http.StatusOK, result)
}

func (h *Handler) PostApiV1OrganizationsOrganizationIDProjectsProjectIDClustermanagers(w http.ResponseWriter, r *http.Request, organizationID openapi.OrganizationIDParameter, projectID openapi.ProjectIDParameter) {
	ctx := r.Context()

	if err := rbac.AllowProjectScope(ctx, "kubernetes:clustermanagers", identityapi.Create, organizationID, projectID); err != nil {
		errors.HandleError(w, r, err)
		return
	}

	request := &openapi.ClusterManagerWrite{}

	if err := util.ReadJSONBody(r, request); err != nil {
		errors.HandleError(w, r, err)
		return
	}

	result, err := clustermanager.NewClient(h.client).Create(ctx, organizationID, projectID, request)
	if err != nil {
		errors.HandleError(w, r, err)
		return
	}

	h.setUncacheable(w)
	util.WriteJSONResponse(w, r, http.StatusAccepted, result)
}

func (h *Handler) DeleteApiV1OrganizationsOrganizationIDProjectsProjectIDClustermanagersClusterManagerID(w http.ResponseWriter, r *http.Request, organizationID openapi.OrganizationIDParameter, projectID openapi.ProjectIDParameter, clusterManagerID openapi.ClusterManagerIDParameter) {
	ctx := r.Context()

	if err := rbac.AllowProjectScope(ctx, "kubernetes:clustermanagers", identityapi.Delete, organizationID, projectID); err != nil {
		errors.HandleError(w, r, err)
		return
	}

	if err := clustermanager.NewClient(h.client).Delete(ctx, organizationID, projectID, clusterManagerID); err != nil {
		errors.HandleError(w, r, err)
		return
	}

	h.setUncacheable(w)
	w.WriteHeader(http.StatusAccepted)
}

func (h *Handler) PutApiV1OrganizationsOrganizationIDProjectsProjectIDClustermanagersClusterManagerID(w http.ResponseWriter, r *http.Request, organizationID openapi.OrganizationIDParameter, projectID openapi.ProjectIDParameter, clusterManagerID openapi.ClusterManagerIDParameter) {
	ctx := r.Context()

	if err := rbac.AllowProjectScope(ctx, "kubernetes:clustermanagers", identityapi.Update, organizationID, projectID); err != nil {
		errors.HandleError(w, r, err)
		return
	}

	request := &openapi.ClusterManagerWrite{}

	if err := util.ReadJSONBody(r, request); err != nil {
		errors.HandleError(w, r, err)
		return
	}

	if err := clustermanager.NewClient(h.client).Update(ctx, organizationID, projectID, clusterManagerID, request); err != nil {
		errors.HandleError(w, r, err)
		return
	}

	h.setUncacheable(w)
	w.WriteHeader(http.StatusAccepted)
}

func (h *Handler) clusterClient(ctx context.Context) (*cluster.Client, error) {
	identity, err := h.identityClient(ctx)
	if err != nil {
		return nil, err
	}

	region, err := h.regionClient(ctx)
	if err != nil {
		return nil, err
	}

	return cluster.NewClient(h.client, h.namespace, &h.options.Cluster, identity, region), nil
}

func (h *Handler) GetApiV1OrganizationsOrganizationIDClusters(w http.ResponseWriter, r *http.Request, organizationID openapi.OrganizationIDParameter) {
	ctx := r.Context()

	clusters, err := h.clusterClient(ctx)
	if err != nil {
		errors.HandleError(w, r, err)
		return
	}

	result, err := clusters.List(ctx, organizationID)
	if err != nil {
		errors.HandleError(w, r, err)
		return
	}

	result = slices.DeleteFunc(result, func(resource openapi.KubernetesClusterRead) bool {
		return rbac.AllowProjectScope(ctx, "kubernetes:clusters", identityapi.Read, organizationID, resource.Metadata.ProjectId) != nil
	})

	h.setUncacheable(w)
	util.WriteJSONResponse(w, r, http.StatusOK, result)
}

func (h *Handler) PostApiV1OrganizationsOrganizationIDProjectsProjectIDClusters(w http.ResponseWriter, r *http.Request, organizationID openapi.OrganizationIDParameter, projectID openapi.ProjectIDParameter) {
	ctx := r.Context()

	if err := rbac.AllowProjectScope(ctx, "kubernetes:clusters", identityapi.Create, organizationID, projectID); err != nil {
		errors.HandleError(w, r, err)
		return
	}

	request := &openapi.KubernetesClusterWrite{}

	if err := util.ReadJSONBody(r, request); err != nil {
		errors.HandleError(w, r, err)
		return
	}

	clusters, err := h.clusterClient(ctx)
	if err != nil {
		errors.HandleError(w, r, err)
		return
	}

	result, err := clusters.Create(ctx, organizationID, projectID, request)
	if err != nil {
		errors.HandleError(w, r, err)
		return
	}

	h.setUncacheable(w)
	util.WriteJSONResponse(w, r, http.StatusAccepted, result)
}

func (h *Handler) DeleteApiV1OrganizationsOrganizationIDProjectsProjectIDClustersClusterID(w http.ResponseWriter, r *http.Request, organizationID openapi.OrganizationIDParameter, projectID openapi.ProjectIDParameter, clusterID openapi.ClusterIDParameter) {
	ctx := r.Context()

	if err := rbac.AllowProjectScope(ctx, "kubernetes:clusters", identityapi.Delete, organizationID, projectID); err != nil {
		errors.HandleError(w, r, err)
		return
	}

	clusters, err := h.clusterClient(ctx)
	if err != nil {
		errors.HandleError(w, r, err)
		return
	}

	if err := clusters.Delete(ctx, organizationID, projectID, clusterID); err != nil {
		errors.HandleError(w, r, err)
		return
	}

	h.setUncacheable(w)
	w.WriteHeader(http.StatusAccepted)
}

//nolint:dupl
func (h *Handler) PutApiV1OrganizationsOrganizationIDProjectsProjectIDClustersClusterID(w http.ResponseWriter, r *http.Request, organizationID openapi.OrganizationIDParameter, projectID openapi.ProjectIDParameter, clusterID openapi.ClusterIDParameter) {
	ctx := r.Context()

	if err := rbac.AllowProjectScope(ctx, "kubernetes:clusters", identityapi.Update, organizationID, projectID); err != nil {
		errors.HandleError(w, r, err)
		return
	}

	request := &openapi.KubernetesClusterWrite{}

	if err := util.ReadJSONBody(r, request); err != nil {
		errors.HandleError(w, r, err)
		return
	}

	clusters, err := h.clusterClient(ctx)
	if err != nil {
		errors.HandleError(w, r, err)
		return
	}

	if err := clusters.Update(ctx, organizationID, projectID, clusterID, request); err != nil {
		errors.HandleError(w, r, err)
		return
	}

	h.setUncacheable(w)
	w.WriteHeader(http.StatusAccepted)
}

func (h *Handler) GetApiV1OrganizationsOrganizationIDProjectsProjectIDClustersClusterIDKubeconfig(w http.ResponseWriter, r *http.Request, organizationID openapi.OrganizationIDParameter, projectID openapi.ProjectIDParameter, clusterID openapi.ClusterIDParameter) {
	ctx := r.Context()

	if err := rbac.AllowProjectScope(ctx, "kubernetes:clusters", identityapi.Read, organizationID, projectID); err != nil {
		errors.HandleError(w, r, err)
		return
	}

	clusters, err := h.clusterClient(ctx)
	if err != nil {
		errors.HandleError(w, r, err)
		return
	}

	result, err := clusters.GetKubeconfig(ctx, organizationID, projectID, clusterID)
	if err != nil {
		errors.HandleError(w, r, err)
		return
	}

	h.setUncacheable(w)
	util.WriteOctetStreamResponse(w, r, http.StatusOK, result)
}

func (h *Handler) virtualClusterClient(ctx context.Context) (*virtualcluster.Client, error) {
	identity, err := h.identityClient(ctx)
	if err != nil {
		return nil, err
	}

	region, err := h.regionClient(ctx)
	if err != nil {
		return nil, err
	}

	return virtualcluster.NewClient(h.client, h.namespace, identity, region), nil
}

func (h *Handler) GetApiV1OrganizationsOrganizationIDVirtualclusters(w http.ResponseWriter, r *http.Request, organizationID openapi.OrganizationIDParameter) {
	ctx := r.Context()

	virtualclusters, err := h.virtualClusterClient(ctx)
	if err != nil {
		errors.HandleError(w, r, err)
		return
	}

	result, err := virtualclusters.List(ctx, organizationID)
	if err != nil {
		errors.HandleError(w, r, err)
		return
	}

	result = slices.DeleteFunc(result, func(resource openapi.VirtualKubernetesClusterRead) bool {
		return rbac.AllowProjectScope(ctx, "kubernetes:virtualclusters", identityapi.Read, organizationID, resource.Metadata.ProjectId) != nil
	})

	h.setUncacheable(w)
	util.WriteJSONResponse(w, r, http.StatusOK, result)
}

func (h *Handler) PostApiV1OrganizationsOrganizationIDProjectsProjectIDVirtualclusters(w http.ResponseWriter, r *http.Request, organizationID openapi.OrganizationIDParameter, projectID openapi.ProjectIDParameter) {
	ctx := r.Context()

	if err := rbac.AllowProjectScope(ctx, "kubernetes:virtualclusters", identityapi.Create, organizationID, projectID); err != nil {
		errors.HandleError(w, r, err)
		return
	}

	request := &openapi.VirtualKubernetesClusterWrite{}

	if err := util.ReadJSONBody(r, request); err != nil {
		errors.HandleError(w, r, err)
		return
	}

	virtualclusters, err := h.virtualClusterClient(ctx)
	if err != nil {
		errors.HandleError(w, r, err)
		return
	}

	result, err := virtualclusters.Create(ctx, organizationID, projectID, request)
	if err != nil {
		errors.HandleError(w, r, err)
		return
	}

	h.setUncacheable(w)
	util.WriteJSONResponse(w, r, http.StatusAccepted, result)
}

func (h *Handler) DeleteApiV1OrganizationsOrganizationIDProjectsProjectIDVirtualclustersClusterID(w http.ResponseWriter, r *http.Request, organizationID openapi.OrganizationIDParameter, projectID openapi.ProjectIDParameter, clusterID openapi.ClusterIDParameter) {
	ctx := r.Context()

	if err := rbac.AllowProjectScope(ctx, "kubernetes:virtualclusters", identityapi.Delete, organizationID, projectID); err != nil {
		errors.HandleError(w, r, err)
		return
	}

	virtualclusters, err := h.virtualClusterClient(ctx)
	if err != nil {
		errors.HandleError(w, r, err)
		return
	}

	if err := virtualclusters.Delete(ctx, organizationID, projectID, clusterID); err != nil {
		errors.HandleError(w, r, err)
		return
	}

	h.setUncacheable(w)
	w.WriteHeader(http.StatusAccepted)
}

//nolint:dupl
func (h *Handler) PutApiV1OrganizationsOrganizationIDProjectsProjectIDVirtualclustersClusterID(w http.ResponseWriter, r *http.Request, organizationID openapi.OrganizationIDParameter, projectID openapi.ProjectIDParameter, clusterID openapi.ClusterIDParameter) {
	ctx := r.Context()

	if err := rbac.AllowProjectScope(ctx, "kubernetes:virtualclusters", identityapi.Update, organizationID, projectID); err != nil {
		errors.HandleError(w, r, err)
		return
	}

	request := &openapi.VirtualKubernetesClusterWrite{}

	if err := util.ReadJSONBody(r, request); err != nil {
		errors.HandleError(w, r, err)
		return
	}

	virtualclusters, err := h.virtualClusterClient(ctx)
	if err != nil {
		errors.HandleError(w, r, err)
		return
	}

	if err := virtualclusters.Update(ctx, organizationID, projectID, clusterID, request); err != nil {
		errors.HandleError(w, r, err)
		return
	}

	h.setUncacheable(w)
	w.WriteHeader(http.StatusAccepted)
}

func (h *Handler) GetApiV1OrganizationsOrganizationIDProjectsProjectIDVirtualclustersClusterIDKubeconfig(w http.ResponseWriter, r *http.Request, organizationID openapi.OrganizationIDParameter, projectID openapi.ProjectIDParameter, clusterID openapi.ClusterIDParameter) {
	ctx := r.Context()

	if err := rbac.AllowProjectScope(ctx, "kubernetes:virtualclusters", identityapi.Read, organizationID, projectID); err != nil {
		errors.HandleError(w, r, err)
		return
	}

	virtualclusters, err := h.virtualClusterClient(ctx)
	if err != nil {
		errors.HandleError(w, r, err)
		return
	}

	result, err := virtualclusters.GetKubeconfig(ctx, organizationID, projectID, clusterID)
	if err != nil {
		errors.HandleError(w, r, err)
		return
	}

	h.setUncacheable(w)
	util.WriteOctetStreamResponse(w, r, http.StatusOK, result)
}
