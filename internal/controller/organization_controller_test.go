package controller_test

import (
	"context"
	"os"
	"testing"
	"time"

	"bitbucket.org/sudosweden/dockyards-backend/internal/controller"
	dockyardsv1 "bitbucket.org/sudosweden/dockyards-backend/pkg/api/v1alpha3"
	authorizationv1 "k8s.io/api/authorization/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
)

func TestOrganizationController(t *testing.T) {
	if os.Getenv("KUBEBUILDER_ASSETS") == "" {
		t.Skip("no kubebuilder assets configured")
	}

	type Review struct {
		name                string
		subjectAccessReview authorizationv1.SubjectAccessReview
		expected            bool
	}

	tt := []struct {
		name         string
		organization dockyardsv1.Organization
		reviews      []Review
	}{
		{
			name: "test single organization",
			organization: dockyardsv1.Organization{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test",
				},
				Spec: dockyardsv1.OrganizationSpec{
					MemberRefs: []dockyardsv1.OrganizationMemberReference{
						{
							TypedLocalObjectReference: corev1.TypedLocalObjectReference{
								Kind: dockyardsv1.UserKind,
								Name: "superuser",
							},
							Role: dockyardsv1.OrganizationMemberRoleSuperUser,
							UID:  "cffbcc36-fd31-4c1a-8d44-fce8b0d69688",
						},
						{
							TypedLocalObjectReference: corev1.TypedLocalObjectReference{
								Kind: dockyardsv1.UserKind,
								Name: "user",
							},
							Role: dockyardsv1.OrganizationMemberRoleUser,
							UID:  "57dc0194-cecd-403a-901a-74dcb4e954e3",
						},
						{
							TypedLocalObjectReference: corev1.TypedLocalObjectReference{
								Kind: dockyardsv1.UserKind,
								Name: "reader",
							},
							Role: dockyardsv1.OrganizationMemberRoleReader,
							UID:  "aa725fea-0907-4ca8-be03-6a3728afd704",
						},
					},
				},
			},
			reviews: []Review{
				{
					name: "test superuser getting organization",
					subjectAccessReview: authorizationv1.SubjectAccessReview{
						Spec: authorizationv1.SubjectAccessReviewSpec{
							User: "cffbcc36-fd31-4c1a-8d44-fce8b0d69688",
							ResourceAttributes: &authorizationv1.ResourceAttributes{
								Verb:     "get",
								Resource: "organizations",
								Group:    dockyardsv1.GroupVersion.Group,
								Name:     "test",
							},
						},
					},
					expected: true,
				},
				{
					name: "test superuser deleting organization",
					subjectAccessReview: authorizationv1.SubjectAccessReview{
						Spec: authorizationv1.SubjectAccessReviewSpec{
							User: "cffbcc36-fd31-4c1a-8d44-fce8b0d69688",
							ResourceAttributes: &authorizationv1.ResourceAttributes{
								Verb:     "delete",
								Resource: "organizations",
								Group:    dockyardsv1.GroupVersion.Group,
								Name:     "test",
							},
						},
					},
					expected: true,
				},
				{
					name: "test superuser patching organization",
					subjectAccessReview: authorizationv1.SubjectAccessReview{
						Spec: authorizationv1.SubjectAccessReviewSpec{
							User: "cffbcc36-fd31-4c1a-8d44-fce8b0d69688",
							ResourceAttributes: &authorizationv1.ResourceAttributes{
								Verb:     "patch",
								Resource: "organizations",
								Group:    dockyardsv1.GroupVersion.Group,
								Name:     "test",
							},
						},
					},
					expected: true,
				},
				{
					name: "test superuser deleting organization without membership",
					subjectAccessReview: authorizationv1.SubjectAccessReview{
						Spec: authorizationv1.SubjectAccessReviewSpec{
							User: "cffbcc36-fd31-4c1a-8d44-fce8b0d69688",
							ResourceAttributes: &authorizationv1.ResourceAttributes{
								Verb:     "delete",
								Resource: "organizations",
								Group:    dockyardsv1.GroupVersion.Group,
								Name:     "not-a-member",
							},
						},
					},
					expected: false,
				},
				{
					name: "test superuser getting cluster",
					subjectAccessReview: authorizationv1.SubjectAccessReview{
						Spec: authorizationv1.SubjectAccessReviewSpec{
							User: "cffbcc36-fd31-4c1a-8d44-fce8b0d69688",
							ResourceAttributes: &authorizationv1.ResourceAttributes{
								Namespace: "REPLACE",
								Verb:      "get",
								Resource:  "clusters",
								Group:     "dockyards.io",
							},
						},
					},
					expected: true,
				},
				{
					name: "test superuser getting cluster without membership",
					subjectAccessReview: authorizationv1.SubjectAccessReview{
						Spec: authorizationv1.SubjectAccessReviewSpec{
							User: "cffbcc36-fd31-4c1a-8d44-fce8b0d69688",
							ResourceAttributes: &authorizationv1.ResourceAttributes{
								Namespace: "not-a-member",
								Verb:      "get",
								Resource:  "clusters",
								Group:     "dockyards.io",
							},
						},
					},
					expected: false,
				},
				{
					name: "test user deleting cluster",
					subjectAccessReview: authorizationv1.SubjectAccessReview{
						Spec: authorizationv1.SubjectAccessReviewSpec{
							User: "57dc0194-cecd-403a-901a-74dcb4e954e3",
							ResourceAttributes: &authorizationv1.ResourceAttributes{
								Namespace: "REPLACE",
								Verb:      "delete",
								Resource:  "clusters",
								Group:     dockyardsv1.GroupVersion.Group,
							},
						},
					},
					expected: true,
				},
				{
					name: "test user getting organization",
					subjectAccessReview: authorizationv1.SubjectAccessReview{
						Spec: authorizationv1.SubjectAccessReviewSpec{
							User: "57dc0194-cecd-403a-901a-74dcb4e954e3",
							ResourceAttributes: &authorizationv1.ResourceAttributes{
								Verb:     "get",
								Resource: "organizations",
								Group:    dockyardsv1.GroupVersion.Group,
								Name:     "test",
							},
						},
					},
					expected: true,
				},
				{
					name: "test user deleting organization",
					subjectAccessReview: authorizationv1.SubjectAccessReview{
						Spec: authorizationv1.SubjectAccessReviewSpec{
							User: "57dc0194-cecd-403a-901a-74dcb4e954e3",
							ResourceAttributes: &authorizationv1.ResourceAttributes{
								Verb:     "delete",
								Resource: "organizations",
								Group:    dockyardsv1.GroupVersion.Group,
								Name:     "test",
							},
						},
					},
					expected: false,
				},
				{
					name: "test user patching organization",
					subjectAccessReview: authorizationv1.SubjectAccessReview{
						Spec: authorizationv1.SubjectAccessReviewSpec{
							User: "57dc0194-cecd-403a-901a-74dcb4e954e3",
							ResourceAttributes: &authorizationv1.ResourceAttributes{
								Verb:     "patch",
								Resource: "organizations",
								Group:    dockyardsv1.GroupVersion.Group,
								Name:     "test",
							},
						},
					},
					expected: false,
				},
				{
					name: "test user getting cluster without membership",
					subjectAccessReview: authorizationv1.SubjectAccessReview{
						Spec: authorizationv1.SubjectAccessReviewSpec{
							User: "57dc0194-cecd-403a-901a-74dcb4e954e3",
							ResourceAttributes: &authorizationv1.ResourceAttributes{
								Namespace: "not-a-member",
								Verb:      "get",
								Resource:  "clusters",
								Group:     "dockyards.io",
							},
						},
					},
					expected: false,
				},
				{
					name: "test reader deleting organization",
					subjectAccessReview: authorizationv1.SubjectAccessReview{
						Spec: authorizationv1.SubjectAccessReviewSpec{
							User: "aa725fea-0907-4ca8-be03-6a3728afd704",
							ResourceAttributes: &authorizationv1.ResourceAttributes{
								Verb:     "delete",
								Resource: "organizations",
								Group:    dockyardsv1.GroupVersion.Group,
								Name:     "test",
							},
						},
					},
					expected: false,
				},
				{
					name: "test reader getting organization",
					subjectAccessReview: authorizationv1.SubjectAccessReview{
						Spec: authorizationv1.SubjectAccessReviewSpec{
							User: "aa725fea-0907-4ca8-be03-6a3728afd704",
							ResourceAttributes: &authorizationv1.ResourceAttributes{
								Verb:     "get",
								Resource: "organizations",
								Group:    "dockyards.io",
								Name:     "test",
							},
						},
					},
					expected: true,
				},
				{
					name: "test reader getting cluster",
					subjectAccessReview: authorizationv1.SubjectAccessReview{
						Spec: authorizationv1.SubjectAccessReviewSpec{
							User: "aa725fea-0907-4ca8-be03-6a3728afd704",
							ResourceAttributes: &authorizationv1.ResourceAttributes{
								Namespace: "REPLACE",
								Verb:      "get",
								Resource:  "clusters",
								Group:     "dockyards.io",
							},
						},
					},
					expected: true,
				},
				{
					name: "test reader getting cluster without membership",
					subjectAccessReview: authorizationv1.SubjectAccessReview{
						Spec: authorizationv1.SubjectAccessReviewSpec{
							User: "aa725fea-0907-4ca8-be03-6a3728afd704",
							ResourceAttributes: &authorizationv1.ResourceAttributes{
								Namespace: "not-a-member",
								Verb:      "get",
								Resource:  "clusters",
								Group:     "dockyards.io",
							},
						},
					},
					expected: false,
				},
				{
					name: "test reader deleting cluster",
					subjectAccessReview: authorizationv1.SubjectAccessReview{
						Spec: authorizationv1.SubjectAccessReviewSpec{
							User: "aa725fea-0907-4ca8-be03-6a3728afd704",
							ResourceAttributes: &authorizationv1.ResourceAttributes{
								Namespace: "REPLACE",
								Verb:      "delete",
								Resource:  "clusters",
								Group:     "dockyards.io",
							},
						},
					},
					expected: false,
				},
				{
					name: "test reader getting deployments",
					subjectAccessReview: authorizationv1.SubjectAccessReview{
						Spec: authorizationv1.SubjectAccessReviewSpec{
							User: "aa725fea-0907-4ca8-be03-6a3728afd704",
							ResourceAttributes: &authorizationv1.ResourceAttributes{
								Namespace: "REPLACE",
								Verb:      "get",
								Resource:  "deployments",
								Group:     dockyardsv1.GroupVersion.Group,
							},
						},
					},
					expected: true,
				},
				{
					name: "test super user creating clusters",
					subjectAccessReview: authorizationv1.SubjectAccessReview{
						Spec: authorizationv1.SubjectAccessReviewSpec{
							User: "cffbcc36-fd31-4c1a-8d44-fce8b0d69688",
							ResourceAttributes: &authorizationv1.ResourceAttributes{
								Namespace: "REPLACE",
								Verb:      "create",
								Resource:  "clusters",
								Group:     dockyardsv1.GroupVersion.Group,
							},
						},
					},
					expected: true,
				},
				{
					name: "test user creating clusters",
					subjectAccessReview: authorizationv1.SubjectAccessReview{
						Spec: authorizationv1.SubjectAccessReviewSpec{
							User: "57dc0194-cecd-403a-901a-74dcb4e954e3",
							ResourceAttributes: &authorizationv1.ResourceAttributes{
								Namespace: "REPLACE",
								Verb:      "create",
								Resource:  "clusters",
								Group:     dockyardsv1.GroupVersion.Group,
							},
						},
					},
					expected: true,
				},
				{
					name: "test reader creating clusters",
					subjectAccessReview: authorizationv1.SubjectAccessReview{
						Spec: authorizationv1.SubjectAccessReviewSpec{
							User: "aa725fea-0907-4ca8-be03-6a3728afd704",
							ResourceAttributes: &authorizationv1.ResourceAttributes{
								Namespace: "REPLACE",
								Verb:      "create",
								Resource:  "clusters",
								Group:     dockyardsv1.GroupVersion.Group,
							},
						},
					},
					expected: false,
				},
			},
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.TODO())

			environment := envtest.Environment{
				CRDDirectoryPaths: []string{
					"../../config/crd",
				},
			}

			cfg, err := environment.Start()
			if err != nil {
				t.Fatalf("error starting test environment: %s", err)
			}

			t.Cleanup(func() {
				cancel()
				environment.Stop()
			})

			scheme := runtime.NewScheme()
			_ = dockyardsv1.AddToScheme(scheme)
			_ = rbacv1.AddToScheme(scheme)
			_ = authorizationv1.AddToScheme(scheme)

			c, err := client.New(cfg, client.Options{Scheme: scheme})
			if err != nil {
				t.Fatalf("error creating test client: %s", err)
			}

			mgr, err := ctrl.NewManager(cfg, ctrl.Options{})
			if err != nil {
				t.Fatalf("error creating test manager: %s", err)
			}

			err = (&controller.OrganizationReconciler{
				Client: mgr.GetClient(),
			}).SetupWithManager(mgr)
			if err != nil {
				t.Fatalf("errror creating organization reconciler: %s", err)
			}

			go func() {
				err := mgr.Start(ctx)
				if err != nil {
					panic(err)
				}
			}()

			err = c.Create(ctx, &tc.organization)
			if err != nil {
				t.Fatalf("error creating test organization: %s", err)
			}

			var reconciledOrganization dockyardsv1.Organization
			for i := 0; i < 5; i++ {
				err := c.Get(ctx, client.ObjectKey{Name: tc.organization.Name}, &reconciledOrganization)
				if err != nil {
					t.Fatalf("error getting reconciled organization: %s", err)
				}
				if reconciledOrganization.Status.NamespaceRef != nil {
					break
				}
				time.Sleep(time.Second)
			}

			if reconciledOrganization.Status.NamespaceRef == nil {
				t.Errorf("expected reconciled organization to have namespace reference")
			}

			var roleBinding rbacv1.RoleBinding
			for i := 0; i < 5; i++ {
				err := c.Get(ctx, client.ObjectKey{Name: "dockyards-user", Namespace: reconciledOrganization.Status.NamespaceRef.Name}, &roleBinding)
				if client.IgnoreNotFound(err) != nil {
					t.Fatalf("error getting reconciled rolebinding: %s", err)
				}
				if !apierrors.IsNotFound(err) {
					break
				}
				time.Sleep(time.Second)
			}

			if roleBinding.CreationTimestamp.IsZero() {
				t.Fatalf("error creating rolebinding")
			}

			var role rbacv1.Role
			err = c.Get(ctx, client.ObjectKey{Name: "dockyards-user", Namespace: reconciledOrganization.Status.NamespaceRef.Name}, &role)
			if err != nil {
				t.Fatalf("error getting role: %s", err)
			}

			if roleBinding.RoleRef.Name != role.Name {
				t.Fatalf("expected rolebinding reference name %s, got %s", role.Name, roleBinding.RoleRef.Name)
			}

			for _, review := range tc.reviews {
				t.Run(review.name, func(t *testing.T) {
					if review.subjectAccessReview.Spec.ResourceAttributes.Namespace == "REPLACE" {
						review.subjectAccessReview.Spec.ResourceAttributes.Namespace = reconciledOrganization.Status.NamespaceRef.Name
					}

					err := c.Create(ctx, &review.subjectAccessReview)
					if err != nil {
						t.Fatalf("error creating subjectaccessreview: %s", err)
					}

					if review.subjectAccessReview.Status.Allowed != review.expected {
						t.Errorf("expected allowed %t, got %t", review.expected, review.subjectAccessReview.Status.Allowed)
					}
				})
			}
		})
	}
}
