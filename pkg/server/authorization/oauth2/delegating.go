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

package oauth2

import (
	"context"
	"crypto/md5" //nolint:gosec
	"crypto/rand"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/spf13/pflag"
	"golang.org/x/oauth2"

	"github.com/spjmurray/unikorn/pkg/server/authorization/jose"
	"github.com/spjmurray/unikorn/pkg/server/authorization/keystone"
	"github.com/spjmurray/unikorn/pkg/server/errors"
	"github.com/spjmurray/unikorn/pkg/server/generated"

	"sigs.k8s.io/controller-runtime/pkg/log"
)

type Options struct {
	// oidcClientID is the client ID for the backend IdP this server
	// is prozying for.
	oidcClientID string

	// oidcIssuer is the expected issuer of OIDC tokens for verification.
	oidcIssuer string

	// oidcAuthorizationEndpoint defines where to get authorization codes
	// from the authentication server.
	oidcAuthorizationEndpoint string

	// oidcTokenEndpoint defines where to exchange authorization codes for
	// authorization tokens with the autorization server.
	oidcTokenEndpoint string

	// oidcJwksURL defines the JWKS endpoint for the authorization server
	// to retrieve signing keys for token validation.
	oidcJwksURL string

	// clientID is the client ID that's expected to be presented by a client
	// during oauth2's authorization flow.
	clientID string

	// redirectURI is the allowed redirect URI for a the client ID.
	redirectURI string
}

// AddFlags to the specified flagset.
func (o *Options) AddFlags(f *pflag.FlagSet) {
	f.StringVar(&o.oidcClientID, "oidc-client-id", "93455590-c733-013b-e155-02ce91db9a85225246", "OIDC client ID.")
	f.StringVar(&o.oidcIssuer, "oidc-issuer", "https://eschercloud-dev.onelogin.com/oidc/2", "Expected OIDC issuer name.")
	f.StringVar(&o.oidcAuthorizationEndpoint, "oidc-autorization-endpoint", "https://eschercloud-dev.onelogin.com/oidc/2/auth", "OIDC authorization endpoint.")
	f.StringVar(&o.oidcTokenEndpoint, "oidc-token-endpoint", "https://eschercloud-dev.onelogin.com/oidc/2/token", "OIDC token endpoint.")
	f.StringVar(&o.oidcJwksURL, "oidc-jwks-url", "https://eschercloud-dev.onelogin.com/oidc/2/certs", "OIDC JWKS endpoint.")
	f.StringVar(&o.clientID, "oauth2-client-id", "9a719e1e-aa85-4a21-a221-324e787efd78", "OAuth2 client ID of server clients.")
	f.StringVar(&o.redirectURI, "oauth2-redirect-uri", "https://kubernetes.eschercloud.com/oauth2/callback", "Exprected redirect URI for the client ID.")
}

// Authenticator provides Keystone authentication functionality.
type Authenticator struct {
	options *Options

	// issuer allows creation and validation of JWT bearer tokens.
	issuer *jose.JWTIssuer

	keystone *keystone.Authenticator
}

// New returns a new authenticator with required fields populated.
// You must call AddFlags after this.
func New(options *Options, issuer *jose.JWTIssuer, keystone *keystone.Authenticator) *Authenticator {
	return &Authenticator{
		options:  options,
		issuer:   issuer,
		keystone: keystone,
	}
}

type Error string

const (
	ErrorInvalidRequest          Error = "invalid_request"
	ErrorUnauthorizedClient      Error = "unauthorized_client"
	ErrorAccessDenied            Error = "access_denied"
	ErrorUnsupportedResponseType Error = "unsupported_response_type"
	ErrorInvalidScope            Error = "invalid_scope"
	ErrorServerError             Error = "server_error"
)

// Scope wraps up scope functionality.
type Scope []string

// NewScope takes a raw scope from a query and return a canonical scope type.
func NewScope(s string) Scope {
	return Scope(strings.Split(s, " "))
}

// Has returns true if a scope exists.
func (s Scope) Has(scope string) bool {
	for _, value := range s {
		if value == scope {
			return true
		}
	}

	return false
}

