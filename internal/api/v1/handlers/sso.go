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
	"errors"
	"fmt"
	"net/http"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/sudoswedenab/dockyards-api/pkg/types"
	"github.com/sudoswedenab/dockyards-backend/api/config"
	dockyardsv1 "github.com/sudoswedenab/dockyards-backend/api/v1alpha3"
	"github.com/sudoswedenab/dockyards-backend/api/v1alpha3/index"
	"github.com/sudoswedenab/dockyards-backend/internal/api/v1/middleware"
	"golang.org/x/oauth2"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func makeNonce() ([]byte, error) {
	src := make([]byte, 18)
	count, err := rand.Read(src)
	if err != nil {
		return nil, err
	}
	if count != len(src) {
		return nil, errors.New("unexpected byte count")
	}

	dst := make([]byte, base64.URLEncoding.EncodedLen(len(src)))
	base64.URLEncoding.Encode(dst, src)

	return dst, nil
}

type stateStruct struct {
	CSRF        string `json:"state"`
	IDP         string `json:"idp"`
	CallbackURL string `json:"callbackURL"`
}

func makeState(identityProvider dockyardsv1.IdentityProvider, callbackURL string) (string, error) {
	src := make([]byte, 18)
	_, err := rand.Read(src)
	if err != nil {
		return "", err
	}

	csrf := base64.URLEncoding.EncodeToString(src)
	stateJSON, err := json.Marshal(stateStruct{
		CSRF: csrf,
		IDP: identityProvider.Name,
		CallbackURL: callbackURL,
	})
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

func (h *handler) getOIDCProvider(ctx context.Context, oidcConfig dockyardsv1.OIDCConfig) (*oidc.Provider, error) {
	if oidcConfig.ProviderConfig != nil {
		c := oidcConfig.ProviderConfig
		pc := oidc.ProviderConfig{
			IssuerURL:  c.Issuer,
			AuthURL:    c.AuthorizationEndpoint,
			TokenURL:   c.TokenEndpoint,
			JWKSURL:    c.JWKSURI,
			Algorithms: c.IDTokenSigningAlgs,
		}

		return pc.NewProvider(ctx), nil
	}

	if oidcConfig.ProviderDiscoveryURL != nil {
		p, err := oidc.NewProvider(ctx, *oidcConfig.ProviderDiscoveryURL)
		if err != nil {
			return nil, err
		}

		return p, nil
	}

	return nil, fmt.Errorf("oidc config needs either provider config or provider discovery url")
}

func (h *handler) getOIDCConfig(ctx context.Context, ref corev1.SecretReference) (dockyardsv1.OIDCConfig, error) {
	secret := corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ref.Name,
			Namespace: ref.Namespace,
		},
	}

	h.logger.Info("getting oidc config", "name", ref.Name, "namespace", ref.Namespace)

	err := h.Get(ctx, client.ObjectKeyFromObject(&secret), &secret)
	if err != nil {
		return dockyardsv1.OIDCConfig{}, err
	}

	obj := make(map[string]any, len(secret.Data))
	for key, value := range secret.Data {
		var v any
		err = json.Unmarshal(value, &v);
		if err != nil {
			return dockyardsv1.OIDCConfig{}, fmt.Errorf("could not unmarshal oidc config key '%s': %w", key, err)
		}
		obj[key] = v
	}

	payload, err := json.Marshal(obj)
	if err != nil {
		return dockyardsv1.OIDCConfig{}, err
	}

	var oidcConfig dockyardsv1.OIDCConfig
	err = json.Unmarshal(payload, &oidcConfig)
	if err != nil {
		return dockyardsv1.OIDCConfig{}, err
	}

	// nolint:nestif // extracting this to its own validating function would make this block more complex
	if oidcConfig.ProviderConfig != nil {
		config := *oidcConfig.ProviderConfig
		if config.Issuer == "" {
			return dockyardsv1.OIDCConfig{}, fmt.Errorf("invalid oidc provider config: issuer field is suspiciously empty")
		}
		if config.AuthorizationEndpoint == "" {
			return dockyardsv1.OIDCConfig{}, fmt.Errorf("invalid oidc provider config: authorizationEndpoint field is suspiciously empty")
		}
		if config.TokenEndpoint == "" {
			return dockyardsv1.OIDCConfig{}, fmt.Errorf("invalid oidc provider config: authorizationEndpoint field is suspiciously empty")
		}
		if config.DeviceAuthorizationEndpoint == "" {
			return dockyardsv1.OIDCConfig{}, fmt.Errorf("invalid oidc provider config: deviceAuthorizationEndpoint field is suspiciously empty")
		}
		if config.UserinfoEndpoint == "" {
			return dockyardsv1.OIDCConfig{}, fmt.Errorf("invalid oidc provider config: userinfoEndpoint field is suspiciously empty")
		}
		if config.JWKSURI == "" {
			return dockyardsv1.OIDCConfig{}, fmt.Errorf("invalid oidc provider config: jwks field is suspiciously empty")
		}
		if len(config.IDTokenSigningAlgs) == 0 {
			return dockyardsv1.OIDCConfig{}, fmt.Errorf("invalid oidc provider config: idTokenSigningAlgs field is suspiciously empty")
		}
	}

	return oidcConfig, nil
}

