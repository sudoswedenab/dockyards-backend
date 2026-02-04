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
	"strings"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/sudoswedenab/dockyards-backend/api/config"
	dockyardsv1 "github.com/sudoswedenab/dockyards-backend/api/v1alpha3"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/oauth2"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (h *handler) config(ctx context.Context, providerName string) (*oauth2.Config, *oidc.Provider, error) {
	publicNamespace := h.Config.GetValueOrDefault(config.KeyPublicNamespace, "dockyards-public")

	var identityProvider dockyardsv1.IdentityProvider
	err := h.Get(ctx, client.ObjectKey{Name: providerName, Namespace: publicNamespace}, &identityProvider)
	if err != nil {
		return nil, nil, fmt.Errorf("could not get provider: %w", err)
	}

	configRef := identityProvider.Spec.OIDCConfigRef
	if configRef == nil {
		return nil, nil, errors.New("oidc config ref was is not set")
	}

	oidcConfig, _, err := h.getOIDCConfig(ctx, *configRef)
	if err != nil {
		return nil, nil, fmt.Errorf("could not get oidc config: %w", err)
	}

	provider, err := h.getOIDCProvider(ctx, oidcConfig)
	if err != nil {
		return nil, nil, fmt.Errorf("could not get oidc provider: %w", err)
	}

	client := oidcConfig.ClientConfig
	config := oauth2.Config{
		ClientID:     client.ClientID,
		ClientSecret: client.ClientSecret,
		Endpoint:     provider.Endpoint(),
		RedirectURL:  client.RedirectURL,
		Scopes:       []string{"openid", "email", "profile"},
	}

	return &config, provider, nil
}

func (h *handler) LoginOIDC(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()

	providerName := r.URL.Query().Get("idp")
	callbackURL := r.URL.Query().Get("callbackURL")

	config, _, err := h.config(ctx, providerName)
	if err != nil {
		return apierrors.NewInternalError(err)
	}

	state, err := makeState(providerName, callbackURL)
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

	// http://localhost/api/backend/v1/callback-sso

	h.logger.Info("redirecting user in sso", "url", url)
	http.Redirect(w, r, url, http.StatusSeeOther)

	return nil
}

func decodeState(data string) (*stateStruct, error) {
	stateJSON, err := base64.URLEncoding.DecodeString(data)
	if err != nil {
		return nil, fmt.Errorf("could not decode state: %w", err)
	}

	var state stateStruct
	err = json.Unmarshal(stateJSON, &state)
	if err != nil {
		return nil, err
	}

	return &state, nil
}

type Claims struct {
	Email string `json:"email"`
	PreferredUsername string `json:"preferred_username"`
	Name string `json:"name"`
}