// IDToken defines an OIDC id_token.
//
//nolint:tagliatelle
type IDToken struct {
	// These are default claims you always get.
	Issuer   string   `json:"iss"`
	Subject  string   `json:"sub"`
	Audience []string `json:"aud"`
	Expiry   int64    `json:"exp"`
	IssuedAt int64    `json:"iat"`
	Nonce    string   `json:"nonce,omitempty"`
	ATHash   string   `json:"at_hash,omitempty"`
	// Optional claims that may be asked for by the "email" scope.
	Email string `json:"email,omitempty"`
	// Optional claims that may be asked for by the "profile" scope.
	Picture string `json:"picture,omitempty"`
}

// State records state across the call to the authorization server.
// This must be encrypted with JWE.
type State struct {
	// Nonce is the one time nonce used to create the token.
	Nonce string `json:"n"`
	// Code verfier is required to prove our identity when
	// exchanging the code with the token endpoint.
	CodeVerfier string `json:"cv"`
	// ClientID is the client identifier.
	ClientID string `json:"cid"`
	// ClientRedirectURI is the redirect URL requested by the client.
	ClientRedirectURI string `json:"cri"`
	// Client state records the client's OAuth state while we interact
	// with the OIDC authorization server.
	ClientState string `json:"cst,omitempty"`
	// ClientCodeChallenge records the client code challenge so we can
	// authenticate we are handing the authorization token back to the
	// correct client.
	ClientCodeChallenge string `json:"ccc"`
	// ClientScope records the requested client scope.
	ClientScope Scope `json:"csc,omitempty"`
	// ClientNonce is injected into a OIDC id_token.
	ClientNonce string `json:"cno,omitempty"`
}

// Code is an authorization code to return to the client that can be
// exchanged for an access token.  Much like how we client things in the oauth2
// state during the OIDC exchange, to mitigate problems with horizonal scaling
// and sharing stuff, we do the same here.
// WARNING: Don't make this too big, the ingress controller will barf if the
// headers are too hefty.
type Code struct {
	// ClientID is the client identifier.
	ClientID string `json:"cid"`
	// ClientRedirectURI is the redirect URL requested by the client.
	ClientRedirectURI string `json:"cri"`
	// ClientCodeChallenge records the client code challenge so we can
	// authenticate we are handing the authorization token back to the
	// correct client.
	ClientCodeChallenge string `json:"ccc"`
	// ClientScope records the requested client scope.
	ClientScope Scope `json:"csc,omitempty"`
	// ClientNonce is injected into a OIDC id_token.
	ClientNonce string `json:"cno,omitempty"`
	// KeystoneToken is the exchanged keystone token.
	KeystoneToken string `json:"kst"`
	// KeystoneUserID is exactly that.
	KeystoneUserID string `json:"kui"`
	// Email is exactly that.
	Email string `json:"email"`
	// Expiry is when the token expires.
	Expiry time.Time `json:"exp"`
}

const (
	// errorTemplate is used to return a verbose error to the client when
	// something is very wrong and cannot be redirected.
	errorTemplate = "<html><body><h1>Oops! Something went wrong.</h1><p><pre>%s</pre></p></body></html>"
)

// htmlError is used in dire situations when we cannot return an error via
// the usual oauth2 flow.
func htmlError(w http.ResponseWriter, r *http.Request, status int, description string) {
	log := log.FromContext(r.Context())

	w.WriteHeader(status)

	if _, err := w.Write([]byte(fmt.Sprintf(errorTemplate, description))); err != nil {
		log.Info("oauth2: failed to write HTML response")
	}
}

// authorizationError redirects to the client's callback URI with an error
// code in the query.
func authorizationError(w http.ResponseWriter, r *http.Request, redirectURI string, kind Error, description string) {
	values := &url.Values{}
	values.Set("error", string(kind))
	values.Set("description", description)

	http.Redirect(w, r, redirectURI+"?"+values.Encode(), http.StatusFound)
}

