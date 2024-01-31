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

package server_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"testing"
	"time"

	chi "github.com/go-chi/chi/v5"
	"github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
	"golang.org/x/oauth2"

	unikornv1 "github.com/unikorn-cloud/unikorn/pkg/apis/unikorn/v1alpha1"
	"github.com/unikorn-cloud/unikorn/pkg/constants"
	"github.com/unikorn-cloud/unikorn/pkg/server"
	"github.com/unikorn-cloud/unikorn/pkg/server/generated"

	coreclient "github.com/unikorn-cloud/core/pkg/client"

	kerrors "k8s.io/apimachinery/pkg/api/errors"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const (
	// privKey is the signing key used for JOSE operations.
	privKey = `-----BEGIN PRIVATE KEY-----
MIHuAgEAMBAGByqGSM49AgEGBSuBBAAjBIHWMIHTAgEBBEIB0k3++pOE0i6sEYVE
Wd2kr2FFJwaCUG+Li6fUuam6QCYGrGN9Jg0A0OY5mP7wn4fbqmHVzqIzc5rIj2Fo
iQgNmjmhgYkDgYYABABlgG7igZ59Kl7I/InXVoWY+fmUzOBeHeayXim25ThlXPuZ
yTVEbD2+5PjIRKq/UAIuYyp3e5ZJpB1Npp2pcLEDygAHqQpVkvDKCyws6jENm5dy
sMxe3kmC4XEq7JPJGLkWjTeOZp1bkLB+N0DiCxAeV12h4ckkkYFQmpjMGVlMAK79
ZA==
-----END PRIVATE KEY-----`

	// pubKey is the public key used for JOSE verification operations.
	pubKey = `-----BEGIN CERTIFICATE-----
MIIB6jCCAUygAwIBAgIRAPfmWlg3c63mehO4VfUsQ5UwCgYIKoZIzj0EAwQwIjEg
MB4GA1UEAxMXVW5pa29ybiBTZXJ2ZXIgSk9TRSBLZXkwHhcNMjMwNjI3MTIwMTEz
WhcNMjMwOTI1MTIwMTEzWjAiMSAwHgYDVQQDExdVbmlrb3JuIFNlcnZlciBKT1NF
IEtleTCBmzAQBgcqhkjOPQIBBgUrgQQAIwOBhgAEAGWAbuKBnn0qXsj8iddWhZj5
+ZTM4F4d5rJeKbblOGVc+5nJNURsPb7k+MhEqr9QAi5jKnd7lkmkHU2mnalwsQPK
AAepClWS8MoLLCzqMQ2bl3KwzF7eSYLhcSrsk8kYuRaNN45mnVuQsH43QOILEB5X
XaHhySSRgVCamMwZWUwArv1koyAwHjAOBgNVHQ8BAf8EBAMCBaAwDAYDVR0TAQH/
BAIwADAKBggqhkjOPQQDBAOBiwAwgYcCQVAfjGe5J0kQUACA0jriLiANL0U74LHz
585rKFwe85AU7zN8XRiAbTiN0qJNoA0DqS5I3t9tg7Xm5JCzz5vUW7k/AkIBjXwr
kzmu+BkD1fagFQ5sJVadfwwf0RwT4Z0lzZ8xle2Af7udWnher5JH444GJtJhPD6c
KjAL9BBqzrnOrLYodEk=
-----END CERTIFICATE-----`

	// privKeyFile is where the signing key will live.
	privKeyFile = "/tmp/unikorn-priv-key.pem"

	// pubKeyFile is where the verification key will live.
	pubKeyFile = "/tmp/unikorn-pub-key.pem"
)

var (
	// debug turns on test debugging.
	//nolint:gochecknoglobals
	debug bool
)

func projectNameFromID(projectID string) string {
	return "unikorn-server-" + projectID
}

// writeFile creates the named file and writes the data to it.
func writeFile(path, data string) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}

	defer file.Close()

	if _, err := io.WriteString(file, data); err != nil {
		return err
	}

	return nil
}

// TestContext provides a common framework for test execution.
type TestContext struct {
	// openstackEndpoint records the TCP address of the mock openstack.
	openstackEndpoint net.Addr

	// openstackServer is the mock openstack server instance.
	openstackServer *http.Server

	// openstackRouter is the router used by the openstack server instance.
	// This allows you to chop and change handlers based on what responses
	// the test expects.
	openstackRouter chi.Router

	// unikornEndpoint records the TCP address of the unikorn server.
	unikornEndpoint net.Addr

	// unikornServer is the unikorn server instance.
	unikornServer *http.Server

	// kubernetesClient allows fake resources to be tested or mutated to
	// trigger various testing scenarios.
	kubernetesClient client.WithWatch
}

func MustNewTestContext(t *testing.T) (*TestContext, func()) {
	t.Helper()

	scheme, err := coreclient.NewScheme(unikornv1.AddToScheme)
	if err != nil {
		t.Fatal(err)
	}

	kubernetesClient := fake.NewClientBuilder().WithScheme(scheme).Build()
	openstackEndpoint, openstackServer, openstackRouter := mustSetupOpenstackServer(t)
	unikornEndpoint, unikornServer := mustSetupUnikornServer(t, openstackEndpoint, kubernetesClient)

	tc := &TestContext{
		openstackEndpoint: openstackEndpoint,
		openstackServer:   openstackServer,
		openstackRouter:   openstackRouter,
		unikornEndpoint:   unikornEndpoint,
		unikornServer:     unikornServer,
		kubernetesClient:  kubernetesClient,
	}

	shutdown := func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := openstackServer.Shutdown(ctx); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		if err := unikornServer.Shutdown(ctx); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	}

	return tc, shutdown
}

func (t *TestContext) OpenstackServerEndpoint() string {
	return t.openstackEndpoint.String()
}

func (t *TestContext) UnikornServerEndpoint() string {
	return t.unikornEndpoint.String()
}

func (t *TestContext) OpenstackRouter() chi.Router {
	return t.openstackRouter
}

func (t *TestContext) KubernetesClient() client.WithWatch {
	return t.kubernetesClient
}

// mustSetupOpenstackServer starts the openstack mock server running.
func mustSetupOpenstackServer(t *testing.T) (net.Addr, *http.Server, chi.Router) {
	t.Helper()

	router := chi.NewRouter()

	if debug {
		loggingMiddleware := func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				fmt.Println(r.Method, r.URL.Path)
				next.ServeHTTP(w, r)
			})
		}

		router.Use(loggingMiddleware)
	}

	listener, err := net.Listen("tcp", "127.0.0.1:")
	if err != nil {
		t.Fatal(err)
	}

	s := &http.Server{
		ReadTimeout:       time.Second,
		ReadHeaderTimeout: time.Second,
		WriteTimeout:      time.Second,
		Handler:           router,
	}

	go func() {
		if err := s.Serve(listener); err != nil {
			if !errors.Is(err, http.ErrServerClosed) {
				fmt.Println(err)
			}
		}
	}()

	return listener.Addr(), s, router
}

// mustSetupUnikornServer starts the unikorn server running.
func mustSetupUnikornServer(t *testing.T, openstack net.Addr, client client.WithWatch) (net.Addr, *http.Server) {
	t.Helper()

	goFlagSet := flag.NewFlagSet(t.Name(), flag.PanicOnError)

	flagSet := pflag.NewFlagSet(t.Name(), pflag.PanicOnError)
	flagSet.AddGoFlagSet(goFlagSet)

	s := &server.Server{}

	s.AddFlags(goFlagSet, flagSet)

	flags := []string{
		"--jose-tls-cert=" + pubKeyFile,
		"--jose-tls-key=" + privKeyFile,
		"--keystone-endpoint=http://" + openstack.String() + "/identity",
		"--image-signing-key=LS0tLS1CRUdJTiBQVUJMSUMgS0VZLS0tLS0KTUhZd0VBWUhLb1pJemowQ0FRWUZLNEVFQUNJRFlnQUVmOGs4RVY1TUg4M1BncThYd0JGUTd5YkU2NTEzRlh0awpHaG1jalp4WmYzbU5QOE0vb3VBbE0vZHdYWGpFeXZTNlJhVHdoT3A0aTdHL3VvbE5ZL0RJSCt1elc2VXNxR3VHClFpSW11Tm9BdzFSS1NQcEtyNWlJVXU2eEc1cDR3U3E5Ci0tLS0tRU5EIFBVQkxJQyBLRVktLS0tLQo=",
		"--flavors-exclude-property=resources:CUSTOM_BAREMETAL",
		"--flavors-gpu-descriptor=property=resources:VGPU,expression=^(\\d+)$",
		"--flavors-gpu-descriptor=property=pci_passthrough:alias,expression=^a100:(\\d+)$",
		"--application-credential-roles=_member_,member,load-balancer_member",
	}

	if err := flagSet.Parse(flags); err != nil {
		t.Fatal(err)
	}

	// Override any flag defaults we need to.
	s.Options.RequestTimeout = 0

	if debug {
		s.SetupLogging()

		if err := s.SetupOpenTelemetry(context.Background()); err != nil {
			t.Fatal(err)
		}
	}

	server, err := s.GetServer(client)
	if err != nil {
		t.Fatal(err)
	}

	listener, err := net.Listen("tcp", "127.0.0.1:")
	if err != nil {
		t.Fatal(err)
	}

	go func() {
		if err := server.Serve(listener); err != nil {
			if !errors.Is(err, http.ErrServerClosed) {
				fmt.Println(err)
			}
		}
	}()

	return listener.Addr(), server
}

