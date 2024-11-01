// Package openapi provides primitives to interact with the openapi HTTP API.
//
// Code generated by github.com/oapi-codegen/oapi-codegen/v2 version v2.4.1 DO NOT EDIT.
package openapi

import (
	"context"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/oapi-codegen/runtime"
)

// ServerInterface represents all server handlers.
type ServerInterface interface {

	// (GET /api/v1/organizations/{organizationID}/clustermanagers)
	GetApiV1OrganizationsOrganizationIDClustermanagers(w http.ResponseWriter, r *http.Request, organizationID OrganizationIDParameter)

	// (GET /api/v1/organizations/{organizationID}/clusters)
	GetApiV1OrganizationsOrganizationIDClusters(w http.ResponseWriter, r *http.Request, organizationID OrganizationIDParameter)

	// (POST /api/v1/organizations/{organizationID}/projects/{projectID}/clustermanagers)
	PostApiV1OrganizationsOrganizationIDProjectsProjectIDClustermanagers(w http.ResponseWriter, r *http.Request, organizationID OrganizationIDParameter, projectID ProjectIDParameter)

	// (DELETE /api/v1/organizations/{organizationID}/projects/{projectID}/clustermanagers/{clusterManagerID})
	DeleteApiV1OrganizationsOrganizationIDProjectsProjectIDClustermanagersClusterManagerID(w http.ResponseWriter, r *http.Request, organizationID OrganizationIDParameter, projectID ProjectIDParameter, clusterManagerID ClusterManagerIDParameter)

	// (PUT /api/v1/organizations/{organizationID}/projects/{projectID}/clustermanagers/{clusterManagerID})
	PutApiV1OrganizationsOrganizationIDProjectsProjectIDClustermanagersClusterManagerID(w http.ResponseWriter, r *http.Request, organizationID OrganizationIDParameter, projectID ProjectIDParameter, clusterManagerID ClusterManagerIDParameter)

	// (POST /api/v1/organizations/{organizationID}/projects/{projectID}/clusters)
	PostApiV1OrganizationsOrganizationIDProjectsProjectIDClusters(w http.ResponseWriter, r *http.Request, organizationID OrganizationIDParameter, projectID ProjectIDParameter)

	// (DELETE /api/v1/organizations/{organizationID}/projects/{projectID}/clusters/{clusterID})
	DeleteApiV1OrganizationsOrganizationIDProjectsProjectIDClustersClusterID(w http.ResponseWriter, r *http.Request, organizationID OrganizationIDParameter, projectID ProjectIDParameter, clusterID ClusterIDParameter)

	// (PUT /api/v1/organizations/{organizationID}/projects/{projectID}/clusters/{clusterID})
	PutApiV1OrganizationsOrganizationIDProjectsProjectIDClustersClusterID(w http.ResponseWriter, r *http.Request, organizationID OrganizationIDParameter, projectID ProjectIDParameter, clusterID ClusterIDParameter)

	// (GET /api/v1/organizations/{organizationID}/projects/{projectID}/clusters/{clusterID}/kubeconfig)
	GetApiV1OrganizationsOrganizationIDProjectsProjectIDClustersClusterIDKubeconfig(w http.ResponseWriter, r *http.Request, organizationID OrganizationIDParameter, projectID ProjectIDParameter, clusterID ClusterIDParameter)

	// (GET /api/v1/organizations/{organizationID}/regions/{regionID}/flavors)
	GetApiV1OrganizationsOrganizationIDRegionsRegionIDFlavors(w http.ResponseWriter, r *http.Request, organizationID OrganizationIDParameter, regionID RegionIDParameter)

	// (GET /api/v1/organizations/{organizationID}/regions/{regionID}/images)
	GetApiV1OrganizationsOrganizationIDRegionsRegionIDImages(w http.ResponseWriter, r *http.Request, organizationID OrganizationIDParameter, regionID RegionIDParameter)
}

// Unimplemented server implementation that returns http.StatusNotImplemented for each endpoint.

type Unimplemented struct{}

// (GET /api/v1/organizations/{organizationID}/clustermanagers)
func (_ Unimplemented) GetApiV1OrganizationsOrganizationIDClustermanagers(w http.ResponseWriter, r *http.Request, organizationID OrganizationIDParameter) {
	w.WriteHeader(http.StatusNotImplemented)
}

// (GET /api/v1/organizations/{organizationID}/clusters)
func (_ Unimplemented) GetApiV1OrganizationsOrganizationIDClusters(w http.ResponseWriter, r *http.Request, organizationID OrganizationIDParameter) {
	w.WriteHeader(http.StatusNotImplemented)
}

