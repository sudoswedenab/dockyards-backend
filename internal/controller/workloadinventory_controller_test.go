package controller_test

import (
	"context"
	"log/slog"
	"os"
	"path"
	"testing"

	"github.com/go-logr/logr"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	dockyardsv1 "github.com/sudoswedenab/dockyards-backend/api/v1alpha3"
	"github.com/sudoswedenab/dockyards-backend/api/v1alpha3/index"
	"github.com/sudoswedenab/dockyards-backend/pkg/testing/testingutil"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestWorkloadInventory_BySelectorIndex(t *testing.T) {
	if os.Getenv("KUBEBUILDER_ASSETS") == "" {
		t.Skip("no kubebuilder assets configured")
	}

	ctx, cancel := context.WithCancel(context.TODO())

	handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError})
	slogr := logr.FromSlogHandler(handler)
	ctrl.SetLogger(slogr)

	testEnvironment, err := testingutil.NewTestEnvironment(ctx, []string{path.Join("../../config/crd")})
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		cancel()
		testEnvironment.GetEnvironment().Stop()
	})

	mgr := testEnvironment.GetManager()
	c := mgr.GetClient()

	err = index.BySelector(ctx, mgr)
	if err != nil {
		t.Fatal(err)
	}

	organization := testEnvironment.MustCreateOrganization(t)
	otherOrganization := testEnvironment.MustCreateOrganization(t)

	workloadInventoryList := dockyardsv1.WorkloadInventoryList{
		Items: []dockyardsv1.WorkloadInventory{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: organization.Spec.NamespaceRef.Name,
				},
				Spec: dockyardsv1.WorkloadInventorySpec{
					Selector: metav1.LabelSelector{
						MatchLabels: map[string]string{
							"backend.dockyards.io/testing-name":      "test",
							"backend.dockyards.io/testing-namespace": "testing",
						},
					},
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: otherOrganization.Spec.NamespaceRef.Name,
				},
				Spec: dockyardsv1.WorkloadInventorySpec{
					Selector: metav1.LabelSelector{
						MatchLabels: map[string]string{
							"backend.dockyards.io/testing-name":      "test",
							"backend.dockyards.io/testing-namespace": "testing",
						},
					},
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "other-test",
					Namespace: otherOrganization.Spec.NamespaceRef.Name,
				},
				Spec: dockyardsv1.WorkloadInventorySpec{
					Selector: metav1.LabelSelector{
						MatchLabels: map[string]string{
							"backend.dockyards.io/testing-name":      "test",
							"backend.dockyards.io/testing-namespace": "other-testing",
						},
					},
				},
			},
		},
	}

	go func() {
		err := mgr.Start(ctx)
		if err != nil {
			panic(err)
		}
	}()

	if !mgr.GetCache().WaitForCacheSync(ctx) {
		t.Fatal("unable to wait for cache sync")
	}

	for _, workloadInventory := range workloadInventoryList.Items {
		err := c.Create(ctx, &workloadInventory)
		if err != nil {
			t.Fatal(err)
		}
	}

	ignoreFields := cmpopts.IgnoreFields(metav1.ObjectMeta{}, "UID", "ResourceVersion", "Generation", "CreationTimestamp", "ManagedFields")

	t.Run("test organization", func(t *testing.T) {
		matchLabels := map[string]string{
			"backend.dockyards.io/testing-name":      "test",
			"backend.dockyards.io/testing-namespace": "testing",
		}

		matchingFields := client.MatchingFields{
			index.SelectorField: index.MatchLabelsSummary(matchLabels),
		}

		var actual dockyardsv1.WorkloadInventoryList

		err := c.List(ctx, &actual, matchingFields, client.InNamespace(organization.Spec.NamespaceRef.Name))
		if err != nil {
			t.Fatal(err)
		}

		expected := []dockyardsv1.WorkloadInventory{
			{
				TypeMeta: metav1.TypeMeta{
					APIVersion: dockyardsv1.GroupVersion.String(),
					Kind:       dockyardsv1.WorkloadInventoryKind,
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: organization.Spec.NamespaceRef.Name,
				},
				Spec: dockyardsv1.WorkloadInventorySpec{
					Selector: metav1.LabelSelector{
						MatchLabels: matchLabels,
					},
				},
			},
		}

		if !cmp.Equal(actual.Items, expected, ignoreFields) {
			t.Errorf("diff: %s", cmp.Diff(expected, actual.Items, ignoreFields))
		}
	})

	t.Run("test other organization", func(t *testing.T) {
		matchLabels := map[string]string{
			"backend.dockyards.io/testing-name":      "test",
			"backend.dockyards.io/testing-namespace": "testing",
		}

		matchingFields := client.MatchingFields{
			index.SelectorField: index.MatchLabelsSummary(matchLabels),
		}

		var actual dockyardsv1.WorkloadInventoryList

		err := c.List(ctx, &actual, matchingFields, client.InNamespace(otherOrganization.Spec.NamespaceRef.Name))
		if err != nil {
			t.Fatal(err)
		}

		expected := []dockyardsv1.WorkloadInventory{
			{
				TypeMeta: metav1.TypeMeta{
					APIVersion: dockyardsv1.GroupVersion.String(),
					Kind:       dockyardsv1.WorkloadInventoryKind,
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: otherOrganization.Spec.NamespaceRef.Name,
				},
				Spec: dockyardsv1.WorkloadInventorySpec{
					Selector: metav1.LabelSelector{
						MatchLabels: matchLabels,
					},
				},
			},
		}

		if !cmp.Equal(actual.Items, expected, ignoreFields) {
			t.Errorf("diff: %s", cmp.Diff(expected, actual.Items, ignoreFields))
		}
	})
}