// TestMain is the entry point to the tests, now ideally this wouldn't exist,
// but I'll hazard a guess that making tests parallelizable will cause issues
// given the inherent dynamic nature of the ports.
func TestMain(m *testing.M) {
	flag.BoolVar(&debug, "debug", false, "Turn on test debugging output")
	flag.Parse()

	// Setup actual things are shared.
	if err := writeFile(privKeyFile, privKey); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	if err := writeFile(pubKeyFile, pubKey); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// Test!
	os.Exit(m.Run())
}

// JSONReader implments io.Reader that does lazy JSON marshaling.
type JSONReader struct {
	data interface{}
	buf  *bytes.Buffer
}

func NewJSONReader(data interface{}) *JSONReader {
	return &JSONReader{
		data: data,
	}
}

func (r *JSONReader) Read(p []byte) (int, error) {
	if r.buf == nil {
		data, err := json.Marshal(r.data)
		if err != nil {
			return 0, err
		}

		r.buf = bytes.NewBuffer(data)
	}

	return r.buf.Read(p)
}

// MustNewUnscopedToken does username/password authentication using oauth2
// and returns the token.
func MustNewUnscopedToken(t *testing.T, tc *TestContext) string {
	t.Helper()

	config := &oauth2.Config{
		Endpoint: oauth2.Endpoint{
			TokenURL: "http://" + tc.UnikornServerEndpoint() + "/api/v1/auth/oauth2/tokens",
		},
	}

	token, err := config.PasswordCredentialsToken(context.TODO(), "foo", "bar")
	assert.NoError(t, err)
	assert.NotEqual(t, token.AccessToken, "")

	return token.AccessToken
}

// bearerTokenInjector allows a generic client to inject a bearer token for authn/authz.
func bearerTokenInjector(token string) generated.RequestEditorFn {
	return func(ctx context.Context, req *http.Request) error {
		req.Header.Set("Authorization", "Bearer "+token)

		return nil
	}
}

// MustNewClient creates a new client, or dies on error.
func MustNewClient(t *testing.T, tc *TestContext, token string) *generated.ClientWithResponses {
	t.Helper()

	client, err := generated.NewClientWithResponses("http://"+tc.UnikornServerEndpoint(), generated.WithRequestEditorFn(bearerTokenInjector(token)))
	assert.NoError(t, err)

	return client
}

// MustNewUnscopedClient creates a new client, or dies on error, implicitly doing
// the oauth2 password grant flow.
func MustNewUnscopedClient(t *testing.T, tc *TestContext) *generated.ClientWithResponses {
	t.Helper()

	return MustNewClient(t, tc, MustNewUnscopedToken(t, tc))
}

// MustNewScopedClient creates a new client, or dies on error, implicitly doing
// the oauth2 password grant flow, then upgrading that to be scoped to a project.
func MustNewScopedClient(t *testing.T, tc *TestContext) *generated.ClientWithResponses {
	t.Helper()

	scope := &generated.TokenScope{
		Project: generated.TokenScopeProject{
			Id: projectID,
		},
	}

	response, err := MustNewUnscopedClient(t, tc).PostApiV1AuthTokensTokenWithBodyWithResponse(context.TODO(), "application/json", NewJSONReader(scope))
	assert.NoError(t, err)
	assert.Equal(t, http.StatusCreated, response.HTTPResponse.StatusCode)

	return MustNewClient(t, tc, response.JSON201.AccessToken)
}

// MustDoRequestWithForm is a helper that forms and executes a HTTP request with form data.
// This is most useful for testing oauth2/oidc.
func MustDoRequestWithForm(t *testing.T, method, url string, form url.Values) *http.Response {
	t.Helper()

	body := bytes.NewBufferString(form.Encode())

	request, err := http.NewRequestWithContext(context.TODO(), method, url, body)
	assert.NoError(t, err)

	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{}

	response, err := client.Do(request)
	assert.NoError(t, err)

	return response
}

// AssertOauth2Error ensures the response contains a valid error and its type is as expected.
func AssertOauth2Error(t *testing.T, response *http.Response, errorType generated.Oauth2ErrorError) {
	t.Helper()

	assert.Equal(t, "application/json", response.Header.Get("Content-Type"))

	// NOTE: you'll need to replace the body if anything else needs to read it.
	decoder := json.NewDecoder(response.Body)

	var serverErr generated.Oauth2Error

	err := decoder.Decode(&serverErr)
	assert.NoError(t, err)

	assert.Equal(t, errorType, serverErr.Error)
}

// TestApiV1AuthOAuth2TokensPassword tests the oauth2 password grant flow works.
func TestApiV1AuthOAuth2TokensPassword(t *testing.T) {
	t.Parallel()

	tc, cleanup := MustNewTestContext(t)
	defer cleanup()

	RegisterIdentityHandlers(tc)

	_ = MustNewUnscopedClient(t, tc)
}

// TestApiV1AuthOAuth2TokensPasswordUnauthorized tests oauth2 password grant failure due to
// a bad username or password.
func TestApiV1AuthOAuth2TokensPasswordUnauthorized(t *testing.T) {
	t.Parallel()

	tc, cleanup := MustNewTestContext(t)
	defer cleanup()

	RegisterIdentityV3AuthTokensPostUnauthorizedHandler(tc)

	endpoint := "http://" + tc.UnikornServerEndpoint() + "/api/v1/auth/oauth2/tokens"

	query := url.Values{}
	query.Set("grant_type", "password")
	query.Set("username", "sahtrshdfda")
	query.Set("password", "fthrdsesgsg")

	response := MustDoRequestWithForm(t, http.MethodPost, endpoint, query)
	assert.Equal(t, http.StatusUnauthorized, response.StatusCode)

	defer response.Body.Close()

	AssertOauth2Error(t, response, generated.AccessDenied)
}

// TestApiV1AuthOAuth2TokensPasswordInvalid tests oauth2 password grant failure due to
// missing POST data.
func TestApiV1AuthOAuth2TokensPasswordInvalid(t *testing.T) {
	t.Parallel()

	tc, cleanup := MustNewTestContext(t)
	defer cleanup()

	endpoint := "http://" + tc.UnikornServerEndpoint() + "/api/v1/auth/oauth2/tokens"

	query := url.Values{}
	query.Set("grant_type", "password")

	response := MustDoRequestWithForm(t, http.MethodPost, endpoint, query)
	assert.Equal(t, http.StatusBadRequest, response.StatusCode)

	defer response.Body.Close()

	AssertOauth2Error(t, response, generated.InvalidRequest)
}

// TestApiV1AuthTokensToken tests an unscoped token can be scoped to a project.
func TestApiV1AuthTokensToken(t *testing.T) {
	t.Parallel()

	tc, cleanup := MustNewTestContext(t)
	defer cleanup()

	RegisterIdentityHandlers(tc)

	_ = MustNewScopedClient(t, tc)
}

// TestApiV1AuthTokensTokenBadRequest tests that the tokens endpoint will error
// correctly when no authorization token is provided.
func TestApiV1AuthTokensTokenBadRequest(t *testing.T) {
	t.Parallel()

	tc, cleanup := MustNewTestContext(t)
	defer cleanup()

	client, err := generated.NewClientWithResponses("http://" + tc.UnikornServerEndpoint())
	assert.NoError(t, err)

	response, err := client.PostApiV1AuthTokensTokenWithBodyWithResponse(context.TODO(), "application/json", NewJSONReader(&generated.TokenScope{}))
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, response.HTTPResponse.StatusCode)
	assert.NotNil(t, response.JSON400)

	serverErr := *response.JSON400

	assert.Equal(t, generated.InvalidRequest, serverErr.Error)
}

// TestApiV1AuthTokensTokenUnauthorized tests that the tokens endpoint will error
// correctly when a bad autorization token is provided.
func TestApiV1AuthTokensTokenUnauthorized(t *testing.T) {
	t.Parallel()

	tc, cleanup := MustNewTestContext(t)
	defer cleanup()

	client, err := generated.NewClientWithResponses("http://"+tc.UnikornServerEndpoint(), generated.WithRequestEditorFn(bearerTokenInjector("garbage")))
	assert.NoError(t, err)

	response, err := client.PostApiV1AuthTokensTokenWithBodyWithResponse(context.TODO(), "application/json", NewJSONReader(&generated.TokenScope{}))
	assert.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, response.HTTPResponse.StatusCode)
	assert.NotNil(t, response.JSON401)

	serverErr := *response.JSON401

	assert.Equal(t, generated.AccessDenied, serverErr.Error)
}