func (h *handler) LoginOIDC(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()

	providerName := r.URL.Query().Get("idp")
	callbackURL := r.URL.Query().Get("callbackURL")

	publicNamespace := h.Config.GetValueOrDefault(config.KeyPublicNamespace, "dockyards-public")

	var identityProvider dockyardsv1.IdentityProvider
	err := h.Get(ctx, client.ObjectKey{Name: providerName, Namespace: publicNamespace}, &identityProvider)
	if err != nil {
		return apierrors.NewInternalError(fmt.Errorf("could not get provider: %w", err))
	}

	configRef := identityProvider.Spec.OIDCConfigRef
	if configRef == nil {
		return apierrors.NewInternalError(errors.New("oidc config ref was is not set"))
	}

	oidcConfig, err := h.getOIDCConfig(ctx, *configRef)
	if err != nil {
		return apierrors.NewInternalError(fmt.Errorf("could not get oidc config: %w", err))
	}

	provider, err := h.getOIDCProvider(ctx, oidcConfig)
	if err != nil {
		return apierrors.NewInternalError(fmt.Errorf("could not get oidc provider: %w", err))
	}

	c := oidcConfig.ClientConfig
	config := oauth2.Config{
		ClientID:     c.ClientID,
		ClientSecret: c.ClientSecret,
		Endpoint:     provider.Endpoint(),
		RedirectURL:  c.RedirectURL,
		Scopes:       []string{"openid", "email"},
	}

	state, err := makeState(identityProvider, callbackURL)
	if err != nil {
		return apierrors.NewInternalError(fmt.Errorf("could not make state: %w", err))
	}

	nonce, err := makeNonce()
	if err != nil {
		return apierrors.NewInternalError(fmt.Errorf("could not make nonce: %w", err))
	}

	setCallbackCookie(w, r, "state", state)
	setCallbackCookie(w, r, "nonce", string(nonce))

	url := config.AuthCodeURL(
		state,
		oauth2.SetAuthURLParam("nonce", fmt.Sprintf("%x", sha256.Sum256(nonce))),
		oauth2.SetAuthURLParam("prompt", "consent"), // HARDCODE ApprovalForce for now
	)

	http.Redirect(w, r, url, http.StatusSeeOther)

	return nil
}

func (h *handler) newUser(ctx context.Context, idToken oidc.IDToken, email string) (*dockyardsv1.User, error) {
	user := dockyardsv1.User{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "dockyards-",
		},
		Spec: dockyardsv1.UserSpec{
			// DisplayName: displayname,
			Email:      email,
			Password:   "$2a$12$R9h/cIPz0gi.URNNX3kh2OPST9/PgBkqq", // should be impossible
			ProviderID: idToken.Subject,                    		// value should be <provider_name>://<subject>
		},
	}

	err := h.Create(ctx, &user)
	if err != nil {
		return nil, fmt.Errorf("could not create user: %w", err)
	}

	return &user, nil
}

