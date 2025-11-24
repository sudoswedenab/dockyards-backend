// Copyright 2024 Sudo Sweden AB
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package handlers

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/coreos/go-oidc/v3/oidc"
	dockyardsv1 "github.com/sudoswedenab/dockyards-backend/api/v1alpha3"
	"github.com/sudoswedenab/dockyards-backend/api/v1alpha3/index"
	"github.com/sudoswedenab/dockyards-backend/internal/api/v1/middleware"
	"golang.org/x/oauth2"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (h *handler) ListIdentityProviders(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logger := middleware.LoggerFrom(ctx)

	var idplist dockyardsv1.IdentityProviderList
	if err := h.List(ctx, &idplist); err != nil {
		logger.Error("missing resource", "err", err)
		w.WriteHeader(http.StatusInternalServerError)

		return
	}
	idps := []dockyardsv1.IdentityProvider{}
	for _, idp := range idplist.Items {
		// Only return objects with some type of config (currently just OIDC)
		if idp.Spec.OIDCConfig == nil {
			logger.Warn("incomplete IdentityProvider", "name", idp.Name)

			continue
		}
		// Only return OIDC objects which have at least one way of configuring an OIDC provider
		if idp.Spec.OIDCConfig != nil && idp.Spec.OIDCConfig.OIDCProviderDiscoveryURL == nil && idp.Spec.OIDCConfig.OIDCProviderConfig == nil {
			logger.Warn("incomplete or misconfigured OIDCConfig", "name", idp.Name)

			continue
		}
		idps = append(idps, dockyardsv1.IdentityProvider{
			ObjectMeta: metav1.ObjectMeta{
				UID:  idp.GetUID(),
				Name: idp.GetName(),
			},
			Spec: dockyardsv1.IdentityProviderSpec{
				DisplayName: idp.Spec.DisplayName,
			},
		})
	}

	b, err := json.Marshal(&idps)
	if err != nil {
		logger.Error("error serializing identity providers", "err", err)
		w.WriteHeader(http.StatusInternalServerError)

		return
	}

	w.WriteHeader(http.StatusOK)
	_, err = w.Write(b)
	if err != nil {
		logger.Error("error writing response data", "err", err)
	}
}

func makeNonce() []byte {
	src := make([]byte, 18)
	_, _ = rand.Read(src)
	dst := make([]byte, base64.URLEncoding.EncodedLen(len(src)))
	base64.URLEncoding.Encode(dst, src)

	return dst
}

type stateStruct struct {
	CSRF string `json:"state"`
	IDP  string `json:"idp"`
}

func makeState(idp *dockyardsv1.IdentityProvider) string {
	src := make([]byte, 18)
	_, _ = rand.Read(src)
	csrf := base64.URLEncoding.EncodeToString(src)
	stateJSON, err := json.Marshal(stateStruct{CSRF: csrf, IDP: idp.Name})
	if err != nil {
		panic("can this happen?")
	}

	return base64.URLEncoding.EncodeToString(stateJSON)
}

func setCallbackCookie(w http.ResponseWriter, r *http.Request, name, value string) {
	c := &http.Cookie{
		Name:     name,
		Value:    value,
		MaxAge:   60 * 60,
		Secure:   r.TLS != nil,
		Path:     "/",
		HttpOnly: true,
	}
	http.SetCookie(w, c)
}

func setAuthCookie(w http.ResponseWriter, r *http.Request, name, value string, exhours int) {
	c := &http.Cookie{
		Name:     name,
		Value:    value,
		MaxAge:   exhours * 60 * 60,
		Secure:   r.TLS != nil,
		Path:     "/",
		SameSite: 1,
		// HttpOnly: true,
	}
	http.SetCookie(w, c)
}

