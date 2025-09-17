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
	"testing"

	"github.com/fluxcd/pkg/runtime/conditions"
	apitypes "github.com/sudoswedenab/dockyards-api/pkg/types"
	dockyardsv1 "github.com/sudoswedenab/dockyards-backend/api/v1alpha3"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestVerificationRequest_Approve(t *testing.T) {
	if os.Getenv("KUBEBUILDER_ASSETS") == "" {
		t.Skip("no kubebuilder assets configured")
	}

	c := testEnvironment.GetClient()

	t.Run("test verify endpoint", func(t *testing.T) {
		vr := dockyardsv1.VerificationRequest{
			ObjectMeta: metav1.ObjectMeta{
				Name: "sign-up-test-user",
			},
			Spec: dockyardsv1.VerificationRequestSpec{
				Code: "test-code",
				UserRef: corev1.TypedLocalObjectReference{
					APIGroup: &dockyardsv1.GroupVersion.Group,
					Kind:     dockyardsv1.UserKind,
					Name:     "test-user",
				},
			},
		}

		err := c.Create(ctx, &vr)
		if err != nil {
			t.Fatal(err)
		}

		verifyOptions := apitypes.VerifyOptions{
			Type: dockyardsv1.RequestTypeAccount,
			Code: vr.Spec.Code,
		}

		b, err := json.Marshal(&verifyOptions)
		if err != nil {
			t.Fatal(err)
		}

		u := url.URL{
			Path: "/v1/verify",
		}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, u.Path, bytes.NewBuffer(b))
		mux.ServeHTTP(w, r)

		statusCode := w.Result().StatusCode
		if statusCode != http.StatusAccepted {
			t.Fatalf("expected status code %d, got %d", http.StatusAccepted, statusCode)
		}

		err = c.Get(ctx, client.ObjectKeyFromObject(&vr), &vr)
		if err != nil {
			t.Fatal(err)
		}

		verified := conditions.Get(&vr, dockyardsv1.VerifiedCondition)
		if verified.Status != metav1.ConditionTrue {
			t.Fatal("expected VerificationRequest to be approved")
		}
	})

	t.Run("test verify wrong code", func(t *testing.T) {
		verifyOptions := apitypes.VerifyOptions{
			Type: dockyardsv1.RequestTypeAccount,
			Code: "wrong-code",
		}

		b, err := json.Marshal(&verifyOptions)
		if err != nil {
			t.Fatal(err)
		}

		u := url.URL{
			Path: "/v1/verify",
		}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, u.Path, bytes.NewBuffer(b))
		mux.ServeHTTP(w, r)

		statusCode := w.Result().StatusCode
		if statusCode != http.StatusUnauthorized {
			t.Fatalf("expected status code %d, got %d", http.StatusUnauthorized, statusCode)
		}
	})

	t.Run("test verify wrong type", func(t *testing.T) {
		verifyOptions := apitypes.VerifyOptions{
			Type: "wrong-type",
			Code: "wrong-code",
		}

		b, err := json.Marshal(&verifyOptions)
		if err != nil {
			t.Fatal(err)
		}

		u := url.URL{
			Path: "/v1/verify",
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
