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

package handlers_test

import (
	"context"
	"crypto/ecdsa"
	"log/slog"
	"net/http"
	"os"
	"path"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/golang-jwt/jwt/v5"
	dockyardsv1 "github.com/sudoswedenab/dockyards-backend/api/v1alpha3"
	"github.com/sudoswedenab/dockyards-backend/api/v1alpha3/index"
	"github.com/sudoswedenab/dockyards-backend/internal/api/v1/handlers"
	"github.com/sudoswedenab/dockyards-backend/pkg/testing/testingutil"
	utiljwt "github.com/sudoswedenab/dockyards-backend/pkg/util/jwt"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	ctx             context.Context
	cancel          context.CancelFunc
	logger          *slog.Logger
	testEnvironment *testingutil.TestEnvironment
	mux             *http.ServeMux
	accessKey       *ecdsa.PrivateKey
	refreshKey      *ecdsa.PrivateKey
	defaultRelease  *dockyardsv1.Release
)

func TestMain(m *testing.M) {
	var err error

	ctx, cancel = context.WithCancel(context.TODO())
	handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError})

	logger = slog.New(handler)
	slogr := logr.FromSlogHandler(handler)

	ctrl.SetLogger(slogr)

	testEnvironment, err = testingutil.NewTestEnvironment(ctx, []string{path.Join("../../../../config/crd")})
	if err != nil {
		slogr.Error(err, "error creating new test environment")

		os.Exit(1)
	}

	mgr := testEnvironment.GetManager()
	c := testEnvironment.GetClient()

	err = index.AddDefaultIndexes(ctx, mgr)
	if err != nil {
		slogr.Error(err, "error adding default indexers")

		os.Exit(1)
	}

	defaultRelease = &dockyardsv1.Release{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{
				dockyardsv1.AnnotationDefaultRelease: "true",
			},
			GenerateName: "test-",
			Namespace:    testEnvironment.GetDockyardsNamespace(),
		},
		Spec: dockyardsv1.ReleaseSpec{
			Ranges: []string{
				"v1.0.x",
				"v1.1.x",
				"v1.2.x",
			},
			Type: dockyardsv1.ReleaseTypeKubernetes,
		},
	}

	err = c.Create(ctx, defaultRelease)
	if err != nil {
		slogr.Error(err, "error creating test release")

		os.Exit(1)
	}

	patch := client.MergeFrom(defaultRelease.DeepCopy())

	defaultRelease.Status.LatestVersion = "v1.2.3"
	defaultRelease.Status.Versions = []string{
		"v1.0.10",
		"v1.1.7",
		"v1.2.3",
	}

	err = c.Status().Patch(ctx, defaultRelease, patch)
	if err != nil {
		slogr.Error(err, "error preparing test release")

		os.Exit(1)
	}

	go func() {
		err := mgr.Start(ctx)
		if err != nil {
			slogr.Error(err, "error starting test manager")

			os.Exit(1)
		}
	}()

	accessKey, refreshKey, err = utiljwt.GetOrGenerateKeys(ctx, c, testEnvironment.GetDockyardsNamespace())
	if err != nil {
		slogr.Error(err, "error preparing test keys")

		os.Exit(1)
	}

	handlerOptions := []handlers.HandlerOption{
		handlers.WithManager(mgr),
		handlers.WithNamespace(testEnvironment.GetDockyardsNamespace()),
		handlers.WithLogger(logger),
		handlers.WithJWTPrivateKeys(accessKey, refreshKey),
	}

	mux = http.NewServeMux()

	err = handlers.RegisterRoutes(mux, handlerOptions...)
	if err != nil {
		slogr.Error(err, "error registring routes for test mux")

		os.Exit(1)
	}

	code := m.Run()

	cancel()
	err = testEnvironment.GetEnvironment().Stop()
	if err != nil {
		slogr.Error(err, "error stopping test environment")

		os.Exit(1)
	}

	os.Exit(code)
}

func SignToken(subject string, key any) (string, error) {
	claims := jwt.RegisteredClaims{
		Subject:   subject,
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Minute * 30)),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodES256, claims)
	signedToken, err := token.SignedString(key)
	if err != nil {
		return "", err
	}

	return signedToken, nil
}

func MustSignToken(t *testing.T, subject string) string {
	signedToken, err := SignToken(subject, accessKey)
	if err != nil {
		t.Fatal(err)
	}

	return signedToken
}

func MustSignRefreshToken(t *testing.T, subject string) string {
	signedToken, err := SignToken(subject, refreshKey)
	if err != nil {
		t.Fatal(err)
	}

	return signedToken
}
