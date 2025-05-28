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
	"path"
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

func TestClusterNodes_List(t *testing.T) {
	mgr := testEnvironment.GetManager()
	c := testEnvironment.GetClient()

	organization := testEnvironment.MustCreateOrganization(t)

	superUser := testEnvironment.MustGetOrganizationUser(t, organization, dockyardsv1.OrganizationMemberRoleSuperUser)

	superUserToken := MustSignToken(t, string(superUser.UID))

	cluster := dockyardsv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "test-",
			Namespace:    organization.Spec.NamespaceRef.Name,
		},
	}

	err := c.Create(ctx, &cluster)
	if err != nil {
		t.Fatal(err)
	}

	node := dockyardsv1.Node{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: cluster.Name + "-test-",
			Labels: map[string]string{
				dockyardsv1.LabelClusterName: cluster.Name,
			},
			Namespace: cluster.Namespace,
		},
	}

	err = c.Create(ctx, &node)
	if err != nil {
		t.Fatal(err)
	}

	updated := dockyardsv1.Node{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: cluster.Name + "-test-",
			Labels: map[string]string{
				dockyardsv1.LabelClusterName: cluster.Name,
			},
			Namespace: cluster.Namespace,
		},
	}

	err = c.Create(ctx, &updated)
	if err != nil {
		t.Fatal(err)
	}

	patch := client.MergeFrom(updated.DeepCopy())

	statusCondition := metav1.Condition{
		Type:               dockyardsv1.ReadyCondition,
		Reason:             "testing",
		Status:             metav1.ConditionTrue,
		LastTransitionTime: metav1.Now(),
	}

	meta.SetStatusCondition(&updated.Status.Conditions, statusCondition)

	err = c.Status().Patch(ctx, &updated, patch)
	if err != nil {
		t.Fatal(err)
	}

	deleted := dockyardsv1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Finalizers: []string{
				"backend.dockyards.io/testing",
			},
			GenerateName: cluster.Name + "-test-",
			Labels: map[string]string{
				dockyardsv1.LabelClusterName: cluster.Name,
			},
			Namespace: cluster.Namespace,
		},
	}

	err = c.Create(ctx, &deleted)
	if err != nil {
		t.Fatal(err)
	}

	err = c.Delete(ctx, &deleted)
	if err != nil {
		t.Fatal(err)
	}

	err = c.Get(ctx, client.ObjectKeyFromObject(&deleted), &deleted)
	if err != nil {
		t.Fatal(err)
	}

	expected := []types.Node{
		{
			CreatedAt: node.CreationTimestamp.Time,
			ID:        string(node.UID),
			Name:      node.Name,
		},
		{
			Condition: &statusCondition.Reason,
			CreatedAt: updated.CreationTimestamp.Time,
			ID:        string(updated.UID),
			Name:      updated.Name,
			UpdatedAt: ptr.To(statusCondition.LastTransitionTime.Time.Truncate(time.Second)),
		},
		{
			CreatedAt: deleted.CreationTimestamp.Time,
			DeletedAt: ptr.To(deleted.DeletionTimestamp.Time),
			ID:        string(deleted.UID),
			Name:      deleted.Name,
		},
	}

	err = testingutil.RetryUntilFound(ctx, mgr.GetClient(), &deleted)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("test as super user", func(t *testing.T) {
		u := url.URL{
			Path: path.Join("/v1/orgs", organization.Name, "clusters", cluster.Name, "nodes"),
		}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, u.Path, nil)

		r.Header.Add("Authorization", "Bearer "+superUserToken)

		mux.ServeHTTP(w, r)

		statusCode := w.Result().StatusCode
		if statusCode != http.StatusOK {
			t.Errorf("expected status code %d, got %d", http.StatusOK, statusCode)
		}

		b, err := io.ReadAll(w.Result().Body)
		if err != nil {
			t.Fatal(err)
		}

		var actual []types.Node
		err = json.Unmarshal(b, &actual)
		if err != nil {
			t.Fatal(err)
		}

		if !cmp.Equal(actual, expected) {
			t.Fatalf("diff: %s", cmp.Diff(expected, actual))
		}
	})

	t.Run("test as other user", func(t *testing.T) {
		otherOrganization := testEnvironment.MustCreateOrganization(t)

		otherUser := testEnvironment.MustGetOrganizationUser(t, otherOrganization, dockyardsv1.OrganizationMemberRoleSuperUser)

		otherUserToken := MustSignToken(t, string(otherUser.UID))


		u := url.URL{
			Path: path.Join("/v1/orgs", organization.Name, "clusters", cluster.Name, "nodes"),
		}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, u.Path, nil)

		r.Header.Add("Authorization", "Bearer "+otherUserToken)

		mux.ServeHTTP(w, r)

		statusCode := w.Result().StatusCode
		if statusCode != http.StatusUnauthorized {
			t.Errorf("expected status code %d, got %d", http.StatusUnauthorized, statusCode)
		}
	})
}
