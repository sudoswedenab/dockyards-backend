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

	"github.com/golang-jwt/jwt/v5"
	"github.com/sudoswedenab/dockyards-backend/api/v1alpha3/index"
	"github.com/sudoswedenab/dockyards-backend/internal/api/v1/handlers"
	"github.com/sudoswedenab/dockyards-backend/pkg/testing/testingutil"
	utiljwt "github.com/sudoswedenab/dockyards-backend/pkg/util/jwt"
)

var (
	ctx             context.Context
	cancel          context.CancelFunc
	logger          *slog.Logger
	testEnvironment *testingutil.TestEnvironment
	mux             *http.ServeMux
	accessKey       *ecdsa.PrivateKey
	refreshKey      *ecdsa.PrivateKey
)

func TestMain(m *testing.M) {
	var err error

	ctx, cancel = context.WithCancel(context.TODO())
	logger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

	testEnvironment, err = testingutil.NewTestEnvironment(ctx, []string{path.Join("../../../../config/crd")})
	if err != nil {
		panic(err)
	}

	mgr := testEnvironment.GetManager()
	c := testEnvironment.GetClient()

	err = index.AddDefaultIndexes(ctx, mgr)
	if err != nil {
		panic(err)
	}

	go func() {
		err := mgr.Start(ctx)
		if err != nil {
			panic(err)
		}
	}()

	accessKey, refreshKey, err = utiljwt.GetOrGenerateKeys(ctx, c, testEnvironment.GetDockyardsNamespace())
	if err != nil {
		panic(err)
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
		panic(err)
	}

	code := m.Run()

	cancel()
	err = testEnvironment.GetEnvironment().Stop()
	if err != nil {
		panic(err)
	}

	os.Exit(code)
}

func SignToken(subject string) (string, error) {
	claims := jwt.RegisteredClaims{
		Subject:   subject,
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Minute * 30)),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodES256, claims)
	signedToken, err := token.SignedString(accessKey)
	if err != nil {
		return "", err
	}

	return signedToken, nil
}

func MustSignToken(t *testing.T, subject string) string {
	signedToken, err := SignToken(subject)
	if err != nil {
		t.Fatal(err)
	}

	return signedToken
}