// TestApiV1ProjectCreate tests that a project scoped token can create a project
// with the correct name, and delete it.
func TestApiV1ProjectCreateAndDelete(t *testing.T) {
	t.Parallel()

	tc, cleanup := MustNewTestContext(t)
	defer cleanup()

	RegisterIdentityHandlers(tc)

	unikornClient := MustNewScopedClient(t, tc)

	createResponse, err := unikornClient.PostApiV1Project(context.TODO())
	assert.NoError(t, err)
	assert.Equal(t, http.StatusAccepted, createResponse.StatusCode)

	defer createResponse.Body.Close()

	var project unikornv1.Project

	assert.NoError(t, tc.KubernetesClient().Get(context.TODO(), client.ObjectKey{Name: projectNameFromID(projectID)}, &project))
	assert.Contains(t, project.Labels, constants.VersionLabel)

	deleteResponse, err := unikornClient.DeleteApiV1Project(context.TODO())
	assert.NoError(t, err)
	assert.Equal(t, http.StatusAccepted, deleteResponse.StatusCode)

	defer deleteResponse.Body.Close()

	var statusErr *kerrors.StatusError

	assert.ErrorAs(t, tc.KubernetesClient().Get(context.TODO(), client.ObjectKey{Name: projectNameFromID(projectID)}, &project), &statusErr)
	assert.Equal(t, http.StatusNotFound, int(statusErr.ErrStatus.Code))
}

// TestApiV1ProjectCreateExisting tests a project cannot be created on top of an
// existing one.
func TestApiV1ProjectCreateExisting(t *testing.T) {
	t.Parallel()

	tc, cleanup := MustNewTestContext(t)
	defer cleanup()

	RegisterIdentityHandlers(tc)

	mustCreateProjectFixture(t, tc, projectID)

	unikornClient := MustNewScopedClient(t, tc)

	response, err := unikornClient.PostApiV1ProjectWithResponse(context.TODO())
	assert.NoError(t, err)
	assert.Equal(t, http.StatusConflict, response.HTTPResponse.StatusCode)
	assert.NotNil(t, response.JSON409)

	serverErr := *response.JSON409

	assert.Equal(t, serverErr.Error, generated.Conflict)
}

// TestApiV1ProjectDeleteNotFound tests a project deletion when there is no
// project errors in the correct way.
func TestApiV1ProjectDeleteNotFound(t *testing.T) {
	t.Parallel()

	tc, cleanup := MustNewTestContext(t)
	defer cleanup()

	RegisterIdentityHandlers(tc)

	unikornClient := MustNewScopedClient(t, tc)

	response, err := unikornClient.DeleteApiV1ProjectWithResponse(context.TODO())
	assert.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, response.HTTPResponse.StatusCode)
	assert.NotNil(t, response.JSON404)

	serverErr := *response.JSON404

	assert.Equal(t, serverErr.Error, generated.NotFound)
}

// TestApiV1ControlPlaneCreate tests that a control plane can be created
// in a project.
func TestApiV1ControlPlanesCreate(t *testing.T) {
	t.Parallel()

	tc, cleanup := MustNewTestContext(t)
	defer cleanup()

	RegisterIdentityHandlers(tc)

	project := mustCreateProjectFixture(t, tc, projectID)

	unikornClient := MustNewScopedClient(t, tc)

	request := &generated.ControlPlane{
		Name: "foo",
		ApplicationBundle: generated.ApplicationBundle{
			Name: "foo",
		},
	}

	response, err := unikornClient.PostApiV1ControlplanesWithBody(context.TODO(), "application/json", NewJSONReader(request))
	assert.NoError(t, err)
	assert.Equal(t, http.StatusAccepted, response.StatusCode)

	defer response.Body.Close()

	var resource unikornv1.ControlPlane

	assert.NoError(t, tc.KubernetesClient().Get(context.TODO(), client.ObjectKey{Namespace: project.Status.Namespace, Name: "foo"}, &resource))
	assert.Contains(t, resource.Labels, constants.VersionLabel)
	assert.Contains(t, resource.Labels, constants.ProjectLabel)
}

// TestApiV1ControlPlanesCreateExisting tests control plane creation when another
// already exists with the same name.
func TestApiV1ControlPlanesCreateExisting(t *testing.T) {
	t.Parallel()

	tc, cleanup := MustNewTestContext(t)
	defer cleanup()

	RegisterIdentityHandlers(tc)

	project := mustCreateProjectFixture(t, tc, projectID)
	mustCreateControlPlaneFixture(t, tc, project.Status.Namespace, "foo")

	unikornClient := MustNewScopedClient(t, tc)

	request := &generated.ControlPlane{
		Name: "foo",
		ApplicationBundle: generated.ApplicationBundle{
			Name: "foo",
		},
	}

	response, err := unikornClient.PostApiV1ControlplanesWithBodyWithResponse(context.TODO(), "application/json", NewJSONReader(request))
	assert.NoError(t, err)
	assert.Equal(t, http.StatusConflict, response.HTTPResponse.StatusCode)
	assert.NotNil(t, response.JSON409)

	serverErr := *response.JSON409

	assert.Equal(t, serverErr.Error, generated.Conflict)
}

// TestApiV1ControlPlaneCreateImplicitProject tests that a control plane can be created
// in a project that does not exist yet.
func TestApiV1ControlPlanesCreateImplicitProject(t *testing.T) {
	t.Parallel()

	tc, cleanup := MustNewTestContext(t)
	defer cleanup()

	RegisterIdentityHandlers(tc)

	unikornClient := MustNewScopedClient(t, tc)

	request := &generated.ControlPlane{
		Name: "foo",
		ApplicationBundle: generated.ApplicationBundle{
			Name: "foo",
		},
	}

	response, err := unikornClient.PostApiV1ControlplanesWithBodyWithResponse(context.TODO(), "application/json", NewJSONReader(request))
	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, response.HTTPResponse.StatusCode)
	assert.NotNil(t, response.JSON500)

	serverErr := *response.JSON500

	assert.Equal(t, generated.ServerError, serverErr.Error)

	// TODO: we should probably emulate the project manager here and allocate a namespace
	// so the handler can progress... However, it' proabably much easier to do this with
	// integration testing.
	var resource unikornv1.Project

	assert.NoError(t, tc.KubernetesClient().Get(context.TODO(), client.ObjectKey{Name: projectNameFromID(projectID)}, &resource))
	assert.Contains(t, resource.Labels, constants.VersionLabel)
}

// TestApiV1ControlPlanesNotCreateImplicitProject checks a project is not created when the
// control plane request is not a create.
func TestApiV1ControlPlanesNotCreateImplicitProject(t *testing.T) {
	t.Parallel()

	tc, cleanup := MustNewTestContext(t)
	defer cleanup()

	RegisterIdentityHandlers(tc)

	unikornClient := MustNewScopedClient(t, tc)

	response, err := unikornClient.GetApiV1ControlplanesControlPlaneNameWithResponse(context.TODO(), "foo")
	assert.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, response.HTTPResponse.StatusCode)
	assert.NotNil(t, response.JSON404)

	serverErr := *response.JSON404

	assert.Equal(t, serverErr.Error, generated.NotFound)

	var resource unikornv1.Project

	assert.Error(t, tc.KubernetesClient().Get(context.TODO(), client.ObjectKey{Name: projectNameFromID(projectID)}, &resource))
}

// TestApiV1ControlPlanesGet tests a control plane can be retrieved and
// that its fields are completed as specified by the schema.  This flexes
// compositing of resources e.g. expansion of application bundles.
func TestApiV1ControlPlanesGet(t *testing.T) {
	t.Parallel()

	tc, cleanup := MustNewTestContext(t)
	defer cleanup()

	RegisterIdentityHandlers(tc)

	project := mustCreateProjectFixture(t, tc, projectID)
	mustCreateControlPlaneFixture(t, tc, project.Status.Namespace, "foo")
	mustCreateControlPlaneApplicationBundleFixture(t, tc)

	unikornClient := MustNewScopedClient(t, tc)

	response, err := unikornClient.GetApiV1ControlplanesControlPlaneNameWithResponse(context.TODO(), "foo")
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, response.HTTPResponse.StatusCode)
	assert.NotNil(t, response.JSON200)

	result := *response.JSON200

	assert.Equal(t, "foo", result.Name)
	assert.NotNil(t, result.Status)
	assert.Equal(t, "Provisioned", result.Status.Status)
	assert.Equal(t, controlPlaneApplicationBundleName, result.ApplicationBundle.Name)
	assert.Equal(t, controlPlaneApplicationBundleVersion, result.ApplicationBundle.Version)
}

