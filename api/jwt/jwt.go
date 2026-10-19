// Copyright 2026 Sudo Sweden AB
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package jwt

import (
	"context"
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"net/http"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"github.com/golang-jwt/jwt/v5"
)

const (
	AccessTokenPrivateKeyKey  = "accessTokenPrivateKey"
	ECPrivateKeyKey           = "EC PRIVATE KEY"
)

func GetAccessKey(ctx context.Context, c client.Client, systemNamespace string) (*ecdsa.PublicKey, error) {
	secret := corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "dockyards-backend-jwt",
			Namespace: systemNamespace,
		},
	}

	fmt.Printf("trying to access %s/%s\n", systemNamespace, "dockyards-backend-jwt")

	err := c.Get(ctx, client.ObjectKeyFromObject(&secret), &secret)
	if err != nil {
		return nil, err
	}
	if secret.Data == nil {
		secret.Data = map[string][]byte{}
	}
	accessTokenPrivateKeyPEM, has := secret.Data[AccessTokenPrivateKeyKey]
	if !has {
		return nil, errors.New("secret did not contain " + AccessTokenPrivateKeyKey)
	}
	accessTokenPrivateKeyDER, _ := pem.Decode(accessTokenPrivateKeyPEM)
	if accessTokenPrivateKeyDER == nil || accessTokenPrivateKeyDER.Type != ECPrivateKeyKey {
		return nil, errors.New("invalid access private key")
	}
	accessTokenPrivateKey, err := x509.ParseECPrivateKey(accessTokenPrivateKeyDER.Bytes)
	if err != nil {
		return nil, err
	}

	return &accessTokenPrivateKey.PublicKey, nil
}

func TokenFromRequest(r *http.Request) (string, error) {
	token := r.URL.Query().Get("token")
	if token != "" {
		return token, nil
	}

	if r.Header.Get("Upgrade") == "websocket" {
		h := strings.TrimSpace(r.Header.Get("Sec-Websocket-Protocol"))
		protocols := strings.Split(h, ",")
		for i := range protocols {
			protocols[i] = strings.TrimSpace(protocols[i])
		}
		for _, protocol := range protocols {
			tokenBase64, found := strings.CutPrefix(protocol, "base64url.bearer.authorization.k8s.io.")
			if !found {
				continue
			}
			token, err := base64.URLEncoding.DecodeString(tokenBase64)
			if err != nil {
				return "", err
			}
			return string(token), nil
		}
		return "", fmt.Errorf("subprotocol 'base64url.bearer.authorization.k8s.io.<base64token>' was not provided")
	}

	authorization := r.Header.Get("Authorization")
	if authorization == "" {
		return "", fmt.Errorf("authorization header was not provided")
	}
	token, ok := strings.CutPrefix(authorization, "Bearer ")
	if !ok {
		return "", fmt.Errorf("expexted bearer token")
	}

	return token, nil
}

func SubjectFromRequest(r *http.Request, accessKey *ecdsa.PublicKey) (string, error) {
	tokenString, err := TokenFromRequest(r)
	if err != nil {
		return "", err
	}
	token, err := jwt.ParseWithClaims(tokenString, &jwt.RegisteredClaims{}, func(t *jwt.Token) (any, error) {
		_, ok := t.Method.(*jwt.SigningMethodECDSA)
		if !ok {
			return nil, jwt.ErrTokenSignatureInvalid
		}

		return accessKey, nil
	})
	if err != nil {
		return "", err
	}

	return token.Claims.GetSubject()
}

type subjectKey struct {}

func WithSubject(ctx context.Context, subject string) context.Context {
	return context.WithValue(ctx, subjectKey{}, subject)
}

func SubjectFrom(ctx context.Context) string {
	value, ok := ctx.Value(subjectKey{}).(string)
	if !ok {
		return ""
	}
	return value
}

type RequestAuthenticator interface {
	client.Client
	GetSystemNamespace() string
}

func WithAuth(a RequestAuthenticator, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// logger := webshell.LoggerFrom(r.Context())

		accessKey, err := GetAccessKey(r.Context(), a, a.GetSystemNamespace())
		if err != nil {
			// logger.Error("could not get access key", "err", err)
			w.WriteHeader(http.StatusInternalServerError)

			return
		}

		subject, err := SubjectFromRequest(r, accessKey)
		if err != nil {
			// logger.Error("could not get subject from request", "err", err)
			w.WriteHeader(http.StatusUnauthorized)

			return
		}

		r = r.WithContext(WithSubject(r.Context(), subject))
		next.ServeHTTP(w, r)
	})
}