func (h *handler) getOIDCProvider(ctx context.Context, oidcConf dockyardsv1.OIDCConfig) *oidc.Provider {
	if oidcConf.OIDCProviderConfig != nil {
		c := oidcConf.OIDCProviderConfig
		pc := oidc.ProviderConfig{
			IssuerURL:  c.Issuer,
			AuthURL:    c.AuthorizationEndpoint,
			TokenURL:   c.TokenEndpoint,
			JWKSURL:    c.JWKSURI,
			Algorithms: c.IDTokenSigningAlgs,
		}

		return pc.NewProvider(ctx)
	} else if oidcConf.OIDCProviderDiscoveryURL != nil {
		p, _ := oidc.NewProvider(ctx, *oidcConf.OIDCProviderDiscoveryURL)

		return p
	}
	panic("Bad config")
}

func (h *handler) LoginOIDC(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	// logger := middleware.LoggerFrom(ctx)

	providerName := r.URL.Query().Get("idp")

	var idp dockyardsv1.IdentityProvider
	err := h.Get(ctx, client.ObjectKey{Name: providerName}, &idp)
	if err != nil {
		http.Error(w, "Invalid or missing IDP", http.StatusBadRequest)

		return
	}

	if idp.Spec.OIDCConfig == nil {
		panic("not an OIDC config")
	}
	provider := h.getOIDCProvider(ctx, *idp.Spec.OIDCConfig)

	c := idp.Spec.OIDCConfig.OIDCClientConfig
	config := oauth2.Config{
		ClientID:     c.ClientID,
		ClientSecret: c.ClientSecret,
		Endpoint:     provider.Endpoint(),
		RedirectURL:  c.RedirectURL,
		Scopes:       []string{"openid", "email"},
	}

	state := makeState(&idp)
	nonce := makeNonce()

	setCallbackCookie(w, r, "state", state)
	setCallbackCookie(w, r, "nonce", string(nonce))

	url := config.AuthCodeURL(
		state,
		oauth2.SetAuthURLParam("nonce", fmt.Sprintf("%x", sha256.Sum256(nonce))),
		oauth2.SetAuthURLParam("prompt", "consent"), // HARDCODE ApprovalForce for now
	)

	http.Redirect(w, r, url, http.StatusFound)
}

func (h *handler) newUser(ctx context.Context, idToken oidc.IDToken, email string) (*dockyardsv1.User, error) {
	logger := middleware.LoggerFrom(ctx)
	// auto create account (copy from serverside signup)
	// Create the user
	user := dockyardsv1.User{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "dockyards-",
			Annotations:  map[string]string{},
		},
		Spec: dockyardsv1.UserSpec{
			// DisplayName: displayname,
			Email:      email,
			Password:   "$2a$12$R9h/cIPz0gi.URNNX3kh2OPST9/PgBkqq", // should be impossible
			ProviderID: ptr.To(idToken.Subject),                    // value should be <provider_name>://<subject>
			// Phone:       phone,
		},
	}

	err := h.Client.Create(ctx, &user)
	if err != nil {
		logger.Error("no make user", "err", err)

		return nil, err
	}

	logger.Info("created user", "name", user.Name)

	logger.Info("Trying to activate")

	patch := client.MergeFrom(user.DeepCopy())
	condition := metav1.Condition{
		Type:    dockyardsv1.ReadyCondition,
		Status:  metav1.ConditionTrue,
		Reason:  "because",
		Message: "Verified by SSO",
	}

	meta.SetStatusCondition(&user.Status.Conditions, condition)

	// FIXME does this need to clean up half-baked user?
	err = h.Client.Status().Patch(ctx, &user, patch)
	if err != nil {
		logger.Error("Failed to set status condition", "name", user.Name)

		return nil, err
	}

	return &user, nil
}