// (POST /api/v1/organizations/{organizationID}/projects/{projectID}/clustermanagers)
func (_ Unimplemented) PostApiV1OrganizationsOrganizationIDProjectsProjectIDClustermanagers(w http.ResponseWriter, r *http.Request, organizationID OrganizationIDParameter, projectID ProjectIDParameter) {
	w.WriteHeader(http.StatusNotImplemented)
}

// (DELETE /api/v1/organizations/{organizationID}/projects/{projectID}/clustermanagers/{clusterManagerID})
func (_ Unimplemented) DeleteApiV1OrganizationsOrganizationIDProjectsProjectIDClustermanagersClusterManagerID(w http.ResponseWriter, r *http.Request, organizationID OrganizationIDParameter, projectID ProjectIDParameter, clusterManagerID ClusterManagerIDParameter) {
	w.WriteHeader(http.StatusNotImplemented)
}

// (PUT /api/v1/organizations/{organizationID}/projects/{projectID}/clustermanagers/{clusterManagerID})
func (_ Unimplemented) PutApiV1OrganizationsOrganizationIDProjectsProjectIDClustermanagersClusterManagerID(w http.ResponseWriter, r *http.Request, organizationID OrganizationIDParameter, projectID ProjectIDParameter, clusterManagerID ClusterManagerIDParameter) {
	w.WriteHeader(http.StatusNotImplemented)
}

// (POST /api/v1/organizations/{organizationID}/projects/{projectID}/clusters)
func (_ Unimplemented) PostApiV1OrganizationsOrganizationIDProjectsProjectIDClusters(w http.ResponseWriter, r *http.Request, organizationID OrganizationIDParameter, projectID ProjectIDParameter) {
	w.WriteHeader(http.StatusNotImplemented)
}

// (DELETE /api/v1/organizations/{organizationID}/projects/{projectID}/clusters/{clusterID})
func (_ Unimplemented) DeleteApiV1OrganizationsOrganizationIDProjectsProjectIDClustersClusterID(w http.ResponseWriter, r *http.Request, organizationID OrganizationIDParameter, projectID ProjectIDParameter, clusterID ClusterIDParameter) {
	w.WriteHeader(http.StatusNotImplemented)
}

// (PUT /api/v1/organizations/{organizationID}/projects/{projectID}/clusters/{clusterID})
func (_ Unimplemented) PutApiV1OrganizationsOrganizationIDProjectsProjectIDClustersClusterID(w http.ResponseWriter, r *http.Request, organizationID OrganizationIDParameter, projectID ProjectIDParameter, clusterID ClusterIDParameter) {
	w.WriteHeader(http.StatusNotImplemented)
}

// (GET /api/v1/organizations/{organizationID}/projects/{projectID}/clusters/{clusterID}/kubeconfig)
func (_ Unimplemented) GetApiV1OrganizationsOrganizationIDProjectsProjectIDClustersClusterIDKubeconfig(w http.ResponseWriter, r *http.Request, organizationID OrganizationIDParameter, projectID ProjectIDParameter, clusterID ClusterIDParameter) {
	w.WriteHeader(http.StatusNotImplemented)
}

// (GET /api/v1/organizations/{organizationID}/regions/{regionID}/flavors)
func (_ Unimplemented) GetApiV1OrganizationsOrganizationIDRegionsRegionIDFlavors(w http.ResponseWriter, r *http.Request, organizationID OrganizationIDParameter, regionID RegionIDParameter) {
	w.WriteHeader(http.StatusNotImplemented)
}

// (GET /api/v1/organizations/{organizationID}/regions/{regionID}/images)
func (_ Unimplemented) GetApiV1OrganizationsOrganizationIDRegionsRegionIDImages(w http.ResponseWriter, r *http.Request, organizationID OrganizationIDParameter, regionID RegionIDParameter) {
	w.WriteHeader(http.StatusNotImplemented)
}

// ServerInterfaceWrapper converts contexts to parameters.
type ServerInterfaceWrapper struct {
	Handler            ServerInterface
	HandlerMiddlewares []MiddlewareFunc
	ErrorHandlerFunc   func(w http.ResponseWriter, r *http.Request, err error)
}

type MiddlewareFunc func(http.Handler) http.Handler

