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

package authorization_test

import (
	"os"
	"testing"

	dockyardsv1 "github.com/sudoswedenab/dockyards-backend/api/v1alpha3"
	"github.com/sudoswedenab/dockyards-backend/pkg/authorization"
	authorizationv1 "k8s.io/api/authorization/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
)

var (
	c client.Client
)

func TestMain(m *testing.M) {
	environment := envtest.Environment{
		CRDDirectoryPaths: []string{
			"../../config/crd",
		},
	}

	cfg, err := environment.Start()
	if err != nil {
		os.Exit(1)
	}

	defer func() {
		err := environment.Stop()
		if err != nil {
			os.Exit(1)
		}
	}()

	scheme := runtime.NewScheme()

	_ = authorizationv1.AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)
	_ = dockyardsv1.AddToScheme(scheme)
	_ = rbacv1.AddToScheme(scheme)

	c, err = client.New(cfg, client.Options{Scheme: scheme})
	if err != nil {
		os.Exit(1)
	}

	code := m.Run()
	os.Exit(code)
}

func TestUserAuthorization(t *testing.T) {
	ctx := t.Context()

	user := &dockyardsv1.User{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "authorization-",
		},
	}

	other := &dockyardsv1.User{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "authorization-",
		},
	}

	for _, u := range []*dockyardsv1.User{user, other} {
		err := c.Create(ctx, u)
		if err != nil {
			t.Fatal(err)
		}

		err = authorization.ReconcileUserAuthorization(ctx, c, *u)
		if err != nil {
			t.Fatal(err)
		}
	}

	t.Run("test user getting self", func(t *testing.T) {
		subjectAccessReview := authorizationv1.SubjectAccessReview{
			Spec: authorizationv1.SubjectAccessReviewSpec{
				User: user.Name,
				ResourceAttributes: &authorizationv1.ResourceAttributes{
					Group:    dockyardsv1.GroupVersion.Group,
					Name:     user.Name,
					Resource: "users",
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

	t.Run("test user getting other", func(t *testing.T) {
		subjectAccessReview := authorizationv1.SubjectAccessReview{
			Spec: authorizationv1.SubjectAccessReviewSpec{
				User: user.Name,
				ResourceAttributes: &authorizationv1.ResourceAttributes{
					Group:    dockyardsv1.GroupVersion.Group,
					Name:     other.Name,
					Resource: "users",
					Verb:     "get",
				},
			},
		}

		err := c.Create(ctx, &subjectAccessReview)
		if err != nil {
			t.Fatal(err)
		}

		if subjectAccessReview.Status.Allowed {
			t.Errorf("expected not allowed, got %t", subjectAccessReview.Status.Allowed)
		}
	})

	t.Run("test user deleting self", func(t *testing.T) {
		subjectAccessReview := authorizationv1.SubjectAccessReview{
			Spec: authorizationv1.SubjectAccessReviewSpec{
				User: user.Name,
				ResourceAttributes: &authorizationv1.ResourceAttributes{
					Group:    dockyardsv1.GroupVersion.Group,
					Name:     user.Name,
					Resource: "users",
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

	t.Run("test user deleting other", func(t *testing.T) {
		subjectAccessReview := authorizationv1.SubjectAccessReview{
			Spec: authorizationv1.SubjectAccessReviewSpec{
				User: user.Name,
				ResourceAttributes: &authorizationv1.ResourceAttributes{
					Group:    dockyardsv1.GroupVersion.Group,
					Name:     other.Name,
					Resource: "users",
					Verb:     "delete",
				},
			},
		}

		err := c.Create(ctx, &subjectAccessReview)
		if err != nil {
			t.Fatal(err)
		}

		if subjectAccessReview.Status.Allowed {
			t.Errorf("expected not allowed, got %t", subjectAccessReview.Status.Allowed)
		}
	})
}

func TestOrganizationAuthorization(t *testing.T) {
	ctx := t.Context()

	superUser := &dockyardsv1.User{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "authorization-",
		},
	}

	user := &dockyardsv1.User{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "authorization-",
		},
	}

	reader := &dockyardsv1.User{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "authorization-",
		},
	}

	otherUser := &dockyardsv1.User{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "authorization-",
		},
	}

	for _, u := range []*dockyardsv1.User{superUser, user, reader, otherUser} {
		err := c.Create(ctx, u)
		if err != nil {
			t.Fatal(err)
		}

		err = authorization.ReconcileUserAuthorization(ctx, c, *u)
		if err != nil {
			t.Fatal(err)
		}
	}

	organization := dockyardsv1.Organization{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "authorization-",
		},
	}

	otherOrganization := dockyardsv1.Organization{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "authorization-",
		},
	}

	tt := []struct {
		Organization *dockyardsv1.Organization
		Members      []dockyardsv1.Member
	}{
		{
			Organization: &organization,
			Members: []dockyardsv1.Member{
				{
					Spec: dockyardsv1.MemberSpec{
						UserRef: corev1.TypedLocalObjectReference{
							Name: superUser.Name,
						},
						Role: dockyardsv1.RoleSuperUser,
					},
				},
				{
					Spec: dockyardsv1.MemberSpec{
						UserRef: corev1.TypedLocalObjectReference{
							Name: user.Name,
						},
						Role: dockyardsv1.RoleUser,
					},
				},
				{
					Spec: dockyardsv1.MemberSpec{
						UserRef: corev1.TypedLocalObjectReference{
							Name: reader.Name,
						},
						Role: dockyardsv1.RoleReader,
					},
				},
			},
		},
		{
			Organization: &otherOrganization,
			Members: []dockyardsv1.Member{
				{
					Spec: dockyardsv1.MemberSpec{
						UserRef: corev1.TypedLocalObjectReference{
							Name: otherUser.Name,
						},
						Role: dockyardsv1.RoleSuperUser,
					},
				},
			},
		},
	}

	for _, tc := range tt {
		namespace := corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: "authorization-",
			},
		}

		err := c.Create(ctx, &namespace)
		if err != nil {
			t.Fatal(err)
		}

		tc.Organization.Spec.NamespaceRef = &corev1.LocalObjectReference{
			Name: namespace.Name,
		}

		err = c.Create(ctx, tc.Organization)
		if err != nil {
			t.Fatal(err)
		}

		for _, member := range tc.Members {
			member.ObjectMeta = metav1.ObjectMeta{
				GenerateName: "authorization-",
				Namespace:    namespace.Name,
			}

			err := c.Create(ctx, &member)
			if err != nil {
				t.Fatal(err)
			}
		}

		err = authorization.ReconcileOrganizationAuthorization(ctx, c, tc.Organization)
		if err != nil {
			t.Fatal(err)
		}
	}

	t.Run("test super user getting organization", func(t *testing.T) {
		subjectAccessReview := authorizationv1.SubjectAccessReview{
			Spec: authorizationv1.SubjectAccessReviewSpec{
				User: superUser.Name,
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
				User: superUser.Name,
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
				User: superUser.Name,
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
				User: superUser.Name,
				ResourceAttributes: &authorizationv1.ResourceAttributes{
					Group:    dockyardsv1.GroupVersion.Group,
					Name:     otherOrganization.Name,
					Resource: "organizations",
					Verb:     "delete",
				},
			},
		}

		err := c.Create(ctx, &subjectAccessReview)
		if err != nil {
			t.Fatal(err)
		}

		if subjectAccessReview.Status.Allowed {
			t.Errorf("expected not allowed, got %t", subjectAccessReview.Status.Allowed)
		}
	})

	t.Run("test user getting organization", func(t *testing.T) {
		subjectAccessReview := authorizationv1.SubjectAccessReview{
			Spec: authorizationv1.SubjectAccessReviewSpec{
				User: user.Name,
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

	t.Run("test user deleting organization", func(t *testing.T) {
		subjectAccessReview := authorizationv1.SubjectAccessReview{
			Spec: authorizationv1.SubjectAccessReviewSpec{
				User: user.Name,
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

		if subjectAccessReview.Status.Allowed {
			t.Errorf("expected not allowed, got %t", subjectAccessReview.Status.Allowed)
		}
	})

	t.Run("test user patching organization", func(t *testing.T) {
		subjectAccessReview := authorizationv1.SubjectAccessReview{
			Spec: authorizationv1.SubjectAccessReviewSpec{
				User: user.Name,
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

		if subjectAccessReview.Status.Allowed {
			t.Errorf("expected not allowed, got %t", subjectAccessReview.Status.Allowed)
		}
	})

	t.Run("test reader deleting organization", func(t *testing.T) {
		subjectAccessReview := authorizationv1.SubjectAccessReview{
			Spec: authorizationv1.SubjectAccessReviewSpec{
				User: reader.Name,
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

		if subjectAccessReview.Status.Allowed {
			t.Errorf("expected not allowed, got %t", subjectAccessReview.Status.Allowed)
		}
	})

	t.Run("test reader getting organization", func(t *testing.T) {
		subjectAccessReview := authorizationv1.SubjectAccessReview{
			Spec: authorizationv1.SubjectAccessReviewSpec{
				User: reader.Name,
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
}

func TestMemberAuthorization(t *testing.T) {
	ctx := t.Context()

	err := authorization.ReconcileClusterAuthorization(ctx, c)
	if err != nil {
		t.Fatal(err)
	}

	superUser := dockyardsv1.User{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "member-",
		},
	}

	user := dockyardsv1.User{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "member-",
		},
	}

	reader := dockyardsv1.User{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "member-",
		},
	}

	otherUser := dockyardsv1.User{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "member-",
		},
	}

	for _, u := range []*dockyardsv1.User{&superUser, &user, &reader, &otherUser} {
		err := c.Create(ctx, u)
		if err != nil {
			t.Fatal(err)
		}
	}

	organization := dockyardsv1.Organization{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "member-",
		},
	}

	otherOrganization := dockyardsv1.Organization{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "member-",
		},
	}

	tt := []struct {
		Organization *dockyardsv1.Organization
		Members      []dockyardsv1.Member
	}{
		{
			Organization: &organization,
			Members: []dockyardsv1.Member{
				{
					Spec: dockyardsv1.MemberSpec{
						UserRef: corev1.TypedLocalObjectReference{
							Name: superUser.Name,
						},
						Role: dockyardsv1.RoleSuperUser,
					},
				},
				{
					Spec: dockyardsv1.MemberSpec{
						UserRef: corev1.TypedLocalObjectReference{
							Name: user.Name,
						},
						Role: dockyardsv1.RoleUser,
					},
				},
				{
					Spec: dockyardsv1.MemberSpec{
						UserRef: corev1.TypedLocalObjectReference{
							Name: reader.Name,
						},
						Role: dockyardsv1.RoleReader,
					},
				},
			},
		},
		{
			Organization: &otherOrganization,
			Members: []dockyardsv1.Member{
				{
					Spec: dockyardsv1.MemberSpec{
						UserRef: corev1.TypedLocalObjectReference{
							Name: otherUser.Name,
						},
						Role: dockyardsv1.RoleSuperUser,
					},
				},
			},
		},
	}

	for _, tc := range tt {
		namespace := corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: "member-",
			},
		}

		err := c.Create(ctx, &namespace)
		if err != nil {
			t.Fatal(err)
		}

		tc.Organization.Spec.NamespaceRef = &corev1.LocalObjectReference{
			Name: namespace.Name,
		}

		err = c.Create(ctx, tc.Organization)
		if err != nil {
			t.Fatal(err)
		}

		for _, member := range tc.Members {
			member.ObjectMeta = metav1.ObjectMeta{
				Name:      member.Spec.UserRef.Name,
				Namespace: namespace.Name,
			}

			err := c.Create(ctx, &member)
			if err != nil {
				t.Fatal(err)
			}

			err = authorization.ReconcileMemberAuthorization(ctx, c, &member)
			if err != nil {
				t.Fatal(err)
			}
		}
	}

	t.Run("test super user getting clusters", func(t *testing.T) {
		subjectAccessReview := authorizationv1.SubjectAccessReview{
			Spec: authorizationv1.SubjectAccessReviewSpec{
				User: superUser.Name,
				ResourceAttributes: &authorizationv1.ResourceAttributes{
					Group:     dockyardsv1.GroupVersion.Group,
					Namespace: organization.Spec.NamespaceRef.Name,
					Resource:  "clusters",
					Verb:      "get",
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

	t.Run("test user deleting clusters", func(t *testing.T) {
		subjectAccessReview := authorizationv1.SubjectAccessReview{
			Spec: authorizationv1.SubjectAccessReviewSpec{
				User: user.Name,
				ResourceAttributes: &authorizationv1.ResourceAttributes{
					Group:     dockyardsv1.GroupVersion.Group,
					Namespace: organization.Spec.NamespaceRef.Name,
					Resource:  "clusters",
					Verb:      "delete",
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

	t.Run("test user deleting clusters other organization", func(t *testing.T) {
		subjectAccessReview := authorizationv1.SubjectAccessReview{
			Spec: authorizationv1.SubjectAccessReviewSpec{
				User: user.Name,
				ResourceAttributes: &authorizationv1.ResourceAttributes{
					Group:     dockyardsv1.GroupVersion.Group,
					Namespace: otherOrganization.Spec.NamespaceRef.Name,
					Resource:  "clusters",
					Verb:      "delete",
				},
			},
		}

		err := c.Create(ctx, &subjectAccessReview)
		if err != nil {
			t.Fatal(err)
		}

		if subjectAccessReview.Status.Allowed {
			t.Errorf("expected not allowed, got %t", subjectAccessReview.Status.Allowed)
		}
	})

	t.Run("test reader getting clusters", func(t *testing.T) {
		subjectAccessReview := authorizationv1.SubjectAccessReview{
			Spec: authorizationv1.SubjectAccessReviewSpec{
				User: reader.Name,
				ResourceAttributes: &authorizationv1.ResourceAttributes{
					Group:     dockyardsv1.GroupVersion.Group,
					Namespace: organization.Spec.NamespaceRef.Name,
					Resource:  "clusters",
					Verb:      "get",
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

	t.Run("test reader getting clusters other organization", func(t *testing.T) {
		subjectAccessReview := authorizationv1.SubjectAccessReview{
			Spec: authorizationv1.SubjectAccessReviewSpec{
				User: reader.Name,
				ResourceAttributes: &authorizationv1.ResourceAttributes{
					Group:     dockyardsv1.GroupVersion.Group,
					Namespace: otherOrganization.Spec.NamespaceRef.Name,
					Resource:  "clusters",
					Verb:      "get",
				},
			},
		}

		err := c.Create(ctx, &subjectAccessReview)
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
				User: reader.Name,
				ResourceAttributes: &authorizationv1.ResourceAttributes{
					Group:     dockyardsv1.GroupVersion.Group,
					Namespace: organization.Spec.NamespaceRef.Name,
					Resource:  "clusters",
					Verb:      "delete",
				},
			},
		}

		err := c.Create(ctx, &subjectAccessReview)
		if err != nil {
			t.Fatal(err)
		}

		if subjectAccessReview.Status.Allowed {
			t.Errorf("expected not allowed, got %t", subjectAccessReview.Status.Allowed)
		}
	})

	t.Run("test reader getting workloads", func(t *testing.T) {
		subjectAccessReview := authorizationv1.SubjectAccessReview{
			Spec: authorizationv1.SubjectAccessReviewSpec{
				User: reader.Name,
				ResourceAttributes: &authorizationv1.ResourceAttributes{
					Group:     dockyardsv1.GroupVersion.Group,
					Namespace: organization.Spec.NamespaceRef.Name,
					Resource:  "workloads",
					Verb:      "get",
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

	t.Run("test super user creating clusters", func(t *testing.T) {
		subjectAccessReview := authorizationv1.SubjectAccessReview{
			Spec: authorizationv1.SubjectAccessReviewSpec{
				User: superUser.Name,
				ResourceAttributes: &authorizationv1.ResourceAttributes{
					Group:     dockyardsv1.GroupVersion.Group,
					Namespace: organization.Spec.NamespaceRef.Name,
					Resource:  "clusters",
					Verb:      "create",
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

	t.Run("test user creating clusters", func(t *testing.T) {
		subjectAccessReview := authorizationv1.SubjectAccessReview{
			Spec: authorizationv1.SubjectAccessReviewSpec{
				User: user.Name,
				ResourceAttributes: &authorizationv1.ResourceAttributes{
					Group:     dockyardsv1.GroupVersion.Group,
					Namespace: organization.Spec.NamespaceRef.Name,
					Resource:  "clusters",
					Verb:      "create",
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

	t.Run("test reader creating clusters", func(t *testing.T) {
		subjectAccessReview := authorizationv1.SubjectAccessReview{
			Spec: authorizationv1.SubjectAccessReviewSpec{
				User: reader.Name,
				ResourceAttributes: &authorizationv1.ResourceAttributes{
					Group:     dockyardsv1.GroupVersion.Group,
					Namespace: organization.Spec.NamespaceRef.Name,
					Resource:  "clusters",
					Verb:      "create",
				},
			},
		}

		err := c.Create(ctx, &subjectAccessReview)
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
				User: reader.Name,
				ResourceAttributes: &authorizationv1.ResourceAttributes{
					Group:     dockyardsv1.GroupVersion.Group,
					Namespace: otherOrganization.Spec.NamespaceRef.Name,
					Resource:  "clusters",
					Verb:      "get",
				},
			},
		}

		err := c.Create(ctx, &subjectAccessReview)
		if err != nil {
			t.Fatal(err)
		}

		if subjectAccessReview.Status.Allowed {
			t.Errorf("expected not allowed, got %t", subjectAccessReview.Status.Allowed)
		}
	})

	t.Run("test super user deleting members", func(t *testing.T) {
		subjectAccessReview := authorizationv1.SubjectAccessReview{
			Spec: authorizationv1.SubjectAccessReviewSpec{
				User: superUser.Name,
				ResourceAttributes: &authorizationv1.ResourceAttributes{
					Group:     dockyardsv1.GroupVersion.Group,
					Namespace: organization.Spec.NamespaceRef.Name,
					Resource:  "members",
					Verb:      "delete",
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

	t.Run("test user deleting members", func(t *testing.T) {
		subjectAccessReview := authorizationv1.SubjectAccessReview{
			Spec: authorizationv1.SubjectAccessReviewSpec{
				User: user.Name,
				ResourceAttributes: &authorizationv1.ResourceAttributes{
					Group:     dockyardsv1.GroupVersion.Group,
					Namespace: organization.Spec.NamespaceRef.Name,
					Resource:  "members",
					Verb:      "delete",
				},
			},
		}

		err := c.Create(ctx, &subjectAccessReview)
		if err != nil {
			t.Fatal(err)
		}

		if subjectAccessReview.Status.Allowed {
			t.Errorf("expected not allowed, got %t", subjectAccessReview.Status.Allowed)
		}
	})

	t.Run("test reader deleting members", func(t *testing.T) {
		subjectAccessReview := authorizationv1.SubjectAccessReview{
			Spec: authorizationv1.SubjectAccessReviewSpec{
				User: reader.Name,
				ResourceAttributes: &authorizationv1.ResourceAttributes{
					Group:     dockyardsv1.GroupVersion.Group,
					Namespace: organization.Spec.NamespaceRef.Name,
					Resource:  "members",
					Verb:      "delete",
				},
			},
		}

		err := c.Create(ctx, &subjectAccessReview)
		if err != nil {
			t.Fatal(err)
		}

		if subjectAccessReview.Status.Allowed {
			t.Errorf("expected not allowed, got %t", subjectAccessReview.Status.Allowed)
		}
	})

	t.Run("test reader deleting self members", func(t *testing.T) {
		subjectAccessReview := authorizationv1.SubjectAccessReview{
			Spec: authorizationv1.SubjectAccessReviewSpec{
				User: reader.Name,
				ResourceAttributes: &authorizationv1.ResourceAttributes{
					Group:     dockyardsv1.GroupVersion.Group,
					Name:      reader.Name,
					Namespace: organization.Spec.NamespaceRef.Name,
					Resource:  "members",
					Verb:      "delete",
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

	t.Run("test reader deleting @me members", func(t *testing.T) {
		subjectAccessReview := authorizationv1.SubjectAccessReview{
			Spec: authorizationv1.SubjectAccessReviewSpec{
				User: reader.Name,
				ResourceAttributes: &authorizationv1.ResourceAttributes{
					Group:     dockyardsv1.GroupVersion.Group,
					Name:      reader.Name,
					Namespace: organization.Spec.NamespaceRef.Name,
					Resource:  "members",
					Verb:      "delete",
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
}