// OAuth2AuthorizationValidateNonRedirecting checks authorization request parameters
// are valid that directly control the ability to redirect, and returns some helpful
// debug in HTML.
func (a *Authenticator) authorizationValidateNonRedirecting(w http.ResponseWriter, r *http.Request) bool {
	query := r.URL.Query()

	var description string

	switch {
	case !query.Has("client_id"):
		description = "client_id is not specified"
	case query.Get("client_id") != a.options.clientID:
		description = "client_id is invalid"
	case !query.Has("redirect_uri"):
		description = "redirect_uri is not specified"
	case query.Get("redirect_uri") != a.options.redirectURI:
		description = "redirect_uri is invalid"
	default:
		return true
	}

	htmlError(w, r, http.StatusBadRequest, description)

	return false
}

// OAuth2AuthorizationValidateRedirecting checks autohorization request parameters after
// the redirect URI has been validated.  If any of these fail, we redirect but with an
// error query rather than a code for the client to pick up and run with.
func (a *Authenticator) authorizationValidateRedirecting(w http.ResponseWriter, r *http.Request) bool {
	query := r.URL.Query()

	var kind Error

	var description string

	switch {
	case query.Get("response_type") != "code":
		kind = ErrorUnsupportedResponseType
		description = "response_type must be 'code'"
	case query.Get("code_challenge_method") != "S256":
		kind = ErrorInvalidRequest
		description = "code_challenge_method must be 'S256'"
	case query.Get("code_challenge") == "":
		kind = ErrorInvalidRequest
		description = "code_challenge must be specified"
	default:
		return true
	}

	authorizationError(w, r, query.Get("redirect_uri"), kind, description)

	return false
}

// oidcConfig returns a oauth2 configuration for the OIDC backend.
func (a *Authenticator) oidcConfig(r *http.Request) *oauth2.Config {
	return &oauth2.Config{
		ClientID: a.options.oidcClientID,
		Endpoint: oauth2.Endpoint{
			AuthURL:  a.options.oidcAuthorizationEndpoint,
			TokenURL: a.options.oidcTokenEndpoint,
		},
		// TODO: the ingress converts this all into a relative URL
		// and adds an X-Forwardered-Host, X-Forwarded-Proto.  You should
		// never use HTTP anyway to be fair...
		RedirectURL: "https://" + r.Host + "/api/v1/auth/oidc/callback",
		Scopes: []string{
			oidc.ScopeOpenID,
			"profile",
			"email",
			"groups",
		},
	}
}

// encodeCodeChallengeS256 performs code verifier to code challenge translation
// for the SHA256 method.
func encodeCodeChallengeS256(codeVerifier string) string {
	hash := sha256.Sum256([]byte(codeVerifier))

	return base64.RawURLEncoding.EncodeToString(hash[:])
}

// randomString creates size bytes of high entropy randomness and base64 URL
// encodes it into a string.  Bear in mind base64 expands the size by 33%, so for example
// an oauth2 code verifier needs to be at least 43 bytes, so youd nee'd a size of 32,
// 32 * 1.33 = 42.66.
func randomString(size int) (string, error) {
	buf := make([]byte, size)

	if _, err := rand.Read(buf); err != nil {
		return "", err
	}

	return base64.RawURLEncoding.EncodeToString(buf), nil
}

