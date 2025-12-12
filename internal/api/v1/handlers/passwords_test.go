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
	"os"
	"path"
	"testing"
	"time"

	"github.com/sudoswedenab/dockyards-api/pkg/types"
	dockyardsv1 "github.com/sudoswedenab/dockyards-backend/api/v1alpha3"
	"github.com/sudoswedenab/dockyards-backend/pkg/authorization"
	"golang.org/x/crypto/bcrypt"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	corev1 "k8s.io/api/core/v1"
)

func TestUserPassword_Update(t *testing.T) {
	if os.Getenv("KUBEBUILDER_ASSETS") == "" {
		t.Skip("no kubebuilder assets configured")
	}

	c := testEnvironment.GetClient()

	t.Run("test update password", func(t *testing.T) {
		hash, err := bcrypt.GenerateFromPassword([]byte("testing"), bcrypt.DefaultCost)
		if err != nil {
			t.Fatal(err)
		}

		user := dockyardsv1.User{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: "dockyards-",
			},
			Spec: dockyardsv1.UserSpec{
				Password: string(hash),
			},
		}

		err = c.Create(ctx, &user)
		if err != nil {
			t.Fatal(err)
		}

		err = authorization.ReconcileUserAuthorization(ctx, c, &user)
		if err != nil {
			t.Fatal(err)
		}

		userToken := MustSignToken(t, user.Name)

		passwordOptions := types.PasswordOptions{
			OldPassword: "testing",
			NewPassword: "update",
		}

		b, err := json.Marshal(&passwordOptions)
		if err != nil {
			t.Fatal(err)
		}

		u := url.URL{
			Path: path.Join("/v1/users", user.Name, "password"),
		}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, u.Path, bytes.NewBuffer(b))

		r.Header.Add("Authorization", "Bearer "+userToken)

		mux.ServeHTTP(w, r)

		statusCode := w.Result().StatusCode
		if statusCode != http.StatusAccepted {
			t.Fatalf("expected status code %d, got %d", http.StatusAccepted, statusCode)
		}

		var actual dockyardsv1.User
		err = c.Get(ctx, client.ObjectKeyFromObject(&user), &actual)
		if err != nil {
			t.Fatal(err)
		}

		err = bcrypt.CompareHashAndPassword([]byte(actual.Spec.Password), []byte("update"))
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("test incorrect old password", func(t *testing.T) {
		hash, err := bcrypt.GenerateFromPassword([]byte("old"), bcrypt.DefaultCost)
		if err != nil {
			t.Fatal(err)
		}

		user := dockyardsv1.User{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: "dockyards-",
			},
			Spec: dockyardsv1.UserSpec{
				Password: string(hash),
			},
		}

		err = c.Create(ctx, &user)
		if err != nil {
			t.Fatal(err)
		}

		err = authorization.ReconcileUserAuthorization(ctx, c, &user)
		if err != nil {
			t.Fatal(err)
		}

		userToken := MustSignToken(t, user.Name)

		passwordOptions := types.PasswordOptions{
			OldPassword: "testing",
			NewPassword: "update",
		}

		b, err := json.Marshal(&passwordOptions)
		if err != nil {
			t.Fatal(err)
		}

		u := url.URL{
			Path: path.Join("/v1/users", user.Name, "password"),
		}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, u.Path, bytes.NewBuffer(b))

		r.Header.Add("Authorization", "Bearer "+userToken)

		mux.ServeHTTP(w, r)

		statusCode := w.Result().StatusCode
		if statusCode != http.StatusUnauthorized {
			t.Fatalf("expected status code %d, got %d", http.StatusUnauthorized, statusCode)
		}
	})
}

