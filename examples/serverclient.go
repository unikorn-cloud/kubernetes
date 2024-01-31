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

package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"

	"golang.org/x/oauth2"

	"github.com/unikorn-cloud/unikorn/pkg/server/generated"
)

const (
	server           = "https://kubernetes.eschercloud.com"
	tokenEndpoint    = "https://kubernetes.eschercloud.com/api/v1/auth/oauth2/tokens"
	exchangeEndpoint = "https://kubernetes.eschercloud.com/api/v1/auth/tokens/token"
	username         = ""
	password         = ""
	projectID        = ""
)

var (
	ErrAPI = errors.New("api returned unexpected status code")
)

// insecureClient should never be used, this is just for testing against a self-signed
// development instance.
func insecureClient() *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}
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

// bearerTokenInjector is a handy function that augments the clients to add
func bearerTokenInjector(token string) generated.RequestEditorFn {
	return func(ctx context.Context, req *http.Request) error {
		req.Header.Set("Authorization", "Bearer "+token)

		return nil
	}
}

// NewClient is a helper to abstract away client authentication.
func NewClient(accessToken string) (*generated.ClientWithResponses, error) {
	return generated.NewClientWithResponses(server, generated.WithHTTPClient(insecureClient()), generated.WithRequestEditorFn(bearerTokenInjector(accessToken)))
}

// Login via oauth2's password grant flow.  But you should never do this.
// See https://tools.ietf.org/html/rfc6749#section-4.3.
func oauth2Authenticate() (*oauth2.Token, error) {
	client := insecureClient()

	ctx := context.WithValue(context.Background(), oauth2.HTTPClient, client)

	config := &oauth2.Config{
		Endpoint: oauth2.Endpoint{
			TokenURL: tokenEndpoint,
		},
	}

	return config.PasswordCredentialsToken(ctx, username, password)
}

// getScopedToken exchanges a token for one with a new project scope.
func getScopedToken(token *oauth2.Token, projectID string) (*generated.Token, error) {
	client, err := NewClient(token.AccessToken)
	if err != nil {
		return nil, err
	}

	scope := &generated.TokenScope{
		Project: generated.TokenScopeProject{
			Id: projectID,
		},
	}

	response, err := client.PostApiV1AuthTokensTokenWithBodyWithResponse(context.TODO(), "application/json", NewJSONReader(scope))
	if err != nil {
		return nil, err
	}

	if response.StatusCode() != 201 {
		return nil, fmt.Errorf("%w: unable to scope token", ErrAPI)
	}

	return response.JSON201, nil
}

func main() {
	token, err := oauth2Authenticate()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	scopedToken, err := getScopedToken(token, projectID)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	fmt.Println(scopedToken.AccessToken)
}
