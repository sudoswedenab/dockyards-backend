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
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/sudoswedenab/dockyards-api/pkg/types"
	"github.com/sudoswedenab/dockyards-backend/api/featurenames"
	dockyardsv1 "github.com/sudoswedenab/dockyards-backend/api/v1alpha3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestGlobalUser_Create(t *testing.T) {
	if os.Getenv("KUBEBUILDER_ASSETS") == "" {
		t.Skip("no kubebuilder assets configured")
	}

	c := testEnvironment.GetClient()

	feature := dockyardsv1.Feature{
		ObjectMeta: metav1.ObjectMeta{
			Name:      featurenames.FeatureUserSignUp,
			Namespace: testEnvironment.GetDockyardsNamespace(),
		},
	}

	err := c.Create(ctx, &feature)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("test provider id", func(t *testing.T) {
		userOptions := types.UserOptions{
			Email:    "test@dockyards.dev",
			Password: "testing",
		}

		b, err := json.Marshal(&userOptions)
		if err != nil {
			t.Fatal(err)
		}

		u := url.URL{
			Path: path.Join("/v1/users"),
		}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, u.Path, bytes.NewBuffer(b))

		mux.ServeHTTP(w, r)

		statusCode := w.Result().StatusCode
		if statusCode != http.StatusCreated {
			t.Fatalf("expected status code %d, got %d", http.StatusCreated, statusCode)
		}

		b, err = io.ReadAll(w.Result().Body)
		if err != nil {
			t.Fatal(err)
		}

		var actual types.User
		err = json.Unmarshal(b, &actual)
		if err != nil {
			t.Fatal(err)
		}

		var user dockyardsv1.User
		err = c.Get(ctx, client.ObjectKey{Name: actual.Name}, &user)
		if err != nil {
			t.Fatal(err)
		}

		expected := types.User{
			ID:         string(user.UID),
			CreatedAt:  &user.CreationTimestamp.Time,
			Email:      user.Spec.Email,
			Name:       user.Name,
			ProviderID: ptr.To(dockyardsv1.ProviderPrefixDockyards),
		}

		if !cmp.Equal(actual, expected) {
			t.Errorf("diff: %s", cmp.Diff(expected, actual))
		}
	})
}
