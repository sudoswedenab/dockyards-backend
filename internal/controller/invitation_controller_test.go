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

func TestInvitationController_ReconcileExpiration(t *testing.T) {
	if os.Getenv("KUBEBUILDER_ASSETS") == "" {
		t.Skip("no kubebuilder assets configured")
	}

	ctx := t.Context()

	handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError})
	slogr := logr.FromSlogHandler(handler)
	ctrl.SetLogger(slogr)

	testEnvironment, err := testingutil.NewTestEnvironment(ctx, []string{path.Join("../../config/crd")})
	if err != nil {
		t.Fatal(err)
	}

	organization := testEnvironment.MustCreateOrganization(t)

	t.Cleanup(func() {
		testEnvironment.GetEnvironment().Stop()
	})

	mgr := testEnvironment.GetManager()
	c := testEnvironment.GetClient()

	err = (&controller.InvitationReconciler{
		Client: mgr.GetClient(),
	}).SetupWithManger(mgr)
	if err != nil {
		t.Fatal(err)
	}

	go func() {
		err := mgr.Start(ctx)
		if err != nil {
			t.Error(err)
		}
	}()

	if !mgr.GetCache().WaitForCacheSync(ctx) {
		t.Fatal("unable to wait for cache sync")
	}

	t.Run("test invitation", func(t *testing.T) {
		invitation := dockyardsv1.Invitation{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: "test-",
				Namespace:    organization.Spec.NamespaceRef.Name,
			},
			Spec: dockyardsv1.InvitationSpec{
				Role: dockyardsv1.RoleReader,
			},
		}

		err := c.Create(ctx, &invitation)
		if err != nil {
			t.Fatal(err)
		}

		var actual dockyardsv1.Invitation

		err = wait.PollUntilContextTimeout(ctx, time.Millisecond*200, time.Second*5, true, func(ctx context.Context) (bool, error) {
			err := c.Get(ctx, client.ObjectKeyFromObject(&invitation), &actual)
			if err != nil {
				return true, err
			}

			return invitation.Status.ExpirationTimestamp == nil, nil
		})
		if err != nil {
			t.Fatal(err)
		}

		expected := dockyardsv1.Invitation{
			ObjectMeta: actual.ObjectMeta,
			Spec:       actual.Spec,
			Status:     dockyardsv1.InvitationStatus{},
		}

		if !cmp.Equal(actual, expected) {
			t.Errorf("diff: %s", cmp.Diff(expected, actual))
		}
	})

	t.Run("test expiration", func(t *testing.T) {
		invitation := dockyardsv1.Invitation{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: "test-",
				Namespace:    organization.Spec.NamespaceRef.Name,
			},
			Spec: dockyardsv1.InvitationSpec{
				Duration: &metav1.Duration{
					Duration: time.Minute * 5,
				},
				Role: dockyardsv1.RoleReader,
			},
		}

		err := c.Create(ctx, &invitation)
		if err != nil {
			t.Fatal(err)
		}

		var actual dockyardsv1.Invitation

		err = wait.PollUntilContextTimeout(ctx, time.Millisecond*200, time.Second*5, true, func(ctx context.Context) (bool, error) {
			err := c.Get(ctx, client.ObjectKeyFromObject(&invitation), &actual)
			if err != nil {
				return true, err
			}

			return actual.Status.ExpirationTimestamp != nil, nil
		})
		if err != nil {
			t.Fatal(err)
		}

		expected := dockyardsv1.Invitation{
			ObjectMeta: actual.ObjectMeta,
			Spec:       actual.Spec,
			Status: dockyardsv1.InvitationStatus{
				ExpirationTimestamp: &metav1.Time{
					Time: invitation.CreationTimestamp.Add(invitation.Spec.Duration.Duration),
				},
			},
		}

		if !cmp.Equal(actual, expected) {
			t.Errorf("diff: %s", cmp.Diff(expected, actual))
		}
	})

	t.Run("test expired", func(t *testing.T) {
		invitation := dockyardsv1.Invitation{
			ObjectMeta: metav1.ObjectMeta{
				Finalizers: []string{
					"backend.dockyards.io/testing",
				},
				GenerateName: "test-",
				Namespace:    organization.Spec.NamespaceRef.Name,
			},
			Spec: dockyardsv1.InvitationSpec{
				Duration: &metav1.Duration{
					Duration: time.Second,
				},
				Role: dockyardsv1.RoleReader,
			},
		}

		err := c.Create(ctx, &invitation)
		if err != nil {
			t.Fatal(err)
		}

		var actual dockyardsv1.Invitation

		err = wait.PollUntilContextTimeout(ctx, time.Millisecond*200, time.Second*5, true, func(ctx context.Context) (bool, error) {
			err := c.Get(ctx, client.ObjectKeyFromObject(&invitation), &actual)
			if err != nil {
				return true, err
			}

			return !actual.DeletionTimestamp.IsZero(), nil
		})
		if err != nil {
			t.Fatal(err)
		}
	})
}