// GetApiV1OrganizationsOrganizationIDClustermanagers operation middleware
func (siw *ServerInterfaceWrapper) GetApiV1OrganizationsOrganizationIDClustermanagers(w http.ResponseWriter, r *http.Request) {

	var err error

	// ------------- Path parameter "organizationID" -------------
	var organizationID OrganizationIDParameter

	err = runtime.BindStyledParameterWithOptions("simple", "organizationID", chi.URLParam(r, "organizationID"), &organizationID, runtime.BindStyledParameterOptions{ParamLocation: runtime.ParamLocationPath, Explode: false, Required: true})
	if err != nil {
		siw.ErrorHandlerFunc(w, r, &InvalidParamFormatError{ParamName: "organizationID", Err: err})
		return
	}

	ctx := r.Context()

	ctx = context.WithValue(ctx, Oauth2AuthenticationScopes, []string{})

	r = r.WithContext(ctx)

	handler := http.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		siw.Handler.GetApiV1OrganizationsOrganizationIDClustermanagers(w, r, organizationID)
	}))

	for _, middleware := range siw.HandlerMiddlewares {
		handler = middleware(handler)
	}

	handler.ServeHTTP(w, r)
}

// GetApiV1OrganizationsOrganizationIDClusters operation middleware
func (siw *ServerInterfaceWrapper) GetApiV1OrganizationsOrganizationIDClusters(w http.ResponseWriter, r *http.Request) {

	var err error

	// ------------- Path parameter "organizationID" -------------
	var organizationID OrganizationIDParameter

	err = runtime.BindStyledParameterWithOptions("simple", "organizationID", chi.URLParam(r, "organizationID"), &organizationID, runtime.BindStyledParameterOptions{ParamLocation: runtime.ParamLocationPath, Explode: false, Required: true})
	if err != nil {
		siw.ErrorHandlerFunc(w, r, &InvalidParamFormatError{ParamName: "organizationID", Err: err})
		return
	}

	ctx := r.Context()

	ctx = context.WithValue(ctx, Oauth2AuthenticationScopes, []string{})

	r = r.WithContext(ctx)

	handler := http.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		siw.Handler.GetApiV1OrganizationsOrganizationIDClusters(w, r, organizationID)
	}))

	for _, middleware := range siw.HandlerMiddlewares {
		handler = middleware(handler)
	}

	handler.ServeHTTP(w, r)
}

// PostApiV1OrganizationsOrganizationIDProjectsProjectIDClustermanagers operation middleware
func (siw *ServerInterfaceWrapper) PostApiV1OrganizationsOrganizationIDProjectsProjectIDClustermanagers(w http.ResponseWriter, r *http.Request) {

	var err error

	// ------------- Path parameter "organizationID" -------------
	var organizationID OrganizationIDParameter

	err = runtime.BindStyledParameterWithOptions("simple", "organizationID", chi.URLParam(r, "organizationID"), &organizationID, runtime.BindStyledParameterOptions{ParamLocation: runtime.ParamLocationPath, Explode: false, Required: true})
	if err != nil {
		siw.ErrorHandlerFunc(w, r, &InvalidParamFormatError{ParamName: "organizationID", Err: err})
		return
	}

	// ------------- Path parameter "projectID" -------------
	var projectID ProjectIDParameter

	err = runtime.BindStyledParameterWithOptions("simple", "projectID", chi.URLParam(r, "projectID"), &projectID, runtime.BindStyledParameterOptions{ParamLocation: runtime.ParamLocationPath, Explode: false, Required: true})
	if err != nil {
		siw.ErrorHandlerFunc(w, r, &InvalidParamFormatError{ParamName: "projectID", Err: err})
		return
	}

	ctx := r.Context()

	ctx = context.WithValue(ctx, Oauth2AuthenticationScopes, []string{})

	r = r.WithContext(ctx)

	handler := http.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		siw.Handler.PostApiV1OrganizationsOrganizationIDProjectsProjectIDClustermanagers(w, r, organizationID, projectID)
	}))

	for _, middleware := range siw.HandlerMiddlewares {
		handler = middleware(handler)
	}

	handler.ServeHTTP(w, r)
}

