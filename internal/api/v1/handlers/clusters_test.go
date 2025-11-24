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

func TestOrganizationClusters_Create(t *testing.T) {
	if os.Getenv("KUBEBUILDER_ASSETS") == "" {
		t.Skip("no kubebuilder assets configured")
	}

	c := testEnvironment.GetClient()

	organization := testEnvironment.MustCreateOrganization(t)

	superUser := testEnvironment.MustGetOrganizationUser(t, organization, dockyardsv1.RoleSuperUser)
	user := testEnvironment.MustGetOrganizationUser(t, organization, dockyardsv1.RoleUser)
	reader := testEnvironment.MustGetOrganizationUser(t, organization, dockyardsv1.RoleReader)

	superUserToken := MustSignToken(t, superUser.Name)
	userToken := MustSignToken(t, user.Name)
	readerToken := MustSignToken(t, reader.Name)

	clusterTemplate := dockyardsv1.ClusterTemplate{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "test-",
			Namespace:    testEnvironment.GetDockyardsNamespace(),
			Annotations: map[string]string{
				dockyardsv1.AnnotationDefaultTemplate: "true",
			},
		},
		Spec: dockyardsv1.ClusterTemplateSpec{
			NodePoolTemplates: []dockyardsv1.NodePoolTemplate{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "controlplane",
					},
					Spec: dockyardsv1.NodePoolSpec{
						Replicas:      ptr.To(int32(3)),
						ControlPlane:  true,
						DedicatedRole: true,
						Resources: corev1.ResourceList{
							corev1.ResourceCPU:     resource.MustParse("2"),
							corev1.ResourceMemory:  resource.MustParse("4096M"),
							corev1.ResourceStorage: resource.MustParse("100G"),
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "worker",
					},
					Spec: dockyardsv1.NodePoolSpec{
						Replicas: ptr.To(int32(2)),
						Resources: corev1.ResourceList{
							corev1.ResourceCPU:     resource.MustParse("4"),
							corev1.ResourceMemory:  resource.MustParse("8192M"),
							corev1.ResourceStorage: resource.MustParse("100G"),
						},
					},
				},
			},
		},
	}

	err := c.Create(ctx, &clusterTemplate)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("test default as super user", func(t *testing.T) {
		clusterOptions := types.ClusterOptions{
			Name:    "test-super-user",
			Version: ptr.To("v1.2.3"),
		}

		b, err := json.Marshal(clusterOptions)
		if err != nil {
			t.Fatal(err)
		}

		u := url.URL{
			Path: path.Join("/v1/orgs", organization.Name, "clusters"),
		}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, u.Path, bytes.NewBuffer(b))

		r.Header.Add("Authorization", "Bearer "+superUserToken)

		mux.ServeHTTP(w, r)

		statusCode := w.Result().StatusCode
		if statusCode != http.StatusCreated {
			t.Fatalf("expected status code %d, got %d", http.StatusCreated, statusCode)
		}

		expectedCluster := dockyardsv1.Cluster{
			ObjectMeta: metav1.ObjectMeta{
				Name:      clusterOptions.Name,
				Namespace: organization.Spec.NamespaceRef.Name,
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
			Spec: dockyardsv1.ClusterSpec{
				Version: "v1.2.3",
			},
		}

		var actualCluster dockyardsv1.Cluster
		err = c.Get(ctx, client.ObjectKeyFromObject(&expectedCluster), &actualCluster)
		if err != nil {
			t.Fatal(err)
		}

		if !cmp.Equal(actualCluster.Spec, expectedCluster.Spec) {
			t.Errorf("diff: %s", cmp.Diff(expectedCluster.Spec, actualCluster.Spec))
		}
	})

	t.Run("test default as user", func(t *testing.T) {
		clusterOptions := types.ClusterOptions{
			Name: "test-user",
		}

		b, err := json.Marshal(clusterOptions)
		if err != nil {
			t.Fatal(err)
		}

		u := url.URL{
			Path: path.Join("/v1/orgs", organization.Name, "clusters"),
		}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, u.Path, bytes.NewBuffer(b))

		r.Header.Add("Authorization", "Bearer "+userToken)

		mux.ServeHTTP(w, r)

		statusCode := w.Result().StatusCode
		if statusCode != http.StatusCreated {
			t.Fatalf("expected status code %d, got %d", http.StatusCreated, statusCode)
		}
	})

	t.Run("test default as reader", func(t *testing.T) {
		clusterOptions := types.ClusterOptions{
			Name: "test-reader",
		}

		b, err := json.Marshal(clusterOptions)
		if err != nil {
			t.Fatal(err)
		}

		u := url.URL{
			Path: path.Join("/v1/orgs", organization.Name, "clusters"),
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

	t.Run("test allocate internal ip", func(t *testing.T) {
		clusterOptions := types.ClusterOptions{
			Name:               "test",
			AllocateInternalIP: ptr.To(true),
		}

		b, err := json.Marshal(clusterOptions)
		if err != nil {
			t.Fatal(err)
		}

		u := url.URL{
			Path: path.Join("/v1/orgs", organization.Name, "clusters"),
		}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, u.Path, bytes.NewBuffer(b))

		r.Header.Add("Authorization", "Bearer "+userToken)

		mux.ServeHTTP(w, r)

		statusCode := w.Result().StatusCode
		if statusCode != http.StatusCreated {
			t.Fatalf("expected status code %d, got %d", http.StatusCreated, statusCode)
		}

		objectKey := client.ObjectKey{
			Name:      clusterOptions.Name,
			Namespace: organization.Spec.NamespaceRef.Name,
		}

		var actual dockyardsv1.Cluster
		err = c.Get(ctx, objectKey, &actual)
		if err != nil {
			t.Fatal(err)
		}

		expected := dockyardsv1.ClusterSpec{
			AllocateInternalIP: true,
		}

		if !cmp.Equal(actual.Spec, expected) {
			t.Errorf("diff: %s", cmp.Diff(expected, actual.Spec))
		}
	})

	t.Run("test cluster template", func(t *testing.T) {
		clusterTemplate := dockyardsv1.ClusterTemplate{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: "test-",
				Namespace:    testEnvironment.GetPublicNamespace(),
			},
			Spec: dockyardsv1.ClusterTemplateSpec{
				NodePoolTemplates: []dockyardsv1.NodePoolTemplate{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "controlplane",
						},
						Spec: dockyardsv1.NodePoolSpec{
							Replicas:      ptr.To(int32(1)),
							ControlPlane:  true,
							DedicatedRole: true,
							Resources: corev1.ResourceList{
								corev1.ResourceCPU:     resource.MustParse("2"),
								corev1.ResourceMemory:  resource.MustParse("3Mi"),
								corev1.ResourceStorage: resource.MustParse("4G"),
							},
						},
					},
				},
			},
		}

		err := c.Create(ctx, &clusterTemplate)
		if err != nil {
			t.Fatal(err)
		}

		clusterOptions := types.ClusterOptions{
			Name:                "test-cluster-template",
			ClusterTemplateName: ptr.To(clusterTemplate.Name),
		}

		b, err := json.Marshal(clusterOptions)
		if err != nil {
			t.Fatal(err)
		}

		u := url.URL{
			Path: path.Join("/v1/orgs", organization.Name, "clusters"),
		}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, u.Path, bytes.NewBuffer(b))

		r.Header.Add("Authorization", "Bearer "+userToken)

		mux.ServeHTTP(w, r)

		statusCode := w.Result().StatusCode
		if statusCode != http.StatusCreated {
			t.Fatalf("expected status code %d, got %d", http.StatusCreated, statusCode)
		}

		expectedCluster := dockyardsv1.ClusterSpec{}

		objectKey := client.ObjectKey{
			Name:      clusterOptions.Name,
			Namespace: organization.Spec.NamespaceRef.Name,
		}

		var actualCluster dockyardsv1.Cluster
		err = c.Get(ctx, objectKey, &actualCluster)
		if err != nil {
			t.Fatal(err)
		}

		if !cmp.Equal(actualCluster.Spec, expectedCluster) {
			t.Errorf("diff: %s", cmp.Diff(expectedCluster, actualCluster.Spec))
		}
	})

	t.Run("test invalid organization", func(t *testing.T) {
		clusterOptions := types.ClusterOptions{
			Name: "test-invalid-organization",
		}

		b, err := json.Marshal(clusterOptions)
		if err != nil {
			t.Fatal(err)
		}

		u := url.URL{
			Path: path.Join("/v1/orgs", "invalid-organization", "clusters"),
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

	t.Run("test invalid name", func(t *testing.T) {
		clusterOptions := types.ClusterOptions{
			Name: "InvalidClusterName",
		}

		b, err := json.Marshal(clusterOptions)
		if err != nil {
			t.Fatal(err)
		}

		u := url.URL{
			Path: path.Join("/v1/orgs", organization.Name, "clusters"),
		}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, u.Path, bytes.NewBuffer(b))

		r.Header.Add("Authorization", "Bearer "+userToken)

		mux.ServeHTTP(w, r)

		statusCode := w.Result().StatusCode
		if statusCode != http.StatusUnprocessableEntity {
			t.Fatalf("expected status code %d, got %d", http.StatusUnprocessableEntity, statusCode)
		}
	})

	t.Run("test invalid node pool name", func(t *testing.T) {
		clusterOptions := types.ClusterOptions{
			Name: "test-node-pool-name",
			NodePoolOptions: ptr.To([]types.NodePoolOptions{
				{
					Name: ptr.To("InvalidNodePoolName"),
				},
			}),
		}

		b, err := json.Marshal(clusterOptions)
		if err != nil {
			t.Fatal(err)
		}

		u := url.URL{
			Path: path.Join("/v1/orgs", organization.Name, "clusters"),
		}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, u.Path, bytes.NewBuffer(b))

		r.Header.Add("Authorization", "Bearer "+userToken)

		mux.ServeHTTP(w, r)

		statusCode := w.Result().StatusCode
		if statusCode != http.StatusUnprocessableEntity {
			t.Fatalf("expected status code %d, got %d", http.StatusUnprocessableEntity, statusCode)
		}
	})

	t.Run("test existing name", func(t *testing.T) {
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

		clusterOptions := types.ClusterOptions{
			Name: cluster.Name,
		}

		b, err := json.Marshal(clusterOptions)
		if err != nil {
			t.Fatal(err)
		}

		u := url.URL{
			Path: path.Join("/v1/orgs", organization.Name, "clusters"),
		}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, u.Path, bytes.NewBuffer(b))

		r.Header.Add("Authorization", "Bearer "+userToken)

		mux.ServeHTTP(w, r)

		statusCode := w.Result().StatusCode
		if statusCode != http.StatusConflict {
			t.Fatalf("expected status code %d, got %d", http.StatusConflict, statusCode)
		}
	})

	t.Run("test high quantity", func(t *testing.T) {
		clusterOptions := types.ClusterOptions{
			Name: "test-high-quantity",
			NodePoolOptions: ptr.To([]types.NodePoolOptions{
				{
					Name:     ptr.To("test"),
					Quantity: ptr.To(123),
				},
			}),
		}

		b, err := json.Marshal(clusterOptions)
		if err != nil {
			t.Fatal(err)
		}

		u := url.URL{
			Path: path.Join("/v1/orgs", organization.Name, "clusters"),
		}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, u.Path, bytes.NewBuffer(b))

		r.Header.Add("Authorization", "Bearer "+userToken)

		mux.ServeHTTP(w, r)

		statusCode := w.Result().StatusCode
		if statusCode != http.StatusUnprocessableEntity {
			t.Fatalf("expected status code %d, got %d", http.StatusUnprocessableEntity, statusCode)
		}
	})

	t.Run("test storage resources", func(t *testing.T) {
		clusterOptions := types.ClusterOptions{
			Name: "test-storage-resources",
			NodePoolOptions: &[]types.NodePoolOptions{
				{
					Name:     ptr.To("worker"),
					Quantity: ptr.To(3),
					DiskSize: ptr.To("4G"),
					RAMSize:  ptr.To("3Mi"),
					CPUCount: ptr.To(2),
					StorageResources: &[]types.StorageResource{
						{
							Name:     "test",
							Quantity: "123",
							Type:     ptr.To("HostPath"),
						},
					},
				},
			},
		}

		b, err := json.Marshal(clusterOptions)
		if err != nil {
			t.Fatal(err)
		}

		u := url.URL{
			Path: path.Join("/v1/orgs", organization.Name, "clusters"),
		}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, u.Path, bytes.NewBuffer(b))

		r.Header.Add("Authorization", "Bearer "+userToken)

		mux.ServeHTTP(w, r)

		statusCode := w.Result().StatusCode
		if statusCode != http.StatusCreated {
			t.Fatalf("expected status code %d, got %d", http.StatusCreated, statusCode)
		}
	})

	t.Run("test duration", func(t *testing.T) {
		clusterOptions := types.ClusterOptions{
			Name:     "test-duration",
			Duration: ptr.To("15m"),
		}

		b, err := json.Marshal(clusterOptions)
		if err != nil {
			t.Fatal(err)
		}

		u := url.URL{
			Path: path.Join("/v1/orgs", organization.Name, "clusters"),
		}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, u.Path, bytes.NewBuffer(b))

		r.Header.Add("Authorization", "Bearer "+userToken)

		mux.ServeHTTP(w, r)

		statusCode := w.Result().StatusCode
		if statusCode != http.StatusCreated {
			t.Fatalf("expected status code %d, got %d", http.StatusCreated, statusCode)
		}
	})

	t.Run("test no default network plugin", func(t *testing.T) {
		clusterOptions := types.ClusterOptions{
			Name:                   "test-network-plugin",
			NoDefaultNetworkPlugin: ptr.To(true),
		}

		b, err := json.Marshal(clusterOptions)
		if err != nil {
			t.Fatal(err)
		}

		u := url.URL{
			Path: path.Join("/v1/orgs", organization.Name, "clusters"),
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
			t.Fatalf("error reading result body: %s", err)
		}

		var actual types.Cluster
		err = json.Unmarshal(b, &actual)
		if err != nil {
			t.Fatalf("error unmarshalling result body: %s", err)
		}

		var actualCluster dockyardsv1.Cluster
		err = c.Get(ctx, client.ObjectKey{Name: actual.Name, Namespace: organization.Spec.NamespaceRef.Name}, &actualCluster)
		if err != nil {
			t.Fatal(err)
		}

		expected := types.Cluster{
			CreatedAt:              actualCluster.CreationTimestamp.Time,
			ID:                     string(actualCluster.UID),
			Name:                   actualCluster.Name,
			NoDefaultNetworkPlugin: ptr.To(true),
			Version:                &actualCluster.Status.Version,
		}

		if !cmp.Equal(actual, expected) {
			t.Errorf("diff: %s", cmp.Diff(expected, actual))
		}
	})

	t.Run("test subnets", func(t *testing.T) {
		clusterOptions := types.ClusterOptions{
			Name: "test-subnets",
			PodSubnets: &[]string{
				"10.244.0.0/16",
			},
			ServiceSubnets: &[]string{
				"10.96.0.0/12",
			},
		}

		b, err := json.Marshal(clusterOptions)
		if err != nil {
			t.Fatal(err)
		}

		u := url.URL{
			Path: path.Join("/v1/orgs", organization.Name, "clusters"),
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
			t.Fatalf("error reading result body: %s", err)
		}

		var actual types.Cluster
		err = json.Unmarshal(b, &actual)
		if err != nil {
			t.Fatalf("error unmarshalling result body: %s", err)
		}

		var actualCluster dockyardsv1.Cluster
		err = c.Get(ctx, client.ObjectKey{Name: actual.Name, Namespace: organization.Spec.NamespaceRef.Name}, &actualCluster)
		if err != nil {
			t.Fatal(err)
		}

		expectedCluster := dockyardsv1.Cluster{
			ObjectMeta: actualCluster.ObjectMeta,
			Spec: dockyardsv1.ClusterSpec{
				PodSubnets:     *clusterOptions.PodSubnets,
				ServiceSubnets: *clusterOptions.ServiceSubnets,
			},
		}

		if !cmp.Equal(actualCluster, expectedCluster) {
			t.Errorf("diff: %s", cmp.Diff(expectedCluster, actualCluster))
		}

		expected := types.Cluster{
			CreatedAt:      actualCluster.CreationTimestamp.Time,
			ID:             string(actualCluster.UID),
			Name:           actualCluster.Name,
			PodSubnets:     &actualCluster.Spec.PodSubnets,
			ServiceSubnets: &actualCluster.Spec.ServiceSubnets,
			Version:        &actualCluster.Status.Version,
		}

		if !cmp.Equal(actual, expected) {
			t.Errorf("diff: %s", cmp.Diff(expected, actual))
		}
	})
}