// TestApiV1ControlPlanesGetNotFound tests control planes behave correctly when
// a control plane doesn't exist.
func TestApiV1ControlPlanesGetNotFound(t *testing.T) {
	t.Parallel()

	tc, cleanup := MustNewTestContext(t)
	defer cleanup()

	RegisterIdentityHandlers(tc)

	mustCreateProjectFixture(t, tc, projectID)

	unikornClient := MustNewScopedClient(t, tc)

	response, err := unikornClient.GetApiV1ControlplanesControlPlaneNameWithResponse(context.TODO(), "foo")
	assert.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, response.HTTPResponse.StatusCode)
	assert.NotNil(t, response.JSON404)

	serverErr := *response.JSON404

	assert.Equal(t, serverErr.Error, generated.NotFound)
}

// TestApiV1ControlPlanesList tests a control planes can be retrieved and
// that their fields are completed as specified by the schema.  This flexes
// compositing of resources e.g. expansion of application bundles.
func TestApiV1ControlPlanesList(t *testing.T) {
	t.Parallel()

	tc, cleanup := MustNewTestContext(t)
	defer cleanup()

	RegisterIdentityHandlers(tc)

	project := mustCreateProjectFixture(t, tc, projectID)
	mustCreateControlPlaneFixture(t, tc, project.Status.Namespace, "foo")
	mustCreateControlPlaneApplicationBundleFixture(t, tc)

	unikornClient := MustNewScopedClient(t, tc)

	response, err := unikornClient.GetApiV1ControlplanesWithResponse(context.TODO())
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, response.HTTPResponse.StatusCode)
	assert.NotNil(t, response.JSON200)

	results := *response.JSON200

	assert.Len(t, results, 1)
	assert.Equal(t, "foo", results[0].Name)
	assert.Equal(t, controlPlaneApplicationBundleName, results[0].ApplicationBundle.Name)
	assert.Equal(t, controlPlaneApplicationBundleVersion, results[0].ApplicationBundle.Version)
}

// TestApiV1ControlPlanesUpdate tests control planes can be updated.
func TestApiV1ControlPlanesUpdate(t *testing.T) {
	t.Parallel()

	tc, cleanup := MustNewTestContext(t)
	defer cleanup()

	RegisterIdentityHandlers(tc)

	project := mustCreateProjectFixture(t, tc, projectID)
	mustCreateControlPlaneFixture(t, tc, project.Status.Namespace, "foo")

	request := &generated.ControlPlane{
		Name: "foo",
		ApplicationBundle: generated.ApplicationBundle{
			Name: "foo",
		},
	}

	unikornClient := MustNewScopedClient(t, tc)

	response, err := unikornClient.PutApiV1ControlplanesControlPlaneNameWithBody(context.TODO(), "foo", "application/json", NewJSONReader(request))
	assert.NoError(t, err)
	assert.Equal(t, http.StatusAccepted, response.StatusCode)

	defer response.Body.Close()

	var resource unikornv1.ControlPlane

	assert.NoError(t, tc.KubernetesClient().Get(context.TODO(), client.ObjectKey{Namespace: project.Status.Namespace, Name: "foo"}, &resource))
	assert.NotNil(t, resource.Spec.ApplicationBundle)
	assert.Equal(t, *resource.Spec.ApplicationBundle, "foo")
}

// TestApiV1ControlPlanesUpdateNotFound tests control planes behave correctly when
// an update request is made for a non-existent resource.
func TestApiV1ControlPlanesUpdateNotFound(t *testing.T) {
	t.Parallel()

	tc, cleanup := MustNewTestContext(t)
	defer cleanup()

	RegisterIdentityHandlers(tc)

	mustCreateProjectFixture(t, tc, projectID)

	request := &generated.ControlPlane{
		Name: "foo",
		ApplicationBundle: generated.ApplicationBundle{
			Name: "foo",
		},
	}

	unikornClient := MustNewScopedClient(t, tc)

	response, err := unikornClient.PutApiV1ControlplanesControlPlaneNameWithBodyWithResponse(context.TODO(), "foo", "application/json", NewJSONReader(request))
	assert.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, response.HTTPResponse.StatusCode)
	assert.NotNil(t, response.JSON404)

	serverErr := *response.JSON404

	assert.Equal(t, serverErr.Error, generated.NotFound)
}

// TestApiV1ControlPlanesDelete tests a control plane can be deleted.
func TestApiV1ControlPlanesDelete(t *testing.T) {
	t.Parallel()

	tc, cleanup := MustNewTestContext(t)
	defer cleanup()

	RegisterIdentityHandlers(tc)

	project := mustCreateProjectFixture(t, tc, projectID)
	mustCreateControlPlaneFixture(t, tc, project.Status.Namespace, "foo")

	unikornClient := MustNewScopedClient(t, tc)

	response, err := unikornClient.DeleteApiV1ControlplanesControlPlaneName(context.TODO(), "foo")
	assert.NoError(t, err)
	assert.Equal(t, http.StatusAccepted, response.StatusCode)

	defer response.Body.Close()

	var resource unikornv1.ControlPlane

	var statusErr *kerrors.StatusError

	assert.ErrorAs(t, tc.KubernetesClient().Get(context.TODO(), client.ObjectKey{Namespace: project.Status.Namespace, Name: "foo"}, &resource), &statusErr)
	assert.Equal(t, http.StatusNotFound, int(statusErr.ErrStatus.Code))
}

// TestApiV1ControlPlanesDeleteNotFound tests that a deletion of a not found resource
// results in the correct response.
func TestApiV1ControlPlanesDeleteNotFound(t *testing.T) {
	t.Parallel()

	tc, cleanup := MustNewTestContext(t)
	defer cleanup()

	RegisterIdentityHandlers(tc)

	mustCreateProjectFixture(t, tc, projectID)

	unikornClient := MustNewScopedClient(t, tc)

	response, err := unikornClient.DeleteApiV1ControlplanesControlPlaneNameWithResponse(context.TODO(), "foo")
	assert.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, response.HTTPResponse.StatusCode)
	assert.NotNil(t, response.JSON404)

	serverErr := *response.JSON404

	assert.Equal(t, serverErr.Error, generated.NotFound)
}

// createClusterRequest is a basic request that can be shared across tests.
// NOTE: obviously if you want to mutate this in any way you may want to find
// a way to deep copy it first!  You would win a prize for removing the nolint
// tag below...
//
//nolint:gochecknoglobals
var createClusterRequest = &generated.KubernetesCluster{
	Name: "foo",
	ApplicationBundle: generated.ApplicationBundle{
		Name: "foo",
	},
	Network: generated.KubernetesClusterNetwork{
		DnsNameservers: []string{
			"8.8.8.8",
		},
		NodePrefix:    "192.168.0.0/24",
		ServicePrefix: "172.16.0.0/12",
		PodPrefix:     "10.0.0.0/8",
	},
	ControlPlane: generated.OpenstackMachinePool{
		Version:    "v1.28.0",
		Replicas:   3,
		ImageName:  "ubuntu-24.04-lts",
		FlavorName: flavorName,
	},
	WorkloadPools: generated.KubernetesClusterWorkloadPools{
		{
			Name: "foo",
			Machine: generated.OpenstackMachinePool{
				Version:    "v1.28.0",
				Replicas:   3,
				ImageName:  "ubuntu-24.04-lts",
				FlavorName: flavorName,
			},
		},
	},
}

// TestApiV1ClustersCreate tests that a cluster can be created
// in a control plane.
func TestApiV1ClustersCreate(t *testing.T) {
	t.Parallel()

	tc, cleanup := MustNewTestContext(t)
	defer cleanup()

	RegisterIdentityHandlers(tc)
	RegisterImageV2Images(tc)
	RegisterComputeV2FlavorsDetail(tc)
	RegisterComputeV2ServerGroups(tc)

	project := mustCreateProjectFixture(t, tc, projectID)
	controlPlane := mustCreateControlPlaneFixture(t, tc, project.Status.Namespace, "foo")

	unikornClient := MustNewScopedClient(t, tc)

	response, err := unikornClient.PostApiV1ControlplanesControlPlaneNameClustersWithBody(context.TODO(), controlPlane.Name, "application/json", NewJSONReader(createClusterRequest))
	assert.NoError(t, err)
	assert.Equal(t, http.StatusAccepted, response.StatusCode)

	defer response.Body.Close()

	var resource unikornv1.KubernetesCluster

	assert.NoError(t, tc.KubernetesClient().Get(context.TODO(), client.ObjectKey{Namespace: controlPlane.Status.Namespace, Name: "foo"}, &resource))
	assert.Contains(t, resource.Labels, constants.VersionLabel)
	assert.Contains(t, resource.Labels, constants.ProjectLabel)
	assert.Contains(t, resource.Labels, constants.ControlPlaneLabel)
}

