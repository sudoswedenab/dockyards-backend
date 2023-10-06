package handlers

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path"
	"testing"
	"time"

	"bitbucket.org/sudosweden/dockyards-backend/pkg/api/v1alpha1"
	"bitbucket.org/sudosweden/dockyards-backend/pkg/util/index"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
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
				&v1alpha1.UserList{
					Items: []v1alpha1.User{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name: "test",
								UID:  "24449fef-e181-42f3-a9c6-10d920024090",
							},
							Spec: v1alpha1.UserSpec{
								Email: "test@dockyards.dev",
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
		},
	}

	accessTokenSecret := []byte("testing-access")
	refreshTokenSecret := []byte("testing-refresh")

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

			refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, tc.claims)
			signedRefreshToken, err := refreshToken.SignedString(refreshTokenSecret)
			if err != nil {
				t.Fatalf("unexpected error signing refresh token: %s", err)
			}

			scheme := scheme.Scheme
			v1alpha1.AddToScheme(scheme)
			fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithLists(tc.lists...).WithIndex(&v1alpha1.User{}, "metadata.uid", index.UIDIndexer).Build()

			h := handler{
				jwtAccessTokenSecret:  accessTokenSecret,
				jwtRefreshTokenSecret: refreshTokenSecret,
				logger:                logger,
				controllerClient:      fakeClient,
				namespace:             "test",
			}

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			u := url.URL{
				Path: path.Join("/v1/refresh"),
			}

			header := http.Header{}
			header.Set("Authorization", "Bearer "+signedRefreshToken)

			c.Request = &http.Request{
				Method: http.MethodPost,
				URL:    &u,
				Header: header,
			}

			h.PostRefresh(c)

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
				&v1alpha1.UserList{
					Items: []v1alpha1.User{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name: "test",
								UID:  "0ad28a56-3f6d-40bf-a3e6-0807f2fd6f86",
							},
							Spec: v1alpha1.UserSpec{
								Email: "test@dockyards.dev",
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
			expected: http.StatusUnauthorized,
		},
	}

	accessTokenSecret := []byte("test-access-errors")
	refreshTokenSecret := []byte("test-refresh-errors")

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError + 1}))

			refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, tc.claims)
			signedRefreshToken, err := refreshToken.SignedString(refreshTokenSecret)
			if err != nil {
				t.Fatalf("unexpected error signing refresh token: %s", err)
			}

			scheme := scheme.Scheme
			v1alpha1.AddToScheme(scheme)
			fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithLists(tc.lists...).WithIndex(&v1alpha1.User{}, "metadata.uid", index.UIDIndexer).Build()

			h := handler{
				jwtAccessTokenSecret:  accessTokenSecret,
				jwtRefreshTokenSecret: refreshTokenSecret,
				logger:                logger,
				controllerClient:      fakeClient,
				namespace:             "test",
			}

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			u := url.URL{
				Path: path.Join("/v1/refresh"),
			}

			header := http.Header{}
			header.Set("Authorization", "Bearer "+signedRefreshToken)

			c.Request = &http.Request{
				Method: http.MethodPost,
				URL:    &u,
				Header: header,
			}

			h.PostRefresh(c)

			statusCode := w.Result().StatusCode
			if statusCode != tc.expected {
				t.Fatalf("expected status code %d, got %d", tc.expected, statusCode)
			}
		})
	}
}
