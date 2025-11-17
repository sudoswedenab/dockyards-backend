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
	dockyardsv1 "github.com/sudoswedenab/dockyards-backend/api/v1alpha3"
	"github.com/sudoswedenab/dockyards-backend/pkg/testing/testingutil"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestClusterNodePools_Get(t *testing.T) {
	if os.Getenv("KUBEBUILDER_ASSETS") == "" {
		t.Skip("no kubebuilder assets configured")
	}

	mgr := testEnvironment.GetManager()
	c := testEnvironment.GetClient()

	organization := testEnvironment.MustCreateOrganization(t)

	superUser := testEnvironment.MustGetOrganizationUser(t, organization, dockyardsv1.RoleSuperUser)
	user := testEnvironment.MustGetOrganizationUser(t, organization, dockyardsv1.RoleUser)
	reader := testEnvironment.MustGetOrganizationUser(t, organization, dockyardsv1.RoleReader)

	superUserToken := MustSignToken(t, superUser.Name)
	userToken := MustSignToken(t, user.Name)
	readerToken := MustSignToken(t, reader.Name)

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

	nodePool := dockyardsv1.NodePool{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "test-",
			Namespace:    organization.Spec.NamespaceRef.Name,
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: dockyardsv1.GroupVersion.String(),
					Kind:       dockyardsv1.ClusterKind,
					Name:       cluster.Name,
					UID:        cluster.UID,
				},
			},
			Labels: map[string]string{
				dockyardsv1.LabelClusterName: cluster.Name,
			},
		},
	}

	err = c.Create(ctx, &nodePool)
	if err != nil {
		t.Fatal(err)
	}

	node := dockyardsv1.Node{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "test-",
			Namespace:    organization.Spec.NamespaceRef.Name,
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: dockyardsv1.GroupVersion.String(),
					Kind:       dockyardsv1.NodePoolKind,
					Name:       nodePool.Name,
					UID:        nodePool.UID,
				},
			},
			Labels: map[string]string{
				dockyardsv1.LabelNodePoolName: nodePool.Name,
			},
		},
	}

	err = c.Create(ctx, &node)
	if err != nil {
		t.Fatal(err)
	}

	err = testingutil.RetryUntilFound(ctx, mgr.GetClient(), &node)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("test as super user", func(t *testing.T) {
		u := url.URL{
			Path: path.Join("/v1/orgs", organization.Name, "clusters", cluster.Name, "node-pools", nodePool.Name),
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
			t.Fatalf("unexpected error reading result body: %s", err)
		}

		var actual types.NodePool
		err = json.Unmarshal(b, &actual)
		if err != nil {
			t.Fatalf("error unmarshalling result body to json: %s", err)
		}

		expected := types.NodePool{
			CreatedAt: nodePool.CreationTimestamp.Time,
			ID:        string(nodePool.UID),
			Name:      nodePool.Name,
			Nodes: &[]types.Node{
				{
					CreatedAt: node.CreationTimestamp.Time,
					ID:        string(node.UID),
					Name:      node.Name,
				},
			},
		}

		if !cmp.Equal(actual, expected) {
			t.Errorf("diff: %s", cmp.Diff(expected, actual))
		}
	})

	t.Run("test as user", func(t *testing.T) {
		u := url.URL{
			Path: path.Join("/v1/orgs", organization.Name, "clusters", cluster.Name, "node-pools", nodePool.Name),
		}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, u.Path, nil)

		r.Header.Add("Authorization", "Bearer "+userToken)

		mux.ServeHTTP(w, r)

		statusCode := w.Result().StatusCode
		if statusCode != http.StatusOK {
			t.Fatalf("expected status code %d, got %d", http.StatusOK, statusCode)
		}

		b, err := io.ReadAll(w.Result().Body)
		if err != nil {
			t.Fatalf("unexpected error reading result body: %s", err)
		}

		var actual types.NodePool
		err = json.Unmarshal(b, &actual)
		if err != nil {
			t.Fatalf("error unmarshalling result body to json: %s", err)
		}

		expected := types.NodePool{
			CreatedAt: nodePool.CreationTimestamp.Time,
			ID:        string(nodePool.UID),
			Name:      nodePool.Name,
			Nodes: &[]types.Node{
				{
					CreatedAt: node.CreationTimestamp.Time,
					ID:        string(node.UID),
					Name:      node.Name,
				},
			},
		}

		if !cmp.Equal(actual, expected) {
			t.Errorf("diff: %s", cmp.Diff(expected, actual))
		}
	})

	t.Run("test as reader", func(t *testing.T) {
		u := url.URL{
			Path: path.Join("/v1/orgs", organization.Name, "clusters", cluster.Name, "node-pools", nodePool.Name),
		}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, u.Path, nil)

		r.Header.Add("Authorization", "Bearer "+readerToken)

		mux.ServeHTTP(w, r)

		statusCode := w.Result().StatusCode
		if statusCode != http.StatusOK {
			t.Fatalf("expected status code %d, got %d", http.StatusOK, statusCode)
		}

		b, err := io.ReadAll(w.Result().Body)
		if err != nil {
			t.Fatalf("unexpected error reading result body: %s", err)
		}

		var actual types.NodePool
		err = json.Unmarshal(b, &actual)
		if err != nil {
			t.Fatalf("error unmarshalling result body to json: %s", err)
		}

		expected := types.NodePool{
			CreatedAt: nodePool.CreationTimestamp.Time,
			ID:        string(nodePool.UID),
			Name:      nodePool.Name,
			Nodes: &[]types.Node{
				{
					CreatedAt: node.CreationTimestamp.Time,
					ID:        string(node.UID),
					Name:      node.Name,
				},
			},
		}

		if !cmp.Equal(actual, expected) {
			t.Errorf("diff: %s", cmp.Diff(expected, actual))
		}
	})

	t.Run("test non-existing node pool", func(t *testing.T) {
		u := url.URL{
			Path: path.Join("/v1/orgs", organization.Name, "clusters", cluster.Name, "node-pools", "non-existing"),
		}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, u.Path, nil)

		r.Header.Add("Authorization", "Bearer "+superUserToken)

		mux.ServeHTTP(w, r)

		statusCode := w.Result().StatusCode
		if statusCode != http.StatusNotFound {
			t.Fatalf("expected status code %d, got %d", http.StatusNotFound, statusCode)
		}
	})

	t.Run("test without membership", func(t *testing.T) {
		namespace := corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: "test-",
			},
		}

		err = c.Create(ctx, &namespace)
		if err != nil {
			t.Fatal(err)
		}

		otherOrganization := dockyardsv1.Organization{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: "test-",
			},
			Spec: dockyardsv1.OrganizationSpec{
				MemberRefs: []dockyardsv1.OrganizationMemberReference{
					{
						Role: dockyardsv1.OrganizationMemberRoleSuperUser,
						UID:  user.UID,
					},
					{
						Role: dockyardsv1.OrganizationMemberRoleUser,
						UID:  reader.UID,
					},
				},
				NamespaceRef: &corev1.LocalObjectReference{
					Name: namespace.Name,
				},
			},
		}

		err := c.Create(ctx, &otherOrganization)
		if err != nil {
			t.Fatal(err)
		}

		otherCluster := dockyardsv1.Cluster{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: "test-",
				Namespace:    namespace.Name,
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion: dockyardsv1.GroupVersion.String(),
						Kind:       dockyardsv1.OrganizationKind,
						Name:       otherOrganization.Name,
						UID:        otherOrganization.UID,
					},
				},
			},
		}

		err = c.Create(ctx, &otherCluster)
		if err != nil {
			t.Fatal(err)
		}

		otherNodePool := dockyardsv1.NodePool{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: "test-",
				Namespace:    namespace.Name,
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion: dockyardsv1.GroupVersion.String(),
						Kind:       dockyardsv1.ClusterKind,
						Name:       otherCluster.Name,
						UID:        otherCluster.UID,
					},
				},
			},
		}

		err = c.Create(ctx, &otherNodePool)
		if err != nil {
			t.Fatal(err)
		}

		err = testingutil.RetryUntilFound(ctx, mgr.GetClient(), &otherNodePool)
		if err != nil {
			t.Fatal(err)
		}

		u := url.URL{
			Path: path.Join("/v1/orgs", otherOrganization.Name, "clusters", otherCluster.Name, "node-pools", otherNodePool.Name),
		}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, u.Path, nil)

		r.Header.Add("Authorization", "Bearer "+superUserToken)

		mux.ServeHTTP(w, r)

		statusCode := w.Result().StatusCode
		if statusCode != http.StatusUnauthorized {
			t.Fatalf("expected status code %d, got %d", http.StatusUnauthorized, statusCode)
		}
	})
}

