package handlers

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"bitbucket.org/sudosweden/dockyards-api/pkg/types"
	"bitbucket.org/sudosweden/dockyards-backend/internal/api/v1/middleware"
	dockyardsv1 "bitbucket.org/sudosweden/dockyards-backend/pkg/api/v1alpha3"
	"bitbucket.org/sudosweden/dockyards-backend/pkg/api/v1alpha3/index"
	"golang.org/x/crypto/bcrypt"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestLogin(t *testing.T) {
	hash, err := bcrypt.GenerateFromPassword([]byte("password"), 10)
	if err != nil {
		t.Fatalf("unexpected error hashing string 'password'")
	}

	tt := []struct {
		name  string
		lists []client.ObjectList
		login types.Login
	}{
		{
			name: "test valid user",
			lists: []client.ObjectList{
				&dockyardsv1.UserList{
					Items: []dockyardsv1.User{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name: "test",
							},
							Spec: dockyardsv1.UserSpec{
								Email:    "test@dockyards.dev",
								Password: string(hash),
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
			login: types.Login{
				Email:    "test@dockyards.dev",
				Password: "password",
			},
		},
		{
			name: "test multiple users",
			lists: []client.ObjectList{
				&dockyardsv1.UserList{
					Items: []dockyardsv1.User{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name: "test1",
							},
							Spec: dockyardsv1.UserSpec{
								Email:    "test1@dockyards.dev",
								Password: string(hash),
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
						{
							ObjectMeta: metav1.ObjectMeta{
								Name: "test2",
							},
							Spec: dockyardsv1.UserSpec{
								Email:    "test2@dockyards.dev",
								Password: string(hash),
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
						{
							ObjectMeta: metav1.ObjectMeta{
								Name: "test3",
							},
							Spec: dockyardsv1.UserSpec{
								Email:    "test3@dockyards.dev",
								Password: string(hash),
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
			login: types.Login{
				Email:    "test3@dockyards.dev",
				Password: "password",
			},
		},
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

	accessPrivateKey, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	refreshPrivateKey, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			scheme := scheme.Scheme
			dockyardsv1.AddToScheme(scheme)
			fakeClient := fake.NewClientBuilder().
				WithScheme(scheme).
				WithLists(tc.lists...).
				WithIndex(&dockyardsv1.User{}, index.EmailField, index.ByEmail).
				Build()

			h := handler{
				logger:               logger,
				Client:               fakeClient,
				jwtAccessPrivateKey:  accessPrivateKey,
				jwtRefreshPrivateKey: refreshPrivateKey,
			}

			b, err := json.Marshal(tc.login)
			if err != nil {
				t.Fatalf("unexpected error marshalling: %s", err)
			}

			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodPost, "/v1/login", bytes.NewBuffer(b))

			ctx := middleware.ContextWithLogger(context.Background(), logger)

			h.Login(w, r.Clone(ctx))

			if w.Code != http.StatusOK {
				t.Errorf("expected code %d, got %d", http.StatusOK, w.Code)
			}
		})
	}
}

func TestLoginErrors(t *testing.T) {
	hash, err := bcrypt.GenerateFromPassword([]byte("password"), 10)
	if err != nil {
		t.Fatalf("unexpected error hashing string 'password'")
	}

	tt := []struct {
		name     string
		lists    []client.ObjectList
		login    types.Login
		expected int
	}{
		{
			name: "test incorrect password",
			lists: []client.ObjectList{
				&dockyardsv1.UserList{
					Items: []dockyardsv1.User{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name: "test",
							},
							Spec: dockyardsv1.UserSpec{
								Email:    "test@dockyards.dev",
								Password: string(hash),
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
			login: types.Login{
				Email:    "test@dockyards.dev",
				Password: "incorrect",
			},
			expected: http.StatusUnauthorized,
		},
		{
			name: "test missing user",
			login: types.Login{
				Email:    "test@dockyards.dev",
				Password: "password",
			},
			expected: http.StatusUnauthorized,
		},
		{
			name: "test unverified user",
			lists: []client.ObjectList{
				&dockyardsv1.UserList{
					Items: []dockyardsv1.User{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name: "test",
							},
							Spec: dockyardsv1.UserSpec{
								Email:    "test@dockyards.dev",
								Password: string(hash),
							},
							Status: dockyardsv1.UserStatus{},
						},
					},
				},
			},
			login: types.Login{
				Email:    "test@dockyards.dev",
				Password: "password",
			},
			expected: http.StatusForbidden,
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError + 1}))

			scheme := scheme.Scheme
			dockyardsv1.AddToScheme(scheme)
			fakeClient := fake.NewClientBuilder().
				WithScheme(scheme).
				WithLists(tc.lists...).
				WithIndex(&dockyardsv1.User{}, index.EmailField, index.ByEmail).
				Build()

			h := handler{
				Client: fakeClient,
			}

			b, err := json.Marshal(tc.login)
			if err != nil {
				t.Fatalf("unexpected error marshalling: %s", err)
			}

			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodPost, "/v1/login", bytes.NewBuffer(b))

			ctx := middleware.ContextWithLogger(context.Background(), logger)

			h.Login(w, r.Clone(ctx))

			if w.Code != tc.expected {
				t.Errorf("expected code %d, got %d", tc.expected, w.Code)
			}
		})
	}
}
