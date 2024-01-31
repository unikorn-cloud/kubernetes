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

package errors

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/unikorn-cloud/unikorn/pkg/server/generated"

	"sigs.k8s.io/controller-runtime/pkg/log"
)

var (
	// ErrRequest is raised for all handler errors.
	ErrRequest = errors.New("request error")
)

// HTTPError wraps ErrRequest with more contextual information that is used to
// propagate and create suitable responses.
type HTTPError struct {
	// status is the HTTP error code.
	status int

	// code us the terse error code to return to the client.
	code generated.Oauth2ErrorError

	// description is a verbose description to log/return to the user.
	description string

	// err is set when the originator was an error.  This is only used
	// for logging so as not to leak server internals to the client.
	err error

	// values are arbitrary key value pairs for logging.
	values []interface{}
}

// newHTTPError returns a new HTTP error.
func newHTTPError(status int, code generated.Oauth2ErrorError, description string) *HTTPError {
	return &HTTPError{
		status:      status,
		code:        code,
		description: description,
	}
}

// WithError augments the error with an error from a library.
func (e *HTTPError) WithError(err error) *HTTPError {
	e.err = err

	return e
}

// WithValues augments the error with a set of K/V pairs.
// Values should not use the "error" key as that's implicitly defined
// by WithError and could collide.
func (e *HTTPError) WithValues(values ...interface{}) *HTTPError {
	e.values = values

	return e
}

// Unwrap implements Go 1.13 errors.
func (e *HTTPError) Unwrap() error {
	return ErrRequest
}

// Error implements the error interface.
func (e *HTTPError) Error() string {
	return e.description
}

// Write returns the error code and description to the client.
func (e *HTTPError) Write(w http.ResponseWriter, r *http.Request) {
	// Log out any detail from the error that shouldn't be
	// reported to the client.  Do it before things can error
	// and return.
	log := log.FromContext(r.Context())

	var details []interface{}

	if e.description != "" {
		details = append(details, "detail", e.description)
	}

	if e.err != nil {
		details = append(details, "error", e.err)
	}

	if e.values != nil {
		details = append(details, e.values...)
	}

	log.Info("error detail", details...)

	// Emit the response to the client.
	w.Header().Add("Cache-Control", "no-cache")
	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(e.status)

	// Emit the response body.
	ge := &generated.Oauth2Error{
		Error:            e.code,
		ErrorDescription: e.description,
	}

	body, err := json.Marshal(ge)
	if err != nil {
		log.Error(err, "failed to marshal error response")

		return
	}

	if _, err := w.Write(body); err != nil {
		log.Error(err, "failed to wirte error response")

		return
	}
}

func HTTPForbidden(description string) *HTTPError {
	return newHTTPError(http.StatusForbidden, generated.Forbidden, description)
}

func HTTPNotFound() *HTTPError {
	return newHTTPError(http.StatusNotFound, generated.NotFound, "resource not found")
}

func IsHTTPNotFound(err error) bool {
	httpError := &HTTPError{}

	if ok := errors.As(err, &httpError); !ok {
		return false
	}

	if httpError.status != http.StatusNotFound {
		return false
	}

	return true
}

func HTTPMethodNotAllowed() *HTTPError {
	return newHTTPError(http.StatusMethodNotAllowed, generated.MethodNotAllowed, "the requested method was not allowed")
}

func HTTPConflict() *HTTPError {
	return newHTTPError(http.StatusConflict, generated.Conflict, "the requested resource already exists")
}

// OAuth2InvalidRequest indicates a client error.
func OAuth2InvalidRequest(description string) *HTTPError {
	return newHTTPError(http.StatusBadRequest, generated.InvalidRequest, description)
}

func OAuth2UnauthorizedClient(description string) *HTTPError {
	return newHTTPError(http.StatusBadRequest, generated.UnauthorizedClient, description)
}

func OAuth2UnsupportedGrantType(description string) *HTTPError {
	return newHTTPError(http.StatusBadRequest, generated.UnsupportedGrantType, description)
}

func OAuth2InvalidGrant(description string) *HTTPError {
	return newHTTPError(http.StatusBadRequest, generated.InvalidGrant, description)
}

func OAuth2InvalidClient(description string) *HTTPError {
	return newHTTPError(http.StatusBadRequest, generated.InvalidClient, description)
}

// OAuth2AccessDenied tells the client the authentication failed e.g.
// username/password are wrong, or a token has expired and needs reauthentication.
func OAuth2AccessDenied(description string) *HTTPError {
	return newHTTPError(http.StatusUnauthorized, generated.AccessDenied, description)
}

// OAuth2ServerError tells the client we are at fault, this should never be seen
// in production.  If so then our testing needs to improve.
func OAuth2ServerError(description string) *HTTPError {
	return newHTTPError(http.StatusInternalServerError, generated.ServerError, description)
}

// OAuth2InvalidScope tells the client it doesn't have the necessary scope
// to access the resource.
func OAuth2InvalidScope(description string) *HTTPError {
	return newHTTPError(http.StatusUnauthorized, generated.InvalidScope, description)
}

// toHTTPError is a handy unwrapper to get a HTTP error from a generic one.
func toHTTPError(err error) *HTTPError {
	var httpErr *HTTPError

	if !errors.As(err, &httpErr) {
		return nil
	}

	return httpErr
}

// HandleError is the top level error handler that should be called from all
// path handlers on error.
func HandleError(w http.ResponseWriter, r *http.Request, err error) {
	log := log.FromContext(r.Context())

	if httpError := toHTTPError(err); httpError != nil {
		httpError.Write(w, r)

		return
	}

	log.Error(err, "unhandled error")

	OAuth2ServerError("unhandled error").Write(w, r)
}