func TestClusterNodePools_Delete(t *testing.T) {
	if os.Getenv("KUBEBUILDER_ASSETS") == "" {
		t.Skip("no kubebuilder assets configured")
	}

	mgr := testEnvironment.GetManager()
	c := testEnvironment.GetClient()

	organization := testEnvironment.MustCreateOrganization(t)

	superUser := testEnvironment.MustGetOrganizationUser(t, organization, dockyardsv1.RoleSuperUser)
	user := testEnvironment.MustGetOrganizationUser(t, organization, dockyardsv1.RoleUser)
	reader := testEnvironment.MustGetOrganizationUser(t, organization, dockyardsv1.RoleReader)

	superUserToken := MustSignToken(t, superUser.Name)
	userToken := MustSignToken(t, user.Name)
	readerToken := MustSignToken(t, reader.Name)

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

	t.Run("test as super user", func(t *testing.T) {
		nodePool := dockyardsv1.NodePool{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: cluster.Name + "-delete-super-user",
				Namespace:    cluster.Namespace,
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion: dockyardsv1.GroupVersion.String(),
						Kind:       dockyardsv1.ClusterKind,
						Name:       cluster.Name,
						UID:        cluster.UID,
					},
				},
			},
		}

		err := c.Create(ctx, &nodePool)
		if err != nil {
			t.Fatal(err)
		}

		err = testingutil.RetryUntilFound(ctx, mgr.GetClient(), &nodePool)
		if err != nil {
			t.Fatal(err)
		}

		u := url.URL{
			Path: path.Join("/v1/orgs", organization.Name, "clusters", cluster.Name, "node-pools", nodePool.Name),
		}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodDelete, u.Path, nil)

		r.Header.Add("Authorization", "Bearer "+superUserToken)

		mux.ServeHTTP(w, r)

		statusCode := w.Result().StatusCode
		if statusCode != http.StatusAccepted {
			t.Fatalf("expected status code %d, got %d", http.StatusAccepted, statusCode)
		}
	})

	t.Run("test as user", func(t *testing.T) {
		nodePool := dockyardsv1.NodePool{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: "delete-as-user",
				Namespace:    cluster.Namespace,
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion: dockyardsv1.GroupVersion.String(),
						Kind:       dockyardsv1.ClusterKind,
						Name:       cluster.Name,
						UID:        cluster.UID,
					},
				},
			},
		}

		err := c.Create(ctx, &nodePool)
		if err != nil {
			t.Fatal(err)
		}

		err = testingutil.RetryUntilFound(ctx, mgr.GetClient(), &nodePool)
		if err != nil {
			t.Fatal(err)
		}

		u := url.URL{
			Path: path.Join("/v1/orgs", organization.Name, "clusters", cluster.Name, "node-pools", nodePool.Name),
		}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodDelete, u.Path, nil)

		r.Header.Add("Authorization", "Bearer "+userToken)

		mux.ServeHTTP(w, r)

		statusCode := w.Result().StatusCode
		if statusCode != http.StatusAccepted {
			t.Fatalf("expected status code %d, got %d", http.StatusAccepted, statusCode)
		}
	})

	t.Run("test as reader", func(t *testing.T) {
		nodePool := dockyardsv1.NodePool{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: "delete-as-reader",
				Namespace:    cluster.Namespace,
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion: dockyardsv1.GroupVersion.String(),
						Kind:       dockyardsv1.ClusterKind,
						Name:       cluster.Name,
						UID:        cluster.UID,
					},
				},
			},
		}

		err := c.Create(ctx, &nodePool)
		if err != nil {
			t.Fatal(err)
		}

		err = testingutil.RetryUntilFound(ctx, mgr.GetClient(), &nodePool)
		if err != nil {
			t.Fatal(err)
		}

		u := url.URL{
			Path: path.Join("/v1/orgs", organization.Name, "clusters", cluster.Name, "node-pools", nodePool.Name),
		}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodDelete, u.Path, nil)

		r.Header.Add("Authorization", "Bearer "+readerToken)

		mux.ServeHTTP(w, r)

		statusCode := w.Result().StatusCode
		if statusCode != http.StatusUnauthorized {
			t.Fatalf("expected status code %d, got %d", http.StatusUnauthorized, statusCode)
		}
	})

	t.Run("test non-existing node pool", func(t *testing.T) {
		u := url.URL{
			Path: path.Join("/v1/orgs", organization.Name, "clusters", cluster.Name, "node-pools", "non-existing"),
		}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodDelete, u.Path, nil)

		r.Header.Add("Authorization", "Bearer "+superUserToken)

		mux.ServeHTTP(w, r)

		statusCode := w.Result().StatusCode
		if statusCode != http.StatusNotFound {
			t.Fatalf("expected status code %d, got %d", http.StatusNotFound, statusCode)
		}
	})

	t.Run("test without membership", func(t *testing.T) {
		namespace := corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: "test-",
			},
		}

		err = c.Create(ctx, &namespace)
		if err != nil {
			t.Fatal(err)
		}

		otherOrganization := dockyardsv1.Organization{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: "test-",
			},
			Spec: dockyardsv1.OrganizationSpec{
				MemberRefs: []dockyardsv1.OrganizationMemberReference{
					{
						Role: dockyardsv1.OrganizationMemberRoleSuperUser,
						UID:  user.UID,
					},
					{
						Role: dockyardsv1.OrganizationMemberRoleUser,
						UID:  reader.UID,
					},
				},
				NamespaceRef: &corev1.LocalObjectReference{
					Name: namespace.Name,
				},
			},
		}

		err := c.Create(ctx, &otherOrganization)
		if err != nil {
			t.Fatal(err)
		}

		otherCluster := dockyardsv1.Cluster{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: "test-",
				Namespace:    namespace.Name,
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion: dockyardsv1.GroupVersion.String(),
						Kind:       dockyardsv1.OrganizationKind,
						Name:       otherOrganization.Name,
						UID:        otherOrganization.UID,
					},
				},
			},
		}

		err = c.Create(ctx, &otherCluster)
		if err != nil {
			t.Fatal(err)
		}

		otherNodePool := dockyardsv1.NodePool{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: "test-",
				Namespace:    namespace.Name,
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion: dockyardsv1.GroupVersion.String(),
						Kind:       dockyardsv1.ClusterKind,
						Name:       otherCluster.Name,
						UID:        otherCluster.UID,
					},
				},
			},
		}

		err = c.Create(ctx, &otherNodePool)
		if err != nil {
			t.Fatal(err)
		}

		err = testingutil.RetryUntilFound(ctx, mgr.GetClient(), &otherNodePool)
		if err != nil {
			t.Fatal(err)
		}

		u := url.URL{
			Path: path.Join("/v1/orgs", otherOrganization.Name, "clusters", otherCluster.Name, "node-pools", otherNodePool.Name),
		}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodDelete, u.Path, nil)

		r.Header.Add("Authorization", "Bearer "+superUserToken)

		mux.ServeHTTP(w, r)

		statusCode := w.Result().StatusCode
		if statusCode != http.StatusUnauthorized {
			t.Fatalf("expected status code %d, got %d", http.StatusUnauthorized, statusCode)
		}
	})
}

