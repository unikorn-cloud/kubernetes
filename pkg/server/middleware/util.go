/*
Copyright 2022-2024 EscherCloud.

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

package middleware

import (
	"net/http"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/routers"
	"github.com/getkin/kin-openapi/routers/gorillamux"

	"github.com/spjmurray/unikorn/pkg/server/errors"
	"github.com/spjmurray/unikorn/pkg/server/generated"
)

// OpenAPI abstracts schema access and validation.
type OpenAPI struct {
	// spec is the full specification.
	spec *openapi3.T

	// router is a router able to process requests and return the
	// route from the spec.
	router routers.Router
}

// NewOpenRpi extracts the swagger document.
// NOTE: this is surprisingly slow, make sure you cache it and reuse it.
func NewOpenAPI() (*OpenAPI, error) {
	spec, err := generated.GetSwagger()
	if err != nil {
		return nil, err
	}

	router, err := gorillamux.NewRouter(spec)
	if err != nil {
		return nil, err
	}

	o := &OpenAPI{
		spec:   spec,
		router: router,
	}

	return o, nil
}

// findRoute looks up the route from the specification.
func (o *OpenAPI) findRoute(r *http.Request) (*routers.Route, map[string]string, error) {
	route, params, err := o.router.FindRoute(r)
	if err != nil {
		return nil, nil, errors.OAuth2ServerError("unable to find route").WithError(err)
	}

	return route, params, nil
}