// DeleteApiV1OrganizationsOrganizationIDProjectsProjectIDClustermanagersClusterManagerID operation middleware
func (siw *ServerInterfaceWrapper) DeleteApiV1OrganizationsOrganizationIDProjectsProjectIDClustermanagersClusterManagerID(w http.ResponseWriter, r *http.Request) {

	var err error

	// ------------- Path parameter "organizationID" -------------
	var organizationID OrganizationIDParameter

	err = runtime.BindStyledParameterWithOptions("simple", "organizationID", chi.URLParam(r, "organizationID"), &organizationID, runtime.BindStyledParameterOptions{ParamLocation: runtime.ParamLocationPath, Explode: false, Required: true})
	if err != nil {
		siw.ErrorHandlerFunc(w, r, &InvalidParamFormatError{ParamName: "organizationID", Err: err})
		return
	}

	// ------------- Path parameter "projectID" -------------
	var projectID ProjectIDParameter

	err = runtime.BindStyledParameterWithOptions("simple", "projectID", chi.URLParam(r, "projectID"), &projectID, runtime.BindStyledParameterOptions{ParamLocation: runtime.ParamLocationPath, Explode: false, Required: true})
	if err != nil {
		siw.ErrorHandlerFunc(w, r, &InvalidParamFormatError{ParamName: "projectID", Err: err})
		return
	}

	// ------------- Path parameter "clusterManagerID" -------------
	var clusterManagerID ClusterManagerIDParameter

	err = runtime.BindStyledParameterWithOptions("simple", "clusterManagerID", chi.URLParam(r, "clusterManagerID"), &clusterManagerID, runtime.BindStyledParameterOptions{ParamLocation: runtime.ParamLocationPath, Explode: false, Required: true})
	if err != nil {
		siw.ErrorHandlerFunc(w, r, &InvalidParamFormatError{ParamName: "clusterManagerID", Err: err})
		return
	}

	ctx := r.Context()

	ctx = context.WithValue(ctx, Oauth2AuthenticationScopes, []string{})

	r = r.WithContext(ctx)

	handler := http.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		siw.Handler.DeleteApiV1OrganizationsOrganizationIDProjectsProjectIDClustermanagersClusterManagerID(w, r, organizationID, projectID, clusterManagerID)
	}))

	for _, middleware := range siw.HandlerMiddlewares {
		handler = middleware(handler)
	}

	handler.ServeHTTP(w, r)
}

// PutApiV1OrganizationsOrganizationIDProjectsProjectIDClustermanagersClusterManagerID operation middleware
func (siw *ServerInterfaceWrapper) PutApiV1OrganizationsOrganizationIDProjectsProjectIDClustermanagersClusterManagerID(w http.ResponseWriter, r *http.Request) {

	var err error

	// ------------- Path parameter "organizationID" -------------
	var organizationID OrganizationIDParameter

	err = runtime.BindStyledParameterWithOptions("simple", "organizationID", chi.URLParam(r, "organizationID"), &organizationID, runtime.BindStyledParameterOptions{ParamLocation: runtime.ParamLocationPath, Explode: false, Required: true})
	if err != nil {
		siw.ErrorHandlerFunc(w, r, &InvalidParamFormatError{ParamName: "organizationID", Err: err})
		return
	}

	// ------------- Path parameter "projectID" -------------
	var projectID ProjectIDParameter

	err = runtime.BindStyledParameterWithOptions("simple", "projectID", chi.URLParam(r, "projectID"), &projectID, runtime.BindStyledParameterOptions{ParamLocation: runtime.ParamLocationPath, Explode: false, Required: true})
	if err != nil {
		siw.ErrorHandlerFunc(w, r, &InvalidParamFormatError{ParamName: "projectID", Err: err})
		return
	}

	// ------------- Path parameter "clusterManagerID" -------------
	var clusterManagerID ClusterManagerIDParameter

	err = runtime.BindStyledParameterWithOptions("simple", "clusterManagerID", chi.URLParam(r, "clusterManagerID"), &clusterManagerID, runtime.BindStyledParameterOptions{ParamLocation: runtime.ParamLocationPath, Explode: false, Required: true})
	if err != nil {
		siw.ErrorHandlerFunc(w, r, &InvalidParamFormatError{ParamName: "clusterManagerID", Err: err})
		return
	}

	ctx := r.Context()

	ctx = context.WithValue(ctx, Oauth2AuthenticationScopes, []string{})

	r = r.WithContext(ctx)

	handler := http.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		siw.Handler.PutApiV1OrganizationsOrganizationIDProjectsProjectIDClustermanagersClusterManagerID(w, r, organizationID, projectID, clusterManagerID)
	}))

	for _, middleware := range siw.HandlerMiddlewares {
		handler = middleware(handler)
	}

	handler.ServeHTTP(w, r)
}

