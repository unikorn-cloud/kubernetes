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
	"fmt"
	"net/http"
	"time"

	"github.com/unikorn-cloud/unikorn/pkg/server/errors"
	"github.com/unikorn-cloud/unikorn/pkg/server/generated"
	"github.com/unikorn-cloud/unikorn/pkg/server/handler/application"
	"github.com/unikorn-cloud/unikorn/pkg/server/handler/applicationbundle"
	"github.com/unikorn-cloud/unikorn/pkg/server/handler/cluster"
	"github.com/unikorn-cloud/unikorn/pkg/server/handler/controlplane"
	"github.com/unikorn-cloud/unikorn/pkg/server/handler/project"
	"github.com/unikorn-cloud/unikorn/pkg/server/handler/region"
	"github.com/unikorn-cloud/unikorn/pkg/server/util"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Handler struct {
	// client gives cached access to Kubernetes.
	client client.Client

	// options allows behaviour to be defined on the CLI.
	options *Options
}

func New(client client.Client, options *Options) (*Handler, error) {
	h := &Handler{
		client:  client,
		options: options,
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

func (h *Handler) PostApiV1Project(w http.ResponseWriter, r *http.Request) {
	if err := project.NewClient(h.client).Create(r.Context()); err != nil {
		errors.HandleError(w, r, err)
		return
	}

	h.setUncacheable(w)
	w.WriteHeader(http.StatusAccepted)
}

func (h *Handler) DeleteApiV1Project(w http.ResponseWriter, r *http.Request) {
	if err := project.NewClient(h.client).Delete(r.Context()); err != nil {
		errors.HandleError(w, r, err)
		return
	}

	h.setUncacheable(w)
	w.WriteHeader(http.StatusAccepted)
}

func (h *Handler) GetApiV1Controlplanes(w http.ResponseWriter, r *http.Request) {
	result, err := controlplane.NewClient(h.client).List(r.Context())
	if err != nil {
		errors.HandleError(w, r, err)
		return
	}

	h.setUncacheable(w)
	util.WriteJSONResponse(w, r, http.StatusOK, result)
}

func (h *Handler) PostApiV1Controlplanes(w http.ResponseWriter, r *http.Request) {
	request := &generated.ControlPlane{}

	if err := util.ReadJSONBody(r, request); err != nil {
		errors.HandleError(w, r, err)
		return
	}

	if err := controlplane.NewClient(h.client).Create(r.Context(), request); err != nil {
		errors.HandleError(w, r, err)
		return
	}

	h.setUncacheable(w)
	w.WriteHeader(http.StatusAccepted)
}

func (h *Handler) DeleteApiV1ControlplanesControlPlaneName(w http.ResponseWriter, r *http.Request, controlPlaneName generated.ControlPlaneNameParameter) {
	if err := controlplane.NewClient(h.client).Delete(r.Context(), controlPlaneName); err != nil {
		errors.HandleError(w, r, err)
		return
	}

	h.setUncacheable(w)
	w.WriteHeader(http.StatusAccepted)
}

func (h *Handler) GetApiV1ControlplanesControlPlaneName(w http.ResponseWriter, r *http.Request, controlPlaneName generated.ControlPlaneNameParameter) {
	result, err := controlplane.NewClient(h.client).Get(r.Context(), controlPlaneName)
	if err != nil {
		errors.HandleError(w, r, err)
		return
	}

	h.setUncacheable(w)
	util.WriteJSONResponse(w, r, http.StatusOK, result)
}

func (h *Handler) PutApiV1ControlplanesControlPlaneName(w http.ResponseWriter, r *http.Request, controlPlaneName generated.ControlPlaneNameParameter) {
	request := &generated.ControlPlane{}

	if err := util.ReadJSONBody(r, request); err != nil {
		errors.HandleError(w, r, err)
		return
	}

	if err := controlplane.NewClient(h.client).Update(r.Context(), controlPlaneName, request); err != nil {
		errors.HandleError(w, r, err)
		return
	}

	h.setUncacheable(w)
	w.WriteHeader(http.StatusAccepted)
}

func (h *Handler) GetApiV1ControlplanesControlPlaneNameClusters(w http.ResponseWriter, r *http.Request, controlPlaneName generated.ControlPlaneNameParameter) {
	provider, err := region.NewClient(h.client).Provider(r.Context(), "uk-manchester")
	if err != nil {
		errors.HandleError(w, r, err)
		return
	}

	result, err := cluster.NewClient(h.client, r, provider).List(r.Context(), controlPlaneName)
	if err != nil {
		errors.HandleError(w, r, err)
		return
	}

	h.setUncacheable(w)
	util.WriteJSONResponse(w, r, http.StatusOK, result)
}

func (h *Handler) PostApiV1ControlplanesControlPlaneNameClusters(w http.ResponseWriter, r *http.Request, controlPlaneName generated.ControlPlaneNameParameter) {
	request := &generated.KubernetesCluster{}

	if err := util.ReadJSONBody(r, request); err != nil {
		errors.HandleError(w, r, err)
		return
	}

	provider, err := region.NewClient(h.client).Provider(r.Context(), "uk-manchester")
	if err != nil {
		errors.HandleError(w, r, err)
		return
	}

	if err := cluster.NewClient(h.client, r, provider).Create(r.Context(), controlPlaneName, request); err != nil {
		errors.HandleError(w, r, err)
		return
	}

	h.setUncacheable(w)
	w.WriteHeader(http.StatusAccepted)
}

func (h *Handler) DeleteApiV1ControlplanesControlPlaneNameClustersClusterName(w http.ResponseWriter, r *http.Request, controlPlaneName generated.ControlPlaneNameParameter, clusterName generated.ClusterNameParameter) {
	provider, err := region.NewClient(h.client).Provider(r.Context(), "uk-manchester")
	if err != nil {
		errors.HandleError(w, r, err)
		return
	}

	if err := cluster.NewClient(h.client, r, provider).Delete(r.Context(), controlPlaneName, clusterName); err != nil {
		errors.HandleError(w, r, err)
		return
	}

	h.setUncacheable(w)
	w.WriteHeader(http.StatusAccepted)
}

func (h *Handler) GetApiV1ControlplanesControlPlaneNameClustersClusterName(w http.ResponseWriter, r *http.Request, controlPlaneName generated.ControlPlaneNameParameter, clusterName generated.ClusterNameParameter) {
	provider, err := region.NewClient(h.client).Provider(r.Context(), "uk-manchester")
	if err != nil {
		errors.HandleError(w, r, err)
		return
	}

	result, err := cluster.NewClient(h.client, r, provider).Get(r.Context(), controlPlaneName, clusterName)
	if err != nil {
		errors.HandleError(w, r, err)
		return
	}

	h.setUncacheable(w)
	util.WriteJSONResponse(w, r, http.StatusOK, result)
}

func (h *Handler) PutApiV1ControlplanesControlPlaneNameClustersClusterName(w http.ResponseWriter, r *http.Request, controlPlaneName generated.ControlPlaneNameParameter, clusterName generated.ClusterNameParameter) {
	request := &generated.KubernetesCluster{}

	if err := util.ReadJSONBody(r, request); err != nil {
		errors.HandleError(w, r, err)
		return
	}

	provider, err := region.NewClient(h.client).Provider(r.Context(), "uk-manchester")
	if err != nil {
		errors.HandleError(w, r, err)
		return
	}

	if err := cluster.NewClient(h.client, r, provider).Update(r.Context(), controlPlaneName, clusterName, request); err != nil {
		errors.HandleError(w, r, err)
		return
	}

	h.setUncacheable(w)
	w.WriteHeader(http.StatusAccepted)
}

func (h *Handler) GetApiV1ControlplanesControlPlaneNameClustersClusterNameKubeconfig(w http.ResponseWriter, r *http.Request, controlPlaneName generated.ControlPlaneNameParameter, clusterName generated.ClusterNameParameter) {
	provider, err := region.NewClient(h.client).Provider(r.Context(), "uk-manchester")
	if err != nil {
		errors.HandleError(w, r, err)
		return
	}

	result, err := cluster.NewClient(h.client, r, provider).GetKubeconfig(r.Context(), controlPlaneName, clusterName)
	if err != nil {
		errors.HandleError(w, r, err)
		return
	}

	h.setUncacheable(w)
	util.WriteOctetStreamResponse(w, r, http.StatusOK, result)
}

func (h *Handler) GetApiV1ApplicationbundlesControlPlane(w http.ResponseWriter, r *http.Request) {
	result, err := applicationbundle.NewClient(h.client).ListControlPlane(r.Context())
	if err != nil {
		errors.HandleError(w, r, err)
		return
	}

	h.setUncacheable(w)
	util.WriteJSONResponse(w, r, http.StatusOK, result)
}

func (h *Handler) GetApiV1ApplicationbundlesCluster(w http.ResponseWriter, r *http.Request) {
	result, err := applicationbundle.NewClient(h.client).ListCluster(r.Context())
	if err != nil {
		errors.HandleError(w, r, err)
		return
	}

	h.setUncacheable(w)
	util.WriteJSONResponse(w, r, http.StatusOK, result)
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

func (h *Handler) GetApiV1RegionsRegionNameAvailabilityZonesCompute(w http.ResponseWriter, r *http.Request, regionName generated.RegionNameParameter) {
	provider, err := region.NewClient(h.client).Provider(r.Context(), regionName)
	if err != nil {
		errors.HandleError(w, r, err)
		return
	}

	result, err := provider.ListAvailabilityZonesCompute(r)
	if err != nil {
		errors.HandleError(w, r, err)
		return
	}

	h.setCacheable(w)
	util.WriteJSONResponse(w, r, http.StatusOK, result)
}

func (h *Handler) GetApiV1RegionsRegionNameAvailabilityZonesBlockStorage(w http.ResponseWriter, r *http.Request, regionName generated.RegionNameParameter) {
	provider, err := region.NewClient(h.client).Provider(r.Context(), regionName)
	if err != nil {
		errors.HandleError(w, r, err)
		return
	}

	result, err := provider.ListAvailabilityZonesBlockStorage(r)
	if err != nil {
		errors.HandleError(w, r, err)
		return
	}

	h.setCacheable(w)
	util.WriteJSONResponse(w, r, http.StatusOK, result)
}

func (h *Handler) GetApiV1RegionsRegionNameExternalNetworks(w http.ResponseWriter, r *http.Request, regionName generated.RegionNameParameter) {
	provider, err := region.NewClient(h.client).Provider(r.Context(), regionName)
	if err != nil {
		errors.HandleError(w, r, err)
		return
	}

	result, err := provider.ListExternalNetworks(r)
	if err != nil {
		errors.HandleError(w, r, err)
		return
	}

	h.setCacheable(w)
	util.WriteJSONResponse(w, r, http.StatusOK, result)
}

func (h *Handler) GetApiV1RegionsRegionNameFlavors(w http.ResponseWriter, r *http.Request, regionName generated.RegionNameParameter) {
	provider, err := region.NewClient(h.client).Provider(r.Context(), regionName)
	if err != nil {
		errors.HandleError(w, r, err)
		return
	}

	result, err := provider.ListFlavors(r)
	if err != nil {
		errors.HandleError(w, r, err)
		return
	}

	h.setCacheable(w)
	util.WriteJSONResponse(w, r, http.StatusOK, result)
}

func (h *Handler) GetApiV1RegionsRegionNameImages(w http.ResponseWriter, r *http.Request, regionName generated.RegionNameParameter) {
	provider, err := region.NewClient(h.client).Provider(r.Context(), regionName)
	if err != nil {
		errors.HandleError(w, r, err)
		return
	}

	result, err := provider.ListImages(r)
	if err != nil {
		errors.HandleError(w, r, err)
		return
	}

	h.setCacheable(w)
	util.WriteJSONResponse(w, r, http.StatusOK, result)
}

func (h *Handler) GetApiV1RegionsRegionNameKeyPairs(w http.ResponseWriter, r *http.Request, regionName generated.RegionNameParameter) {
	provider, err := region.NewClient(h.client).Provider(r.Context(), regionName)
	if err != nil {
		errors.HandleError(w, r, err)
		return
	}

	result, err := provider.ListKeyPairs(r)
	if err != nil {
		errors.HandleError(w, r, err)
		return
	}

	h.setUncacheable(w)
	util.WriteJSONResponse(w, r, http.StatusOK, result)
}
