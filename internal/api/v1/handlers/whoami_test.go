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
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/sudoswedenab/dockyards-api/pkg/types"
	dockyardsv1 "github.com/sudoswedenab/dockyards-backend/api/v1alpha3"
	"github.com/sudoswedenab/dockyards-backend/pkg/testing/testingutil"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestWhoami_Get(t *testing.T) {
	mgr := testEnvironment.GetManager()
	c := mgr.GetClient()

	organization := testEnvironment.MustCreateOrganization(t)
	superUser := testEnvironment.MustGetOrganizationUser(t, organization, dockyardsv1.OrganizationMemberRoleSuperUser)
	superUserToken := MustSignToken(t, superUser.Name)

	t.Run("test as super user", func(t *testing.T) {
		u := url.URL{
			Path: "/v1/whoami",
		}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, u.Path, nil)

		r.Header.Add("Authorization", "Bearer "+superUserToken)

		mux.ServeHTTP(w, r)

		statusCode := w.Result().StatusCode
		if statusCode != http.StatusOK {
			t.Fatalf("expected status code %d, got %d", http.StatusOK, statusCode)
		}

		b, err := io.ReadAll(w.Result().Body)
		if err != nil {
			t.Fatal(err)
		}

		var actual types.User
		err = json.Unmarshal(b, &actual)
		if err != nil {
			t.Fatal(err)
		}

		expected := types.User{
			CreatedAt: &superUser.CreationTimestamp.Time,
			Email:     superUser.Spec.Email,
			ID:        string(superUser.UID),
			Name:      superUser.Name,
		}

		if !cmp.Equal(actual, expected) {
			t.Errorf("diff: %s", cmp.Diff(expected, actual))
		}
	})

	t.Run("test as other user", func(t *testing.T) {
		otherUser := dockyardsv1.User{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: "other-",
			},
			Spec: dockyardsv1.UserSpec{
				DisplayName: "other",
				Email:       "other@dockyards.dev",
				ProviderID:  ptr.To("testing://"),
			},
		}

		err := c.Create(ctx, &otherUser)
		if err != nil {
			t.Fatal(err)
		}

		readyCondition := metav1.Condition{
			Type:               dockyardsv1.ReadyCondition,
			Status:             metav1.ConditionTrue,
			Reason:             "testing",
			LastTransitionTime: metav1.Now(),
		}

		patch := client.MergeFrom(otherUser.DeepCopy())

		meta.SetStatusCondition(&otherUser.Status.Conditions, readyCondition)

		err = c.Status().Patch(ctx, &otherUser, patch)
		if err != nil {
			t.Fatal(err)
		}

		otherUserToken := MustSignToken(t, otherUser.Name)

		err = testingutil.RetryUntilFound(ctx, mgr.GetClient(), &otherUser)
		if err != nil {
			t.Fatal(err)
		}

		u := url.URL{
			Path: "/v1/whoami",
		}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, u.Path, nil)

		r.Header.Add("Authorization", "Bearer "+otherUserToken)

		mux.ServeHTTP(w, r)

		statusCode := w.Result().StatusCode
		if statusCode != http.StatusOK {
			t.Fatalf("expected status code %d, got %d", http.StatusOK, statusCode)
		}

		b, err := io.ReadAll(w.Result().Body)
		if err != nil {
			t.Fatal(err)
		}

		var actual types.User
		err = json.Unmarshal(b, &actual)
		if err != nil {
			t.Fatal(err)
		}

		expected := types.User{
			CreatedAt:   &otherUser.CreationTimestamp.Time,
			DisplayName: &otherUser.Spec.DisplayName,
			Email:       otherUser.Spec.Email,
			ID:          string(otherUser.UID),
			Name:        otherUser.Name,
			ProviderID:  otherUser.Spec.ProviderID,
			UpdatedAt:   ptr.To(readyCondition.LastTransitionTime.Time.Truncate(time.Second)),
		}

		if !cmp.Equal(actual, expected) {
			t.Errorf("diff: %s", cmp.Diff(expected, actual))
		}
	})
}