// Authorization redirects the client to the OIDC autorization endpoint
// to get an authorization code.  Note that this function is responsible for
// either returning an authorization grant or error via a HTTP 302 redirect,
// or returning a HTML fragment for errors that cannot follow the provided
// redirect URI.
func (a *Authenticator) Authorization(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()

	if !a.authorizationValidateNonRedirecting(w, r) {
		return
	}

	if !a.authorizationValidateRedirecting(w, r) {
		return
	}

	clientRedirectURI := query.Get("redirect_uri")

	// OIDC requires a nonce, just some random data base64 URL encoded will suffice.
	nonce, err := randomString(16)
	if err != nil {
		authorizationError(w, r, clientRedirectURI, ErrorServerError, "unable to create oidc nonce: "+err.Error())
		return
	}

	// We pass a hashed code challenge to the OIDC authorization endpoint when
	// requesting an authentication code.  When we exchange that for a token we
	// send the initial code challenge verifier so the token endpoint can validate
	// it's talking to the same client.
	codeVerifier, err := randomString(32)
	if err != nil {
		authorizationError(w, r, clientRedirectURI, ErrorServerError, "unable to create oauth2 code verifier: "+err.Error())
		return
	}

	codeChallenge := encodeCodeChallengeS256(codeVerifier)

	// Rather than cache any state we require after the oauth rediretion dance, which
	// requires persistent state at the minimum, and a database in the case of multi-head
	// deployments, just encrypt it and send with the authoriation request.
	oidcState := &State{
		Nonce:               nonce,
		CodeVerfier:         codeVerifier,
		ClientID:            query.Get("client_id"),
		ClientRedirectURI:   query.Get("redirect_uri"),
		ClientState:         query.Get("state"),
		ClientCodeChallenge: query.Get("code_challenge"),
	}

	// To implement OIDC we need a copy of the scopes.
	if query.Has("scope") {
		oidcState.ClientScope = NewScope(query.Get("scope"))
	}

	if query.Has("nonce") {
		oidcState.ClientNonce = query.Get("nonce")
	}

	state, err := a.issuer.EncodeJWEToken(oidcState)
	if err != nil {
		authorizationError(w, r, clientRedirectURI, ErrorServerError, "failed to encode oidc state: "+err.Error())
		return
	}

	// Finally generate the redirection URL and send back to the client.
	authURLParams := []oauth2.AuthCodeOption{
		oauth2.SetAuthURLParam("code_challenge_method", "S256"),
		oauth2.SetAuthURLParam("code_challenge", codeChallenge),
		oidc.Nonce(nonce),
	}

	http.Redirect(w, r, a.oidcConfig(r).AuthCodeURL(state, authURLParams...), http.StatusFound)
}

// oidcExtractIDToken wraps up token verification against the JWKS service and conversion
// to a concrete type.
func (a *Authenticator) oidcExtractIDToken(ctx context.Context, token string) (*oidc.IDToken, error) {
	// Verify the ID token, and then extract information required by the client
	// e.g. email addresses etc.
	oidcConfig := &oidc.Config{
		ClientID: a.options.oidcClientID,
	}

	remoteKeySet := oidc.NewRemoteKeySet(ctx, a.options.oidcJwksURL)
	idTokenVerifier := oidc.NewVerifier(a.options.oidcIssuer, remoteKeySet, oidcConfig)

	idToken, err := idTokenVerifier.Verify(ctx, token)
	if err != nil {
		return nil, err
	}

	return idToken, nil
}

