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
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	dockyardsv1 "github.com/sudoswedenab/dockyards-backend/api/v1alpha3"
	"github.com/sudoswedenab/dockyards-backend/internal/controller"
	"github.com/sudoswedenab/dockyards-backend/pkg/testing/testingutil"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestUserReconciler_Reconcile(t *testing.T) {
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

	reconciler := controller.UserReconciler{
		Client:               c,
		DockyardsExternalURL: "test.com",
	}

	t.Run("test verification request creation", func(t *testing.T) {
		user := dockyardsv1.User{
			TypeMeta: metav1.TypeMeta{
				Kind:       dockyardsv1.UserKind,
				APIVersion: dockyardsv1.GroupVersion.String(),
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: "test",
			},
			Spec: dockyardsv1.UserSpec{
				DisplayName: "test",
				Email:       "test@test.com",
				Password:    "test",
			},
		}
		if err := c.Create(ctx, &user); err != nil {
			t.Fatal(err)
		}

		// create VerificationRequest
		_, err := reconciler.Reconcile(ctx, ctrl.Request{NamespacedName: types.NamespacedName{Name: "test"}})
		if err != nil {
			t.Fatal(err)
		}

		// create VerificationRequest Verified condition
		_, err = reconciler.Reconcile(ctx, ctrl.Request{NamespacedName: types.NamespacedName{Name: "test"}})
		if err != nil {
			t.Fatal(err)
		}

		vr := dockyardsv1.VerificationRequest{ObjectMeta: metav1.ObjectMeta{Name: "sign-up-" + user.Name}}
		err = testingutil.RetryUntilFound(ctx, c, &vr)
		if err != nil {
			t.Fatal(err)
		}

		if vr.Spec.Code == "" {
			t.Fatal("No VerificationRequest created for user")
		}

		expectedOwner := []metav1.OwnerReference{
			{
				APIVersion: dockyardsv1.GroupVersion.String(),
				Kind:       dockyardsv1.UserKind,
				Name:       user.Name,
				UID:        user.UID,
			},
		}

		if !cmp.Equal(expectedOwner, vr.ObjectMeta.OwnerReferences) {
			t.Errorf("VerificationRequest is missing OwnerReferences.\nDiff: %s", cmp.Diff(expectedOwner, vr.ObjectMeta.OwnerReferences))
		}

		expectedUserRef := corev1.TypedLocalObjectReference{
			APIGroup: &dockyardsv1.GroupVersion.Group,
			Kind:     dockyardsv1.UserKind,
			Name:     user.Name,
		}

		if !cmp.Equal(expectedUserRef, vr.Spec.UserRef) {
			t.Errorf("VerificationRequest is missing userReferences.\nDiff: %s", cmp.Diff(expectedUserRef, vr.Spec.UserRef))
		}

		ignoreFields := cmpopts.IgnoreFields(metav1.Condition{}, "LastTransitionTime", "ObservedGeneration")
		expectedCondition := metav1.Condition{
			Type:    dockyardsv1.VerifiedCondition,
			Status:  metav1.ConditionFalse,
			Reason:  "NotVerified",
			Message: "",
		}

		actualCondition := conditions.Get(&vr, dockyardsv1.VerifiedCondition)
		if actualCondition == nil {
			t.Errorf("VerificationRequest does not have %s condition", dockyardsv1.VerifiedCondition)
		}

		if !cmp.Equal(expectedCondition, *actualCondition, ignoreFields) {
			t.Errorf("VerificationRequest %s condition is not as expected.\nDiff: %s", dockyardsv1.VerifiedCondition, cmp.Diff(expectedCondition, *actualCondition, ignoreFields))
		}

		err = c.Get(ctx, client.ObjectKeyFromObject(&user), &user)
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("test user status reconciliation", func(t *testing.T) {
		vr := dockyardsv1.VerificationRequest{
			ObjectMeta: metav1.ObjectMeta{
				Name: "sign-up-test",
			},
		}

		if err := c.Get(ctx, client.ObjectKeyFromObject(&vr), &vr); err != nil {
			t.Fatal(err)
		}

		verifiedCond := metav1.Condition{
			Type:               dockyardsv1.VerifiedCondition,
			Status:             metav1.ConditionTrue,
			Reason:             "TestVerified",
			Message:            "",
			LastTransitionTime: metav1.Now(),
		}

		conditions.Set(&vr, &verifiedCond)

		if err := c.Status().Update(ctx, &vr); err != nil {
			t.Fatal(err)
		}

		// Update User Ready condition to match VerificationRequest Verified condition
		_, err := reconciler.Reconcile(ctx, ctrl.Request{NamespacedName: types.NamespacedName{Name: "test"}})
		if err != nil {
			t.Fatal(err)
		}

		expected := metav1.Condition{
			Type:               dockyardsv1.ReadyCondition,
			Status:             metav1.ConditionTrue,
			Reason:             verifiedCond.Reason,
			Message:            verifiedCond.Message,
			LastTransitionTime: verifiedCond.LastTransitionTime,
		}

		var user dockyardsv1.User
		if err := c.Get(ctx, client.ObjectKey{Name: "test"}, &user); err != nil {
			t.Fatal(err)
		}

		ignoreFields := cmpopts.IgnoreFields(metav1.Condition{}, "ObservedGeneration")
		actual := conditions.Get(&user, dockyardsv1.ReadyCondition)
		if !cmp.Equal(expected, *actual, ignoreFields) {
			t.Errorf("User %s condition is not as expected.\nDiff: %s", dockyardsv1.ReadyCondition, cmp.Diff(expected, *actual, ignoreFields))
		}
	})

	t.Run("test verificationrequest cleanup", func(t *testing.T) {
		// Trigger VerificationRequest deletion
		_, err := reconciler.Reconcile(ctx, ctrl.Request{NamespacedName: types.NamespacedName{Name: "test"}})
		if err != nil {
			t.Fatal(err)
		}

		e := dockyardsv1.VerificationRequest{
			ObjectMeta: metav1.ObjectMeta{
				Name: "sign-up-test",
			},
		}
		var actual dockyardsv1.VerificationRequest
		err = c.Get(ctx, client.ObjectKeyFromObject(&e), &actual)
		if err != nil && !apierrors.IsNotFound(err) {
			t.Fatal(err)
		}

		if !cmp.Equal(dockyardsv1.VerificationRequest{}, actual) {
			t.Errorf("VerificationRequest has not been deleted after the user has been marked as %s", dockyardsv1.ReadyCondition)
		}
	})
}