// PostApiV1OrganizationsOrganizationIDProjectsProjectIDClusters operation middleware
func (siw *ServerInterfaceWrapper) PostApiV1OrganizationsOrganizationIDProjectsProjectIDClusters(w http.ResponseWriter, r *http.Request) {

	var err error

	// ------------- Path parameter "organizationID" -------------
	var organizationID OrganizationIDParameter

	err = runtime.BindStyledParameterWithOptions("simple", "organizationID", chi.URLParam(r, "organizationID"), &organizationID, runtime.BindStyledParameterOptions{ParamLocation: runtime.ParamLocationPath, Explode: false, Required: true})
	if err != nil {
		siw.ErrorHandlerFunc(w, r, &InvalidParamFormatError{ParamName: "organizationID", Err: err})
		return
	}

	// ------------- Path parameter "projectID" -------------
	var projectID ProjectIDParameter

	err = runtime.BindStyledParameterWithOptions("simple", "projectID", chi.URLParam(r, "projectID"), &projectID, runtime.BindStyledParameterOptions{ParamLocation: runtime.ParamLocationPath, Explode: false, Required: true})
	if err != nil {
		siw.ErrorHandlerFunc(w, r, &InvalidParamFormatError{ParamName: "projectID", Err: err})
		return
	}

	ctx := r.Context()

	ctx = context.WithValue(ctx, Oauth2AuthenticationScopes, []string{})

	r = r.WithContext(ctx)

	handler := http.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		siw.Handler.PostApiV1OrganizationsOrganizationIDProjectsProjectIDClusters(w, r, organizationID, projectID)
	}))

	for _, middleware := range siw.HandlerMiddlewares {
		handler = middleware(handler)
	}

	handler.ServeHTTP(w, r)
}

// DeleteApiV1OrganizationsOrganizationIDProjectsProjectIDClustersClusterID operation middleware
func (siw *ServerInterfaceWrapper) DeleteApiV1OrganizationsOrganizationIDProjectsProjectIDClustersClusterID(w http.ResponseWriter, r *http.Request) {

	var err error

	// ------------- Path parameter "organizationID" -------------
	var organizationID OrganizationIDParameter

	err = runtime.BindStyledParameterWithOptions("simple", "organizationID", chi.URLParam(r, "organizationID"), &organizationID, runtime.BindStyledParameterOptions{ParamLocation: runtime.ParamLocationPath, Explode: false, Required: true})
	if err != nil {
		siw.ErrorHandlerFunc(w, r, &InvalidParamFormatError{ParamName: "organizationID", Err: err})
		return
	}

	// ------------- Path parameter "projectID" -------------
	var projectID ProjectIDParameter

	err = runtime.BindStyledParameterWithOptions("simple", "projectID", chi.URLParam(r, "projectID"), &projectID, runtime.BindStyledParameterOptions{ParamLocation: runtime.ParamLocationPath, Explode: false, Required: true})
	if err != nil {
		siw.ErrorHandlerFunc(w, r, &InvalidParamFormatError{ParamName: "projectID", Err: err})
		return
	}

	// ------------- Path parameter "clusterID" -------------
	var clusterID ClusterIDParameter

	err = runtime.BindStyledParameterWithOptions("simple", "clusterID", chi.URLParam(r, "clusterID"), &clusterID, runtime.BindStyledParameterOptions{ParamLocation: runtime.ParamLocationPath, Explode: false, Required: true})
	if err != nil {
		siw.ErrorHandlerFunc(w, r, &InvalidParamFormatError{ParamName: "clusterID", Err: err})
		return
	}

	ctx := r.Context()

	ctx = context.WithValue(ctx, Oauth2AuthenticationScopes, []string{})

	r = r.WithContext(ctx)

	handler := http.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		siw.Handler.DeleteApiV1OrganizationsOrganizationIDProjectsProjectIDClustersClusterID(w, r, organizationID, projectID, clusterID)
	}))

	for _, middleware := range siw.HandlerMiddlewares {
		handler = middleware(handler)
	}

	handler.ServeHTTP(w, r)
}

