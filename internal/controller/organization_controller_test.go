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
	dockyardsv1 "bitbucket.org/sudosweden/dockyards-backend/pkg/api/v1alpha3"
	"bitbucket.org/sudosweden/dockyards-backend/pkg/testing/testingutil"
	"github.com/go-logr/logr"
	authorizationv1 "k8s.io/api/authorization/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestOrganizationController(t *testing.T) {
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

	err = (&controller.OrganizationReconciler{
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

	superUser := testEnvironment.GetSuperUser()
	user := testEnvironment.GetUser()
	reader := testEnvironment.GetReader()

	organization := dockyardsv1.Organization{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "test-",
		},
		Spec: dockyardsv1.OrganizationSpec{
			MemberRefs: []dockyardsv1.OrganizationMemberReference{
				{
					UID:  superUser.UID,
					Role: dockyardsv1.OrganizationMemberRoleSuperUser,
				},
				{
					UID:  user.UID,
					Role: dockyardsv1.OrganizationMemberRoleUser,
				},
				{
					UID:  reader.UID,
					Role: dockyardsv1.OrganizationMemberRoleReader,
				},
			},
		},
	}

	err = c.Create(ctx, &organization)
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
					UID:  "dd82971a-c83e-4db4-85c4-00d7c6165d06",
					Role: dockyardsv1.OrganizationMemberRoleSuperUser,
				},
				{
					UID:  superUser.UID,
					Role: dockyardsv1.OrganizationMemberRoleUser,
				},
				{
					UID:  user.UID,
					Role: dockyardsv1.OrganizationMemberRoleReader,
				},
			},
		},
	}

	err = c.Create(ctx, &otherOrganization)
	if err != nil {
		t.Fatal(err)
	}

	if !mgr.GetCache().WaitForCacheSync(ctx) {
		t.Fatal("unable to wait for cache sync")
	}

	for i := 0; i < 5; i++ {
		err := c.Get(ctx, client.ObjectKeyFromObject(&organization), &organization)
		if err != nil {
			t.Fatal(err)
		}

		if organization.Status.NamespaceRef != nil {
			break
		}

		time.Sleep(time.Second)
	}

	for i := 0; i < 5; i++ {
		err := c.Get(ctx, client.ObjectKeyFromObject(&otherOrganization), &otherOrganization)
		if err != nil {
			t.Fatal(err)
		}

		if otherOrganization.Status.NamespaceRef != nil {
			break
		}

		time.Sleep(time.Second)
	}

	t.Run("test super user getting organization", func(t *testing.T) {
		subjectAccessReview := authorizationv1.SubjectAccessReview{
			Spec: authorizationv1.SubjectAccessReviewSpec{
				User: string(superUser.UID),
				ResourceAttributes: &authorizationv1.ResourceAttributes{
					Group:    dockyardsv1.GroupVersion.Group,
					Name:     organization.Name,
					Resource: "organizations",
					Verb:     "get",
				},
			},
		}

		err := c.Create(ctx, &subjectAccessReview)
		if err != nil {
			t.Fatal(err)
		}

		if !subjectAccessReview.Status.Allowed {
			t.Errorf("expected allowed, got %t", subjectAccessReview.Status.Allowed)
		}
	})

	t.Run("test super user deleting organization", func(t *testing.T) {
		subjectAccessReview := authorizationv1.SubjectAccessReview{
			Spec: authorizationv1.SubjectAccessReviewSpec{
				User: string(superUser.UID),
				ResourceAttributes: &authorizationv1.ResourceAttributes{
					Group:    dockyardsv1.GroupVersion.Group,
					Name:     organization.Name,
					Resource: "organizations",
					Verb:     "delete",
				},
			},
		}

		err := c.Create(ctx, &subjectAccessReview)
		if err != nil {
			t.Fatal(err)
		}

		if !subjectAccessReview.Status.Allowed {
			t.Errorf("expected allowed, got %t", subjectAccessReview.Status.Allowed)
		}
	})

	t.Run("test super user patching organization", func(t *testing.T) {
		subjectAccessReview := authorizationv1.SubjectAccessReview{
			Spec: authorizationv1.SubjectAccessReviewSpec{
				User: string(superUser.UID),
				ResourceAttributes: &authorizationv1.ResourceAttributes{
					Group:    dockyardsv1.GroupVersion.Group,
					Name:     organization.Name,
					Resource: "organizations",
					Verb:     "patch",
				},
			},
		}

		err := c.Create(ctx, &subjectAccessReview)
		if err != nil {
			t.Fatal(err)
		}

		if !subjectAccessReview.Status.Allowed {
			t.Errorf("expected allowed, got %t", subjectAccessReview.Status.Allowed)
		}
	})

	t.Run("test super user deleting other organization", func(t *testing.T) {
		subjectAccessReview := authorizationv1.SubjectAccessReview{
			Spec: authorizationv1.SubjectAccessReviewSpec{
				User: string(superUser.UID),
				ResourceAttributes: &authorizationv1.ResourceAttributes{
					Group:    dockyardsv1.GroupVersion.Group,
					Name:     otherOrganization.Name,
					Resource: "organizations",
					Verb:     "delete",
				},
			},
		}

		err = c.Create(ctx, &subjectAccessReview)
		if err != nil {
			t.Fatal(err)
		}

		if subjectAccessReview.Status.Allowed {
			t.Errorf("expected not allowed, got %t", subjectAccessReview.Status.Allowed)
		}
	})

	t.Run("test super user getting clusters", func(t *testing.T) {
		subjectAccessReview := authorizationv1.SubjectAccessReview{
			Spec: authorizationv1.SubjectAccessReviewSpec{
				User: string(superUser.UID),
				ResourceAttributes: &authorizationv1.ResourceAttributes{
					Group:     dockyardsv1.GroupVersion.Group,
					Namespace: organization.Status.NamespaceRef.Name,
					Resource:  "clusters",
					Verb:      "get",
				},
			},
		}

		err = c.Create(ctx, &subjectAccessReview)
		if err != nil {
			t.Fatal(err)
		}

		if !subjectAccessReview.Status.Allowed {
			t.Errorf("expected allowed, got %t", subjectAccessReview.Status.Allowed)
		}
	})

	t.Run("test super user deleting other organization", func(t *testing.T) {
		subjectAccessReview := authorizationv1.SubjectAccessReview{
			Spec: authorizationv1.SubjectAccessReviewSpec{
				User: string(superUser.UID),
				ResourceAttributes: &authorizationv1.ResourceAttributes{
					Group:    dockyardsv1.GroupVersion.Group,
					Name:     otherOrganization.Name,
					Resource: "organizations",
					Verb:     "delete",
				},
			},
		}

		err = c.Create(ctx, &subjectAccessReview)
		if err != nil {
			t.Fatal(err)
		}

		if subjectAccessReview.Status.Allowed {
			t.Errorf("expected not allowed, got %t", subjectAccessReview.Status.Allowed)
		}
	})

	t.Run("test user deleting clusters", func(t *testing.T) {
		subjectAccessReview := authorizationv1.SubjectAccessReview{
			Spec: authorizationv1.SubjectAccessReviewSpec{
				User: string(user.UID),
				ResourceAttributes: &authorizationv1.ResourceAttributes{
					Group:     dockyardsv1.GroupVersion.Group,
					Namespace: organization.Status.NamespaceRef.Name,
					Resource:  "clusters",
					Verb:      "delete",
				},
			},
		}

		err = c.Create(ctx, &subjectAccessReview)
		if err != nil {
			t.Fatal(err)
		}

		if !subjectAccessReview.Status.Allowed {
			t.Errorf("expected allowed, got %t", subjectAccessReview.Status.Allowed)
		}
	})

	t.Run("test user getting organization", func(t *testing.T) {
		subjectAccessReview := authorizationv1.SubjectAccessReview{
			Spec: authorizationv1.SubjectAccessReviewSpec{
				User: string(user.UID),
				ResourceAttributes: &authorizationv1.ResourceAttributes{
					Group:    dockyardsv1.GroupVersion.Group,
					Name:     organization.Name,
					Resource: "organizations",
					Verb:     "get",
				},
			},
		}

		err = c.Create(ctx, &subjectAccessReview)
		if err != nil {
			t.Fatal(err)
		}

		if !subjectAccessReview.Status.Allowed {
			t.Errorf("expected allowed, got %t", subjectAccessReview.Status.Allowed)
		}
	})

	t.Run("test user deleting organization", func(t *testing.T) {
		subjectAccessReview := authorizationv1.SubjectAccessReview{
			Spec: authorizationv1.SubjectAccessReviewSpec{
				User: string(user.UID),
				ResourceAttributes: &authorizationv1.ResourceAttributes{
					Group:    dockyardsv1.GroupVersion.Group,
					Name:     organization.Name,
					Resource: "organizations",
					Verb:     "delete",
				},
			},
		}

		err = c.Create(ctx, &subjectAccessReview)
		if err != nil {
			t.Fatal(err)
		}

		if subjectAccessReview.Status.Allowed {
			t.Errorf("expected not allowed, got %t", subjectAccessReview.Status.Allowed)
		}
	})

	t.Run("test user patching organization", func(t *testing.T) {
		subjectAccessReview := authorizationv1.SubjectAccessReview{
			Spec: authorizationv1.SubjectAccessReviewSpec{
				User: string(user.UID),
				ResourceAttributes: &authorizationv1.ResourceAttributes{
					Group:    dockyardsv1.GroupVersion.Group,
					Name:     organization.Name,
					Resource: "organizations",
					Verb:     "patch",
				},
			},
		}

		err = c.Create(ctx, &subjectAccessReview)
		if err != nil {
			t.Fatal(err)
		}

		if subjectAccessReview.Status.Allowed {
			t.Errorf("expected not allowed, got %t", subjectAccessReview.Status.Allowed)
		}
	})

	t.Run("test user deleting clusters other organization", func(t *testing.T) {
		subjectAccessReview := authorizationv1.SubjectAccessReview{
			Spec: authorizationv1.SubjectAccessReviewSpec{
				User: string(user.UID),
				ResourceAttributes: &authorizationv1.ResourceAttributes{
					Group:     dockyardsv1.GroupVersion.Group,
					Namespace: otherOrganization.Status.NamespaceRef.Name,
					Resource:  "clusters",
					Verb:      "delete",
				},
			},
		}

		err = c.Create(ctx, &subjectAccessReview)
		if err != nil {
			t.Fatal(err)
		}

		if subjectAccessReview.Status.Allowed {
			t.Errorf("expected not allowed, got %t", subjectAccessReview.Status.Allowed)
		}
	})

	t.Run("test reader deleting organization", func(t *testing.T) {
		subjectAccessReview := authorizationv1.SubjectAccessReview{
			Spec: authorizationv1.SubjectAccessReviewSpec{
				User: string(reader.UID),
				ResourceAttributes: &authorizationv1.ResourceAttributes{
					Group:    dockyardsv1.GroupVersion.Group,
					Name:     organization.Name,
					Resource: "organizations",
					Verb:     "delete",
				},
			},
		}

		err = c.Create(ctx, &subjectAccessReview)
		if err != nil {
			t.Fatal(err)
		}

		if subjectAccessReview.Status.Allowed {
			t.Errorf("expected not allowed, got %t", subjectAccessReview.Status.Allowed)
		}
	})

	t.Run("test reader getting organization", func(t *testing.T) {
		subjectAccessReview := authorizationv1.SubjectAccessReview{
			Spec: authorizationv1.SubjectAccessReviewSpec{
				User: string(reader.UID),
				ResourceAttributes: &authorizationv1.ResourceAttributes{
					Group:    dockyardsv1.GroupVersion.Group,
					Name:     organization.Name,
					Resource: "organizations",
					Verb:     "get",
				},
			},
		}

		err = c.Create(ctx, &subjectAccessReview)
		if err != nil {
			t.Fatal(err)
		}

		if !subjectAccessReview.Status.Allowed {
			t.Errorf("expected allowed, got %t", subjectAccessReview.Status.Allowed)
		}
	})

	t.Run("test reader getting clusters", func(t *testing.T) {
		subjectAccessReview := authorizationv1.SubjectAccessReview{
			Spec: authorizationv1.SubjectAccessReviewSpec{
				User: string(reader.UID),
				ResourceAttributes: &authorizationv1.ResourceAttributes{
					Group:     dockyardsv1.GroupVersion.Group,
					Namespace: organization.Status.NamespaceRef.Name,
					Resource:  "clusters",
					Verb:      "get",
				},
			},
		}

		err = c.Create(ctx, &subjectAccessReview)
		if err != nil {
			t.Fatal(err)
		}

		if !subjectAccessReview.Status.Allowed {
			t.Errorf("expected allowed, got %t", subjectAccessReview.Status.Allowed)
		}
	})

	t.Run("test reader getting clusters other organization", func(t *testing.T) {
		subjectAccessReview := authorizationv1.SubjectAccessReview{
			Spec: authorizationv1.SubjectAccessReviewSpec{
				User: string(reader.UID),
				ResourceAttributes: &authorizationv1.ResourceAttributes{
					Group:     dockyardsv1.GroupVersion.Group,
					Namespace: otherOrganization.Status.NamespaceRef.Name,
					Resource:  "clusters",
					Verb:      "get",
				},
			},
		}

		err = c.Create(ctx, &subjectAccessReview)
		if err != nil {
			t.Fatal(err)
		}

		if subjectAccessReview.Status.Allowed {
			t.Errorf("expected not allowed, got %t", subjectAccessReview.Status.Allowed)
		}
	})

	t.Run("test reader deleting clusters", func(t *testing.T) {
		subjectAccessReview := authorizationv1.SubjectAccessReview{
			Spec: authorizationv1.SubjectAccessReviewSpec{
				User: string(reader.UID),
				ResourceAttributes: &authorizationv1.ResourceAttributes{
					Group:     dockyardsv1.GroupVersion.Group,
					Namespace: organization.Status.NamespaceRef.Name,
					Resource:  "clusters",
					Verb:      "delete",
				},
			},
		}

		err = c.Create(ctx, &subjectAccessReview)
		if err != nil {
			t.Fatal(err)
		}

		if subjectAccessReview.Status.Allowed {
			t.Errorf("expected not allowed, got %t", subjectAccessReview.Status.Allowed)
		}
	})

	t.Run("test reader getting deployments", func(t *testing.T) {
		subjectAccessReview := authorizationv1.SubjectAccessReview{
			Spec: authorizationv1.SubjectAccessReviewSpec{
				User: string(reader.UID),
				ResourceAttributes: &authorizationv1.ResourceAttributes{
					Group:     dockyardsv1.GroupVersion.Group,
					Namespace: organization.Status.NamespaceRef.Name,
					Resource:  "deployments",
					Verb:      "get",
				},
			},
		}

		err = c.Create(ctx, &subjectAccessReview)
		if err != nil {
			t.Fatal(err)
		}

		if !subjectAccessReview.Status.Allowed {
			t.Errorf("expected allowed, got %t", subjectAccessReview.Status.Allowed)
		}
	})

	t.Run("test super user creating clusters", func(t *testing.T) {
		subjectAccessReview := authorizationv1.SubjectAccessReview{
			Spec: authorizationv1.SubjectAccessReviewSpec{
				User: string(superUser.UID),
				ResourceAttributes: &authorizationv1.ResourceAttributes{
					Group:     dockyardsv1.GroupVersion.Group,
					Namespace: organization.Status.NamespaceRef.Name,
					Resource:  "clusters",
					Verb:      "create",
				},
			},
		}

		err = c.Create(ctx, &subjectAccessReview)
		if err != nil {
			t.Fatal(err)
		}

		if !subjectAccessReview.Status.Allowed {
			t.Errorf("expected allowed, got %t", subjectAccessReview.Status.Allowed)
		}
	})

	t.Run("test user creating clusters", func(t *testing.T) {
		subjectAccessReview := authorizationv1.SubjectAccessReview{
			Spec: authorizationv1.SubjectAccessReviewSpec{
				User: string(user.UID),
				ResourceAttributes: &authorizationv1.ResourceAttributes{
					Group:     dockyardsv1.GroupVersion.Group,
					Namespace: organization.Status.NamespaceRef.Name,
					Resource:  "clusters",
					Verb:      "create",
				},
			},
		}

		err = c.Create(ctx, &subjectAccessReview)
		if err != nil {
			t.Fatal(err)
		}

		if !subjectAccessReview.Status.Allowed {
			t.Errorf("expected allowed, got %t", subjectAccessReview.Status.Allowed)
		}
	})

	t.Run("test reader creating clusters", func(t *testing.T) {
		subjectAccessReview := authorizationv1.SubjectAccessReview{
			Spec: authorizationv1.SubjectAccessReviewSpec{
				User: string(reader.UID),
				ResourceAttributes: &authorizationv1.ResourceAttributes{
					Group:     dockyardsv1.GroupVersion.Group,
					Namespace: organization.Status.NamespaceRef.Name,
					Resource:  "clusters",
					Verb:      "create",
				},
			},
		}

		err = c.Create(ctx, &subjectAccessReview)
		if err != nil {
			t.Fatal(err)
		}

		if subjectAccessReview.Status.Allowed {
			t.Errorf("expected not allowed, got %t", subjectAccessReview.Status.Allowed)
		}
	})

	t.Run("test reader getting clusters other organization", func(t *testing.T) {
		subjectAccessReview := authorizationv1.SubjectAccessReview{
			Spec: authorizationv1.SubjectAccessReviewSpec{
				User: string(reader.UID),
				ResourceAttributes: &authorizationv1.ResourceAttributes{
					Group:     dockyardsv1.GroupVersion.Group,
					Namespace: otherOrganization.Status.NamespaceRef.Name,
					Resource:  "clusters",
					Verb:      "get",
				},
			},
		}

		err = c.Create(ctx, &subjectAccessReview)
		if err != nil {
			t.Fatal(err)
		}

		if subjectAccessReview.Status.Allowed {
			t.Errorf("expected not allowed, got %t", subjectAccessReview.Status.Allowed)
		}
	})
}
