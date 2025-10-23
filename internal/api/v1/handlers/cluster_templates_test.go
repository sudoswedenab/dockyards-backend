package handlers_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/sudoswedenab/dockyards-api/pkg/types"
	dockyardsv1 "github.com/sudoswedenab/dockyards-backend/api/v1alpha3"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
)

func TestGlobalClusterTemplates_List(t *testing.T) {
	c := testEnvironment.GetClient()

	organization := testEnvironment.MustCreateOrganization(t)
	reader := testEnvironment.MustGetOrganizationUser(t, organization, dockyardsv1.OrganizationMemberRoleReader)
	readerToken := MustSignToken(t, reader.Name)

	t.Run("test empty", func(t *testing.T) {
		u := url.URL{
			Path: "/v1/cluster-templates",
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
			t.Fatal(err)
		}

		expected := []types.ClusterTemplate{}

		var actual []types.ClusterTemplate
		err = json.Unmarshal(b, &actual)
		if err != nil {
			t.Fatal(err)
		}

		if !cmp.Equal(actual, expected) {
			t.Errorf("diff: %s", cmp.Diff(expected, actual))
		}
	})

	t.Run("test single template", func(t *testing.T) {
		clusterTemplate := dockyardsv1.ClusterTemplate{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: "test-",
				Namespace:    testEnvironment.GetDockyardsNamespace(),
			},
			Spec: dockyardsv1.ClusterTemplateSpec{
				NodePoolTemplates: []dockyardsv1.NodePoolTemplate{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "node-pool",
						},
						Spec: dockyardsv1.NodePoolSpec{
							ControlPlane: true,
							Replicas:     ptr.To(int32(3)),
							Resources: corev1.ResourceList{
								corev1.ResourceCPU:     resource.MustParse("2"),
								corev1.ResourceMemory:  resource.MustParse("3M"),
								corev1.ResourceStorage: resource.MustParse("4Gi"),
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

		u := url.URL{
			Path: "/v1/cluster-templates",
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
			t.Fatal(err)
		}

		expected := []types.ClusterTemplate{
			{
				Name:      clusterTemplate.Name,
				IsDefault: false,
				ClusterOptions: types.ClusterOptions{
					NodePoolOptions: &[]types.NodePoolOptions{
						{
							ControlPlane: ptr.To(true),
							CPUCount:     ptr.To(2),
							DiskSize:     ptr.To("4Gi"),
							Name:         ptr.To("node-pool"),
							Quantity:     ptr.To(3),
							RAMSize:      ptr.To("3M"),
						},
					},
				},
			},
		}

		var actual []types.ClusterTemplate
		err = json.Unmarshal(b, &actual)
		if err != nil {
			t.Fatal(err)
		}

		if !cmp.Equal(actual, expected) {
			t.Errorf("diff: %s", cmp.Diff(expected, actual))
		}
	})
}