// TestApiV1ClustersCreateUnauthorized tests a keystone token expiring during a
// request errors in the right way.
// NOTE: this assumes other implicit calls such as those to images, server groups
// and application credentials all use the same error processing.
func TestApiV1ClustersCreateUnauthorized(t *testing.T) {
	t.Parallel()

	tc, cleanup := MustNewTestContext(t)
	defer cleanup()

	RegisterIdentityHandlers(tc)
	RegisterImageV2Images(tc)
	RegisterComputeV2FlavorsDetailUnauthorized(tc)
	RegisterComputeV2ServerGroups(tc)

	project := mustCreateProjectFixture(t, tc, projectID)
	controlPlane := mustCreateControlPlaneFixture(t, tc, project.Status.Namespace, "foo")

	unikornClient := MustNewScopedClient(t, tc)

	response, err := unikornClient.PostApiV1ControlplanesControlPlaneNameClustersWithBodyWithResponse(context.TODO(), controlPlane.Name, "application/json", NewJSONReader(createClusterRequest))
	assert.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, response.HTTPResponse.StatusCode)
	assert.NotNil(t, response.JSON401)

	serverErr := *response.JSON401

	assert.Equal(t, generated.AccessDenied, serverErr.Error)
}

// TestApiV1ClustersCreateForbidden tests the use of an API that, while authenticated,
// is not allowed by current user role assignments.  We typically see this with
// federation where you cannot create an application credential at all.
func TestApiV1ClustersCreateForbidden(t *testing.T) {
	t.Parallel()

	tc, cleanup := MustNewTestContext(t)
	defer cleanup()

	RegisterIdentityHandlers(tc)
	RegisterIdentityV3UserApplicationCredentialsForbidden(tc)
	RegisterImageV2Images(tc)
	RegisterComputeV2FlavorsDetail(tc)
	RegisterComputeV2ServerGroups(tc)

	project := mustCreateProjectFixture(t, tc, projectID)
	controlPlane := mustCreateControlPlaneFixture(t, tc, project.Status.Namespace, "foo")

	unikornClient := MustNewScopedClient(t, tc)

	response, err := unikornClient.PostApiV1ControlplanesControlPlaneNameClustersWithBodyWithResponse(context.TODO(), controlPlane.Name, "application/json", NewJSONReader(createClusterRequest))
	assert.NoError(t, err)
	assert.Equal(t, http.StatusForbidden, response.HTTPResponse.StatusCode)
	assert.NotNil(t, response.JSON403)

	serverErr := *response.JSON403

	assert.Equal(t, generated.Forbidden, serverErr.Error)
}

// TestApiV1ClustersCreateExisting tests creating a cluster when one exists
// errors in the right way.
func TestApiV1ClustersCreateExisting(t *testing.T) {
	t.Parallel()

	tc, cleanup := MustNewTestContext(t)
	defer cleanup()

	RegisterIdentityHandlers(tc)
	RegisterImageV2Images(tc)
	RegisterComputeV2FlavorsDetail(tc)
	RegisterComputeV2ServerGroups(tc)

	project := mustCreateProjectFixture(t, tc, projectID)
	controlPlane := mustCreateControlPlaneFixture(t, tc, project.Status.Namespace, "foo")
	mustCreateKubernetesClusterFixture(t, tc, controlPlane.Status.Namespace, "foo")

	unikornClient := MustNewScopedClient(t, tc)

	response, err := unikornClient.PostApiV1ControlplanesControlPlaneNameClustersWithBodyWithResponse(context.TODO(), controlPlane.Name, "application/json", NewJSONReader(createClusterRequest))
	assert.NoError(t, err)
	assert.Equal(t, http.StatusConflict, response.HTTPResponse.StatusCode)
	assert.NotNil(t, response.JSON409)

	serverErr := *response.JSON409

	assert.Equal(t, serverErr.Error, generated.Conflict)
}

// TestApiV1ClustersCreateImplicitControlPlane tests that a cluster can be created
// in a control plane that doesn't exist yet.
func TestApiV1ClustersCreateImplicitControlPlane(t *testing.T) {
	t.Parallel()

	tc, cleanup := MustNewTestContext(t)
	defer cleanup()

	RegisterIdentityHandlers(tc)
	RegisterImageV2Images(tc)
	RegisterComputeV2FlavorsDetail(tc)
	RegisterComputeV2ServerGroups(tc)

	project := mustCreateProjectFixture(t, tc, projectID)
	mustCreateControlPlaneApplicationBundleFixture(t, tc)

	unikornClient := MustNewScopedClient(t, tc)

	response, err := unikornClient.PostApiV1ControlplanesControlPlaneNameClustersWithBodyWithResponse(context.TODO(), "foo", "application/json", NewJSONReader(createClusterRequest))
	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, response.HTTPResponse.StatusCode)
	assert.NotNil(t, response.JSON500)

	serverErr := *response.JSON500

	assert.Equal(t, generated.ServerError, serverErr.Error)

	var resource unikornv1.ControlPlane

	assert.NoError(t, tc.KubernetesClient().Get(context.TODO(), client.ObjectKey{Namespace: project.Status.Namespace, Name: "foo"}, &resource))
	assert.Contains(t, resource.Labels, constants.VersionLabel)
	assert.Contains(t, resource.Labels, constants.ProjectLabel)
}

// TestApiV1ClustersNotCreateImplicitControlPlane tests a control plane is not created
// when the request is not a create.
func TestApiV1ClustersNotCreateImplicitControlPlane(t *testing.T) {
	t.Parallel()

	tc, cleanup := MustNewTestContext(t)
	defer cleanup()

	RegisterIdentityHandlers(tc)
	RegisterImageV2Images(tc)
	RegisterComputeV2FlavorsDetail(tc)
	RegisterComputeV2ServerGroups(tc)

	project := mustCreateProjectFixture(t, tc, projectID)
	mustCreateControlPlaneApplicationBundleFixture(t, tc)

	unikornClient := MustNewScopedClient(t, tc)

	response, err := unikornClient.GetApiV1ControlplanesControlPlaneNameClustersClusterNameWithResponse(context.TODO(), "foo", "foo")
	assert.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, response.HTTPResponse.StatusCode)
	assert.NotNil(t, response.JSON404)

	serverErr := *response.JSON404

	assert.Equal(t, generated.NotFound, serverErr.Error)

	var resource unikornv1.ControlPlane

	assert.Error(t, tc.KubernetesClient().Get(context.TODO(), client.ObjectKey{Namespace: project.Status.Namespace, Name: "foo"}, &resource))
}

// TestApiV1ClustersGet tests a cluster can be referenced by name.
func TestApiV1ClustersGet(t *testing.T) {
	t.Parallel()

	tc, cleanup := MustNewTestContext(t)
	defer cleanup()

	RegisterIdentityHandlers(tc)
	RegisterImageV2Images(tc)
	RegisterComputeV2FlavorsDetail(tc)
	RegisterComputeV2ServerGroups(tc)

	project := mustCreateProjectFixture(t, tc, projectID)
	controlPlane := mustCreateControlPlaneFixture(t, tc, project.Status.Namespace, "foo")
	mustCreateKubernetesClusterFixture(t, tc, controlPlane.Status.Namespace, "foo")
	mustCreateKubernetesClusterApplicationBundleFixture(t, tc)

	unikornClient := MustNewScopedClient(t, tc)

	response, err := unikornClient.GetApiV1ControlplanesControlPlaneNameClustersClusterNameWithResponse(context.TODO(), controlPlane.Name, "foo")
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, response.HTTPResponse.StatusCode)
	assert.NotNil(t, response.JSON200)

	result := *response.JSON200

	assert.Equal(t, "foo", result.Name)
	assert.NotNil(t, result.Status)
	assert.Equal(t, "Provisioned", result.Status.Status)
	assert.Equal(t, kubernetesClusterApplicationBundleName, result.ApplicationBundle.Name)
	assert.Equal(t, kubernetesClusterApplicationBundleVersion, result.ApplicationBundle.Version)
	assert.Equal(t, clusterComputeFailureDomain, result.Openstack.ComputeAvailabilityZone)
	assert.Equal(t, clusterStorageFailureDomain, result.Openstack.VolumeAvailabilityZone)
	assert.Equal(t, clusterExternalNetworkID, result.Openstack.ExternalNetworkID)
	assert.NotNil(t, result.Openstack.SshKeyName)
	assert.Equal(t, clusterSSHKeyName, *result.Openstack.SshKeyName)
	assert.Equal(t, clusterNodeNetwork, result.Network.NodePrefix)
	assert.Equal(t, clusterServiceNetwork, result.Network.ServicePrefix)
	assert.Equal(t, clusterPodNetwork, result.Network.PodPrefix)
	assert.Len(t, result.Network.DnsNameservers, 1)
	assert.Equal(t, clusterDNSNameserver, result.Network.DnsNameservers[0])
	assert.Equal(t, "v"+imageK8sVersion, result.ControlPlane.Version)
	assert.Equal(t, imageName, result.ControlPlane.ImageName)
	assert.Equal(t, flavorName, result.ControlPlane.FlavorName)
	assert.Equal(t, clusterControlPlaneReplicas, result.ControlPlane.Replicas)
	assert.Len(t, result.WorkloadPools, 1)
	assert.Equal(t, clusterWorkloadPoolName, result.WorkloadPools[0].Name)
	assert.Equal(t, "v"+imageK8sVersion, result.WorkloadPools[0].Machine.Version)
	assert.Equal(t, imageName, result.WorkloadPools[0].Machine.ImageName)
	assert.Equal(t, flavorName, result.WorkloadPools[0].Machine.FlavorName)
	assert.Equal(t, clusterWorkloadPoolReplicas, result.WorkloadPools[0].Machine.Replicas)
}

