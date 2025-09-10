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

	"github.com/go-logr/logr"
	"github.com/google/go-cmp/cmp"
	dockyardsv1 "github.com/sudoswedenab/dockyards-backend/api/v1alpha3"
	"github.com/sudoswedenab/dockyards-backend/internal/controller"
	"github.com/sudoswedenab/dockyards-backend/pkg/testing/testingutil"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
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

	mgr := testEnvironment.GetManager()
	c := testEnvironment.GetClient()

	err = (&controller.UserReconciler{
		Client:               mgr.GetClient(),
		DockyardsExternalURL: "localhost",
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

	t.Run("test verification request creation", func(t *testing.T) {
		user := dockyardsv1.User{
			ObjectMeta: v1.ObjectMeta{
				Name: "test",
			},
			Spec: dockyardsv1.UserSpec{
				DisplayName: "test",
				Email:       "test@test.com",
				Password:    "test",
			},
		}
		err := c.Create(ctx, &user)
		if err != nil {
			t.Fatal(err)
		}

		vr := dockyardsv1.VerificationRequest{ObjectMeta: v1.ObjectMeta{Name: "sign-up-" + user.Name}}
		err = testingutil.RetryUntilFound(ctx, c, &vr)
		if err != nil {
			t.Fatal(err)
		}

		if vr.Spec.Code == "" {
			t.Fatal("No VerificationRequest created for user")
		}

		expectedOwner := []v1.OwnerReference{
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
	})
}
