package handlers

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"bitbucket.org/sudosweden/dockyards-backend/api/v1"
	"bitbucket.org/sudosweden/dockyards-backend/pkg/api/v1alpha1"
	"bitbucket.org/sudosweden/dockyards-backend/pkg/api/v1alpha1/index"
	"github.com/gin-gonic/gin"
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
		login v1.Login
	}{
		{
			name: "test valid user",
			lists: []client.ObjectList{
				&v1alpha1.UserList{
					Items: []v1alpha1.User{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name: "test",
							},
							Spec: v1alpha1.UserSpec{
								Email:    "test@dockyards.dev",
								Password: string(hash),
							},
							Status: v1alpha1.UserStatus{
								Conditions: []metav1.Condition{
									{
										Type:   v1alpha1.VerifiedCondition,
										Status: metav1.ConditionTrue,
										Reason: v1alpha1.UserVerifiedReason,
									},
								},
							},
						},
					},
				},
			},
			login: v1.Login{
				Email:    "test@dockyards.dev",
				Password: "password",
			},
		},
		{
			name: "test multiple users",
			lists: []client.ObjectList{
				&v1alpha1.UserList{
					Items: []v1alpha1.User{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name: "test1",
							},
							Spec: v1alpha1.UserSpec{
								Email:    "test1@dockyards.dev",
								Password: string(hash),
							},
							Status: v1alpha1.UserStatus{
								Conditions: []metav1.Condition{
									{
										Type:   v1alpha1.VerifiedCondition,
										Status: metav1.ConditionTrue,
										Reason: v1alpha1.UserVerifiedReason,
									},
								},
							},
						},
						{
							ObjectMeta: metav1.ObjectMeta{
								Name: "test2",
							},
							Spec: v1alpha1.UserSpec{
								Email:    "test2@dockyards.dev",
								Password: string(hash),
							},
							Status: v1alpha1.UserStatus{
								Conditions: []metav1.Condition{
									{
										Type:   v1alpha1.VerifiedCondition,
										Status: metav1.ConditionTrue,
										Reason: v1alpha1.UserVerifiedReason,
									},
								},
							},
						},
						{
							ObjectMeta: metav1.ObjectMeta{
								Name: "test3",
							},
							Spec: v1alpha1.UserSpec{
								Email:    "test3@dockyards.dev",
								Password: string(hash),
							},
							Status: v1alpha1.UserStatus{
								Conditions: []metav1.Condition{
									{
										Type:   v1alpha1.VerifiedCondition,
										Status: metav1.ConditionTrue,
										Reason: v1alpha1.UserVerifiedReason,
									},
								},
							},
						},
					},
				},
			},
			login: v1.Login{
				Email:    "test3@dockyards.dev",
				Password: "password",
			},
		},
	}

	gin.SetMode(gin.TestMode)

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

	accessPrivateKey, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	refreshPrivateKey, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			scheme := scheme.Scheme
			v1alpha1.AddToScheme(scheme)
			fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithLists(tc.lists...).WithIndex(&v1alpha1.User{}, "spec.email", index.EmailIndexer).Build()

			h := handler{
				logger:               logger,
				controllerClient:     fakeClient,
				jwtAccessPrivateKey:  accessPrivateKey,
				jwtRefreshPrivateKey: refreshPrivateKey,
			}

			r := gin.New()
			r.POST("/test", h.Login)

			b, err := json.Marshal(tc.login)
			if err != nil {
				t.Fatalf("unexpected error marshalling: %s", err)
			}

			w := httptest.NewRecorder()
			req, err := http.NewRequest("POST", "/test", bytes.NewBuffer(b))
			if err != nil {
				t.Fatalf("unexpected error preparing request: %s", err)
			}
			req.Header.Add("content-type", "application/json")

			r.ServeHTTP(w, req)

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
		login    v1.Login
		expected int
	}{
		{
			name: "test incorrect password",
			lists: []client.ObjectList{
				&v1alpha1.UserList{
					Items: []v1alpha1.User{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name: "test",
							},
							Spec: v1alpha1.UserSpec{
								Email:    "test@dockyards.dev",
								Password: string(hash),
							},
							Status: v1alpha1.UserStatus{
								Conditions: []metav1.Condition{
									{
										Type:   v1alpha1.VerifiedCondition,
										Status: metav1.ConditionTrue,
										Reason: v1alpha1.UserVerifiedReason,
									},
								},
							},
						},
					},
				},
			},
			login: v1.Login{
				Email:    "test@dockyards.dev",
				Password: "incorrect",
			},
			expected: http.StatusUnauthorized,
		},
		{
			name: "test missing user",
			login: v1.Login{
				Email:    "test@dockyards.dev",
				Password: "password",
			},
			expected: http.StatusUnauthorized,
		},
		{
			name: "test unverified user",
			lists: []client.ObjectList{
				&v1alpha1.UserList{
					Items: []v1alpha1.User{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name: "test",
							},
							Spec: v1alpha1.UserSpec{
								Email:    "test@dockyards.dev",
								Password: string(hash),
							},
							Status: v1alpha1.UserStatus{},
						},
					},
				},
			},
			login: v1.Login{
				Email:    "test@dockyards.dev",
				Password: "password",
			},
			expected: http.StatusForbidden,
		},
	}

	gin.SetMode(gin.TestMode)

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError + 1}))

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			scheme := scheme.Scheme
			v1alpha1.AddToScheme(scheme)
			fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithLists(tc.lists...).WithIndex(&v1alpha1.User{}, "spec.email", index.EmailIndexer).Build()

			h := handler{
				logger:           logger,
				controllerClient: fakeClient,
			}

			r := gin.New()
			r.POST("/test", h.Login)

			b, err := json.Marshal(tc.login)
			if err != nil {
				t.Fatalf("unexpected error marshalling: %s", err)
			}

			w := httptest.NewRecorder()
			req, err := http.NewRequest("POST", "/test", bytes.NewBuffer(b))
			if err != nil {
				t.Fatalf("unexpected error preparing request: %s", err)
			}
			req.Header.Add("content-type", "application/json")

			r.ServeHTTP(w, req)

			if w.Code != tc.expected {
				t.Errorf("expected code %d, got %d", tc.expected, w.Code)
			}
		})
	}
}