// TestApiV1ClustersGetNotFound tests a request for a non-existent cluster returns the
// correct error.
func TestApiV1ClustersGetNotFound(t *testing.T) {
	t.Parallel()

	tc, cleanup := MustNewTestContext(t)
	defer cleanup()

	RegisterIdentityHandlers(tc)

	project := mustCreateProjectFixture(t, tc, projectID)
	controlPlane := mustCreateControlPlaneFixture(t, tc, project.Status.Namespace, "foo")

	unikornClient := MustNewScopedClient(t, tc)

	response, err := unikornClient.GetApiV1ControlplanesControlPlaneNameClustersClusterNameWithResponse(context.TODO(), controlPlane.Name, "foo")
	assert.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, response.HTTPResponse.StatusCode)
	assert.NotNil(t, response.JSON404)

	result := *response.JSON404

	assert.Equal(t, generated.NotFound, result.Error)
}

// TestApiV1ClustersList tests clusters can be listed.
func TestApiV1ClustersList(t *testing.T) {
	t.Parallel()

	tc, cleanup := MustNewTestContext(t)
	defer cleanup()

	RegisterIdentityHandlers(tc)
	RegisterImageV2Images(tc)
	RegisterComputeV2FlavorsDetail(tc)
	RegisterComputeV2ServerGroups(tc)

	project := mustCreateProjectFixture(t, tc, projectID)
	controlPlane := mustCreateControlPlaneFixture(t, tc, project.Status.Namespace, "foo")
	mustCreateKubernetesClusterFixture(t, tc, controlPlane.Status.Namespace, "foo")
	mustCreateKubernetesClusterApplicationBundleFixture(t, tc)

	unikornClient := MustNewScopedClient(t, tc)

	response, err := unikornClient.GetApiV1ControlplanesControlPlaneNameClustersWithResponse(context.TODO(), controlPlane.Name)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, response.HTTPResponse.StatusCode)
	assert.NotNil(t, response.JSON200)

	results := *response.JSON200

	assert.Len(t, results, 1)
	assert.Equal(t, "foo", results[0].Name)
	assert.NotNil(t, results[0].Status)
	assert.Equal(t, "Provisioned", results[0].Status.Status)
	assert.Equal(t, kubernetesClusterApplicationBundleName, results[0].ApplicationBundle.Name)
	assert.Equal(t, kubernetesClusterApplicationBundleVersion, results[0].ApplicationBundle.Version)
	assert.Equal(t, clusterComputeFailureDomain, results[0].Openstack.ComputeAvailabilityZone)
	assert.Equal(t, clusterStorageFailureDomain, results[0].Openstack.VolumeAvailabilityZone)
	assert.Equal(t, clusterExternalNetworkID, results[0].Openstack.ExternalNetworkID)
	assert.NotNil(t, results[0].Openstack.SshKeyName)
	assert.Equal(t, clusterSSHKeyName, *results[0].Openstack.SshKeyName)
	assert.Equal(t, clusterNodeNetwork, results[0].Network.NodePrefix)
	assert.Equal(t, clusterServiceNetwork, results[0].Network.ServicePrefix)
	assert.Equal(t, clusterPodNetwork, results[0].Network.PodPrefix)
	assert.Len(t, results[0].Network.DnsNameservers, 1)
	assert.Equal(t, clusterDNSNameserver, results[0].Network.DnsNameservers[0])
	assert.Equal(t, "v"+imageK8sVersion, results[0].ControlPlane.Version)
	assert.Equal(t, imageName, results[0].ControlPlane.ImageName)
	assert.Equal(t, flavorName, results[0].ControlPlane.FlavorName)
	assert.Equal(t, clusterControlPlaneReplicas, results[0].ControlPlane.Replicas)
	assert.Len(t, results[0].WorkloadPools, 1)
	assert.Equal(t, clusterWorkloadPoolName, results[0].WorkloadPools[0].Name)
	assert.Equal(t, "v"+imageK8sVersion, results[0].WorkloadPools[0].Machine.Version)
	assert.Equal(t, imageName, results[0].WorkloadPools[0].Machine.ImageName)
	assert.Equal(t, flavorName, results[0].WorkloadPools[0].Machine.FlavorName)
	assert.Equal(t, clusterWorkloadPoolReplicas, results[0].WorkloadPools[0].Machine.Replicas)
}

// TestApiV1ClustersUpdate tests clusters can be updated.
func TestApiV1ClustersUpdate(t *testing.T) {
	t.Parallel()

	tc, cleanup := MustNewTestContext(t)
	defer cleanup()

	RegisterIdentityHandlers(tc)
	RegisterImageV2Images(tc)
	RegisterComputeV2FlavorsDetail(tc)
	RegisterComputeV2ServerGroups(tc)

	project := mustCreateProjectFixture(t, tc, projectID)
	controlPlane := mustCreateControlPlaneFixture(t, tc, project.Status.Namespace, "foo")
	mustCreateKubernetesClusterFixture(t, tc, controlPlane.Status.Namespace, "foo")

	unikornClient := MustNewScopedClient(t, tc)

	request := &generated.KubernetesCluster{
		Name: "foo",
		ApplicationBundle: generated.ApplicationBundle{
			Name: "foo",
		},
		Network: generated.KubernetesClusterNetwork{
			DnsNameservers: []string{
				"8.8.8.8",
			},
			NodePrefix:    "192.168.0.0/24",
			ServicePrefix: "172.16.0.0/12",
			PodPrefix:     "10.0.0.0/8",
		},
		ControlPlane: generated.OpenstackMachinePool{
			Version:    "v1.28.0",
			Replicas:   3,
			ImageName:  "ubuntu-24.04-lts",
			FlavorName: flavorName,
		},
		WorkloadPools: generated.KubernetesClusterWorkloadPools{
			{
				Name: "foo",
				Machine: generated.OpenstackMachinePool{
					Version:    "v1.28.0",
					Replicas:   3,
					ImageName:  "ubuntu-24.04-lts",
					FlavorName: flavorName,
				},
			},
		},
	}

	response, err := unikornClient.PutApiV1ControlplanesControlPlaneNameClustersClusterNameWithBody(context.TODO(), controlPlane.Name, "foo", "application/json", NewJSONReader(request))
	assert.NoError(t, err)
	assert.Equal(t, http.StatusAccepted, response.StatusCode)

	defer response.Body.Close()

	var resource unikornv1.KubernetesCluster

	assert.NoError(t, tc.KubernetesClient().Get(context.TODO(), client.ObjectKey{Namespace: controlPlane.Status.Namespace, Name: "foo"}, &resource))
	assert.Equal(t, *resource.Spec.ApplicationBundle, "foo")
}

// TestApiV1ClustersUpdateNotFound tests clusters return the correct error if they don't
// exist on update.
func TestApiV1ClustersUpdateNotFound(t *testing.T) {
	t.Parallel()

	tc, cleanup := MustNewTestContext(t)
	defer cleanup()

	RegisterIdentityHandlers(tc)
	RegisterImageV2Images(tc)
	RegisterComputeV2FlavorsDetail(tc)
	RegisterComputeV2ServerGroups(tc)

	project := mustCreateProjectFixture(t, tc, projectID)
	controlPlane := mustCreateControlPlaneFixture(t, tc, project.Status.Namespace, "foo")

	unikornClient := MustNewScopedClient(t, tc)

	request := &generated.KubernetesCluster{
		Name: "foo",
		ApplicationBundle: generated.ApplicationBundle{
			Name: "foo",
		},
		Network: generated.KubernetesClusterNetwork{
			DnsNameservers: []string{
				"8.8.8.8",
			},
			NodePrefix:    "192.168.0.0/24",
			ServicePrefix: "172.16.0.0/12",
			PodPrefix:     "10.0.0.0/8",
		},
		ControlPlane: generated.OpenstackMachinePool{
			Version:    "v1.28.0",
			Replicas:   3,
			ImageName:  "ubuntu-24.04-lts",
			FlavorName: flavorName,
		},
		WorkloadPools: generated.KubernetesClusterWorkloadPools{
			{
				Name: "foo",
				Machine: generated.OpenstackMachinePool{
					Version:    "v1.28.0",
					Replicas:   3,
					ImageName:  "ubuntu-24.04-lts",
					FlavorName: flavorName,
				},
			},
		},
	}

	response, err := unikornClient.PutApiV1ControlplanesControlPlaneNameClustersClusterNameWithBodyWithResponse(context.TODO(), controlPlane.Name, "foo", "application/json", NewJSONReader(request))
	assert.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, response.HTTPResponse.StatusCode)
	assert.NotNil(t, response.JSON404)

	serverErr := *response.JSON404

	assert.Equal(t, serverErr.Error, generated.NotFound)
}

