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
	"reflect"
	"testing"

	"github.com/go-logr/logr"
	dyconfig "github.com/sudoswedenab/dockyards-backend/api/config"
	dockyardsv1 "github.com/sudoswedenab/dockyards-backend/api/v1alpha3"
	"github.com/sudoswedenab/dockyards-backend/internal/controller"
	"github.com/sudoswedenab/dockyards-backend/pkg/testing/testingutil"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestUserReconciler_Reconcile(t *testing.T) {
	if os.Getenv("KUBEBUILDER_ASSETS") == "" {
		t.Skip("no kubebuilder assets configured")
	}

	ctx, cancel := context.WithCancel(t.Context())

	handler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo})
	log := logr.FromSlogHandler(handler)
	ctrl.SetLogger(log)

	testEnvironment, err := testingutil.NewTestEnvironment(ctx, []string{path.Join("../../config/crd")})
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		cancel()
		testEnvironment.GetEnvironment().Stop()
	})

	c := testEnvironment.GetClient()

	config := dyconfig.NewFakeConfigManager(map[string]string{
		string(dyconfig.KeyExternalURL): "http://test.com",
	})
	if err != nil {
		t.Fatal(err)
	}

	reconciler := controller.UserReconciler{
		Client: c,
		Config: config,
	}

	reconcileUserCalls := 0
	reconcileUserWithName := func(t *testing.T, name string) {
		// Reconcile as long as the reconciliation yields a different object, but with a limit as to avoid infinite loops.
		defer func() {
			reconcileUserCalls++
		}()

		limit := 10
		i := 0
		for i = 0; i < limit; i++ {
			t.Logf("reconcileUserWithName(%d): reconciling user %s: iteration: %d", reconcileUserCalls, name, i)

			var user dockyardsv1.User
			err := c.Get(ctx, types.NamespacedName{Name: name}, &user)
			if err != nil {
				t.Fatalf("reconcileUserWithName(%d): %s", reconcileUserCalls, err)
			}
			result, err := reconciler.Reconcile(ctx, ctrl.Request{NamespacedName: types.NamespacedName{Name: name}})
			if err != nil {
				t.Fatalf("reconcileUserWithName(%d): %s", reconcileUserCalls, err)
			}
			if result.Requeue {
				t.Logf("reconcileUserWithName(%d): reconciler requested requeue", reconcileUserCalls)
				continue
			}
			if result.RequeueAfter.Nanoseconds() != 0 {
				// We do not delay, since we should avoid treating the passage of time as a syncronization primitive (see: https://xeiaso.net/blog/nosleep/)
				continue
			}

			var user2 dockyardsv1.User
			err = c.Get(ctx, types.NamespacedName{Name: name}, &user2)
			if err != nil {
				if errors.IsNotFound(err) {
					break
				}
				t.Fatal(err)
			}
			if !reflect.DeepEqual(&user, &user2) {
				continue
			}

			t.Logf("reconcileUserWithName(%d): object has stopped changing", reconcileUserCalls)
			break
		}
		if i == limit {
			t.Fatalf("reconcileUserWithName(%d): reconciler loop limit reached", reconcileUserCalls)
		}
	}

	t.Run("creating a dockyards user eventually creates a verification request", func(t *testing.T) {
		user := dockyardsv1.User{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-0245e314",
			},
			Spec: dockyardsv1.UserSpec{
				DisplayName: "test",
				Email:       "test+0245e314@test.com",
				Password:    "test",
				ProviderID:  dockyardsv1.ProviderPrefixDockyards,
			},
		}
		err := c.Create(ctx, &user)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() {
			c.Delete(ctx, &user)
		})

		reconcileUserWithName(t, user.Name)

		vr := dockyardsv1.VerificationRequest{ObjectMeta: metav1.ObjectMeta{Name: "sign-up-" + user.Name}}
		err = testingutil.RetryUntilFound(ctx, c, &vr)
		if err != nil {
			t.Fatalf("expected to find verification request for user %s, but got err: %s", user.Name, err)
		}
	})

	t.Run("creating a dockyards user adds a ready false to it", func(t *testing.T) {
		user := dockyardsv1.User{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-517d5fa0",
			},
			Spec: dockyardsv1.UserSpec{
				DisplayName: "test",
				Email:       "test+517d5fa0@test.com",
				Password:    "test",
				ProviderID:  dockyardsv1.ProviderPrefixDockyards,
			},
		}
		err := c.Create(ctx, &user)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() {
			c.Delete(ctx, &user)
		})

		reconcileUserWithName(t, user.Name)

		err = c.Get(ctx, client.ObjectKeyFromObject(&user), &user)
		if err != nil {
			t.Fatal(err)
		}

		ready := meta.FindStatusCondition(user.GetConditions(), dockyardsv1.ReadyCondition)
		if ready == nil {
			t.Fatal("expected to find ready condition on dockyards user")
		}

		if ready.Status != metav1.ConditionFalse {
			t.Fatal("expected user ready condition to be False")
		}
	})

	t.Run("user is marked as ready when verification request is verified", func(t *testing.T) {
		user := dockyardsv1.User{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-0b92e552",
			},
			Spec: dockyardsv1.UserSpec{
				DisplayName: "test",
				Email:       "test+0b92e552@test.com",
				Password:    "test",
				ProviderID:  dockyardsv1.ProviderPrefixDockyards,
			},
		}
		err := c.Create(ctx, &user)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() {
			c.Delete(ctx, &user)
		})

		reconcileUserWithName(t, user.Name)

		err = c.Get(ctx, client.ObjectKeyFromObject(&user), &user)
		if err != nil {
			t.Fatal(err)
		}

		ready := meta.FindStatusCondition(user.Status.Conditions, dockyardsv1.ReadyCondition)
		if ready.Status != metav1.ConditionFalse {
			t.Fatalf("expected user ready condition to be False, but found %s", ready.Status)
		}

		verificationRequest := dockyardsv1.VerificationRequest{
			ObjectMeta: metav1.ObjectMeta{
				Name: "sign-up-" + user.Name,
			},
		}

		err = c.Get(ctx, client.ObjectKeyFromObject(&verificationRequest), &verificationRequest)
		if err != nil {
			// Verification request should exist
			t.Fatal(err)
		}

		patch := client.MergeFrom(verificationRequest.DeepCopy())

		meta.SetStatusCondition(&verificationRequest.Status.Conditions, metav1.Condition{
			Type:    dockyardsv1.VerifiedCondition,
			Status:  metav1.ConditionTrue,
			Reason:  dockyardsv1.VerificationReasonVerified,
			Message: "Verified by some test",
		})

		err = c.Status().Patch(ctx, &verificationRequest, patch)
		if err != nil {
			t.Fatal(err)
		}

		reconcileUserWithName(t, user.Name)

		err = c.Get(ctx, client.ObjectKeyFromObject(&user), &user)
		if err != nil {
			t.Fatal(err)
		}

		ready = meta.FindStatusCondition(user.Status.Conditions, dockyardsv1.ReadyCondition)
		if ready == nil {
			t.Fatal("expected user ready condition to be True, but found nil")
		}
		if ready.Status != metav1.ConditionTrue {
			t.Fatalf("expected user ready condition to be True, but found %s", ready.Status)
		}
	})

	t.Run("verification request is deleted when user is marked as ready", func(t *testing.T) {
		user := dockyardsv1.User{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-4b24d182",
			},
			Spec: dockyardsv1.UserSpec{
				DisplayName: "test",
				Email:       "test+4b24d182@test.com",
				Password:    "test",
				ProviderID:  dockyardsv1.ProviderPrefixDockyards,
			},
		}
		err := c.Create(ctx, &user)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() {
			c.Delete(ctx, &user)
		})

		reconcileUserWithName(t, user.Name)

		verificationRequest := dockyardsv1.VerificationRequest{
			ObjectMeta: metav1.ObjectMeta{
				Name: "sign-up-" + user.Name,
			},
		}

		err = c.Get(ctx, client.ObjectKeyFromObject(&verificationRequest), &verificationRequest)
		if err != nil {
			// Verification request should exist
			t.Fatal(err)
		}

		patch := client.MergeFrom(user.DeepCopy())

		meta.SetStatusCondition(&user.Status.Conditions, metav1.Condition{
			Type:    dockyardsv1.ReadyCondition,
			Status:  metav1.ConditionTrue,
			Reason:  dockyardsv1.VerificationReasonVerified,
			Message: "Verified by some test",
		})

		err = c.Status().Patch(ctx, &user, patch)
		if err != nil {
			t.Fatal(err)
		}

		err = c.Get(ctx, client.ObjectKeyFromObject(&user), &user)
		if err != nil {
			t.Fatal(err)
		}

		ready := meta.FindStatusCondition(user.Status.Conditions, dockyardsv1.ReadyCondition)
		if ready == nil {
			t.Fatal("expected user ready condition to be True, but found nil")
		}
		if ready.Status != metav1.ConditionTrue {
			t.Fatalf("expected user ready condition to be True, but found %s", ready.Status)
		}

		reconcileUserWithName(t, user.Name)

		err = c.Get(ctx, client.ObjectKeyFromObject(&verificationRequest), &verificationRequest)
		if !errors.IsNotFound(err) {
			t.Fatal("expected verification request to be deleted after user has been marked as ready")
		}
	})
}
