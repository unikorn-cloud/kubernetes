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
	"sort"
	"time"

	"github.com/unikorn-cloud/core/pkg/server/errors"
	coreutil "github.com/unikorn-cloud/core/pkg/util"
	"github.com/unikorn-cloud/unikorn/pkg/server/generated"
	"github.com/unikorn-cloud/unikorn/pkg/server/handler/application"
	"github.com/unikorn-cloud/unikorn/pkg/server/handler/cluster"
	"github.com/unikorn-cloud/unikorn/pkg/server/handler/controlplane"
	"github.com/unikorn-cloud/unikorn/pkg/server/handler/organization"
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

func (h *Handler) PostApiV1Organization(w http.ResponseWriter, r *http.Request) {
	if err := organization.NewClient(h.client).Create(r.Context()); err != nil {
		errors.HandleError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusAccepted)
}

func (h *Handler) DeleteApiV1Organization(w http.ResponseWriter, r *http.Request) {
	if err := organization.NewClient(h.client).Delete(r.Context()); err != nil {
		errors.HandleError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusAccepted)
}

func (h *Handler) GetApiV1Projects(w http.ResponseWriter, r *http.Request) {
	result, err := project.NewClient(h.client).List(r.Context())
	if err != nil {
		errors.HandleError(w, r, err)
		return
	}

	h.setUncacheable(w)
	util.WriteJSONResponse(w, r, http.StatusOK, result)
}

func (h *Handler) PostApiV1Projects(w http.ResponseWriter, r *http.Request) {
	request := &generated.Project{}

	if err := util.ReadJSONBody(r, request); err != nil {
		errors.HandleError(w, r, err)
		return
	}

	if err := project.NewClient(h.client).Create(r.Context(), request); err != nil {
		errors.HandleError(w, r, err)
		return
	}

	h.setUncacheable(w)
	w.WriteHeader(http.StatusAccepted)
}

func (h *Handler) DeleteApiV1ProjectsProjectName(w http.ResponseWriter, r *http.Request, projectName generated.ProjectNameParameter) {
	if err := project.NewClient(h.client).Delete(r.Context(), projectName); err != nil {
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

func (h *Handler) PostApiV1ProjectsProjectNameControlplanes(w http.ResponseWriter, r *http.Request, projectName generated.ProjectNameParameter) {
	request := &generated.ControlPlane{}

	if err := util.ReadJSONBody(r, request); err != nil {
		errors.HandleError(w, r, err)
		return
	}

	if err := controlplane.NewClient(h.client).Create(r.Context(), projectName, request); err != nil {
		errors.HandleError(w, r, err)
		return
	}

	h.setUncacheable(w)
	w.WriteHeader(http.StatusAccepted)
}

func (h *Handler) DeleteApiV1ProjectsProjectNameControlplanesControlPlaneName(w http.ResponseWriter, r *http.Request, projectName generated.ProjectNameParameter, controlPlaneName generated.ControlPlaneNameParameter) {
	if err := controlplane.NewClient(h.client).Delete(r.Context(), projectName, controlPlaneName); err != nil {
		errors.HandleError(w, r, err)
		return
	}

	h.setUncacheable(w)
	w.WriteHeader(http.StatusAccepted)
}

func (h *Handler) PutApiV1ProjectsProjectNameControlplanesControlPlaneName(w http.ResponseWriter, r *http.Request, projectName generated.ProjectNameParameter, controlPlaneName generated.ControlPlaneNameParameter) {
	request := &generated.ControlPlane{}

	if err := util.ReadJSONBody(r, request); err != nil {
		errors.HandleError(w, r, err)
		return
	}

	if err := controlplane.NewClient(h.client).Update(r.Context(), projectName, controlPlaneName, request); err != nil {
		errors.HandleError(w, r, err)
		return
	}

	h.setUncacheable(w)
	w.WriteHeader(http.StatusAccepted)
}

func (h *Handler) GetApiV1Clusters(w http.ResponseWriter, r *http.Request) {
	result, err := cluster.NewClient(h.client, &h.options.Cluster).List(r.Context())
	if err != nil {
		errors.HandleError(w, r, err)
		return
	}

	h.setUncacheable(w)
	util.WriteJSONResponse(w, r, http.StatusOK, result)
}

func (h *Handler) PostApiV1ProjectsProjectNameControlplanesControlPlaneNameClusters(w http.ResponseWriter, r *http.Request, projectName generated.ProjectNameParameter, controlPlaneName generated.ControlPlaneNameParameter) {
	request := &generated.KubernetesCluster{}

	if err := util.ReadJSONBody(r, request); err != nil {
		errors.HandleError(w, r, err)
		return
	}

	if err := cluster.NewClient(h.client, &h.options.Cluster).Create(r.Context(), projectName, controlPlaneName, request); err != nil {
		errors.HandleError(w, r, err)
		return
	}

	h.setUncacheable(w)
	w.WriteHeader(http.StatusAccepted)
}

func (h *Handler) DeleteApiV1ProjectsProjectNameControlplanesControlPlaneNameClustersClusterName(w http.ResponseWriter, r *http.Request, projectName generated.ProjectNameParameter, controlPlaneName generated.ControlPlaneNameParameter, clusterName generated.ClusterNameParameter) {
	if err := cluster.NewClient(h.client, &h.options.Cluster).Delete(r.Context(), projectName, controlPlaneName, clusterName); err != nil {
		errors.HandleError(w, r, err)
		return
	}

	h.setUncacheable(w)
	w.WriteHeader(http.StatusAccepted)
}

func (h *Handler) PutApiV1ProjectsProjectNameControlplanesControlPlaneNameClustersClusterName(w http.ResponseWriter, r *http.Request, projectName generated.ProjectNameParameter, controlPlaneName generated.ControlPlaneNameParameter, clusterName generated.ClusterNameParameter) {
	request := &generated.KubernetesCluster{}

	if err := util.ReadJSONBody(r, request); err != nil {
		errors.HandleError(w, r, err)
		return
	}

	if err := cluster.NewClient(h.client, &h.options.Cluster).Update(r.Context(), projectName, controlPlaneName, clusterName, request); err != nil {
		errors.HandleError(w, r, err)
		return
	}

	h.setUncacheable(w)
	w.WriteHeader(http.StatusAccepted)
}

func (h *Handler) GetApiV1ProjectsProjectNameControlplanesControlPlaneNameClustersClusterNameKubeconfig(w http.ResponseWriter, r *http.Request, projectName generated.ProjectNameParameter, controlPlaneName generated.ControlPlaneNameParameter, clusterName generated.ClusterNameParameter) {
	result, err := cluster.NewClient(h.client, &h.options.Cluster).GetKubeconfig(r.Context(), projectName, controlPlaneName, clusterName)
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

	out := make(generated.OpenstackFlavors, 0, len(result))

	for _, r := range result {
		t := generated.OpenstackFlavor{
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

	out := make(generated.OpenstackImages, 0, len(result))

	for _, r := range result {
		out = append(out, generated.OpenstackImage{
			Name:     r.Name,
			Created:  r.Created,
			Modified: r.Modified,
			Versions: generated.OpenstackImageVersions{
				Kubernetes: r.KubernetesVersion,
			},
		})
	}

	h.setCacheable(w)
	util.WriteJSONResponse(w, r, http.StatusOK, out)
}
