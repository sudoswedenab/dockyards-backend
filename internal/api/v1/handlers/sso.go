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
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/sudoswedenab/dockyards-api/pkg/types"
	dockyardsv1 "github.com/sudoswedenab/dockyards-backend/api/v1alpha3"
	"github.com/sudoswedenab/dockyards-backend/api/v1alpha3/index"
	"github.com/sudoswedenab/dockyards-backend/internal/api/v1/middleware"
	"golang.org/x/oauth2"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func makeNonce() []byte {
	src := make([]byte, 18)
	_, _ = rand.Read(src)
	dst := make([]byte, base64.URLEncoding.EncodedLen(len(src)))
	base64.URLEncoding.Encode(dst, src)

	return dst
}

type stateStruct struct {
	CSRF        string `json:"state"`
	IDP         string `json:"idp"`
	CallbackURL string `json:"callbackURL"`
}

func makeState(idp *dockyardsv1.IdentityProvider, callbackURL string) (string, error) {
	src := make([]byte, 18)
	_, err := rand.Read(src)
	if err != nil {
		return "", err
	}

	csrf := base64.URLEncoding.EncodeToString(src)
	stateJSON, err := json.Marshal(stateStruct{CSRF: csrf, IDP: idp.Name, CallbackURL: callbackURL})
	if err != nil {
		return "", err
	}

	return base64.URLEncoding.EncodeToString(stateJSON), nil
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

func (h *handler) getOIDCProvider(ctx context.Context, oidcConf dockyardsv1.OIDCConfig) (*oidc.Provider, error) {
	if oidcConf.OIDCProviderConfig != nil {
		c := oidcConf.OIDCProviderConfig
		pc := oidc.ProviderConfig{
			IssuerURL:  c.Issuer,
			AuthURL:    c.AuthorizationEndpoint,
			TokenURL:   c.TokenEndpoint,
			JWKSURL:    c.JWKSURI,
			Algorithms: c.IDTokenSigningAlgs,
		}

		return pc.NewProvider(ctx), nil
	} else if oidcConf.OIDCProviderDiscoveryURL != nil {
		p, _ := oidc.NewProvider(ctx, *oidcConf.OIDCProviderDiscoveryURL)

		return p, nil
	}
	return nil, fmt.Errorf("Bad config")
}

func (h *handler) enrichOIDCConfig(ctx context.Context, cfg *dockyardsv1.OIDCConfig) error {
	if cfg.OIDCProviderConfig != nil || cfg.OIDCProviderDiscoveryURL == nil {
		return nil
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, *cfg.OIDCProviderDiscoveryURL, nil)
	if err != nil {
		return fmt.Errorf("build discovery request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("fetch discovery document: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("discovery endpoint returned %s", resp.Status)
	}

	var discovered dockyardsv1.OIDCProviderConfig
	if err := json.NewDecoder(resp.Body).Decode(&discovered); err != nil {
		return fmt.Errorf("decode discovery document: %w", err)
	}

	cfg.OIDCProviderConfig = &discovered
	return nil
}

func (h *handler) getOIDCConfig(ctx context.Context, idp dockyardsv1.IdentityProvider) (*dockyardsv1.OIDCConfig, error) {
	secret := corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      idp.Spec.OIDCConfig.Name,
			Namespace: idp.Spec.OIDCConfig.Namespace,
		},
	}

	if err := h.Get(ctx, client.ObjectKeyFromObject(&secret), &secret); err != nil {
		return nil, err
	}

	cfg, err := decodeSecretData[dockyardsv1.OIDCConfig](secret.Data)
	if err != nil {
		return nil, err
	}

	return cfg, nil
}

func decodeSecretData[T any](data map[string][]byte) (*T, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("secret does not contain any data")
	}

	obj := make(map[string]any, len(data))
	for key, value := range data {
		trimmed := bytes.TrimSpace(value)
		if len(trimmed) == 0 {
			obj[key] = ""

			continue
		}

		if json.Valid(trimmed) {
			var v any
			if err := json.Unmarshal(trimmed, &v); err != nil {
				return nil, fmt.Errorf("secret field %q is not valid JSON: %w", key, err)
			}
			obj[key] = v

			continue
		}

		obj[key] = string(value)
	}

	payload, err := json.Marshal(obj)
	if err != nil {
		return nil, err
	}

	var out T
	if err := json.Unmarshal(payload, &out); err != nil {
		return nil, err
	}

	return &out, nil
}

