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

package v2_test

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
	v2 "github.com/sudoswedenab/dockyards-backend/internal/api/v2"
	"github.com/sudoswedenab/dockyards-backend/pkg/testing/testingutil"
	utiljwt "github.com/sudoswedenab/dockyards-backend/pkg/util/jwt"
	ctrl "sigs.k8s.io/controller-runtime"
)

var (
	environment *testingutil.TestEnvironment
	mux         *http.ServeMux
	accessKey   *ecdsa.PrivateKey
)

func TestMain(m *testing.M) {
	ctx, cancel := context.WithCancel(context.Background())

	handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError})
	slogr := logr.FromSlogHandler(handler)

	ctrl.SetLogger(slogr)

	var err error

	environment, err = testingutil.NewTestEnvironment(ctx, []string{path.Join("../../../config/crd")})
	if err != nil {
		panic(err)
	}

	mgr := environment.GetManager()
	c := environment.GetClient()

	go func() {
		err := mgr.Start(ctx)
		if err != nil {
			panic(err)
		}
	}()

	accessKey, _, err = utiljwt.GetOrGenerateKeys(ctx, c, environment.GetDockyardsNamespace())
	if err != nil {
		panic(err)
	}

	mux = http.NewServeMux()

	a := v2.NewAPI(mgr, &accessKey.PublicKey)
	a.RegisterRoutes(mux)

	code := m.Run()

	cancel()
	err = environment.GetEnvironment().Stop()
	if err != nil {
		panic(err)
	}

	os.Exit(code)
}

func MustSignToken(user *dockyardsv1.User) string {
	claims := jwt.RegisteredClaims{
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Minute)),
		Subject:   user.Name,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodES256, claims)
	signed, err := token.SignedString(accessKey)
	if err != nil {
		panic(err)
	}

	return signed
}