// OIDCCallback is called by the authorization endpoint in order to return an
// authorization back to us.  We then exchange the code for an ID token, and
// refresh token.  Remember, as far as the client is concerned we're still doing
// the code grant, so return errors in the redirect query.
//
//nolint:cyclop
func (a *Authenticator) OIDCCallback(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()

	// This should always be present, if not then we are boned and cannot
	// send an error back to the redirectURI, cos that's in the state!
	if !query.Has("state") {
		htmlError(w, r, http.StatusBadRequest, "oidc state is required")
		return
	}

	// Extract our state for the next part...
	state := &State{}

	if err := a.issuer.DecodeJWEToken(query.Get("state"), state); err != nil {
		htmlError(w, r, http.StatusBadRequest, "oidc state failed to decode")
		return
	}

	if query.Has("error") {
		authorizationError(w, r, state.ClientRedirectURI, Error(query.Get("error")), query.Get("description"))
		return
	}

	if !query.Has("code") {
		authorizationError(w, r, state.ClientRedirectURI, ErrorServerError, "oidc callback does not contain an authorization code")
		return
	}

	// Exchange the code for an id_token, access_token and refresh_token with
	// the extracted code verifier.
	authURLParams := []oauth2.AuthCodeOption{
		oauth2.SetAuthURLParam("client_id", state.ClientID),
		oauth2.SetAuthURLParam("code_verifier", state.CodeVerfier),
	}

	tokens, err := a.oidcConfig(r).Exchange(r.Context(), query.Get("code"), authURLParams...)
	if err != nil {
		authorizationError(w, r, state.ClientRedirectURI, ErrorServerError, "oidc code exchange failed: "+err.Error())
		return
	}

	idTokenRaw, ok := tokens.Extra("id_token").(string)
	if !ok {
		authorizationError(w, r, state.ClientRedirectURI, ErrorServerError, "oidc response missing id_token")
		return
	}

	idToken, err := a.oidcExtractIDToken(r.Context(), idTokenRaw)
	if err != nil {
		authorizationError(w, r, state.ClientRedirectURI, ErrorServerError, "id_token verification failed: "+err.Error())
		return
	}

	var claims struct {
		Email string `json:"email"`
	}

	if err := idToken.Claims(&claims); err != nil {
		authorizationError(w, r, state.ClientRedirectURI, ErrorServerError, "failed to extract id_token email claims: "+err.Error())
		return
	}

	// Next up, exchange the ID token with keystone for an API token...
	// TODO: Once we have an unscoped token we should scope it to a project to avoid
	// the extra round trip.  We can possibly provide a hint in the ouath2 authorization
	// request query, and have that persisted by persistent storage in the browser.
	token, tokenMeta, err := a.keystone.OIDCTokenExchange(r.Context(), idTokenRaw)
	if err != nil {
		authorizationError(w, r, state.ClientRedirectURI, ErrorServerError, "keystone token exchange failed: "+err.Error())
		return
	}

	oauth2Code := &Code{
		ClientID:            state.ClientID,
		ClientRedirectURI:   state.ClientRedirectURI,
		ClientCodeChallenge: state.ClientCodeChallenge,
		ClientScope:         state.ClientScope,
		ClientNonce:         state.ClientNonce,
		KeystoneToken:       token,
		KeystoneUserID:      tokenMeta.Token.User.ID,
		Email:               claims.Email,
		Expiry:              tokenMeta.Token.ExpiresAt,
	}

	code, err := a.issuer.EncodeJWEToken(oauth2Code)
	if err != nil {
		authorizationError(w, r, state.ClientRedirectURI, ErrorServerError, "failed to encode authorization code: "+err.Error())
		return
	}

	q := &url.Values{}
	q.Set("code", code)

	if state.ClientState != "" {
		q.Set("state", state.ClientState)
	}

	http.Redirect(w, r, state.ClientRedirectURI+"?"+q.Encode(), http.StatusFound)
}

// tokenValidate does any request validation when issuing a token.
func tokenValidate(r *http.Request) error {
	if r.Form.Get("grant_type") != "authorization_code" {
		return errors.OAuth2UnsupportedGrantType("grant_type must be 'authorization_code'")
	}

	required := []string{
		"client_id",
		"redirect_uri",
		"code",
		"code_verifier",
	}

	for _, parameter := range required {
		if !r.Form.Has(parameter) {
			return errors.OAuth2InvalidRequest(parameter + " must be specified")
		}
	}

	return nil
}

// tokenValidateCode validates the request against the parsed code.
func tokenValidateCode(code *Code, r *http.Request) error {
	if code.ClientID != r.Form.Get("client_id") {
		return errors.OAuth2InvalidGrant("client_id mismatch")
	}

	if code.ClientRedirectURI != r.Form.Get("redirect_uri") {
		return errors.OAuth2InvalidGrant("redirect_uri mismatch")
	}

	if code.ClientCodeChallenge != encodeCodeChallengeS256(r.Form.Get("code_verifier")) {
		return errors.OAuth2InvalidClient("code_verfier invalid")
	}

	return nil
}

// oidcHash is used to create at_hash and c_hash values.
// TODO: this is very much tied to the algorithm defined (hard coded) in
// the JOSE package.
func oidcHash(value string) string {
	sum := sha512.Sum512([]byte(value))

	return base64.RawURLEncoding.EncodeToString(sum[:sha512.Size>>1])
}

// oidcPicture returns a URL to a picture for the user.
func oidcPicture(email string) string {
	//nolint:gosec
	return fmt.Sprintf("https://www.gravatar.com/avatar/%x", md5.Sum([]byte(email)))
}

// oidcIDToken builds an OIDC ID token.
func (a *Authenticator) oidcIDToken(r *http.Request, scope Scope, expiry time.Time, atHash, clientID, email string) (*string, error) {
	//nolint:nilnil
	if !scope.Has("openid") {
		return nil, nil
	}

	claims := &IDToken{
		Issuer:  "https://" + r.Host,
		Subject: email,
		Audience: []string{
			clientID,
		},
		Expiry:   expiry.Unix(),
		IssuedAt: time.Now().Unix(),
		ATHash:   atHash,
	}

	if scope.Has("email") {
		claims.Email = email
	}

	if scope.Has("profile") {
		claims.Picture = oidcPicture(email)
	}

	idToken, err := a.issuer.EncodeJWT(claims)
	if err != nil {
		return nil, err
	}

	return &idToken, nil
}

