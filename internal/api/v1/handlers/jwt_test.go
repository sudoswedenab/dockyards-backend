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
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	dockyardsv1 "github.com/sudoswedenab/dockyards-backend/api/v1alpha3"
	"github.com/sudoswedenab/dockyards-backend/api/v1alpha3/index"
	"github.com/sudoswedenab/dockyards-backend/internal/api/v1/middleware"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestPostRefresh(t *testing.T) {
	tt := []struct {
		name   string
		claims jwt.RegisteredClaims
		lists  []client.ObjectList
	}{
		{
			name: "test simple",
			claims: jwt.RegisteredClaims{
				Subject:   "24449fef-e181-42f3-a9c6-10d920024090",
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Minute * 5)),
			},
			lists: []client.ObjectList{
				&dockyardsv1.UserList{
					Items: []dockyardsv1.User{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name: "test",
								UID:  "24449fef-e181-42f3-a9c6-10d920024090",
							},
							Spec: dockyardsv1.UserSpec{
								Email: "test@dockyards.dev",
							},
							Status: dockyardsv1.UserStatus{
								Conditions: []metav1.Condition{
									{
										Type:   dockyardsv1.ReadyCondition,
										Status: metav1.ConditionTrue,
										Reason: "testing",
									},
								},
							},
						},
					},
				},
			},
		},
	}

	accessPrivateKey, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	refreshPrivateKey, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

			refreshToken := jwt.NewWithClaims(jwt.SigningMethodES256, tc.claims)
			signedRefreshToken, err := refreshToken.SignedString(refreshPrivateKey)
			if err != nil {
				t.Fatalf("unexpected error signing refresh token: %s", err)
			}

			scheme := scheme.Scheme
			dockyardsv1.AddToScheme(scheme)
			fakeClient := fake.NewClientBuilder().
				WithScheme(scheme).
				WithLists(tc.lists...).
				WithIndex(&dockyardsv1.User{}, index.UIDField, index.ByUID).
				Build()

			h := handler{
				logger:               logger,
				Client:               fakeClient,
				jwtAccessPrivateKey:  accessPrivateKey,
				jwtRefreshPrivateKey: refreshPrivateKey,
				jwtRefreshPublicKey:  &refreshPrivateKey.PublicKey,
			}

			u := url.URL{
				Path: path.Join("/v1/refresh"),
			}

			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodPost, u.Path, nil)

			r.Header.Set("Authorization", "Bearer "+signedRefreshToken)

			ctx := middleware.ContextWithLogger(context.Background(), logger)

			h.PostRefresh(w, r.Clone(ctx))

			statusCode := w.Result().StatusCode
			if statusCode != http.StatusOK {
				t.Fatalf("expected status code %d, got %d", http.StatusOK, statusCode)
			}
		})
	}
}

func TestPostRefreshErrors(t *testing.T) {
	tt := []struct {
		name     string
		claims   jwt.RegisteredClaims
		lists    []client.ObjectList
		expected int
	}{
		{
			name: "test expired token",
			claims: jwt.RegisteredClaims{
				Subject:   "02d52af7-409f-4452-b551-274a372476aa",
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(-5) * time.Minute)),
			},
			expected: http.StatusUnauthorized,
		},
		{
			name: "test missing subject",
			claims: jwt.RegisteredClaims{
				Subject:   "9de2904f-3e60-4097-ad16-bda5ebfbd452",
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
			},
			lists: []client.ObjectList{
				&dockyardsv1.UserList{
					Items: []dockyardsv1.User{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name: "test",
								UID:  "0ad28a56-3f6d-40bf-a3e6-0807f2fd6f86",
							},
							Spec: dockyardsv1.UserSpec{
								Email: "test@dockyards.dev",
							},
							Status: dockyardsv1.UserStatus{
								Conditions: []metav1.Condition{
									{
										Type:   dockyardsv1.ReadyCondition,
										Status: metav1.ConditionTrue,
										Reason: "testing",
									},
								},
							},
						},
					},
				},
			},
			expected: http.StatusUnauthorized,
		},
	}

	accessPrivateKey, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	refreshPrivateKey, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError + 1}))

			refreshToken := jwt.NewWithClaims(jwt.SigningMethodES256, tc.claims)
			signedRefreshToken, err := refreshToken.SignedString(refreshPrivateKey)
			if err != nil {
				t.Fatalf("unexpected error signing refresh token: %s", err)
			}

			scheme := scheme.Scheme
			dockyardsv1.AddToScheme(scheme)
			fakeClient := fake.NewClientBuilder().
				WithScheme(scheme).
				WithLists(tc.lists...).
				WithIndex(&dockyardsv1.User{}, index.UIDField, index.ByUID).
				Build()

			h := handler{
				logger:               logger,
				Client:               fakeClient,
				jwtAccessPrivateKey:  accessPrivateKey,
				jwtRefreshPrivateKey: refreshPrivateKey,
				jwtRefreshPublicKey:  &refreshPrivateKey.PublicKey,
			}

			u := url.URL{
				Path: path.Join("/v1/refresh"),
			}

			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodPost, u.Path, nil)

			r.Header.Set("Authorization", "Bearer "+signedRefreshToken)

			ctx := middleware.ContextWithLogger(context.Background(), logger)

			h.PostRefresh(w, r.Clone(ctx))

			statusCode := w.Result().StatusCode
			if statusCode != tc.expected {
				t.Fatalf("expected status code %d, got %d", tc.expected, statusCode)
			}
		})
	}
}
