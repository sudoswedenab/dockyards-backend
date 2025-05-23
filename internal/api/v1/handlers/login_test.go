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
	"net/http"
	"net/http/httptest"
	"net/url"
	"path"
	"testing"

	"bitbucket.org/sudosweden/dockyards-api/pkg/types"
	dockyardsv1 "bitbucket.org/sudosweden/dockyards-backend/pkg/api/v1alpha3"
	"bitbucket.org/sudosweden/dockyards-backend/pkg/testing/testingutil"
	"golang.org/x/crypto/bcrypt"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestGlobalTokens_Create(t *testing.T) {
	hash, err := bcrypt.GenerateFromPassword([]byte("password"), 10)
	if err != nil {
		t.Fatalf("unexpected error hashing string 'password'")
	}

	mgr := testEnvironment.GetManager()
	c := testEnvironment.GetClient()

	user := dockyardsv1.User{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "test-",
		},
		Spec: dockyardsv1.UserSpec{
			Password: string(hash),
		},
		Status: dockyardsv1.UserStatus{
			Conditions: []metav1.Condition{},
		},
	}

	err = c.Create(ctx, &user)
	if err != nil {
		t.Fatal(err)
	}

	patch := client.MergeFrom(user.DeepCopy())

	user.Spec.Email = user.Name + "@dockyards.dev"

	err = c.Patch(ctx, &user, patch)
	if err != nil {
		t.Fatal(err)
	}

	patch = client.MergeFrom(user.DeepCopy())

	user.Status.Conditions = []metav1.Condition{
		{
			Type:               dockyardsv1.ReadyCondition,
			Status:             metav1.ConditionTrue,
			Reason:             "testing",
			LastTransitionTime: metav1.Now(),
		},
	}

	err = c.Status().Patch(ctx, &user, patch)
	if err != nil {
		t.Fatal(err)
	}

	err = testingutil.RetryUntilFound(ctx, mgr.GetClient(), &user)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("test password", func(t *testing.T) {
		options := types.LoginOptions{
			Email:    user.Spec.Email,
			Password: "password",
		}

		b, err := json.Marshal(&options)
		if err != nil {
			t.Fatal(err)
		}

		u := url.URL{
			Path: path.Join("/v1/login"),
		}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, u.Path, bytes.NewBuffer(b))

		mux.ServeHTTP(w, r)

		statusCode := w.Result().StatusCode
		if statusCode != http.StatusCreated {
			t.Fatalf("expected status code %d, got %d", http.StatusCreated, statusCode)
		}
	})

	t.Run("test invalid password", func(t *testing.T) {
		options := types.LoginOptions{
			Email:    user.Spec.Email,
			Password: "invalid",
		}

		b, err := json.Marshal(&options)
		if err != nil {
			t.Fatal(err)
		}

		u := url.URL{
			Path: path.Join("/v1/login"),
		}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, u.Path, bytes.NewBuffer(b))

		mux.ServeHTTP(w, r)

		statusCode := w.Result().StatusCode
		if statusCode != http.StatusUnauthorized {
			t.Fatalf("expected status code %d, got %d", http.StatusUnauthorized, statusCode)
		}
	})

	t.Run("test non-existing user", func(t *testing.T) {
		options := types.LoginOptions{
			Email:    "non-existing@dockyards.dev",
			Password: "password",
		}

		b, err := json.Marshal(&options)
		if err != nil {
			t.Fatal(err)
		}

		u := url.URL{
			Path: path.Join("/v1/login"),
		}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, u.Path, bytes.NewBuffer(b))

		mux.ServeHTTP(w, r)

		statusCode := w.Result().StatusCode
		if statusCode != http.StatusUnauthorized {
			t.Fatalf("expected status code %d, got %d", http.StatusUnauthorized, statusCode)
		}

	})

	t.Run("test missing condition", func(t *testing.T) {
		otherUser := dockyardsv1.User{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: "test-",
			},
			Spec: dockyardsv1.UserSpec{
				Password: string(hash),
			},
		}

		err := c.Create(ctx, &otherUser)
		if err != nil {
			t.Fatal(err)
		}

		patch := client.MergeFrom(otherUser.DeepCopy())

		user.Spec.Email = otherUser.Name + "@dockyards.dev"

		err = c.Patch(ctx, &otherUser, patch)
		if err != nil {
			t.Fatal(err)
		}

		err = testingutil.RetryUntilFound(ctx, mgr.GetClient(), &otherUser)
		if err != nil {
			t.Fatal(err)
		}

		options := types.LoginOptions{
			Email:    otherUser.Spec.Email,
			Password: "password",
		}

		b, err := json.Marshal(&options)
		if err != nil {
			t.Fatal(err)
		}

		u := url.URL{
			Path: path.Join("/v1/login"),
		}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, u.Path, bytes.NewBuffer(b))

		mux.ServeHTTP(w, r)

		statusCode := w.Result().StatusCode
		if statusCode != http.StatusUnauthorized {
			t.Fatalf("expected status code %d, got %d", http.StatusUnauthorized, statusCode)
		}
	})
}