func TestUserPassword_Reset(t *testing.T) {
	if os.Getenv("KUBEBUILDER_ASSETS") == "" {
		t.Skip("no kubebuilder assets configured")
	}

	c := testEnvironment.GetClient()

	t.Run("cannot reset password of user without provider", func(t *testing.T) {
		user := dockyardsv1.User{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: "dockyards-",
			},
			Spec: dockyardsv1.UserSpec{
				Email: "user-without-provider@localhost.local",
			},
		}

		err := c.Create(ctx, &user)
		if err != nil {
			t.Fatal(err)
		}

		var sanityCheck dockyardsv1.VerificationRequestList
		err = c.List(ctx, &sanityCheck)
		if err != nil {
			t.Fatal(err)
		}
		if len(sanityCheck.Items) != 0 {
			t.Fatal("verification request list already has items")
		}

		passwordResetRequestOptions := types.PasswordResetRequestOptions{
			Email: user.Spec.Email,
		}
		b, err := json.Marshal(&passwordResetRequestOptions)
		if err != nil {
			t.Fatal(err)
		}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, "/v1/password-reset-request", bytes.NewBuffer(b))
		mux.ServeHTTP(w, r)

		statusCode := w.Result().StatusCode
		if statusCode != http.StatusInternalServerError {
			t.Fatalf("expected status code %d, got %d", http.StatusInternalServerError, statusCode)
		}

		var result dockyardsv1.VerificationRequestList
		err = c.List(ctx, &result)
		if err != nil {
			t.Fatal(err)
		}
		if len(sanityCheck.Items) != 0 {
			t.Fatal("expected verification request to still not exist after failed create")
		}
	})

	t.Run("create password reset request", func(t *testing.T) {
		user := dockyardsv1.User{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: "dockyards-",
			},
			Spec: dockyardsv1.UserSpec{
				Email: "user-with-provider@localhost.local",
				ProviderID: dockyardsv1.ProviderPrefixDockyards,
			},
		}

		err := c.Create(ctx, &user)
		if err != nil {
			t.Fatal(err)
		}
		var sanityCheck dockyardsv1.VerificationRequestList
		err = c.List(ctx, &sanityCheck)
		if err != nil {
			t.Fatal(err)
		}
		if len(sanityCheck.Items) != 0 {
			t.Fatal("verification request list already has items")
		}

		passwordResetRequestOptions := types.PasswordResetRequestOptions{
			Email: user.Spec.Email,
		}
		b, err := json.Marshal(&passwordResetRequestOptions)
		if err != nil {
			t.Fatal(err)
		}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, "/v1/password-reset-request", bytes.NewBuffer(b))
		mux.ServeHTTP(w, r)

		statusCode := w.Result().StatusCode
		if statusCode != http.StatusAccepted {
			t.Fatalf("expected status code %d, got %d", http.StatusAccepted, statusCode)
		}

		var result dockyardsv1.VerificationRequestList
		err = c.List(ctx, &result)
		if err != nil {
			t.Fatal(err)
		}
		if len(result.Items) != 1 {
			t.Fatalf("expected verification request to exist after create")
		}
	})

	t.Run("use password reset request", func(t *testing.T) {
		user := dockyardsv1.User{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: "dockyards-",
			},
			Spec: dockyardsv1.UserSpec{
				Email: "password-reset@localhost.local",
				ProviderID: dockyardsv1.ProviderPrefixDockyards,
			},
		}

		err := c.Create(ctx, &user)
		if err != nil {
			t.Fatal(err)
		}

		passwordResetRequestOptions := types.PasswordResetRequestOptions{
			Email: user.Spec.Email,
		}
		b, err := json.Marshal(&passwordResetRequestOptions)
		if err != nil {
			t.Fatal(err)
		}

		obj := dockyardsv1.VerificationRequest{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: "password-reset-",
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion: dockyardsv1.GroupVersion.String(),
						Kind:       dockyardsv1.UserKind,
						Name:       user.Name,
						UID:        user.UID,
					},
				},
			},
			Spec: dockyardsv1.VerificationRequestSpec{
				Code: "password-reset-request",
				Duration: &metav1.Duration{Duration: 10 * time.Second},
				UserRef: corev1.TypedLocalObjectReference{
					APIGroup: &dockyardsv1.GroupVersion.Group,
					Kind:     dockyardsv1.UserKind,
					Name:     user.Name,
				},
			},
		}
		err = c.Create(ctx, &obj)
		if err != nil {
			t.Fatal(err)
		}

		passwordResetOptions := types.ResetPasswordOptions{
			ResetCode: "password-reset-request",
			NewPassword: "Foobar2000!",
		}

		b, err = json.Marshal(&passwordResetOptions)
		if err != nil {
			t.Fatal(err)
		}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, "/v1/reset-password", bytes.NewBuffer(b))
		mux.ServeHTTP(w, r)

		statusCode := w.Result().StatusCode
		if statusCode != http.StatusAccepted {
			t.Fatalf("expected status code %d, got %d", http.StatusAccepted, statusCode)
		}

		var actual dockyardsv1.User
		err = c.Get(ctx, client.ObjectKeyFromObject(&user), &actual)
		if err != nil {
			t.Fatal(err)
		}

		err = bcrypt.CompareHashAndPassword([]byte(actual.Spec.Password), []byte("Foobar2000!"))
		if err != nil {
			t.Fatal(err)
		}
	})
}