// PutApiV1OrganizationsOrganizationIDProjectsProjectIDClustersClusterID operation middleware
func (siw *ServerInterfaceWrapper) PutApiV1OrganizationsOrganizationIDProjectsProjectIDClustersClusterID(w http.ResponseWriter, r *http.Request) {

	var err error

	// ------------- Path parameter "organizationID" -------------
	var organizationID OrganizationIDParameter

	err = runtime.BindStyledParameterWithOptions("simple", "organizationID", chi.URLParam(r, "organizationID"), &organizationID, runtime.BindStyledParameterOptions{ParamLocation: runtime.ParamLocationPath, Explode: false, Required: true})
	if err != nil {
		siw.ErrorHandlerFunc(w, r, &InvalidParamFormatError{ParamName: "organizationID", Err: err})
		return
	}

	// ------------- Path parameter "projectID" -------------
	var projectID ProjectIDParameter

	err = runtime.BindStyledParameterWithOptions("simple", "projectID", chi.URLParam(r, "projectID"), &projectID, runtime.BindStyledParameterOptions{ParamLocation: runtime.ParamLocationPath, Explode: false, Required: true})
	if err != nil {
		siw.ErrorHandlerFunc(w, r, &InvalidParamFormatError{ParamName: "projectID", Err: err})
		return
	}

	// ------------- Path parameter "clusterID" -------------
	var clusterID ClusterIDParameter

	err = runtime.BindStyledParameterWithOptions("simple", "clusterID", chi.URLParam(r, "clusterID"), &clusterID, runtime.BindStyledParameterOptions{ParamLocation: runtime.ParamLocationPath, Explode: false, Required: true})
	if err != nil {
		siw.ErrorHandlerFunc(w, r, &InvalidParamFormatError{ParamName: "clusterID", Err: err})
		return
	}

	ctx := r.Context()

	ctx = context.WithValue(ctx, Oauth2AuthenticationScopes, []string{})

	r = r.WithContext(ctx)

	handler := http.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		siw.Handler.PutApiV1OrganizationsOrganizationIDProjectsProjectIDClustersClusterID(w, r, organizationID, projectID, clusterID)
	}))

	for _, middleware := range siw.HandlerMiddlewares {
		handler = middleware(handler)
	}

	handler.ServeHTTP(w, r)
}

// GetApiV1OrganizationsOrganizationIDProjectsProjectIDClustersClusterIDKubeconfig operation middleware
func (siw *ServerInterfaceWrapper) GetApiV1OrganizationsOrganizationIDProjectsProjectIDClustersClusterIDKubeconfig(w http.ResponseWriter, r *http.Request) {

	var err error

	// ------------- Path parameter "organizationID" -------------
	var organizationID OrganizationIDParameter

	err = runtime.BindStyledParameterWithOptions("simple", "organizationID", chi.URLParam(r, "organizationID"), &organizationID, runtime.BindStyledParameterOptions{ParamLocation: runtime.ParamLocationPath, Explode: false, Required: true})
	if err != nil {
		siw.ErrorHandlerFunc(w, r, &InvalidParamFormatError{ParamName: "organizationID", Err: err})
		return
	}

	// ------------- Path parameter "projectID" -------------
	var projectID ProjectIDParameter

	err = runtime.BindStyledParameterWithOptions("simple", "projectID", chi.URLParam(r, "projectID"), &projectID, runtime.BindStyledParameterOptions{ParamLocation: runtime.ParamLocationPath, Explode: false, Required: true})
	if err != nil {
		siw.ErrorHandlerFunc(w, r, &InvalidParamFormatError{ParamName: "projectID", Err: err})
		return
	}

	// ------------- Path parameter "clusterID" -------------
	var clusterID ClusterIDParameter

	err = runtime.BindStyledParameterWithOptions("simple", "clusterID", chi.URLParam(r, "clusterID"), &clusterID, runtime.BindStyledParameterOptions{ParamLocation: runtime.ParamLocationPath, Explode: false, Required: true})
	if err != nil {
		siw.ErrorHandlerFunc(w, r, &InvalidParamFormatError{ParamName: "clusterID", Err: err})
		return
	}

	ctx := r.Context()

	ctx = context.WithValue(ctx, Oauth2AuthenticationScopes, []string{})

	r = r.WithContext(ctx)

	handler := http.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		siw.Handler.GetApiV1OrganizationsOrganizationIDProjectsProjectIDClustersClusterIDKubeconfig(w, r, organizationID, projectID, clusterID)
	}))

	for _, middleware := range siw.HandlerMiddlewares {
		handler = middleware(handler)
	}

	handler.ServeHTTP(w, r)
}