// TestApiV1ClustersDelete tests clusters can be deleted.
func TestApiV1ClustersDelete(t *testing.T) {
	t.Parallel()

	tc, cleanup := MustNewTestContext(t)
	defer cleanup()

	RegisterIdentityHandlers(tc)

	project := mustCreateProjectFixture(t, tc, projectID)
	controlPlane := mustCreateControlPlaneFixture(t, tc, project.Status.Namespace, "foo")
	mustCreateKubernetesClusterFixture(t, tc, controlPlane.Status.Namespace, "foo")
	mustCreateKubernetesClusterApplicationBundleFixture(t, tc)

	unikornClient := MustNewScopedClient(t, tc)

	response, err := unikornClient.DeleteApiV1ControlplanesControlPlaneNameClustersClusterName(context.TODO(), controlPlane.Name, "foo")
	assert.NoError(t, err)
	assert.Equal(t, http.StatusAccepted, response.StatusCode)

	defer response.Body.Close()

	var resource unikornv1.KubernetesCluster

	var statusErr *kerrors.StatusError

	assert.ErrorAs(t, tc.KubernetesClient().Get(context.TODO(), client.ObjectKey{Namespace: controlPlane.Status.Namespace, Name: "foo"}, &resource), &statusErr)
	assert.Equal(t, http.StatusNotFound, int(statusErr.ErrStatus.Code))
}

// TestApiV1ClustersDeleteNotFound tests that the deletion of a non-existent cluster
// results in the correct error.
func TestApiV1ClustersDeleteNotFound(t *testing.T) {
	t.Parallel()

	tc, cleanup := MustNewTestContext(t)
	defer cleanup()

	RegisterIdentityHandlers(tc)

	project := mustCreateProjectFixture(t, tc, projectID)
	controlPlane := mustCreateControlPlaneFixture(t, tc, project.Status.Namespace, "foo")

	unikornClient := MustNewScopedClient(t, tc)

	response, err := unikornClient.DeleteApiV1ControlplanesControlPlaneNameClustersClusterNameWithResponse(context.TODO(), controlPlane.Name, "foo")
	assert.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, response.HTTPResponse.StatusCode)
	assert.NotNil(t, response.JSON404)

	serverErr := *response.JSON404

	assert.Equal(t, serverErr.Error, generated.NotFound)
}

// TestApiV1ApplicationBundlesListControlPlane tests control plane application bundles can
// be listed.
func TestApiV1ApplicationBundlesListControlPlane(t *testing.T) {
	t.Parallel()

	tc, cleanup := MustNewTestContext(t)
	defer cleanup()

	RegisterIdentityHandlers(tc)

	mustCreateControlPlaneApplicationBundleFixture(t, tc)

	unikornClient := MustNewScopedClient(t, tc)

	response, err := unikornClient.GetApiV1ApplicationbundlesControlPlaneWithResponse(context.TODO())
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, response.HTTPResponse.StatusCode)
	assert.NotNil(t, response.JSON200)

	results := *response.JSON200

	assert.Len(t, results, 1)
	assert.Equal(t, controlPlaneApplicationBundleName, results[0].Name)
	assert.Equal(t, controlPlaneApplicationBundleVersion, results[0].Version)
}

// TestApiV1ApplicationBundlesListCluster tests cluster application bundles can be listed.
func TestApiV1ApplicationBundlesListCluster(t *testing.T) {
	t.Parallel()

	tc, cleanup := MustNewTestContext(t)
	defer cleanup()

	RegisterIdentityHandlers(tc)

	mustCreateKubernetesClusterApplicationBundleFixture(t, tc)

	unikornClient := MustNewScopedClient(t, tc)

	response, err := unikornClient.GetApiV1ApplicationbundlesClusterWithResponse(context.TODO())
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, response.HTTPResponse.StatusCode)
	assert.NotNil(t, response.JSON200)

	results := *response.JSON200

	assert.Len(t, results, 1)
	assert.Equal(t, kubernetesClusterApplicationBundleName, results[0].Name)
	assert.Equal(t, kubernetesClusterApplicationBundleVersion, results[0].Version)
}

// TestApiV1ApplicationsList tests applications can be listed.
func TestApiV1ApplicationsList(t *testing.T) {
	t.Parallel()

	tc, cleanup := MustNewTestContext(t)
	defer cleanup()

	RegisterIdentityHandlers(tc)

	mustCreateHelmApplicationFixture(t, tc)

	unikornClient := MustNewScopedClient(t, tc)

	response, err := unikornClient.GetApiV1ApplicationsWithResponse(context.TODO())
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, response.HTTPResponse.StatusCode)
	assert.NotNil(t, response.JSON200)

	results := *response.JSON200

	assert.Len(t, results, 1)
	assert.Equal(t, applicationName, results[0].Name)
	assert.Equal(t, applicationHumanReadableName, results[0].HumanReadableName)
	assert.Equal(t, applicationDescription, results[0].Description)
	assert.Equal(t, applicationDocumentation, results[0].Documentation)
	assert.Equal(t, applicationLicense, results[0].License)
	assert.Equal(t, []byte(applicationIcon), results[0].Icon)
	assert.Len(t, results[0].Versions, 1)
	assert.Equal(t, applicationVersion, results[0].Versions[0].Version)
}

// TestApiV1ProvidersOpenstackProjects tests OpenStack projects can be listed.
func TestApiV1ProvidersOpenstackProjects(t *testing.T) {
	t.Parallel()

	tc, cleanup := MustNewTestContext(t)
	defer cleanup()

	RegisterIdentityHandlers(tc)

	unikornClient := MustNewScopedClient(t, tc)

	response, err := unikornClient.GetApiV1ProvidersOpenstackProjectsWithResponse(context.TODO())
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, response.HTTPResponse.StatusCode)
	assert.NotNil(t, response.JSON200)

	results := *response.JSON200

	assert.Len(t, results, 1)
	assert.Equal(t, projectID, results[0].Id)
	assert.Equal(t, projectName, results[0].Name)
}

// TestApiV1ProvidersOpenstackProjectsUnauthorized tests an unauthorized response
// from a request is propagated to the client correctly.
func TestApiV1ProvidersOpenstackProjectsUnauthorized(t *testing.T) {
	t.Parallel()

	tc, cleanup := MustNewTestContext(t)
	defer cleanup()

	RegisterIdentityHandlers(tc)

	unikornClient := MustNewScopedClient(t, tc)

	// Override...
	RegisterIdentityV3AuthProjectsUnauthorized(tc)

	response, err := unikornClient.GetApiV1ProvidersOpenstackProjectsWithResponse(context.TODO())
	assert.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, response.HTTPResponse.StatusCode)
	assert.NotNil(t, response.JSON401)

	serverErr := *response.JSON401

	assert.Equal(t, generated.AccessDenied, serverErr.Error)
}

// TestApiV1ProvidersOpenstackFlavors tests OpenStack flavors can be listed.
func TestApiV1ProvidersOpenstackFlavors(t *testing.T) {
	t.Parallel()

	tc, cleanup := MustNewTestContext(t)
	defer cleanup()

	RegisterIdentityHandlers(tc)
	RegisterComputeV2FlavorsDetail(tc)

	unikornClient := MustNewScopedClient(t, tc)

	response, err := unikornClient.GetApiV1ProvidersOpenstackFlavorsWithResponse(context.TODO())
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, response.HTTPResponse.StatusCode)
	assert.NotNil(t, response.JSON200)

	results := *response.JSON200

	// NOTE: server converts from MiB to GiB of memory.
	assert.Len(t, results, 3)
	assert.Equal(t, flavorID, results[0].Id)
	assert.Equal(t, flavorName, results[0].Name)
	assert.Equal(t, flavorCpus, results[0].Cpus)
	assert.Equal(t, flavorMemory>>10, results[0].Memory)
	assert.Equal(t, flavorDisk, results[0].Disk)
	assert.Equal(t, flavorName2, results[1].Name)
	assert.Equal(t, flavorName3, results[2].Name)
}

// TestApiV1ProvidersOpenstackFlavorsUnauthorized tests an unauthorized response
// from a request is propagated to the client correctly.
func TestApiV1ProvidersOpenstackFlavorsUnauthorized(t *testing.T) {
	t.Parallel()

	tc, cleanup := MustNewTestContext(t)
	defer cleanup()

	RegisterIdentityHandlers(tc)
	RegisterComputeV2FlavorsDetailUnauthorized(tc)

	unikornClient := MustNewScopedClient(t, tc)

	response, err := unikornClient.GetApiV1ProvidersOpenstackFlavorsWithResponse(context.TODO())
	assert.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, response.HTTPResponse.StatusCode)
	assert.NotNil(t, response.JSON401)

	serverErr := *response.JSON401

	assert.Equal(t, generated.AccessDenied, serverErr.Error)
}