// Token issues an OAuth2 access token from the provided autorization code.
func (a *Authenticator) Token(w http.ResponseWriter, r *http.Request) (*generated.Token, error) {
	if err := r.ParseForm(); err != nil {
		return nil, errors.OAuth2InvalidRequest("failed to parse form data: " + err.Error())
	}

	// TODO: DELETE ME!!!!! See comments below.
	if r.Form.Get("grant_type") == "password" {
		return a.tokenPassword(r)
	}

	if err := tokenValidate(r); err != nil {
		return nil, err
	}

	code := &Code{}

	if err := a.issuer.DecodeJWEToken(r.Form.Get("code"), code); err != nil {
		return nil, errors.OAuth2InvalidRequest("failed to parse code: " + err.Error())
	}

	if err := tokenValidateCode(code, r); err != nil {
		return nil, err
	}

	claims := &UnikornClaims{
		Token: code.KeystoneToken,
		User:  code.KeystoneUserID,
	}

	accessToken, err := Issue(a.issuer, r, code.Email, claims, nil, code.Expiry)
	if err != nil {
		return nil, err
	}

	// Handle OIDC.
	idToken, err := a.oidcIDToken(r, code.ClientScope, code.Expiry, oidcHash(accessToken), r.Form.Get("client_id"), code.Email)
	if err != nil {
		return nil, err
	}

	result := &generated.Token{
		TokenType:   "Bearer",
		AccessToken: accessToken,
		IdToken:     idToken,
		ExpiresIn:   int(time.Until(code.Expiry).Seconds()),
	}

	return result, nil
}

// tokenPasswordValidate does any request validation when issuing a token.
func tokenPasswordValidate(r *http.Request) error {
	required := []string{
		"username",
		"password",
	}

	for _, parameter := range required {
		if !r.Form.Has(parameter) {
			return errors.OAuth2InvalidRequest(parameter + " must be specified")
		}
	}

	// TODO: if the openid scope is defined, then we need a client ID also to set
	// the audience in the id_token.
	return nil
}

// tokenPassword does username/password style logins.  This should never, ever be used, and
// is dropped as of oauth2.1.  Part of the problem is that unlike the authorization flow callback
// there are potentially a whole load of 3rd party scripts running to inject a supply chain attack
// and steal either the credentials or the access token.
func (a *Authenticator) tokenPassword(r *http.Request) (*generated.Token, error) {
	if err := tokenPasswordValidate(r); err != nil {
		return nil, err
	}

	token, user, err := a.keystone.Basic(r.Context(), r.Form.Get("username"), r.Form.Get("password"))
	if err != nil {
		return nil, errors.OAuth2AccessDenied("authentication failed").WithError(err)
	}

	userDetail, err := a.keystone.GetUser(r.Context(), token.ID, user.ID)
	if err != nil {
		return nil, errors.OAuth2ServerError("unable to get user detail").WithError(err)
	}

	claims := &UnikornClaims{
		Token: token.ID,
		User:  user.ID,
	}

	accessToken, err := Issue(a.issuer, r, r.Form.Get("username"), claims, nil, token.ExpiresAt)
	if err != nil {
		return nil, err
	}

	email, ok := userDetail.Extra["email"].(string)
	if !ok {
		return nil, errors.OAuth2ServerError("unable to get user email")
	}

	idToken, err := a.oidcIDToken(r, NewScope(r.Form.Get("scope")), token.ExpiresAt, oidcHash(accessToken), r.Form.Get("client_id"), email)
	if err != nil {
		return nil, err
	}

	result := &generated.Token{
		TokenType:   "Bearer",
		AccessToken: accessToken,
		IdToken:     idToken,
		ExpiresIn:   int(time.Until(token.ExpiresAt).Seconds()),
	}

	return result, nil
}
