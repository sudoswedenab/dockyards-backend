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

package controller_test

import (
	"context"
	"log/slog"
	"os"
	"path"
	"testing"
	"time"

	"github.com/fluxcd/pkg/runtime/conditions"
	"github.com/go-logr/logr"
	"github.com/google/go-cmp/cmp"
	dockyardsv1 "github.com/sudoswedenab/dockyards-backend/api/v1alpha3"
	"github.com/sudoswedenab/dockyards-backend/internal/controller"
	"github.com/sudoswedenab/dockyards-backend/pkg/testing/testingutil"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestClusterController_Upgrades(t *testing.T) {
	if os.Getenv("KUBEBUILDER_ASSETS") == "" {
		t.Skip("no kubebuilder assets configured")
	}

	handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError})
	slogr := logr.FromSlogHandler(handler)
	ctrl.SetLogger(slogr)

	ctx, cancel := context.WithCancel(context.TODO())

	testEnvironment, err := testingutil.NewTestEnvironment(ctx, []string{path.Join("../../config/crd")})
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		cancel()
		testEnvironment.GetEnvironment().Stop()
	})

	mgr := testEnvironment.GetManager()
	c := testEnvironment.GetClient()

	organization := testEnvironment.MustCreateOrganization(t)

	release := dockyardsv1.Release{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: testEnvironment.GetDockyardsNamespace(),
			Annotations: map[string]string{
				dockyardsv1.AnnotationDefaultRelease: "true",
			},
		},
		Spec: dockyardsv1.ReleaseSpec{
			Type: dockyardsv1.ReleaseTypeKubernetes,
		},
	}

	err = c.Create(ctx, &release)
	if err != nil {
		t.Fatal(err)
	}

	patch := client.MergeFrom(release.DeepCopy())

	release.Status.Versions = []string{
		"v1.30.1",
		"v1.29.5",
		"v1.28.10",
		"v1.27.14",
	}

	err = c.Status().Patch(ctx, &release, patch)
	if err != nil {
		t.Fatal(err)
	}

	err = (&controller.ClusterReconciler{
		Client:             mgr.GetClient(),
		DockyardsNamespace: testEnvironment.GetDockyardsNamespace(),
	}).SetupWithManager(mgr)
	if err != nil {
		t.Fatal(err)
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

	err = testingutil.RetryUntilFound(ctx, mgr.GetClient(), &release)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("test latest version", func(t *testing.T) {
		cluster := dockyardsv1.Cluster{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-latest-version",
				Namespace: organization.Spec.NamespaceRef.Name,
			},
			Spec: dockyardsv1.ClusterSpec{
				Version: "v1.30.1",
			},
		}

		err := c.Create(ctx, &cluster)
		if err != nil {
			t.Fatal(err)
		}

		var actual dockyardsv1.Cluster

		err = wait.PollUntilContextTimeout(ctx, time.Millisecond*200, time.Second*5, true, func(ctx context.Context) (bool, error) {
			err := c.Get(ctx, client.ObjectKeyFromObject(&cluster), &actual)
			if err != nil {
				return true, err
			}

			if conditions.IsTrue(&actual, dockyardsv1.ClusterUpgradesReadyCondition) {
				return true, nil
			}

			return false, nil
		})
		if err != nil {
			t.Fatal(err)
		}

		var expected []dockyardsv1.ClusterUpgrade

		if !cmp.Equal(actual.Spec.Upgrades, expected) {
			t.Errorf("diff: %s", cmp.Diff(expected, actual.Spec.Upgrades))
		}
	})

	t.Run("test patch version", func(t *testing.T) {
		cluster := dockyardsv1.Cluster{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-patch-upgrade",
				Namespace: organization.Spec.NamespaceRef.Name,
			},
			Spec: dockyardsv1.ClusterSpec{
				Version: "v1.30.0",
			},
		}

		err := c.Create(ctx, &cluster)
		if err != nil {
			t.Fatal(err)
		}

		var actual dockyardsv1.Cluster

		expected := []dockyardsv1.ClusterUpgrade{
			{
				To: "v1.30.1",
			},
		}

		err = wait.PollUntilContextTimeout(ctx, time.Millisecond*200, time.Second*5, true, func(ctx context.Context) (bool, error) {
			err := c.Get(ctx, client.ObjectKeyFromObject(&cluster), &actual)
			if err != nil {
				return true, err
			}

			return cmp.Equal(actual.Spec.Upgrades, expected), nil
		})
		if err != nil {
			t.Fatal(err)
		}

		if !cmp.Equal(actual.Spec.Upgrades, expected) {
			t.Errorf("diff: %s", cmp.Diff(expected, actual.Spec.Upgrades))
		}
	})

	t.Run("test minor version", func(t *testing.T) {
		cluster := dockyardsv1.Cluster{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-minor-version",
				Namespace: organization.Spec.NamespaceRef.Name,
			},
			Spec: dockyardsv1.ClusterSpec{
				Version: "v1.29.5",
			},
		}

		err := c.Create(ctx, &cluster)
		if err != nil {
			t.Fatal(err)
		}

		var actual dockyardsv1.Cluster

		expected := []dockyardsv1.ClusterUpgrade{
			{
				To: "v1.30.1",
			},
		}

		err = wait.PollUntilContextTimeout(ctx, time.Millisecond*200, time.Second*5, true, func(ctx context.Context) (bool, error) {
			err := c.Get(ctx, client.ObjectKeyFromObject(&cluster), &actual)
			if err != nil {
				return true, err
			}

			return cmp.Equal(actual.Spec.Upgrades, expected), nil
		})
		if err != nil {
			t.Fatal(err)
		}

		if !cmp.Equal(actual.Spec.Upgrades, expected) {
			t.Errorf("diff: %s", cmp.Diff(expected, actual.Spec.Upgrades))
		}
	})

	t.Run("test minor and patch versions", func(t *testing.T) {
		cluster := dockyardsv1.Cluster{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-minor-patch-version",
				Namespace: organization.Spec.NamespaceRef.Name,
			},
			Spec: dockyardsv1.ClusterSpec{
				Version: "v1.29.4",
			},
		}

		err := c.Create(ctx, &cluster)
		if err != nil {
			t.Fatal(err)
		}

		var actual dockyardsv1.Cluster

		expected := []dockyardsv1.ClusterUpgrade{
			{
				To: "v1.30.1",
			},
			{
				To: "v1.29.5",
			},
		}

		err = wait.PollUntilContextTimeout(ctx, time.Millisecond*200, time.Second*5, true, func(ctx context.Context) (bool, error) {
			err := c.Get(ctx, client.ObjectKeyFromObject(&cluster), &actual)
			if err != nil {
				return true, err
			}

			return cmp.Equal(actual.Spec.Upgrades, expected), nil
		})
		if err != nil {
			t.Fatal(err)
		}

		if !cmp.Equal(actual.Spec.Upgrades, expected) {
			t.Errorf("diff: %s", cmp.Diff(expected, actual.Spec.Upgrades))
		}
	})

	t.Run("test penultimate version", func(t *testing.T) {
		cluster := dockyardsv1.Cluster{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-penultimate-version",
				Namespace: organization.Spec.NamespaceRef.Name,
			},
			Spec: dockyardsv1.ClusterSpec{
				Version: "v1.28.10",
			},
		}

		err := c.Create(ctx, &cluster)
		if err != nil {
			t.Fatal(err)
		}

		var actual dockyardsv1.Cluster

		expected := []dockyardsv1.ClusterUpgrade{
			{
				To: "v1.29.5",
			},
		}

		err = wait.PollUntilContextTimeout(ctx, time.Millisecond*200, time.Second*5, true, func(ctx context.Context) (bool, error) {
			err := c.Get(ctx, client.ObjectKeyFromObject(&cluster), &actual)
			if err != nil {
				return true, err
			}

			return cmp.Equal(actual.Spec.Upgrades, expected), nil
		})
		if err != nil {
			t.Fatal(err)
		}

		if !cmp.Equal(actual.Spec.Upgrades, expected) {
			t.Errorf("diff: %s", cmp.Diff(expected, actual.Spec.Upgrades))
		}
	})

	t.Run("test old version", func(t *testing.T) {
		cluster := dockyardsv1.Cluster{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-old-version",
				Namespace: organization.Spec.NamespaceRef.Name,
			},
			Spec: dockyardsv1.ClusterSpec{
				Version: "v1.24.17",
			},
		}

		err := c.Create(ctx, &cluster)
		if err != nil {
			t.Fatal(err)
		}

		var actual dockyardsv1.Cluster

		err = wait.PollUntilContextTimeout(ctx, time.Millisecond*200, time.Second*5, true, func(ctx context.Context) (bool, error) {
			err := c.Get(ctx, client.ObjectKeyFromObject(&cluster), &actual)
			if err != nil {
				return true, err
			}

			if conditions.IsTrue(&actual, dockyardsv1.ClusterUpgradesReadyCondition) {
				return true, nil
			}

			return false, nil
		})
		if err != nil {
			t.Fatal(err)
		}

		var expected []dockyardsv1.ClusterUpgrade

		if !cmp.Equal(actual.Spec.Upgrades, expected) {
			t.Errorf("diff: %s", cmp.Diff(expected, actual.Spec.Upgrades))
		}
	})

	t.Run("test below supported version", func(t *testing.T) {
		cluster := dockyardsv1.Cluster{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-below-supported-version",
				Namespace: organization.Spec.NamespaceRef.Name,
			},
			Spec: dockyardsv1.ClusterSpec{
				Version: "v1.26.15",
			},
		}

		err := c.Create(ctx, &cluster)
		if err != nil {
			t.Fatal(err)
		}

		var actual dockyardsv1.Cluster

		expected := []dockyardsv1.ClusterUpgrade{
			{
				To: "v1.27.14",
			},
		}

		err = wait.PollUntilContextTimeout(ctx, time.Millisecond*200, time.Second*5, true, func(ctx context.Context) (bool, error) {
			err := c.Get(ctx, client.ObjectKeyFromObject(&cluster), &actual)
			if err != nil {
				return true, err
			}

			return cmp.Equal(actual.Spec.Upgrades, expected), nil
		})
		if err != nil {
			t.Fatal(err)
		}

		if !cmp.Equal(actual.Spec.Upgrades, expected) {
			t.Errorf("diff: %s", cmp.Diff(expected, actual.Spec.Upgrades))
		}
	})
}