// GetApiV1OrganizationsOrganizationIDRegionsRegionIDFlavors operation middleware
func (siw *ServerInterfaceWrapper) GetApiV1OrganizationsOrganizationIDRegionsRegionIDFlavors(w http.ResponseWriter, r *http.Request) {

	var err error

	// ------------- Path parameter "organizationID" -------------
	var organizationID OrganizationIDParameter

	err = runtime.BindStyledParameterWithOptions("simple", "organizationID", chi.URLParam(r, "organizationID"), &organizationID, runtime.BindStyledParameterOptions{ParamLocation: runtime.ParamLocationPath, Explode: false, Required: true})
	if err != nil {
		siw.ErrorHandlerFunc(w, r, &InvalidParamFormatError{ParamName: "organizationID", Err: err})
		return
	}

	// ------------- Path parameter "regionID" -------------
	var regionID RegionIDParameter

	err = runtime.BindStyledParameterWithOptions("simple", "regionID", chi.URLParam(r, "regionID"), &regionID, runtime.BindStyledParameterOptions{ParamLocation: runtime.ParamLocationPath, Explode: false, Required: true})
	if err != nil {
		siw.ErrorHandlerFunc(w, r, &InvalidParamFormatError{ParamName: "regionID", Err: err})
		return
	}

	ctx := r.Context()

	ctx = context.WithValue(ctx, Oauth2AuthenticationScopes, []string{})

	r = r.WithContext(ctx)

	handler := http.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		siw.Handler.GetApiV1OrganizationsOrganizationIDRegionsRegionIDFlavors(w, r, organizationID, regionID)
	}))

	for _, middleware := range siw.HandlerMiddlewares {
		handler = middleware(handler)
	}

	handler.ServeHTTP(w, r)
}

// GetApiV1OrganizationsOrganizationIDRegionsRegionIDImages operation middleware
func (siw *ServerInterfaceWrapper) GetApiV1OrganizationsOrganizationIDRegionsRegionIDImages(w http.ResponseWriter, r *http.Request) {

	var err error

	// ------------- Path parameter "organizationID" -------------
	var organizationID OrganizationIDParameter

	err = runtime.BindStyledParameterWithOptions("simple", "organizationID", chi.URLParam(r, "organizationID"), &organizationID, runtime.BindStyledParameterOptions{ParamLocation: runtime.ParamLocationPath, Explode: false, Required: true})
	if err != nil {
		siw.ErrorHandlerFunc(w, r, &InvalidParamFormatError{ParamName: "organizationID", Err: err})
		return
	}

	// ------------- Path parameter "regionID" -------------
	var regionID RegionIDParameter

	err = runtime.BindStyledParameterWithOptions("simple", "regionID", chi.URLParam(r, "regionID"), &regionID, runtime.BindStyledParameterOptions{ParamLocation: runtime.ParamLocationPath, Explode: false, Required: true})
	if err != nil {
		siw.ErrorHandlerFunc(w, r, &InvalidParamFormatError{ParamName: "regionID", Err: err})
		return
	}

	ctx := r.Context()

	ctx = context.WithValue(ctx, Oauth2AuthenticationScopes, []string{})

	r = r.WithContext(ctx)

	handler := http.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		siw.Handler.GetApiV1OrganizationsOrganizationIDRegionsRegionIDImages(w, r, organizationID, regionID)
	}))

	for _, middleware := range siw.HandlerMiddlewares {
		handler = middleware(handler)
	}

	handler.ServeHTTP(w, r)
}

type UnescapedCookieParamError struct {
	ParamName string
	Err       error
}

func (e *UnescapedCookieParamError) Error() string {
	return fmt.Sprintf("error unescaping cookie parameter '%s'", e.ParamName)
}

func (e *UnescapedCookieParamError) Unwrap() error {
	return e.Err
}

type UnmarshalingParamError struct {
	ParamName string
	Err       error
}

func (e *UnmarshalingParamError) Error() string {
	return fmt.Sprintf("Error unmarshaling parameter %s as JSON: %s", e.ParamName, e.Err.Error())
}

func (e *UnmarshalingParamError) Unwrap() error {
	return e.Err
}

type RequiredParamError struct {
	ParamName string
}

func (e *RequiredParamError) Error() string {
	return fmt.Sprintf("Query argument %s is required, but not found", e.ParamName)
}

type RequiredHeaderError struct {
	ParamName string
	Err       error
}

func (e *RequiredHeaderError) Error() string {
	return fmt.Sprintf("Header parameter %s is required, but not found", e.ParamName)
}

func (e *RequiredHeaderError) Unwrap() error {
	return e.Err
}

type InvalidParamFormatError struct {
	ParamName string
	Err       error
}

func (e *InvalidParamFormatError) Error() string {
	return fmt.Sprintf("Invalid format for parameter %s: %s", e.ParamName, e.Err.Error())
}

func (e *InvalidParamFormatError) Unwrap() error {
	return e.Err
}

type TooManyValuesForParamError struct {
	ParamName string
	Count     int
}

func (e *TooManyValuesForParamError) Error() string {
	return fmt.Sprintf("Expected one value for %s, got %d", e.ParamName, e.Count)
}

// Handler creates http.Handler with routing matching OpenAPI spec.
func Handler(si ServerInterface) http.Handler {
	return HandlerWithOptions(si, ChiServerOptions{})
}

