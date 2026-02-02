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

	"github.com/fluxcd/pkg/runtime/conditions"
	"github.com/go-logr/logr"
	dyconfig "github.com/sudoswedenab/dockyards-backend/api/config"
	dockyardsv1 "github.com/sudoswedenab/dockyards-backend/api/v1alpha3"
	"github.com/sudoswedenab/dockyards-backend/internal/controller"
	"github.com/sudoswedenab/dockyards-backend/pkg/testing/testingutil"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestUserReconciler_Reconcile(t *testing.T) {
	if os.Getenv("KUBEBUILDER_ASSETS") == "" {
		t.Skip("no kubebuilder assets configured")
	}

	userName := "test-eadcb6eb" // Some random name to avoid collisions with other tests

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

	t.Run("creating dockyards user creates verification request", func(t *testing.T) {
		user := dockyardsv1.User{
			ObjectMeta: metav1.ObjectMeta{
				Name: userName,
			},
			Spec: dockyardsv1.UserSpec{
				DisplayName: "test",
				Email:       "test@test.com",
				Password:    "test",
				ProviderID:  dockyardsv1.ProviderPrefixDockyards,
			},
		}
		err := c.Create(ctx, &user)
		if err != nil {
			t.Fatal(err)
		}

		_, err = reconciler.Reconcile(ctx, ctrl.Request{NamespacedName: types.NamespacedName{Name: userName}})
		if err != nil {
			t.Fatal(err)
		}

		vr := dockyardsv1.VerificationRequest{ObjectMeta: metav1.ObjectMeta{Name: "sign-up-" + user.Name}}
		err = testingutil.RetryUntilFound(ctx, c, &vr)
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("dockyards user is not ready until verification request has been confirmed", func(t *testing.T) {
		user := dockyardsv1.User{
			ObjectMeta: metav1.ObjectMeta{
				Name: userName,
			},
		}

		err := testingutil.RetryUntilFound(ctx, c, &user)
		if err != nil {
			t.Fatal(err)
		}

		// Reconcile one extra time for verified = false
		_, err = reconciler.Reconcile(ctx, ctrl.Request{NamespacedName: types.NamespacedName{Name: userName}})
		if err != nil {
			t.Fatal(err)
		}

		err = testingutil.RetryUntilFound(ctx, c, &user)
		if err != nil {
			t.Fatal(err)
		}

		userVerifiedCondition := conditions.Get(&user, dockyardsv1.VerifiedCondition)
		if userVerifiedCondition == nil {
			t.Fatalf("expected user verified condition to be False, but found nil")
		}
		if userVerifiedCondition.Status != metav1.ConditionFalse {
			t.Fatalf("expected user verified condition to be False, but found %s", userVerifiedCondition.Status)
		}

		verificationRequest := dockyardsv1.VerificationRequest{
			ObjectMeta: metav1.ObjectMeta{
				Name: "sign-up-" + userName,
			},
		}

		err = testingutil.RetryUntilFound(ctx, c, &verificationRequest)
		if err != nil {
			t.Fatal(err)
		}

		verified := conditions.Get(&verificationRequest, dockyardsv1.VerifiedCondition)
		if verified == nil {
			t.Fatal("expected verification request verified condition to be False, but found nil")
		}
		if verified.Status != metav1.ConditionFalse {
			t.Fatalf("expected verification request verified condition to be False, but found %s", verified.Status)
		}

		verified = &metav1.Condition{
			Type:               dockyardsv1.VerifiedCondition,
			Status:             metav1.ConditionTrue,
			Reason:             dockyardsv1.VerificationReasonVerified,
			Message:            "",
			LastTransitionTime: metav1.Now(),
		}

		conditions.Set(&verificationRequest, verified)

		err = c.Status().Update(ctx, &verificationRequest)
		if err != nil {
			t.Fatal(err)
		}

		_, err = reconciler.Reconcile(ctx, ctrl.Request{NamespacedName: types.NamespacedName{Name: userName}})
		if err != nil {
			t.Fatal(err)
		}

		err = c.Get(ctx, client.ObjectKey{Name: userName}, &user)
		if err != nil {
			t.Fatal(err)
		}

		userVerifiedCondition = conditions.Get(&user, dockyardsv1.VerifiedCondition)
		if userVerifiedCondition == nil {
			t.Fatalf("expected user verified condition to be True, but found nil")
		}
		if userVerifiedCondition.Status != metav1.ConditionTrue {
			t.Fatalf("expected user verified condition to be True, but found %s", userVerifiedCondition.Status)
		}
	})
}