func (h *handler) Callback(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logger := middleware.LoggerFrom(ctx)

	// verify state
	state, err := r.Cookie("state")
	if err != nil {
		http.Error(w, "state not found", http.StatusBadRequest)

		return
	}
	if r.URL.Query().Get("state") != state.Value {
		http.Error(w, "state mismatch", http.StatusUnauthorized) // 401? 403?

		return
	}
	logger.Debug("state matched")

	stateJSON, err := base64.URLEncoding.DecodeString(state.Value)
	if err != nil {
		panic("fix me")
	}
	var x stateStruct
	err = json.Unmarshal(stateJSON, &x)
	if err != nil {
		panic("fix me")
	}

	var idp dockyardsv1.IdentityProvider
	err = h.Get(ctx, client.ObjectKey{Name: x.IDP}, &idp)
	if err != nil {
		panic("fix me")
	}

	if idp.Spec.OIDCConfig == nil {
		panic("not an OIDC config")
	}
	provider := h.getOIDCProvider(ctx, *idp.Spec.OIDCConfig)

	c := idp.Spec.OIDCConfig.OIDCClientConfig
	config := oauth2.Config{
		ClientID:     c.ClientID,
		ClientSecret: c.ClientSecret,
		Endpoint:     provider.Endpoint(),
		RedirectURL:  c.RedirectURL,
		Scopes:       []string{"openid", "email"},
	}

	// get id_token
	oauth2Token, err := config.Exchange(ctx, r.URL.Query().Get("code"))
	if err != nil {
		http.Error(w, "Failed to exchange token: "+err.Error(), http.StatusInternalServerError)

		return
	}

	rawIDToken, ok := oauth2Token.Extra("id_token").(string)
	if !ok {
		http.Error(w, "No id_token field in oauth2 token.", http.StatusInternalServerError)

		return
	}

	// Got the id_token, not yet verified

	// verify id_token
	verifier := provider.Verifier(&oidc.Config{
		ClientID: config.ClientID,
	})
	idToken, err := verifier.Verify(ctx, rawIDToken)
	if err != nil {
		http.Error(w, "Failed to verify ID Token: "+err.Error(), http.StatusInternalServerError)

		return
	}

	nonce, err := r.Cookie("nonce")
	if err != nil {
		http.Error(w, "nonce not found", http.StatusBadRequest)

		return
	}
	if idToken.Nonce != fmt.Sprintf("%x", sha256.Sum256([]byte(nonce.Value))) {
		http.Error(w, "nonce did not match", http.StatusBadRequest)

		return
	}

	// At this point id_token is verified
	logger.Debug(idToken.Subject)

	// Extract the email for a hint
	var claims struct {
		Email string `json:"email"`
	}
	if err := idToken.Claims(&claims); err != nil {
		logger.Error("problem reading extra claims", "err", err)
		http.Error(w, "?", http.StatusInternalServerError)
	}

	logger.Info("", "email", claims.Email)

	// See if user exists
	matchingFields := client.MatchingFields{
		index.ProviderIDField: string(idToken.Subject),
	}
	var userList dockyardsv1.UserList
	err = h.List(ctx, &userList, matchingFields)
	if err != nil {
		logger.Error("error getting user from kubernetes", "err", err)
		w.WriteHeader(http.StatusUnauthorized)

		return
	}

	// TODO put unique email check here
	// email := dummy@sudosweden.com
	email := claims.Email

	switch len(userList.Items) {
	case 1:
		user := userList.Items[0]
		logger.Info("Found user", "user", user.Spec.Email)

		tokens, err := h.generateTokens(ptr.To(user))
		if err != nil {
			logger.Error("error generating tokens", "err", err)
			w.WriteHeader(http.StatusInternalServerError)

			return
		}

		// TODO double check the parameters againts cookies set by JS
		setAuthCookie(w, r, "AccessToken", tokens.AccessToken, 3)
		setAuthCookie(w, r, "RefreshToken", tokens.RefreshToken, 12)

		// FIXME: read the resume redirect from the state?
		http.Redirect(w, r, "http://localhost:8000", http.StatusFound)

	case 0:
		logger.Info("Gotta sign up!")
		user, err := h.newUser(ctx, *idToken, email)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)

			return
		}

		// dumb extra redirect
		tokens, err := h.generateTokens(user)
		if err != nil {
			logger.Error("error generating tokens", "err", err)
			w.WriteHeader(http.StatusInternalServerError)

			return
		}

		setAuthCookie(w, r, "AccessToken", tokens.AccessToken, 3)
		setAuthCookie(w, r, "RefreshToken", tokens.RefreshToken, 12)

		http.Redirect(w, r, "http://localhost:8000", http.StatusFound)
	default:
		logger.Error("expected exactly one user from kubernetes", "users", len(userList.Items))
		w.WriteHeader(http.StatusUnauthorized)

		return
	}
}