type ChiServerOptions struct {
	BaseURL          string
	BaseRouter       chi.Router
	Middlewares      []MiddlewareFunc
	ErrorHandlerFunc func(w http.ResponseWriter, r *http.Request, err error)
}

// HandlerFromMux creates http.Handler with routing matching OpenAPI spec based on the provided mux.
func HandlerFromMux(si ServerInterface, r chi.Router) http.Handler {
	return HandlerWithOptions(si, ChiServerOptions{
		BaseRouter: r,
	})
}

func HandlerFromMuxWithBaseURL(si ServerInterface, r chi.Router, baseURL string) http.Handler {
	return HandlerWithOptions(si, ChiServerOptions{
		BaseURL:    baseURL,
		BaseRouter: r,
	})
}

// HandlerWithOptions creates http.Handler with additional options
func HandlerWithOptions(si ServerInterface, options ChiServerOptions) http.Handler {
	r := options.BaseRouter

	if r == nil {
		r = chi.NewRouter()
	}
	if options.ErrorHandlerFunc == nil {
		options.ErrorHandlerFunc = func(w http.ResponseWriter, r *http.Request, err error) {
			http.Error(w, err.Error(), http.StatusBadRequest)
		}
	}
	wrapper := ServerInterfaceWrapper{
		Handler:            si,
		HandlerMiddlewares: options.Middlewares,
		ErrorHandlerFunc:   options.ErrorHandlerFunc,
	}

	r.Group(func(r chi.Router) {
		r.Get(options.BaseURL+"/api/v1/organizations/{organizationID}/clustermanagers", wrapper.GetApiV1OrganizationsOrganizationIDClustermanagers)
	})
	r.Group(func(r chi.Router) {
		r.Get(options.BaseURL+"/api/v1/organizations/{organizationID}/clusters", wrapper.GetApiV1OrganizationsOrganizationIDClusters)
	})
	r.Group(func(r chi.Router) {
		r.Post(options.BaseURL+"/api/v1/organizations/{organizationID}/projects/{projectID}/clustermanagers", wrapper.PostApiV1OrganizationsOrganizationIDProjectsProjectIDClustermanagers)
	})
	r.Group(func(r chi.Router) {
		r.Delete(options.BaseURL+"/api/v1/organizations/{organizationID}/projects/{projectID}/clustermanagers/{clusterManagerID}", wrapper.DeleteApiV1OrganizationsOrganizationIDProjectsProjectIDClustermanagersClusterManagerID)
	})
	r.Group(func(r chi.Router) {
		r.Put(options.BaseURL+"/api/v1/organizations/{organizationID}/projects/{projectID}/clustermanagers/{clusterManagerID}", wrapper.PutApiV1OrganizationsOrganizationIDProjectsProjectIDClustermanagersClusterManagerID)
	})
	r.Group(func(r chi.Router) {
		r.Post(options.BaseURL+"/api/v1/organizations/{organizationID}/projects/{projectID}/clusters", wrapper.PostApiV1OrganizationsOrganizationIDProjectsProjectIDClusters)
	})
	r.Group(func(r chi.Router) {
		r.Delete(options.BaseURL+"/api/v1/organizations/{organizationID}/projects/{projectID}/clusters/{clusterID}", wrapper.DeleteApiV1OrganizationsOrganizationIDProjectsProjectIDClustersClusterID)
	})
	r.Group(func(r chi.Router) {
		r.Put(options.BaseURL+"/api/v1/organizations/{organizationID}/projects/{projectID}/clusters/{clusterID}", wrapper.PutApiV1OrganizationsOrganizationIDProjectsProjectIDClustersClusterID)
	})
	r.Group(func(r chi.Router) {
		r.Get(options.BaseURL+"/api/v1/organizations/{organizationID}/projects/{projectID}/clusters/{clusterID}/kubeconfig", wrapper.GetApiV1OrganizationsOrganizationIDProjectsProjectIDClustersClusterIDKubeconfig)
	})
	r.Group(func(r chi.Router) {
		r.Get(options.BaseURL+"/api/v1/organizations/{organizationID}/regions/{regionID}/flavors", wrapper.GetApiV1OrganizationsOrganizationIDRegionsRegionIDFlavors)
	})
	r.Group(func(r chi.Router) {
		r.Get(options.BaseURL+"/api/v1/organizations/{organizationID}/regions/{regionID}/images", wrapper.GetApiV1OrganizationsOrganizationIDRegionsRegionIDImages)
	})

	return r
}