func (h *handler) Callback(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()

	code := r.URL.Query().Get("code")

	nonceCookie, err := r.Cookie("nonce")
	if err != nil {
		return apierrors.NewInternalError(errors.New("nonce cookie was not set"))
	}
	expectedNonce := fmt.Sprintf("%x", sha256.Sum256([]byte(nonceCookie.Value)))

	stateCookie, err := r.Cookie("state")
	if err != nil {
		return apierrors.NewBadRequest("state cookie not found")
	}

	queryState := r.URL.Query().Get("state")
	if queryState == "" {
		return apierrors.NewBadRequest("state query not found")
	}

	if stateCookie.Value != queryState {
		return apierrors.NewBadRequest("state cookie did not match query parameter")
	}

	state, err := decodeState(queryState)
	if err != nil {
		return apierrors.NewInternalError(err)
	}

	config, provider, err := h.config(ctx, state.IDP)
	if err != nil {
		return apierrors.NewInternalError(fmt.Errorf("could not get config: %w", err))
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

	var claims Claims
	err = token.Claims(&claims)
	if err != nil {
		return apierrors.NewInternalError(fmt.Errorf("could not parse claims: %w", err))
	}

	user, err := h.getOrCreateUser(ctx, state.IDP, claims)
	if err != nil {
		return apierrors.NewInternalError(fmt.Errorf("could not get or create user: %w", err))
	}

	tokens, err := h.generateTokens(user)
	if err != nil {
		return apierrors.NewInternalError(fmt.Errorf("could not generate tokens: %w", err))
	}

	url := state.CallbackURL + "/" + tokens.RefreshToken
	http.Redirect(w, r, url, http.StatusFound)

	return nil
}

func (h *handler) getOrCreateUser(ctx context.Context, providerName string, claims Claims) (*dockyardsv1.User, error) {
	name := providerName + "-" + claims.PreferredUsername

	var user dockyardsv1.User
	err := h.Get(ctx, client.ObjectKey{Name: name}, &user)
	if err == nil {
		if !strings.HasPrefix(user.Spec.ProviderID, providerName + "://") {
			return nil, errors.New("user provider id was invalid")
		}
		if user.Labels[dockyardsv1.LabelProviderName] != providerName {
			return nil, errors.New("user provider name was invalid")
		}

		return &user, nil
	}

	if !apierrors.IsNotFound(err) {
		return nil, err
	}

	password, err := randomPassword() // Should be impossible to guess.
	if err != nil {
		return nil, fmt.Errorf("could not create random password: %w", err)
	}

	user = dockyardsv1.User{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			Labels: map[string]string{
				dockyardsv1.LabelProviderName: providerName,
			},
		},
		Spec: dockyardsv1.UserSpec{
			DisplayName: claims.Name,
			Password: password,
			Email: claims.Email,
			ProviderID: providerName + "://" + claims.PreferredUsername,
		},
		Status: dockyardsv1.UserStatus{
			Conditions: []metav1.Condition{
				metav1.Condition{
					Type: dockyardsv1.ReadyCondition,
					Status: metav1.ConditionTrue,
					Reason: dockyardsv1.VerificationReasonVerified,
					Message: "Verified by OIDC",
					LastTransitionTime: metav1.Now(),
				},
			},
		},
	}
	err = h.Create(ctx, &user)
	if err != nil {
		return nil, fmt.Errorf("could not create user: %w", err)
	}

	return &user, nil
}

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
func makeState(identityProviderName string, callbackURL string) (string, error) {
	src := make([]byte, 18)
	_, err := rand.Read(src)
	if err != nil {
		return "", err
	}

	csrf := base64.URLEncoding.EncodeToString(src)
	stateJSON, err := json.Marshal(stateStruct{
		CSRF: csrf,
		IDP: identityProviderName,
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

func (h *handler) getOIDCConfig(ctx context.Context, ref corev1.SecretReference) (dockyardsv1.OIDCConfig, corev1.Secret, error) {
	secret := corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ref.Name,
			Namespace: ref.Namespace,
		},
	}

	h.logger.Info("getting oidc config", "name", ref.Name, "namespace", ref.Namespace)

	err := h.Get(ctx, client.ObjectKeyFromObject(&secret), &secret)
	if err != nil {
		return dockyardsv1.OIDCConfig{}, corev1.Secret{}, err
	}

	obj := make(map[string]any, len(secret.Data))
	for key, value := range secret.Data {
		var v any
		err = json.Unmarshal(value, &v);
		if err != nil {
			return dockyardsv1.OIDCConfig{}, corev1.Secret{}, fmt.Errorf("could not unmarshal oidc config key '%s': %w", key, err)
		}
		obj[key] = v
	}

	payload, err := json.Marshal(obj)
	if err != nil {
		return dockyardsv1.OIDCConfig{}, corev1.Secret{}, err
	}

	var oidcConfig dockyardsv1.OIDCConfig
	err = json.Unmarshal(payload, &oidcConfig)
	if err != nil {
		return dockyardsv1.OIDCConfig{}, corev1.Secret{}, err
	}

	// nolint:nestif // extracting this to its own validating function would make this block more complex
	if oidcConfig.ProviderConfig != nil {
		config := *oidcConfig.ProviderConfig
		if config.Issuer == "" {
			return dockyardsv1.OIDCConfig{}, corev1.Secret{}, fmt.Errorf("invalid oidc provider config: issuer field is suspiciously empty")
		}
		if config.AuthorizationEndpoint == "" {
			return dockyardsv1.OIDCConfig{}, corev1.Secret{}, fmt.Errorf("invalid oidc provider config: authorization_endpoint field is suspiciously empty")
		}
		if config.TokenEndpoint == "" {
			return dockyardsv1.OIDCConfig{}, corev1.Secret{}, fmt.Errorf("invalid oidc provider config: token_endpoint field is suspiciously empty")
		}
		if config.DeviceAuthorizationEndpoint == "" {
			return dockyardsv1.OIDCConfig{}, corev1.Secret{}, fmt.Errorf("invalid oidc provider config: device_authorization_endpoint field is suspiciously empty")
		}
		if config.UserinfoEndpoint == "" {
			return dockyardsv1.OIDCConfig{}, corev1.Secret{}, fmt.Errorf("invalid oidc provider config: userinfo_endpoint field is suspiciously empty")
		}
		if config.JWKSURI == "" {
			return dockyardsv1.OIDCConfig{}, corev1.Secret{}, fmt.Errorf("invalid oidc provider config: jwks_uri field is suspiciously empty")
		}
		if len(config.IDTokenSigningAlgs) == 0 {
			return dockyardsv1.OIDCConfig{}, corev1.Secret{}, fmt.Errorf("invalid oidc provider config: id_token_signing_alg_values_supported field is suspiciously empty")
		}
	}

	return oidcConfig, secret, nil
}

func randomPassword() (string, error) {
	bytes := make([]byte, 64)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", err
	}
	pw, err := bcrypt.GenerateFromPassword(bytes, bcrypt.DefaultCost)

	return string(pw), err
}
