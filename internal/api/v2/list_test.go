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
	"encoding/json"
	"os"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	dockyardsv1 "github.com/sudoswedenab/dockyards-backend/api/v1alpha3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestNamespacedResource_List(t *testing.T) {
	if os.Getenv("USE_EXISTING_CLUSTER") != "true" {
		t.Skip("cannot run test in epehemeral cluster")
	}

	ctx := t.Context()

	organization := environment.MustCreateOrganization(t)
	superUser := environment.MustGetOrganizationUser(t, organization, dockyardsv1.RoleSuperUser)

	c := environment.GetClient()

	clusters := []dockyardsv1.Cluster{
		{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: "test-",
				Namespace:    organization.Spec.NamespaceRef.Name,
			},
		},
	}

	for i := range clusters {
		err := c.Create(ctx, &clusters[i])
		if err != nil {
			t.Fatal(err)
		}
	}

	nodes := []dockyardsv1.Node{
		{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: "test-",
				Labels: map[string]string{
					dockyardsv1.LabelClusterName: "a",
				},
				Namespace: organization.Spec.NamespaceRef.Name,
			},
		},
		{
			TypeMeta: metav1.TypeMeta{
				Kind:       dockyardsv1.NodeKind,
				APIVersion: dockyardsv1.GroupVersion.String(),
			},
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: "test-",
				Labels: map[string]string{
					dockyardsv1.LabelClusterName: "b",
				},
				Namespace: organization.Spec.NamespaceRef.Name,
			},
		},
	}

	for i := range nodes {
		err := c.Create(ctx, &nodes[i])
		if err != nil {
			t.Fatal(err)
		}
	}

	ignoreTypeMeta := cmpopts.IgnoreTypes(metav1.TypeMeta{})
	ignoreListMeta := cmpopts.IgnoreFields(metav1.ListMeta{}, "ResourceVersion")

	sortNodeByID := cmpopts.SortSlices(func(a, b dockyardsv1.Node) bool {
		return a.UID < b.UID
	})

	t.Run("test clusters as super user", func(t *testing.T) {
		target, err := url.JoinPath("/v2",
			"group", dockyardsv1.GroupVersion.Group,
			"version", dockyardsv1.GroupVersion.Version,
			"kind", "clusters",
			"namespace", organization.Spec.NamespaceRef.Name,
		)
		if err != nil {
			t.Fatal(err)
		}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, target, nil)

		r.Header.Add("Authorization", "Bearer "+MustSignToken(superUser))

		mux.ServeHTTP(w, r)

		if w.Result().StatusCode != http.StatusOK {
			t.Fatalf("unexpected status code %d", w.Result().StatusCode)
		}

		b, err := io.ReadAll(w.Result().Body)
		if err != nil {
			t.Fatal(err)
		}

		var actual dockyardsv1.ClusterList
		err = json.Unmarshal(b, &actual)
		if err != nil {
			t.Fatal(err)
		}

		expected := dockyardsv1.ClusterList{
			TypeMeta: metav1.TypeMeta{
				Kind:       "ClusterList",
				APIVersion: dockyardsv1.GroupVersion.String(),
			},
			Items: clusters,
		}

		if !cmp.Equal(actual, expected, ignoreListMeta, ignoreTypeMeta) {
			t.Errorf("diff: %s", cmp.Diff(expected, actual, ignoreListMeta, ignoreTypeMeta))
		}
	})

	t.Run("test clusters as other user", func(t *testing.T) {
		otherOrganization := environment.MustCreateOrganization(t)
		otherUser := environment.MustGetOrganizationUser(t, otherOrganization, dockyardsv1.RoleSuperUser)

		target, err := url.JoinPath("/v2",
			"group", dockyardsv1.GroupVersion.Group,
			"version", dockyardsv1.GroupVersion.Version,
			"kind", "clusters",
			"namespace", organization.Spec.NamespaceRef.Name,
		)
		if err != nil {
			t.Fatal(err)
		}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, target, nil)

		r.Header.Add("Authorization", "Bearer "+MustSignToken(otherUser))

		mux.ServeHTTP(w, r)

		if w.Result().StatusCode != http.StatusUnauthorized {
			t.Fatalf("unexpected status code %d", w.Result().StatusCode)
		}
	})

	t.Run("test nodes as user", func(t *testing.T) {
		target, err := url.JoinPath("/v2",
			"group", dockyardsv1.GroupVersion.Group,
			"version", dockyardsv1.GroupVersion.Version,
			"kind", "nodes",
			"namespace", organization.Spec.NamespaceRef.Name,
		)
		if err != nil {
			t.Fatal(err)
		}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, target, nil)

		r.Header.Add("Authorization", "Bearer "+MustSignToken(superUser))

		mux.ServeHTTP(w, r)

		if w.Result().StatusCode != http.StatusOK {
			t.Fatalf("unexpected status code %d", w.Result().StatusCode)
		}

		b, err := io.ReadAll(w.Result().Body)
		if err != nil {
			t.Fatal(err)
		}

		var actual dockyardsv1.NodeList
		err = json.Unmarshal(b, &actual)
		if err != nil {
			t.Fatal(err)
		}

		expected := dockyardsv1.NodeList{
			TypeMeta: metav1.TypeMeta{
				Kind:       "NodeList",
				APIVersion: dockyardsv1.GroupVersion.String(),
			},
			Items: nodes,
		}

		if !cmp.Equal(actual, expected, ignoreListMeta, sortNodeByID, ignoreTypeMeta) {
			t.Errorf("diff: %s", cmp.Diff(expected, actual, ignoreListMeta, sortNodeByID, ignoreTypeMeta))
		}
	})

	t.Run("test nodes label selector", func(t *testing.T) {
		target, err := url.JoinPath("/v2",
			"group", dockyardsv1.GroupVersion.Group,
			"version", dockyardsv1.GroupVersion.Version,
			"kind", "nodes",
			"namespace", organization.Spec.NamespaceRef.Name,
		)
		if err != nil {
			t.Fatal(err)
		}

		v := url.Values{}
		v.Add("labelSelector", "dockyards.io/cluster-name=a")

		u := url.URL{
			Path:     target,
			RawQuery: v.Encode(),
		}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, u.String(), nil)

		r.Header.Add("Authorization", "Bearer "+MustSignToken(superUser))

		mux.ServeHTTP(w, r)

		if w.Result().StatusCode != http.StatusOK {
			t.Fatalf("unexpected status code %d", w.Result().StatusCode)
		}

		b, err := io.ReadAll(w.Result().Body)
		if err != nil {
			t.Fatal(err)
		}
		
		var actual dockyardsv1.NodeList
		err = json.Unmarshal(b, &actual)
		if err != nil {
			t.Fatal(err)
		}

		expected := dockyardsv1.NodeList{
			TypeMeta: metav1.TypeMeta{
				Kind:       "NodeList",
				APIVersion: dockyardsv1.GroupVersion.String(),
			},
			Items: []dockyardsv1.Node{
				nodes[0],
			},
		}

		if !cmp.Equal(actual, expected, ignoreListMeta, sortNodeByID, ignoreTypeMeta) {
			t.Errorf("diff: %s", cmp.Diff(expected, actual, ignoreListMeta, sortNodeByID, ignoreTypeMeta))
		}
	})

	t.Run("test invalid version", func(t *testing.T) {
		target, err := url.JoinPath("/v2",
			"group", dockyardsv1.GroupVersion.Group,
			"version", "v1alpha1",
			"kind", "nodes",
			"namespace", organization.Spec.NamespaceRef.Name,
		)
		if err != nil {
			t.Fatal(err)
		}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, target, nil)

		r.Header.Add("Authorization", "Bearer "+MustSignToken(superUser))

		mux.ServeHTTP(w, r)

		if w.Result().StatusCode != http.StatusNotFound {
			t.Fatalf("unexpected status code %d", w.Result().StatusCode)
		}
	})
}