// TestApiV1ProvidersOpenstackImages tests OpenStack images can be listed.
func TestApiV1ProvidersOpenstackImages(t *testing.T) {
	t.Parallel()

	tc, cleanup := MustNewTestContext(t)
	defer cleanup()

	RegisterIdentityHandlers(tc)
	RegisterImageV2Images(tc)

	unikornClient := MustNewScopedClient(t, tc)

	response, err := unikornClient.GetApiV1ProvidersOpenstackImagesWithResponse(context.TODO())
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, response.HTTPResponse.StatusCode)
	assert.NotNil(t, response.JSON200)

	results := *response.JSON200

	ts, err := time.Parse(time.RFC3339, imageTimestamp)
	assert.NoError(t, err)

	// NOTE: server converts kubernetes version to a proper semver so it's compatible
	// with the CAPI kubeadm controller.
	assert.Len(t, results, 2)
	assert.Equal(t, imageID, results[0].Id)
	assert.Equal(t, imageName, results[0].Name)
	assert.Equal(t, ts, results[0].Created)
	assert.Equal(t, ts, results[0].Modified)
	assert.Equal(t, "v"+imageK8sVersion, results[0].Versions.Kubernetes)
	assert.Equal(t, imageGpuVersion, results[0].Versions.NvidiaDriver)
}

// TestApiV1ProvidersOpenstackImagesUnauthorized tests an unauthorized response
// from a request is propagated to the client correctly.
func TestApiV1ProvidersOpenstackImagesUnauthorized(t *testing.T) {
	t.Parallel()

	tc, cleanup := MustNewTestContext(t)
	defer cleanup()

	RegisterIdentityHandlers(tc)
	RegisterImageV2ImagesUnauthorized(tc)

	unikornClient := MustNewScopedClient(t, tc)

	response, err := unikornClient.GetApiV1ProvidersOpenstackImagesWithResponse(context.TODO())
	assert.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, response.HTTPResponse.StatusCode)
	assert.NotNil(t, response.JSON401)

	serverErr := *response.JSON401

	assert.Equal(t, generated.AccessDenied, serverErr.Error)
}

// TestApiV1ProvidersOpenstackAvailabilityZonesCompute tests OpenStack compute AZscan be listed.
func TestApiV1ProvidersOpenstackAvailabilityZonesCompute(t *testing.T) {
	t.Parallel()

	tc, cleanup := MustNewTestContext(t)
	defer cleanup()

	RegisterIdentityHandlers(tc)
	RegisterComputeV2AvailabilityZone(tc)

	unikornClient := MustNewScopedClient(t, tc)

	response, err := unikornClient.GetApiV1ProvidersOpenstackAvailabilityZonesComputeWithResponse(context.TODO())
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, response.HTTPResponse.StatusCode)
	assert.NotNil(t, response.JSON200)

	results := *response.JSON200

	assert.Len(t, results, 1)
	assert.Equal(t, computeAvailabilityZoneName, results[0].Name)
}

// TestApiV1ProvidersOpenstackAvailabilityZonesComputeUnauthorized tests an unauthorized response
// from a request is propagated to the client correctly.
func TestApiV1ProvidersOpenstackAvailabilityZonesComputeUnauthorized(t *testing.T) {
	t.Parallel()

	tc, cleanup := MustNewTestContext(t)
	defer cleanup()

	RegisterIdentityHandlers(tc)
	RegisterComputeV2AvailabilityZoneUnauthorized(tc)

	unikornClient := MustNewScopedClient(t, tc)

	response, err := unikornClient.GetApiV1ProvidersOpenstackAvailabilityZonesComputeWithResponse(context.TODO())
	assert.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, response.HTTPResponse.StatusCode)
	assert.NotNil(t, response.JSON401)

	serverErr := *response.JSON401

	assert.Equal(t, generated.AccessDenied, serverErr.Error)
}

// TestApiV1ProvidersOpenstackAvailabilityZonesBlockStorage tests OpenStack block storage AZscan be listed.
func TestApiV1ProvidersOpenstackAvailabilityZonesBlockStorage(t *testing.T) {
	t.Parallel()

	tc, cleanup := MustNewTestContext(t)
	defer cleanup()

	RegisterIdentityHandlers(tc)
	RegisterBlockStorageV3AvailabilityZone(tc)

	unikornClient := MustNewScopedClient(t, tc)

	response, err := unikornClient.GetApiV1ProvidersOpenstackAvailabilityZonesBlockStorageWithResponse(context.TODO())
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, response.HTTPResponse.StatusCode)
	assert.NotNil(t, response.JSON200)

	results := *response.JSON200

	assert.Len(t, results, 1)
	assert.Equal(t, blockStorageAvailabilityZone, results[0].Name)
}

// TestApiV1ProvidersOpenstackAvailabilityZonesBlockStorageUnauthorized tests an unauthorized response
// from a request is propagated to the client correctly.
func TestApiV1ProvidersOpenstackAvailabilityZonesBlockStorageUnauthorized(t *testing.T) {
	t.Parallel()

	tc, cleanup := MustNewTestContext(t)
	defer cleanup()

	RegisterIdentityHandlers(tc)
	RegisterBlockStorageV3AvailabilityZoneUnauthorized(tc)

	unikornClient := MustNewScopedClient(t, tc)

	response, err := unikornClient.GetApiV1ProvidersOpenstackAvailabilityZonesBlockStorageWithResponse(context.TODO())
	assert.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, response.HTTPResponse.StatusCode)
	assert.NotNil(t, response.JSON401)

	serverErr := *response.JSON401

	assert.Equal(t, generated.AccessDenied, serverErr.Error)
}

// TestApiV1ProvidersOpenstackExternalNetworks tests OpenStack external networks can be listed.
func TestApiV1ProvidersOpenstackExternalNetworks(t *testing.T) {
	t.Parallel()

	tc, cleanup := MustNewTestContext(t)
	defer cleanup()

	RegisterIdentityHandlers(tc)
	RegisterNetworkV2Networks(tc)

	unikornClient := MustNewScopedClient(t, tc)

	response, err := unikornClient.GetApiV1ProvidersOpenstackExternalNetworksWithResponse(context.TODO())
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, response.HTTPResponse.StatusCode)
	assert.NotNil(t, response.JSON200)

	results := *response.JSON200

	assert.Len(t, results, 1)
	assert.Equal(t, externalNetworkID, results[0].Id)
}

// TestApiV1ProvidersOpenstackExternalNetworksUnauthorized tests an unauthorized response
// from a request is propagated to the client correctly.
func TestApiV1ProvidersOpenstackExternalNetworksUnauthorized(t *testing.T) {
	t.Parallel()

	tc, cleanup := MustNewTestContext(t)
	defer cleanup()

	RegisterIdentityHandlers(tc)
	RegisterNetworkV2NetworksUnauthorized(tc)

	unikornClient := MustNewScopedClient(t, tc)

	response, err := unikornClient.GetApiV1ProvidersOpenstackExternalNetworksWithResponse(context.TODO())
	assert.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, response.HTTPResponse.StatusCode)
	assert.NotNil(t, response.JSON401)

	serverErr := *response.JSON401

	assert.Equal(t, generated.AccessDenied, serverErr.Error)
}

// TestApiV1ProvidersOpenstackKeyPairs tests OpenStack key pairs can be listed.
func TestApiV1ProvidersOpenstackKeyPairs(t *testing.T) {
	t.Parallel()

	tc, cleanup := MustNewTestContext(t)
	defer cleanup()

	RegisterIdentityHandlers(tc)
	RegisterComputeV2Keypairs(tc)

	unikornClient := MustNewScopedClient(t, tc)

	response, err := unikornClient.GetApiV1ProvidersOpenstackKeyPairsWithResponse(context.TODO())
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, response.HTTPResponse.StatusCode)
	assert.NotNil(t, response.JSON200)

	results := *response.JSON200

	assert.Len(t, results, 1)
	assert.Equal(t, keyPairName, results[0].Name)
}

// TestApiV1ProvidersOpenstackKeyPairsUnauthorized tests an unauthorized response
// from a request is propagated to the client correctly.
func TestApiV1ProvidersOpenstackKeyPairsUnauthorized(t *testing.T) {
	t.Parallel()

	tc, cleanup := MustNewTestContext(t)
	defer cleanup()

	RegisterIdentityHandlers(tc)
	RegisterComputeV2KeypairsUnauthorized(tc)

	unikornClient := MustNewScopedClient(t, tc)

	response, err := unikornClient.GetApiV1ProvidersOpenstackKeyPairsWithResponse(context.TODO())
	assert.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, response.HTTPResponse.StatusCode)
	assert.NotNil(t, response.JSON401)

	serverErr := *response.JSON401

	assert.Equal(t, generated.AccessDenied, serverErr.Error)
}