func TestClusterNodePools_Update(t *testing.T) {
	if os.Getenv("KUBEBUILDER_ASSETS") == "" {
		t.Skip("no kubebuilder assets configured")
	}

	mgr := testEnvironment.GetManager()
	c := testEnvironment.GetClient()

	organization := testEnvironment.MustCreateOrganization(t)

	superUser := testEnvironment.MustGetOrganizationUser(t, organization, dockyardsv1.RoleSuperUser)
	user := testEnvironment.MustGetOrganizationUser(t, organization, dockyardsv1.RoleUser)
	reader := testEnvironment.MustGetOrganizationUser(t, organization, dockyardsv1.RoleReader)

	superUserToken := MustSignToken(t, superUser.Name)
	userToken := MustSignToken(t, user.Name)
	readerToken := MustSignToken(t, reader.Name)

	cluster := dockyardsv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "test-update-node-pool-",
			Namespace:    organization.Spec.NamespaceRef.Name,
		},
	}

	err := c.Create(ctx, &cluster)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("test cpu as super user", func(t *testing.T) {
		nodePool := dockyardsv1.NodePool{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: "test-cpu-super-user-",
				Namespace:    cluster.Namespace,
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion: dockyardsv1.GroupVersion.String(),
						Kind:       dockyardsv1.ClusterKind,
						Name:       cluster.Name,
						UID:        cluster.UID,
					},
				},
			},
			Spec: dockyardsv1.NodePoolSpec{
				Resources: corev1.ResourceList{
					corev1.ResourceCPU: resource.MustParse("2"),
				},
			},
		}

		err := c.Create(ctx, &nodePool)
		if err != nil {
			t.Fatal(err)
		}

		err = testingutil.RetryUntilFound(ctx, mgr.GetClient(), &nodePool)
		if err != nil {
			t.Fatal(err)
		}

		update := types.NodePoolOptions{
			CPUCount: ptr.To(3),
		}

		u := url.URL{
			Path: path.Join("/v1/orgs", organization.Name, "clusters", cluster.Name, "node-pools", nodePool.Name),
		}

		w := httptest.NewRecorder()

		b, err := json.Marshal(update)
		if err != nil {
			t.Fatal(err)
		}

		r := httptest.NewRequest(http.MethodPatch, u.Path, bytes.NewBuffer(b))

		r.Header.Add("Authorization", "Bearer "+superUserToken)

		mux.ServeHTTP(w, r)

		statusCode := w.Result().StatusCode
		if statusCode != http.StatusAccepted {
			t.Fatalf("expected status code %d, got %d", http.StatusAccepted, statusCode)
		}

		var actual dockyardsv1.NodePool
		err = c.Get(ctx, client.ObjectKeyFromObject(&nodePool), &actual)
		if err != nil {
			t.Fatal(err)
		}

		expected := dockyardsv1.NodePool{
			ObjectMeta: actual.ObjectMeta,
			Status:     actual.Status,
			Spec: dockyardsv1.NodePoolSpec{
				Resources: corev1.ResourceList{
					corev1.ResourceCPU: resource.MustParse("3"),
				},
			},
		}

		if !cmp.Equal(actual, expected) {
			t.Errorf("diff: %s", cmp.Diff(expected, actual))
		}
	})

	t.Run("test cpu as user", func(t *testing.T) {
		nodePool := dockyardsv1.NodePool{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: "test-cpu-user-",
				Namespace:    cluster.Namespace,
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion: dockyardsv1.GroupVersion.String(),
						Kind:       dockyardsv1.ClusterKind,
						Name:       cluster.Name,
						UID:        cluster.UID,
					},
				},
			},
			Spec: dockyardsv1.NodePoolSpec{
				Resources: corev1.ResourceList{
					corev1.ResourceCPU: resource.MustParse("2"),
				},
			},
		}

		err := c.Create(ctx, &nodePool)
		if err != nil {
			t.Fatal(err)
		}

		err = testingutil.RetryUntilFound(ctx, mgr.GetClient(), &nodePool)
		if err != nil {
			t.Fatal(err)
		}

		update := types.NodePoolOptions{
			CPUCount: ptr.To(3),
		}

		u := url.URL{
			Path: path.Join("/v1/orgs", organization.Name, "clusters", cluster.Name, "node-pools", nodePool.Name),
		}

		w := httptest.NewRecorder()

		b, err := json.Marshal(update)
		if err != nil {
			t.Fatal(err)
		}

		r := httptest.NewRequest(http.MethodPatch, u.Path, bytes.NewBuffer(b))

		r.Header.Add("Authorization", "Bearer "+userToken)

		mux.ServeHTTP(w, r)

		statusCode := w.Result().StatusCode
		if statusCode != http.StatusAccepted {
			t.Fatalf("expected status code %d, got %d", http.StatusAccepted, statusCode)
		}

		var actual dockyardsv1.NodePool
		err = c.Get(ctx, client.ObjectKeyFromObject(&nodePool), &actual)
		if err != nil {
			t.Fatal(err)
		}

		expected := dockyardsv1.NodePool{
			ObjectMeta: actual.ObjectMeta,
			Status:     actual.Status,
			Spec: dockyardsv1.NodePoolSpec{
				Resources: corev1.ResourceList{
					corev1.ResourceCPU: resource.MustParse("3"),
				},
			},
		}

		if !cmp.Equal(actual, expected) {
			t.Errorf("diff: %s", cmp.Diff(expected, actual))
		}
	})

	t.Run("test cpu as reader", func(t *testing.T) {
		nodePool := dockyardsv1.NodePool{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: "test-cpu-reader-",
				Namespace:    cluster.Namespace,
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion: dockyardsv1.GroupVersion.String(),
						Kind:       dockyardsv1.ClusterKind,
						Name:       cluster.Name,
						UID:        cluster.UID,
					},
				},
			},
			Spec: dockyardsv1.NodePoolSpec{
				Resources: corev1.ResourceList{
					corev1.ResourceCPU: resource.MustParse("2"),
				},
			},
		}

		err := c.Create(ctx, &nodePool)
		if err != nil {
			t.Fatal(err)
		}

		err = testingutil.RetryUntilFound(ctx, mgr.GetClient(), &nodePool)
		if err != nil {
			t.Fatal(err)
		}

		update := types.NodePoolOptions{
			CPUCount: ptr.To(3),
		}

		u := url.URL{
			Path: path.Join("/v1/orgs", organization.Name, "clusters", cluster.Name, "node-pools", nodePool.Name),
		}

		w := httptest.NewRecorder()

		b, err := json.Marshal(update)
		if err != nil {
			t.Fatal(err)
		}

		r := httptest.NewRequest(http.MethodPatch, u.Path, bytes.NewBuffer(b))

		r.Header.Add("Authorization", "Bearer "+readerToken)

		mux.ServeHTTP(w, r)

		statusCode := w.Result().StatusCode
		if statusCode != http.StatusUnauthorized {
			t.Fatalf("expected status code %d, got %d", http.StatusUnauthorized, statusCode)
		}
	})

	t.Run("test quantity", func(t *testing.T) {
		nodePool := dockyardsv1.NodePool{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: "test-quantity-",
				Namespace:    cluster.Namespace,
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion: dockyardsv1.GroupVersion.String(),
						Kind:       dockyardsv1.ClusterKind,
						Name:       cluster.Name,
						UID:        cluster.UID,
					},
				},
			},
			Spec: dockyardsv1.NodePoolSpec{
				Replicas: ptr.To(int32(1)),
				Resources: corev1.ResourceList{
					corev1.ResourceCPU: resource.MustParse("2"),
				},
			},
		}

		err := c.Create(ctx, &nodePool)
		if err != nil {
			t.Fatal(err)
		}

		err = testingutil.RetryUntilFound(ctx, mgr.GetClient(), &nodePool)
		if err != nil {
			t.Fatal(err)
		}

		update := types.NodePoolOptions{
			Quantity: ptr.To(2),
		}

		u := url.URL{
			Path: path.Join("/v1/orgs", organization.Name, "clusters", cluster.Name, "node-pools", nodePool.Name),
		}

		w := httptest.NewRecorder()

		b, err := json.Marshal(update)
		if err != nil {
			t.Fatal(err)
		}

		r := httptest.NewRequest(http.MethodPatch, u.Path, bytes.NewBuffer(b))

		r.Header.Add("Authorization", "Bearer "+superUserToken)

		mux.ServeHTTP(w, r)

		statusCode := w.Result().StatusCode
		if statusCode != http.StatusAccepted {
			t.Fatalf("expected status code %d, got %d", http.StatusAccepted, statusCode)
		}

		var actual dockyardsv1.NodePool
		err = c.Get(ctx, client.ObjectKeyFromObject(&nodePool), &actual)
		if err != nil {
			t.Fatal(err)
		}

		expected := dockyardsv1.NodePool{
			ObjectMeta: actual.ObjectMeta,
			Status:     actual.Status,
			Spec: dockyardsv1.NodePoolSpec{
				Replicas: ptr.To(int32(2)),
				Resources: corev1.ResourceList{
					corev1.ResourceCPU: resource.MustParse("2"),
				},
			},
		}

		if !cmp.Equal(actual, expected) {
			t.Errorf("diff: %s", cmp.Diff(expected, actual))
		}
	})

	t.Run("test storage resources", func(t *testing.T) {
		nodePool := dockyardsv1.NodePool{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: "test-storage-resources-",
				Namespace:    cluster.Namespace,
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion: dockyardsv1.GroupVersion.String(),
						Kind:       dockyardsv1.ClusterKind,
						Name:       cluster.Name,
						UID:        cluster.UID,
					},
				},
			},
			Spec: dockyardsv1.NodePoolSpec{
				StorageResources: []dockyardsv1.NodePoolStorageResource{
					{
						Name: "this-should-be-removed",
					},
				},
			},
		}

		err := c.Create(ctx, &nodePool)
		if err != nil {
			t.Fatal(err)
		}

		err = testingutil.RetryUntilFound(ctx, mgr.GetClient(), &nodePool)
		if err != nil {
			t.Fatal(err)
		}

		update := types.NodePoolOptions{
			StorageResources: ptr.To([]types.StorageResource{
				{
					Name:     "foo",
					Quantity: "1",
					Type:     ptr.To(dockyardsv1.StorageResourceTypeHostPath),
				},
			}),
		}

		u := url.URL{
			Path: path.Join("/v1/orgs", organization.Name, "clusters", cluster.Name, "node-pools", nodePool.Name),
		}

		w := httptest.NewRecorder()

		b, err := json.Marshal(update)
		if err != nil {
			t.Fatal(err)
		}

		r := httptest.NewRequest(http.MethodPatch, u.Path, bytes.NewBuffer(b))

		r.Header.Add("Authorization", "Bearer "+userToken)

		mux.ServeHTTP(w, r)

		statusCode := w.Result().StatusCode
		if statusCode != http.StatusAccepted {
			t.Fatalf("expected status code %d, got %d", http.StatusAccepted, statusCode)
		}

		b, err = io.ReadAll(w.Result().Body)
		if err != nil {
			t.Fatal(err)
		}

		var actual dockyardsv1.NodePool
		err = c.Get(ctx, client.ObjectKeyFromObject(&nodePool), &actual)
		if err != nil {
			t.Fatal(err)
		}

		expected := dockyardsv1.NodePool{
			ObjectMeta: actual.ObjectMeta,
			Status:     actual.Status,
			Spec: dockyardsv1.NodePoolSpec{
				StorageResources: []dockyardsv1.NodePoolStorageResource{
					{
						Name:     "foo",
						Quantity: resource.MustParse("1"),
						Type:     dockyardsv1.StorageResourceTypeHostPath,
					},
				},
			},
		}

		if !cmp.Equal(actual, expected) {
			t.Errorf("diff: %s", cmp.Diff(expected, actual))
		}
	})

	t.Run("test change name", func(t *testing.T) {
		nodePool := dockyardsv1.NodePool{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: "test-change-name-",
				Namespace:    cluster.Namespace,
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion: dockyardsv1.GroupVersion.String(),
						Kind:       dockyardsv1.ClusterKind,
						Name:       cluster.Name,
						UID:        cluster.UID,
					},
				},
			},
		}

		err := c.Create(ctx, &nodePool)
		if err != nil {
			t.Fatal(err)
		}

		err = testingutil.RetryUntilFound(ctx, mgr.GetClient(), &nodePool)
		if err != nil {
			t.Fatal(err)
		}

		update := types.NodePoolOptions{
			Name: ptr.To("hello"),
		}

		u := url.URL{
			Path: path.Join("/v1/orgs", organization.Name, "clusters", cluster.Name, "node-pools", nodePool.Name),
		}

		w := httptest.NewRecorder()

		b, err := json.Marshal(update)
		if err != nil {
			t.Fatal(err)
		}

		r := httptest.NewRequest(http.MethodPatch, u.Path, bytes.NewBuffer(b))

		r.Header.Add("Authorization", "Bearer "+userToken)

		mux.ServeHTTP(w, r)

		statusCode := w.Result().StatusCode
		if statusCode != http.StatusUnprocessableEntity {
			t.Fatalf("expected status code %d, got %d", http.StatusUnprocessableEntity, statusCode)
		}
	})

	t.Run("test non-existing node pool", func(t *testing.T) {
		update := types.NodePoolOptions{
			Quantity: ptr.To(3),
		}

		u := url.URL{
			Path: path.Join("/v1/orgs", organization.Name, "clusters", cluster.Name, "node-pools", "test-non-existing"),
		}

		w := httptest.NewRecorder()

		b, err := json.Marshal(update)
		if err != nil {
			t.Fatal(err)
		}

		r := httptest.NewRequest(http.MethodPatch, u.Path, bytes.NewBuffer(b))

		r.Header.Add("Authorization", "Bearer "+userToken)

		mux.ServeHTTP(w, r)

		statusCode := w.Result().StatusCode
		if statusCode != http.StatusNotFound {
			t.Fatalf("expected status code %d, got %d", http.StatusNotFound, statusCode)
		}
	})

	t.Run("test invalid storage resource type", func(t *testing.T) {
		nodePool := dockyardsv1.NodePool{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: "test-invalid-type-",
				Namespace:    cluster.Namespace,
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion: dockyardsv1.GroupVersion.String(),
						Kind:       dockyardsv1.ClusterKind,
						Name:       cluster.Name,
						UID:        cluster.UID,
					},
				},
			},
		}

		err := c.Create(ctx, &nodePool)
		if err != nil {
			t.Fatal(err)
		}

		err = testingutil.RetryUntilFound(ctx, mgr.GetClient(), &nodePool)
		if err != nil {
			t.Fatal(err)
		}

		update := types.NodePoolOptions{
			StorageResources: ptr.To([]types.StorageResource{
				{
					Name:     "foo",
					Quantity: "100Gi",
					Type:     ptr.To("this-type-does-not-exist"),
				},
			}),
		}

		u := url.URL{
			Path: path.Join("/v1/orgs", organization.Name, "clusters", cluster.Name, "node-pools", nodePool.Name),
		}

		w := httptest.NewRecorder()

		b, err := json.Marshal(update)
		if err != nil {
			t.Fatal(err)
		}

		r := httptest.NewRequest(http.MethodPatch, u.Path, bytes.NewBuffer(b))

		r.Header.Add("Authorization", "Bearer "+userToken)

		mux.ServeHTTP(w, r)

		statusCode := w.Result().StatusCode
		if statusCode != http.StatusUnprocessableEntity {
			t.Fatalf("expected status code %d, got %d", http.StatusUnprocessableEntity, statusCode)
		}
	})

	t.Run("test invalid storage resource quantity", func(t *testing.T) {
		nodePool := dockyardsv1.NodePool{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: "test-invalid-type-",
				Namespace:    cluster.Namespace,
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion: dockyardsv1.GroupVersion.String(),
						Kind:       dockyardsv1.ClusterKind,
						Name:       cluster.Name,
						UID:        cluster.UID,
					},
				},
			},
		}

		err := c.Create(ctx, &nodePool)
		if err != nil {
			t.Fatal(err)
		}

		err = testingutil.RetryUntilFound(ctx, mgr.GetClient(), &nodePool)
		if err != nil {
			t.Fatal(err)
		}

		update := types.NodePoolOptions{
			StorageResources: ptr.To([]types.StorageResource{
				{
					Name:     "foo",
					Quantity: "invalid-quantity",
					Type:     ptr.To(dockyardsv1.StorageResourceTypeHostPath),
				},
			}),
		}

		u := url.URL{
			Path: path.Join("/v1/orgs", organization.Name, "clusters", cluster.Name, "node-pools", nodePool.Name),
		}

		w := httptest.NewRecorder()

		b, err := json.Marshal(update)
		if err != nil {
			t.Fatal(err)
		}

		r := httptest.NewRequest(http.MethodPatch, u.Path, bytes.NewBuffer(b))

		r.Header.Add("Authorization", "Bearer "+userToken)

		mux.ServeHTTP(w, r)

		statusCode := w.Result().StatusCode
		if statusCode != http.StatusUnprocessableEntity {
			t.Fatalf("expected status code %d, got %d", http.StatusUnprocessableEntity, statusCode)
		}
	})

	t.Run("test invalid disk size", func(t *testing.T) {
		nodePool := dockyardsv1.NodePool{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: "test-invalid-disk-size-",
				Namespace:    cluster.Namespace,
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion: dockyardsv1.GroupVersion.String(),
						Kind:       dockyardsv1.ClusterKind,
						Name:       cluster.Name,
						UID:        cluster.UID,
					},
				},
			},
		}

		err := c.Create(ctx, &nodePool)
		if err != nil {
			t.Fatal(err)
		}

		err = testingutil.RetryUntilFound(ctx, mgr.GetClient(), &nodePool)
		if err != nil {
			t.Fatal(err)
		}

		update := types.NodePoolOptions{
			DiskSize: ptr.To("foobar"),
		}

		u := url.URL{
			Path: path.Join("/v1/orgs", organization.Name, "clusters", cluster.Name, "node-pools", nodePool.Name),
		}

		w := httptest.NewRecorder()

		b, err := json.Marshal(update)
		if err != nil {
			t.Fatal(err)
		}

		r := httptest.NewRequest(http.MethodPatch, u.Path, bytes.NewBuffer(b))

		r.Header.Add("Authorization", "Bearer "+userToken)

		mux.ServeHTTP(w, r)

		statusCode := w.Result().StatusCode
		if statusCode != http.StatusUnprocessableEntity {
			t.Fatalf("expected status code %d, got %d", http.StatusUnprocessableEntity, statusCode)
		}
	})

	t.Run("test invalid ram size", func(t *testing.T) {
		nodePool := dockyardsv1.NodePool{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: "test-invalid-ram-size-",
				Namespace:    cluster.Namespace,
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion: dockyardsv1.GroupVersion.String(),
						Kind:       dockyardsv1.ClusterKind,
						Name:       cluster.Name,
						UID:        cluster.UID,
					},
				},
			},
		}

		err := c.Create(ctx, &nodePool)
		if err != nil {
			t.Fatal(err)
		}

		err = testingutil.RetryUntilFound(ctx, mgr.GetClient(), &nodePool)
		if err != nil {
			t.Fatal(err)
		}

		update := types.NodePoolOptions{
			RAMSize: ptr.To("foobar"),
		}

		u := url.URL{
			Path: path.Join("/v1/orgs", organization.Name, "clusters", cluster.Name, "node-pools", nodePool.Name),
		}

		w := httptest.NewRecorder()

		b, err := json.Marshal(update)
		if err != nil {
			t.Fatal(err)
		}

		r := httptest.NewRequest(http.MethodPatch, u.Path, bytes.NewBuffer(b))

		r.Header.Add("Authorization", "Bearer "+userToken)

		mux.ServeHTTP(w, r)

		statusCode := w.Result().StatusCode
		if statusCode != http.StatusUnprocessableEntity {
			t.Fatalf("expected status code %d, got %d", http.StatusUnprocessableEntity, statusCode)
		}
	})

	t.Run("test empty storage resource name", func(t *testing.T) {
		nodePool := dockyardsv1.NodePool{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: "test-storage-resource-name-",
				Namespace:    cluster.Namespace,
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion: dockyardsv1.GroupVersion.String(),
						Kind:       dockyardsv1.ClusterKind,
						Name:       cluster.Name,
						UID:        cluster.UID,
					},
				},
			},
		}

		err := c.Create(ctx, &nodePool)
		if err != nil {
			t.Fatal(err)
		}

		err = testingutil.RetryUntilFound(ctx, mgr.GetClient(), &nodePool)
		if err != nil {
			t.Fatal(err)
		}

		update := types.NodePoolOptions{
			StorageResources: ptr.To([]types.StorageResource{
				{
					Name:     "",
					Quantity: "100Gi",
					Type:     ptr.To(dockyardsv1.StorageResourceTypeHostPath),
				},
			}),
		}

		u := url.URL{
			Path: path.Join("/v1/orgs", organization.Name, "clusters", cluster.Name, "node-pools", nodePool.Name),
		}

		w := httptest.NewRecorder()

		b, err := json.Marshal(update)
		if err != nil {
			t.Fatal(err)
		}

		r := httptest.NewRequest(http.MethodPatch, u.Path, bytes.NewBuffer(b))

		r.Header.Add("Authorization", "Bearer "+userToken)

		mux.ServeHTTP(w, r)

		statusCode := w.Result().StatusCode
		if statusCode != http.StatusUnprocessableEntity {
			t.Fatalf("expected status code %d, got %d", http.StatusUnprocessableEntity, statusCode)
		}
	})

	t.Run("test invalid storage resource name", func(t *testing.T) {
		nodePool := dockyardsv1.NodePool{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: "test-storage-resource-name-",
				Namespace:    cluster.Namespace,
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion: dockyardsv1.GroupVersion.String(),
						Kind:       dockyardsv1.ClusterKind,
						Name:       cluster.Name,
						UID:        cluster.UID,
					},
				},
			},
		}

		err := c.Create(ctx, &nodePool)
		if err != nil {
			t.Fatal(err)
		}

		err = testingutil.RetryUntilFound(ctx, mgr.GetClient(), &nodePool)
		if err != nil {
			t.Fatal(err)
		}

		update := types.NodePoolOptions{
			StorageResources: ptr.To([]types.StorageResource{
				{
					Name:     "<script>giveMeYourCookies()</script>",
					Quantity: "100Gi",
					Type:     ptr.To(dockyardsv1.StorageResourceTypeHostPath),
				},
			}),
		}

		u := url.URL{
			Path: path.Join("/v1/orgs", organization.Name, "clusters", cluster.Name, "node-pools", nodePool.Name),
		}

		w := httptest.NewRecorder()

		b, err := json.Marshal(update)
		if err != nil {
			t.Fatal(err)
		}

		r := httptest.NewRequest(http.MethodPatch, u.Path, bytes.NewBuffer(b))

		r.Header.Add("Authorization", "Bearer "+userToken)

		mux.ServeHTTP(w, r)

		statusCode := w.Result().StatusCode
		if statusCode != http.StatusUnprocessableEntity {
			t.Fatalf("expected status code %d, got %d", http.StatusUnprocessableEntity, statusCode)
		}
	})

	t.Run("test invalid cpu count", func(t *testing.T) {
		nodePool := dockyardsv1.NodePool{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: "test-cpu-count-",
				Namespace:    cluster.Namespace,
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion: dockyardsv1.GroupVersion.String(),
						Kind:       dockyardsv1.ClusterKind,
						Name:       cluster.Name,
						UID:        cluster.UID,
					},
				},
			},
		}

		err := c.Create(ctx, &nodePool)
		if err != nil {
			t.Fatal(err)
		}

		err = testingutil.RetryUntilFound(ctx, mgr.GetClient(), &nodePool)
		if err != nil {
			t.Fatal(err)
		}

		update := types.NodePoolOptions{
			CPUCount: ptr.To(-1),
		}

		u := url.URL{
			Path: path.Join("/v1/orgs", organization.Name, "clusters", cluster.Name, "node-pools", nodePool.Name),
		}

		w := httptest.NewRecorder()

		b, err := json.Marshal(update)
		if err != nil {
			t.Fatal(err)
		}

		r := httptest.NewRequest(http.MethodPatch, u.Path, bytes.NewBuffer(b))

		r.Header.Add("Authorization", "Bearer "+userToken)

		mux.ServeHTTP(w, r)

		statusCode := w.Result().StatusCode
		if statusCode != http.StatusUnprocessableEntity {
			t.Fatalf("expected status code %d, got %d", http.StatusUnprocessableEntity, statusCode)
		}
	})
}