func TestOrganizationClusters_Delete(t *testing.T) {
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

	t.Run("test as super user", func(t *testing.T) {
		cluster := dockyardsv1.Cluster{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: "test-super-user-",
				Namespace:    organization.Spec.NamespaceRef.Name,
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion: dockyardsv1.GroupVersion.String(),
						Kind:       dockyardsv1.OrganizationKind,
						Name:       organization.Name,
						UID:        organization.UID,
					},
				},
			},
		}

		err := c.Create(ctx, &cluster)
		if err != nil {
			t.Fatal(err)
		}

		err = testingutil.RetryUntilFound(ctx, mgr.GetClient(), &cluster)
		if err != nil {
			t.Fatal(err)
		}

		u := url.URL{
			Path: path.Join("/v1/orgs", organization.Name, "clusters", cluster.Name),
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
		cluster := dockyardsv1.Cluster{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: "test-user-",
				Namespace:    organization.Spec.NamespaceRef.Name,
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion: dockyardsv1.GroupVersion.String(),
						Kind:       dockyardsv1.OrganizationKind,
						Name:       organization.Name,
						UID:        organization.UID,
					},
				},
			},
		}

		err := c.Create(ctx, &cluster)
		if err != nil {
			t.Fatal(err)
		}

		err = testingutil.RetryUntilFound(ctx, mgr.GetClient(), &cluster)
		if err != nil {
			t.Fatal(err)
		}

		u := url.URL{
			Path: path.Join("/v1/orgs", organization.Name, "clusters", cluster.Name),
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
		cluster := dockyardsv1.Cluster{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: "test-reader-",
				Namespace:    organization.Spec.NamespaceRef.Name,
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion: dockyardsv1.GroupVersion.String(),
						Kind:       dockyardsv1.OrganizationKind,
						Name:       organization.Name,
						UID:        organization.UID,
					},
				},
			},
		}

		err := c.Create(ctx, &cluster)
		if err != nil {
			t.Fatal(err)
		}

		err = testingutil.RetryUntilFound(ctx, mgr.GetClient(), &cluster)
		if err != nil {
			t.Fatal(err)
		}

		u := url.URL{
			Path: path.Join("/v1/orgs", organization.Name, "clusters", cluster.Name),
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

	t.Run("test non-existing cluster", func(t *testing.T) {
		u := url.URL{
			Path: path.Join("/v1/orgs", organization.Name, "clusters", "non-existing"),
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

		err := c.Create(ctx, &namespace)
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

		err = c.Create(ctx, &otherOrganization)
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

		u := url.URL{
			Path: path.Join("/v1/orgs", otherOrganization.Name, "clusters", otherCluster.Name),
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

func TestOrganizationClusters_Get(t *testing.T) {
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
					APIVersion: dockyardsv1.GroupVersion.String(),
					Kind:       dockyardsv1.OrganizationKind,
					Name:       organization.Name,
					UID:        organization.UID,
				},
			},
		},
	}

	err := c.Create(ctx, &cluster)
	if err != nil {
		t.Fatal(err)
	}

	patch := client.MergeFrom(cluster.DeepCopy())

	cluster.Status.Version = "v1.2.3"

	err = c.Status().Patch(ctx, &cluster, patch)
	if err != nil {
		t.Fatal(err)
	}

	nodePool := dockyardsv1.NodePool{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "test-",
			Namespace:    cluster.Namespace,
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

	err = testingutil.RetryUntilFound(ctx, mgr.GetClient(), &nodePool)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("test as super user", func(t *testing.T) {
		u := url.URL{
			Path: path.Join("/v1/orgs", organization.Name, "clusters", cluster.Name),
		}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, u.Path, nil)

		r.SetPathValue("organizationName", organization.Name)
		r.SetPathValue("resourceName", cluster.Name)

		r.Header.Add("Authorization", "Bearer "+superUserToken)

		mux.ServeHTTP(w, r)

		statusCode := w.Result().StatusCode
		if statusCode != http.StatusOK {
			t.Fatalf("expected status code %d, got %d", http.StatusOK, statusCode)
		}

		b, err := io.ReadAll(w.Result().Body)
		if err != nil {
			t.Fatalf("error reading result body: %s", err)
		}

		var actual types.Cluster
		err = json.Unmarshal(b, &actual)
		if err != nil {
			t.Fatalf("error unmarshalling result body: %s", err)
		}

		expected := types.Cluster{
			CreatedAt:      cluster.CreationTimestamp.Time,
			ID:             string(cluster.UID),
			Name:           cluster.Name,
			NodePoolsCount: ptr.To(1),
			Version:        &cluster.Status.Version,
		}

		if !cmp.Equal(actual, expected) {
			t.Errorf("diff: %s", cmp.Diff(expected, actual))
		}
	})

	t.Run("test as user", func(t *testing.T) {
		u := url.URL{
			Path: path.Join("/v1/orgs", organization.Name, "clusters", cluster.Name),
		}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, u.Path, nil)

		r.SetPathValue("organizationName", organization.Name)
		r.SetPathValue("resourceName", cluster.Name)

		r.Header.Add("Authorization", "Bearer "+userToken)

		mux.ServeHTTP(w, r)

		statusCode := w.Result().StatusCode
		if statusCode != http.StatusOK {
			t.Fatalf("expected status code %d, got %d", http.StatusOK, statusCode)
		}

		b, err := io.ReadAll(w.Result().Body)
		if err != nil {
			t.Fatalf("error reading result body: %s", err)
		}

		var actual types.Cluster
		err = json.Unmarshal(b, &actual)
		if err != nil {
			t.Fatalf("error unmarshalling result body: %s", err)
		}

		expected := types.Cluster{
			CreatedAt:      cluster.CreationTimestamp.Time,
			ID:             string(cluster.UID),
			Name:           cluster.Name,
			NodePoolsCount: ptr.To(1),
			Version:        &cluster.Status.Version,
		}

		if !cmp.Equal(actual, expected) {
			t.Errorf("diff: %s", cmp.Diff(expected, actual))
		}
	})

	t.Run("test as reader", func(t *testing.T) {
		u := url.URL{
			Path: path.Join("/v1/orgs", organization.Name, "clusters", cluster.Name),
		}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, u.Path, nil)

		r.SetPathValue("organizationName", organization.Name)
		r.SetPathValue("resourceName", cluster.Name)

		r.Header.Add("Authorization", "Bearer "+readerToken)

		mux.ServeHTTP(w, r)

		statusCode := w.Result().StatusCode
		if statusCode != http.StatusOK {
			t.Fatalf("expected status code %d, got %d", http.StatusOK, statusCode)
		}

		b, err := io.ReadAll(w.Result().Body)
		if err != nil {
			t.Fatalf("error reading result body: %s", err)
		}

		var actual types.Cluster
		err = json.Unmarshal(b, &actual)
		if err != nil {
			t.Fatalf("error unmarshalling result body: %s", err)
		}

		expected := types.Cluster{
			CreatedAt:      cluster.CreationTimestamp.Time,
			ID:             string(cluster.UID),
			Name:           cluster.Name,
			NodePoolsCount: ptr.To(1),
			Version:        &cluster.Status.Version,
		}

		if !cmp.Equal(actual, expected) {
			t.Errorf("diff: %s", cmp.Diff(expected, actual))
		}
	})

	t.Run("test non-existing cluster", func(t *testing.T) {
		u := url.URL{
			Path: path.Join("/v1/orgs", organization.Name, "clusters", "non-existing"),
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

		u := url.URL{
			Path: path.Join("/v1/orgs", otherOrganization.Name, "clusters", otherCluster.Name),
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
