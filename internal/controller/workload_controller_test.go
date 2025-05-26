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

	"bitbucket.org/sudosweden/dockyards-backend/internal/controller"
	dockyardsv1 "bitbucket.org/sudosweden/dockyards-backend/api/v1alpha3"
	"bitbucket.org/sudosweden/dockyards-backend/pkg/testing/testingutil"
	"github.com/go-logr/logr"
	"github.com/google/go-cmp/cmp"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestWorkloadReconciler_Inventory(t *testing.T) {
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
	c := testEnvironment.GetClient()

	organization := testEnvironment.MustCreateOrganization(t)

	err = (&controller.WorkloadReconciler{
		Client: mgr.GetClient(),
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

	t.Run("test without workload inventory", func(t *testing.T) {
		workload := dockyardsv1.Workload{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: "test-",
				Namespace:    organization.Spec.NamespaceRef.Name,
			},
			Spec: dockyardsv1.WorkloadSpec{
				Provenience: dockyardsv1.ProvenienceDockyards,
			},
		}

		err := c.Create(ctx, &workload)
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("test empty workload inventory", func(t *testing.T) {
		workload := dockyardsv1.Workload{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: "test-",
				Namespace:    organization.Spec.NamespaceRef.Name,
			},
			Spec: dockyardsv1.WorkloadSpec{
				Provenience: dockyardsv1.ProvenienceDockyards,
			},
		}

		err := c.Create(ctx, &workload)
		if err != nil {
			t.Fatal(err)
		}

		workloadURL := dockyardsv1.WorkloadInventory{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: "test-",
				Namespace:    organization.Spec.NamespaceRef.Name,
				Labels: map[string]string{
					dockyardsv1.LabelWorkloadName: workload.Name,
				},
			},
		}

		err = c.Create(ctx, &workloadURL)
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("test workload inventory urls", func(t *testing.T) {
		workload := dockyardsv1.Workload{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: "test-",
				Namespace:    organization.Spec.NamespaceRef.Name,
			},
			Spec: dockyardsv1.WorkloadSpec{
				Provenience: dockyardsv1.ProvenienceDockyards,
			},
		}

		err := c.Create(ctx, &workload)
		if err != nil {
			t.Fatal(err)
		}

		workloadInventory := dockyardsv1.WorkloadInventory{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: "test-",
				Namespace:    organization.Spec.NamespaceRef.Name,
				Labels: map[string]string{
					dockyardsv1.LabelWorkloadName: workload.Name,
				},
			},
			Spec: dockyardsv1.WorkloadInventorySpec{
				URLs: []string{
					"http://localhost:1234",
				},
			},
		}

		err = c.Create(ctx, &workloadInventory)
		if err != nil {
			t.Fatal(err)
		}

		err = testingutil.RetryUntilFound(ctx, mgr.GetClient(), &workloadInventory)
		if err != nil {
			t.Fatal(err)
		}

		var actual dockyardsv1.Workload
		err = c.Get(ctx, client.ObjectKeyFromObject(&workload), &actual)
		if err != nil {
			t.Fatal(err)
		}

		err = wait.PollUntilContextTimeout(ctx, time.Millisecond*200, time.Second*5, true, func(ctx context.Context) (bool, error) {
			err := c.Get(ctx, client.ObjectKeyFromObject(&workload), &actual)
			if err != nil {
				return true, err
			}

			return len(actual.Status.URLs) > 0, nil
		})
		if err != nil {
			t.Fatal(err)
		}

		expected := dockyardsv1.Workload{
			ObjectMeta: actual.ObjectMeta,
			Spec:       actual.Spec,
			Status: dockyardsv1.WorkloadStatus{
				URLs: []string{
					"http://localhost:1234",
				},
				//
				Conditions: actual.Status.Conditions,
			},
		}

		if !cmp.Equal(actual, expected) {
			t.Errorf("diff: %s", cmp.Diff(expected, actual))
		}
	})
}