func (h *handler) LoginOIDC(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logger := middleware.LoggerFrom(ctx)

	r.Body.Close()
	providerName := r.URL.Query().Get("idp")
	callbackURL := r.URL.Query().Get("callbackURL")

	var idp dockyardsv1.IdentityProvider
	err := h.Get(ctx, client.ObjectKey{Name: providerName}, &idp)
	if err != nil {
		msg := "Invalid or missing IDP"
		logger.Warn(msg, "idp", idp.Name)
		http.Error(w, msg, http.StatusBadRequest)

		return
	}

	if idp.Spec.OIDCConfig == nil {
		msg := "IDP does not contain a reference to valid OIDC config"
		logger.Error(msg, "idp", idp.Name)
		http.Error(w, msg, http.StatusInternalServerError)

		return
	}

	configOIDC, err := h.getOIDCConfig(ctx, idp)
	if err != nil {
		msg := "Invalid or missing OIDC configuration"
		logger.Error(msg, "idp", idp.Name, "err", err)
		http.Error(w, msg, http.StatusInternalServerError)

		return
	}

	provider, err := h.getOIDCProvider(ctx, *configOIDC)
	if err != nil {
		msg := "Unable to parse OIDC provider from configuration"
		logger.Error(msg, "idp", idp.Name)
		http.Error(w, msg, http.StatusInternalServerError)

		return
	}

	c := configOIDC.OIDCClientConfig
	config := oauth2.Config{
		ClientID:     c.ClientID,
		ClientSecret: c.ClientSecret,
		Endpoint:     provider.Endpoint(),
		RedirectURL:  c.RedirectURL,
		Scopes:       []string{"openid", "email"},
	}

	state, err := makeState(&idp, callbackURL)
	if err != nil {
		panic(err)
	}

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
		msg := "state not found"
		logger.Error(msg, "http status", http.StatusBadRequest)
		http.Error(w, msg, http.StatusBadRequest)

		return
	}
	if r.URL.Query().Get("state") != state.Value {
		msg := "state mismatch"
		logger.Error(msg, "http status", http.StatusUnauthorized)
		http.Error(w, msg, http.StatusUnauthorized) // 401? 403?

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
		panic("no reference to OIDC config")
	}

	configOIDC, err := h.getOIDCConfig(ctx, idp)
	if err != nil {
		panic("not an OIDC config")
	}

	provider, err := h.getOIDCProvider(ctx, *configOIDC)
	if err != nil {
	}

	c := configOIDC.OIDCClientConfig
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
		msg := "Failed to verify ID token"
		logger.Error(msg, "err", err)
		http.Error(w, "Failed to verify ID Token", http.StatusInternalServerError)

		return
	}

	nonce, err := r.Cookie("nonce")
	if err != nil {
		msg := "nonce not found"
		logger.Warn(msg, "err", err)
		http.Error(w, msg, http.StatusBadRequest)

		return
	}
	if idToken.Nonce != fmt.Sprintf("%x", sha256.Sum256([]byte(nonce.Value))) {
		msg := "nonce did not match"
		logger.Warn(msg)
		http.Error(w, msg, http.StatusBadRequest)

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

	logger.Info("Processing SSO callback", "email", claims.Email)

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

	if len(userList.Items) > 1 {
		logger.Error("expected exactly one user from kubernetes", "users", len(userList.Items))
		w.WriteHeader(http.StatusUnauthorized)

		return
	}

	var tokens *types.Tokens
	var user *dockyardsv1.User

	switch len(userList.Items) {
	case 1:
		user = ptr.To(userList.Items[0])
		logger.Info("Found user", "user", user.Spec.Email)

	case 0:
		logger.Info("Gotta sign up!")
		user, err = h.newUser(ctx, *idToken, email)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)

			return
		}
	}

	// dumb extra redirect
	tokens, err = h.generateTokens(user)
	if err != nil {
		logger.Error("error generating tokens", "err", err)
		w.WriteHeader(http.StatusInternalServerError)

		return
	}

	url := x.CallbackURL + "/" + tokens.RefreshToken
	http.Redirect(w, r, url, http.StatusFound)
}
