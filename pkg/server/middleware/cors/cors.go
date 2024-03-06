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

package cors

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/spf13/pflag"

	"github.com/unikorn-cloud/core/pkg/util"
	"github.com/unikorn-cloud/unikorn/pkg/server/errors"
	"github.com/unikorn-cloud/unikorn/pkg/server/middleware/openapi"
)

type Options struct {
	AllowedOrigins []string
	MaxAge         int
}

func (o *Options) AddFlags(f *pflag.FlagSet) {
	f.StringSliceVar(&o.AllowedOrigins, "cors-allow-origin", []string{"*"}, "CORS allowed origins")
	f.IntVar(&o.MaxAge, "cors-max-age", 86400, "CORS maximum age (may be overridden by the browser)")
}

func Middleware(schema *openapi.Schema, options *Options) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// All requests get the allow origin header.
			for _, origin := range options.AllowedOrigins {
				w.Header().Add("Access-Control-Allow-Origin", origin)
			}

			// For normal requests handle them.
			if r.Method != http.MethodOptions {
				next.ServeHTTP(w, r)
				return
			}

			// Handle preflight
			method := r.Header.Get("Access-Control-Request-Method")
			if method == "" {
				errors.HandleError(w, r, errors.OAuth2InvalidRequest("OPTIONS missing Access-Control-Request-Method header"))
				return
			}

			request := r.Clone(r.Context())
			request.Method = method

			route, _, err := schema.FindRoute(request)
			if err != nil {
				errors.HandleError(w, r, err)
				return
			}

			// TODO: add OPTIONS to the schema?
			methods := util.Keys(route.PathItem.Operations())
			methods = append(methods, http.MethodOptions)

			// TODO: I've tried adding them to the schema, but the generator
			// adds them to the hander function signatures, which is superfluous
			// to requirements.
			headers := []string{
				"Authorization",
				"traceparent",
				"tracestate",
			}

			w.Header().Add("Access-Control-Allow-Methods", strings.Join(methods, ", "))
			w.Header().Add("Access-Control-Allow-Headers", strings.Join(headers, ", "))
			w.Header().Add("Access-Control-Max-Age", strconv.Itoa(options.MaxAge))
			w.WriteHeader(http.StatusNoContent)
		})
	}
}