func (h *handler) Callback(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	logger := middleware.LoggerFrom(ctx)

	code := r.URL.Query().Get("code")

	nonceCookie, err := r.Cookie("nonce")
	if err != nil {
		return apierrors.NewInternalError(errors.New("nonce cookie was not set"))
	}
	expectedNonce := fmt.Sprintf("%x", sha256.Sum256([]byte(nonceCookie.Value)))

	stateCookie, err := r.Cookie("state")
	if err != nil {
		return apierrors.NewBadRequest("state not found")
	}

	queryState := r.URL.Query().Get("state")

	if stateCookie.Value != queryState {
		return apierrors.NewBadRequest("state cookie did not match query parameter")
	}
	state := queryState

	stateJSON, err := base64.URLEncoding.DecodeString(state)
	if err != nil {
		return apierrors.NewInternalError(fmt.Errorf("could not decode state: %w", err))
	}

	var decodedState stateStruct
	err = json.Unmarshal(stateJSON, &decodedState)
	if err != nil {
		return apierrors.NewInternalError(err)
	}

	var identityProvider dockyardsv1.IdentityProvider
	err = h.Get(ctx, client.ObjectKey{Name: decodedState.IDP}, &identityProvider)
	if err != nil {
		return apierrors.NewInternalError(err)
	}

	configRef := identityProvider.Spec.OIDCConfigRef
	if configRef == nil {
		return apierrors.NewInternalError(errors.New("identity provider did not contain oidc config ref"))
	}

	oidcConfig, err := h.getOIDCConfig(ctx, *configRef)
	if err != nil {
		return apierrors.NewInternalError(fmt.Errorf("could not get OIDC config: %w", err))
	}

	provider, err := h.getOIDCProvider(ctx, oidcConfig)
	if err != nil {
		return apierrors.NewInternalError(fmt.Errorf("could not get OIDC provider: %w", err))
	}

	c := oidcConfig.ClientConfig
	config := oauth2.Config{
		ClientID:     c.ClientID,
		ClientSecret: c.ClientSecret,
		Endpoint:     provider.Endpoint(),
		RedirectURL:  c.RedirectURL,
		Scopes:       []string{"openid", "email"},
	}

	oauth2Token, err := config.Exchange(ctx, code)
	if err != nil {
		return apierrors.NewInternalError(fmt.Errorf("could not exchange tokens: %w", err))
	}

	rawIDToken, ok := oauth2Token.Extra("id_token").(string)
	if !ok {
		return apierrors.NewInternalError(errors.New("could not find id_token field in oauth2 token"))
	}

	verifier := provider.Verifier(&oidc.Config{
		ClientID: config.ClientID,
	})

	token, err := verifier.Verify(ctx, rawIDToken)
	if err != nil {
		return apierrors.NewInternalError(err)
	}

	if token.Nonce != expectedNonce {
		return apierrors.NewInternalError(errors.New("token nonce did not match cookie nonce"))
	}

	// Extract the email for a hint
	var claims struct {
		Email string `json:"email"`
	}

	err = token.Claims(&claims)
	if err != nil {
		return apierrors.NewInternalError(err)
	}

	matchingFields := client.MatchingFields{
		index.ProviderIDField: string(token.Subject),
	}
	var userList dockyardsv1.UserList
	err = h.List(ctx, &userList, matchingFields)
	if err != nil {
		return apierrors.NewInternalError(fmt.Errorf("could not get users: %w", err))
	}

	email := claims.Email

	if len(userList.Items) > 1 {
		return apierrors.NewInternalError(fmt.Errorf("expected exactly one user, but found %d", len(userList.Items)))
	}

	var tokens *types.Tokens
	var user *dockyardsv1.User

	switch len(userList.Items) {
	case 1:
		user = &userList.Items[0]
		logger.Info("Found user", "user", user.Spec.Email)

	case 0:
		logger.Info("Gotta sign up!")
		user, err = h.newUser(ctx, *token, email)
		if err != nil {
			return apierrors.NewInternalError(err)
		}
	}

	tokens, err = h.generateTokens(user)
	if err != nil {
		return apierrors.NewInternalError(fmt.Errorf("could not generate tokens: %w", err))
	}

	url := decodedState.CallbackURL + "/" + tokens.RefreshToken
	http.Redirect(w, r, url, http.StatusFound)

	return nil
}