func TestClusterNodePools_Create(t *testing.T) {
	if os.Getenv("KUBEBUILDER_ASSETS") == "" {
		t.Skip("no kubebuilder assets configured")
	}

	mgr := testEnvironment.GetManager()
	c := testEnvironment.GetClient()

	organization := testEnvironment.MustCreateOrganization(t)

	superUser := testEnvironment.MustGetOrganizationUser(t, organization, dockyardsv1.RoleSuperUser)
	user := testEnvironment.MustGetOrganizationUser(t, organization, dockyardsv1.RoleUser)
	reader := testEnvironment.MustGetOrganizationUser(t, organization, dockyardsv1.RoleReader)

	superUserToken := MustSignToken(t, superUser.Name)
	userToken := MustSignToken(t, user.Name)
	readerToken := MustSignToken(t, reader.Name)

	cluster := dockyardsv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "test-",
			Namespace:    organization.Spec.NamespaceRef.Name,
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion:         dockyardsv1.GroupVersion.String(),
					Kind:               dockyardsv1.OrganizationKind,
					Name:               organization.Name,
					UID:                organization.UID,
					BlockOwnerDeletion: ptr.To(true),
				},
			},
		},
	}

	err := c.Create(ctx, &cluster)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("test as super user", func(t *testing.T) {
		nodePoolOptions := types.NodePoolOptions{
			Name:     ptr.To("test-super-user"),
			Quantity: ptr.To(0),
		}

		u := url.URL{
			Path: path.Join("/v1/orgs", organization.Name, "clusters", cluster.Name, "node-pools"),
		}

		b, err := json.Marshal(nodePoolOptions)
		if err != nil {
			t.Fatalf("unexpected error marshalling test options: %s", err)
		}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, u.Path, bytes.NewBuffer(b))

		r.Header.Add("Authorization", "Bearer "+superUserToken)

		mux.ServeHTTP(w, r)

		statusCode := w.Result().StatusCode
		if statusCode != http.StatusCreated {
			t.Fatalf("expected status code %d, got %d", http.StatusCreated, statusCode)
		}

		b, err = io.ReadAll(w.Result().Body)
		if err != nil {
			t.Fatalf("unexpected error reading result body: %s", err)
		}

		var actual types.NodePool
		err = json.Unmarshal(b, &actual)
		if err != nil {
			t.Fatalf("error unmarshalling result body to json: %s", err)
		}

		objectKey := client.ObjectKey{
			Name:      cluster.Name + "-" + *nodePoolOptions.Name,
			Namespace: cluster.Namespace,
		}

		var nodePool dockyardsv1.NodePool
		err = c.Get(ctx, objectKey, &nodePool)
		if err != nil {
			t.Fatal(err)
		}

		clusterName, hasLabel := nodePool.Labels[dockyardsv1.LabelClusterName]
		if !hasLabel || clusterName != cluster.Name {
			t.Errorf("expected cluster name label %s, got %s", cluster.Name, clusterName)
		}

		expected := types.NodePool{
			CreatedAt: nodePool.CreationTimestamp.Time,
			ID:        string(nodePool.UID),
			Name:      nodePool.Name,
			Quantity:  ptr.To(0),
		}

		if !cmp.Equal(actual, expected) {
			t.Errorf("diff: %s", cmp.Diff(expected, actual))
		}
	})

	t.Run("test as user", func(t *testing.T) {
		nodePoolOptions := types.NodePoolOptions{
			Name:     ptr.To("test-user"),
			Quantity: ptr.To(0),
		}

		u := url.URL{
			Path: path.Join("/v1/orgs", organization.Name, "clusters", cluster.Name, "node-pools"),
		}

		b, err := json.Marshal(nodePoolOptions)
		if err != nil {
			t.Fatalf("unexpected error marshalling test options: %s", err)
		}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, u.Path, bytes.NewBuffer(b))

		r.Header.Add("Authorization", "Bearer "+userToken)

		mux.ServeHTTP(w, r)

		statusCode := w.Result().StatusCode
		if statusCode != http.StatusCreated {
			t.Fatalf("expected status code %d, got %d", http.StatusCreated, statusCode)
		}

		b, err = io.ReadAll(w.Result().Body)
		if err != nil {
			t.Fatalf("unexpected error reading result body: %s", err)
		}

		var actual types.NodePool
		err = json.Unmarshal(b, &actual)
		if err != nil {
			t.Fatalf("error unmarshalling result body to json: %s", err)
		}

		objectKey := client.ObjectKey{
			Name:      cluster.Name + "-" + *nodePoolOptions.Name,
			Namespace: cluster.Namespace,
		}

		var nodePool dockyardsv1.NodePool
		err = c.Get(ctx, objectKey, &nodePool)
		if err != nil {
			t.Fatal(err)
		}

		clusterName, hasLabel := nodePool.Labels[dockyardsv1.LabelClusterName]
		if !hasLabel || clusterName != cluster.Name {
			t.Errorf("expected cluster name label %s, got %s", cluster.Name, clusterName)
		}

		expected := types.NodePool{
			CreatedAt: nodePool.CreationTimestamp.Time,
			ID:        string(nodePool.UID),
			Name:      nodePool.Name,
			Quantity:  ptr.To(0),
		}

		if !cmp.Equal(actual, expected) {
			t.Errorf("diff: %s", cmp.Diff(expected, actual))
		}
	})

	t.Run("test as reader", func(t *testing.T) {
		nodePoolOptions := types.NodePoolOptions{
			Name:     ptr.To("test-super-user"),
			Quantity: ptr.To(0),
		}

		u := url.URL{
			Path: path.Join("/v1/orgs", organization.Name, "clusters", cluster.Name, "node-pools"),
		}

		b, err := json.Marshal(nodePoolOptions)
		if err != nil {
			t.Fatalf("unexpected error marshalling test options: %s", err)
		}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, u.Path, bytes.NewBuffer(b))

		r.Header.Add("Authorization", "Bearer "+readerToken)

		mux.ServeHTTP(w, r)

		statusCode := w.Result().StatusCode
		if statusCode != http.StatusUnauthorized {
			t.Fatalf("expected status code %d, got %d", http.StatusUnauthorized, statusCode)
		}
	})

	t.Run("test complex options", func(t *testing.T) {
		nodePoolOptions := types.NodePoolOptions{
			Name:                       ptr.To("test2"),
			Quantity:                   ptr.To(3),
			LoadBalancer:               ptr.To(true),
			ControlPlaneComponentsOnly: ptr.To(true),
			RAMSize:                    ptr.To("1234M"),
			CPUCount:                   ptr.To(12),
			DiskSize:                   ptr.To("123Gi"),
		}

		u := url.URL{
			Path: path.Join("/v1/orgs", organization.Name, "clusters", cluster.Name, "node-pools"),
		}

		b, err := json.Marshal(nodePoolOptions)
		if err != nil {
			t.Fatalf("unexpected error marshalling test options: %s", err)
		}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, u.Path, bytes.NewBuffer(b))

		r.Header.Add("Authorization", "Bearer "+superUserToken)

		mux.ServeHTTP(w, r)

		statusCode := w.Result().StatusCode
		if statusCode != http.StatusCreated {
			t.Fatalf("expected status code %d, got %d", http.StatusCreated, statusCode)
		}

		b, err = io.ReadAll(w.Result().Body)
		if err != nil {
			t.Fatalf("unexpected error reading result body: %s", err)
		}

		var actual types.NodePool
		err = json.Unmarshal(b, &actual)
		if err != nil {
			t.Fatalf("error unmarshalling result body to json: %s", err)
		}

		objectKey := client.ObjectKey{
			Name:      cluster.Name + "-" + *nodePoolOptions.Name,
			Namespace: cluster.Namespace,
		}

		var nodePool dockyardsv1.NodePool
		err = c.Get(ctx, objectKey, &nodePool)
		if err != nil {
			t.Fatal(err)
		}

		clusterName, hasLabel := nodePool.Labels[dockyardsv1.LabelClusterName]
		if !hasLabel || clusterName != cluster.Name {
			t.Errorf("expected cluster name label %s, got %s", cluster.Name, clusterName)
		}

		expected := types.NodePool{
			CreatedAt:                  nodePool.CreationTimestamp.Time,
			ControlPlaneComponentsOnly: ptr.To(true),
			CPUCount:                   ptr.To(12),
			DiskSize:                   ptr.To("123Gi"),
			ID:                         string(nodePool.UID),
			LoadBalancer:               ptr.To(true),
			Name:                       nodePool.Name,
			Quantity:                   ptr.To(3),
			RAMSize:                    ptr.To("1234M"),
		}

		if !cmp.Equal(actual, expected) {
			t.Errorf("diff: %s", cmp.Diff(expected, actual))
		}
	})

	t.Run("test storege resource without type", func(t *testing.T) {
		nodePoolOptions := types.NodePoolOptions{
			Name:     ptr.To("storage-resources"),
			Quantity: ptr.To(3),
			StorageResources: &[]types.StorageResource{
				{
					Name:     "test",
					Quantity: "123",
				},
			},
		}

		u := url.URL{
			Path: path.Join("/v1/orgs", organization.Name, "clusters", cluster.Name, "node-pools"),
		}

		b, err := json.Marshal(nodePoolOptions)
		if err != nil {
			t.Fatalf("unexpected error marshalling test options: %s", err)
		}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, u.Path, bytes.NewBuffer(b))

		r.Header.Add("Authorization", "Bearer "+superUserToken)

		mux.ServeHTTP(w, r)

		statusCode := w.Result().StatusCode
		if statusCode != http.StatusCreated {
			t.Fatalf("expected status code %d, got %d", http.StatusCreated, statusCode)
		}

		b, err = io.ReadAll(w.Result().Body)
		if err != nil {
			t.Fatalf("unexpected error reading result body: %s", err)
		}

		var actual types.NodePool
		err = json.Unmarshal(b, &actual)
		if err != nil {
			t.Fatalf("error unmarshalling result body to json: %s", err)
		}

		objectKey := client.ObjectKey{
			Name:      cluster.Name + "-" + *nodePoolOptions.Name,
			Namespace: cluster.Namespace,
		}

		var nodePool dockyardsv1.NodePool
		err = c.Get(ctx, objectKey, &nodePool)
		if err != nil {
			t.Fatal(err)
		}

		clusterName, hasLabel := nodePool.Labels[dockyardsv1.LabelClusterName]
		if !hasLabel || clusterName != cluster.Name {
			t.Errorf("expected cluster name label %s, got %s", cluster.Name, clusterName)
		}

		expected := types.NodePool{
			CreatedAt: nodePool.CreationTimestamp.Time,
			ID:        string(nodePool.UID),
			Name:      nodePool.Name,
			Quantity:  ptr.To(3),
			StorageResources: &[]types.StorageResource{
				{
					Name:     "test",
					Quantity: "123",
				},
			},
		}

		if !cmp.Equal(actual, expected) {
			t.Errorf("diff: %s", cmp.Diff(expected, actual))
		}
	})

	t.Run("test invalid name", func(t *testing.T) {
		nodePoolOptions := types.NodePoolOptions{
			Name:     ptr.To("InvalidName"),
			Quantity: ptr.To(0),
		}

		u := url.URL{
			Path: path.Join("/v1/orgs", organization.Name, "clusters", cluster.Name, "node-pools"),
		}

		b, err := json.Marshal(nodePoolOptions)
		if err != nil {
			t.Fatalf("unexpected error marshalling test options: %s", err)
		}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, u.Path, bytes.NewBuffer(b))

		r.Header.Add("Authorization", "Bearer "+superUserToken)

		mux.ServeHTTP(w, r)

		statusCode := w.Result().StatusCode
		if statusCode != http.StatusUnprocessableEntity {
			t.Fatalf("expected status code %d, got %d", http.StatusUnprocessableEntity, statusCode)
		}
	})

	t.Run("test conflict name", func(t *testing.T) {
		nodePool := dockyardsv1.NodePool{
			ObjectMeta: metav1.ObjectMeta{
				Name:      cluster.Name + "-test-conflict",
				Namespace: organization.Spec.NamespaceRef.Name,
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion: dockyardsv1.GroupVersion.String(),
						Kind:       dockyardsv1.ClusterKind,
						Name:       cluster.Name,
						UID:        cluster.UID,
					},
				},
			},
		}

		err := c.Create(ctx, &nodePool)
		if err != nil {
			t.Fatal(err)
		}

		err = testingutil.RetryUntilFound(ctx, mgr.GetClient(), &nodePool)
		if err != nil {
			t.Fatal(err)
		}

		nodePoolOptions := types.NodePoolOptions{
			Name:     ptr.To("test-conflict"),
			Quantity: ptr.To(0),
		}

		u := url.URL{
			Path: path.Join("/v1/orgs", organization.Name, "clusters", cluster.Name, "node-pools"),
		}

		b, err := json.Marshal(nodePoolOptions)
		if err != nil {
			t.Fatalf("unexpected error marshalling test options: %s", err)
		}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, u.Path, bytes.NewBuffer(b))

		r.Header.Add("Authorization", "Bearer "+superUserToken)

		mux.ServeHTTP(w, r)

		statusCode := w.Result().StatusCode
		if statusCode != http.StatusConflict {
			t.Fatalf("expected status code %d, got %d", http.StatusConflict, statusCode)
		}
	})

	t.Run("test without membership", func(t *testing.T) {
		namespace := corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: "test-",
			},
		}

		err = c.Create(ctx, &namespace)
		if err != nil {
			t.Fatal(err)
		}

		otherOrganization := dockyardsv1.Organization{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: "test-",
			},
			Spec: dockyardsv1.OrganizationSpec{
				MemberRefs: []dockyardsv1.OrganizationMemberReference{
					{
						Role: dockyardsv1.OrganizationMemberRoleSuperUser,
						UID:  user.UID,
					},
					{
						Role: dockyardsv1.OrganizationMemberRoleUser,
						UID:  reader.UID,
					},
				},
				NamespaceRef: &corev1.LocalObjectReference{
					Name: namespace.Name,
				},
			},
		}

		err := c.Create(ctx, &otherOrganization)
		if err != nil {
			t.Fatal(err)
		}

		otherCluster := dockyardsv1.Cluster{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: "test-",
				Namespace:    namespace.Name,
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion: dockyardsv1.GroupVersion.String(),
						Kind:       dockyardsv1.OrganizationKind,
						Name:       otherOrganization.Name,
						UID:        otherOrganization.UID,
					},
				},
			},
		}

		err = c.Create(ctx, &otherCluster)
		if err != nil {
			t.Fatal(err)
		}

		err = testingutil.RetryUntilFound(ctx, mgr.GetClient(), &otherCluster)
		if err != nil {
			t.Fatal(err)
		}

		nodePoolOptions := types.NodePoolOptions{
			Name:     ptr.To("test"),
			Quantity: ptr.To(0),
		}

		u := url.URL{
			Path: path.Join("/v1/orgs", otherOrganization.Name, "clusters", otherCluster.Name, "node-pools"),
		}

		b, err := json.Marshal(nodePoolOptions)
		if err != nil {
			t.Fatal(err)
		}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, u.Path, bytes.NewBuffer(b))

		r.Header.Add("Authorization", "Bearer "+superUserToken)

		mux.ServeHTTP(w, r)

		statusCode := w.Result().StatusCode
		if statusCode != http.StatusUnauthorized {
			t.Fatalf("expected status code %d, got %d", http.StatusUnauthorized, statusCode)
		}
	})

	t.Run("test high quantity", func(t *testing.T) {
		nodePoolOptions := types.NodePoolOptions{
			Name:     ptr.To("test"),
			Quantity: ptr.To(50),
		}

		u := url.URL{
			Path: path.Join("/v1/orgs", organization.Name, "clusters", cluster.Name, "node-pools"),
		}

		b, err := json.Marshal(nodePoolOptions)
		if err != nil {
			t.Fatal(err)
		}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, u.Path, bytes.NewBuffer(b))

		r.Header.Add("Authorization", "Bearer "+superUserToken)

		mux.ServeHTTP(w, r)

		statusCode := w.Result().StatusCode
		if statusCode != http.StatusUnprocessableEntity {
			t.Fatalf("expected status code %d, got %d", http.StatusUnprocessableEntity, statusCode)
		}
	})
}
