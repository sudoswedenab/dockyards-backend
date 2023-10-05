package openstack

import (
	"errors"
	"log/slog"
	"os"
	"testing"

	"bitbucket.org/sudosweden/dockyards-backend/pkg/api/v1alpha1"
	openstackv1alpha1 "bitbucket.org/sudosweden/dockyards-openstack/api/v1alpha1"
	"github.com/google/go-cmp/cmp"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestGetOpenstackProject(t *testing.T) {
	tt := []struct {
		name         string
		organization v1alpha1.Organization
		lists        []client.ObjectList
		expected     openstackv1alpha1.OpenstackProject
	}{
		{
			name: "test single project",
			organization: v1alpha1.Organization{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test",
					UID:  "9bdaf261-659c-4786-bd07-d6a8a8e04243",
				},
				Spec: v1alpha1.OrganizationSpec{
					CloudRef: &v1alpha1.CloudReference{
						APIVersion: openstackv1alpha1.GroupVersion.String(),
						Kind:       openstackv1alpha1.OpenstackProjectKind,
						Name:       "project1",
						Namespace:  "testing",
					},
				},
				Status: v1alpha1.OrganizationStatus{
					NamespaceRef: "testing-123",
				},
			},
			lists: []client.ObjectList{
				&openstackv1alpha1.OpenstackProjectList{
					Items: []openstackv1alpha1.OpenstackProject{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "project1",
								Namespace: "testing",
								UID:       "4d16ce9a-f5ba-44c3-80c1-e11186ee8c52",
							},
							Spec: openstackv1alpha1.OpenstackProjectSpec{
								ProjectID: "ee38bb89dbef19d7d9b60b1967794ad2",
							},
						},
					},
				},
			},
			expected: openstackv1alpha1.OpenstackProject{
				TypeMeta: metav1.TypeMeta{
					APIVersion: openstackv1alpha1.GroupVersion.String(),
					Kind:       openstackv1alpha1.OpenstackProjectKind,
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:            "project1",
					Namespace:       "testing",
					UID:             "4d16ce9a-f5ba-44c3-80c1-e11186ee8c52",
					ResourceVersion: "999",
				},
				Spec: openstackv1alpha1.OpenstackProjectSpec{
					ProjectID: "ee38bb89dbef19d7d9b60b1967794ad2",
				},
			},
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError + 1}))

			scheme := scheme.Scheme
			v1alpha1.AddToScheme(scheme)
			openstackv1alpha1.AddToScheme(scheme)
			fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithLists(tc.lists...).Build()

			s := openStackService{
				logger:           logger,
				controllerClient: fakeClient,
			}

			actual, err := s.getOpenstackProject(&tc.organization)
			if err != nil {
				t.Fatalf("error getting openstack project: %s", err)
			}

			if !cmp.Equal(actual, &tc.expected) {
				t.Errorf("diff: %s", cmp.Diff(&tc.expected, actual))
			}
		})
	}
}

func TestGetOpenstackProjectErrors(t *testing.T) {
	tt := []struct {
		name         string
		lists        []client.ObjectList
		organization v1alpha1.Organization
		expected     error
	}{
		{
			name: "test organization without cloud ref",
			organization: v1alpha1.Organization{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: "testing",
					UID:       "9a966287-a084-409c-8d85-94eca04dda9b",
				},
				Spec: v1alpha1.OrganizationSpec{},
			},
			expected: ErrNoCloudReference,
		},
		{
			name: "test organization with unsupported kind",
			organization: v1alpha1.Organization{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: "testing",
					UID:       "ea48801a-7448-42db-b5de-31aa9c9bd23e",
				},
				Spec: v1alpha1.OrganizationSpec{
					CloudRef: &v1alpha1.CloudReference{
						APIVersion: "cloud.dockyards.io/v1alpha1",
						Kind:       "NoCloudProject",
						Name:       "test",
					},
				},
			},
			expected: ErrNoOpenstackKind,
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError + 1}))

			scheme := scheme.Scheme
			v1alpha1.AddToScheme(scheme)
			openstackv1alpha1.AddToScheme(scheme)
			fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithLists(tc.lists...).Build()

			s := openStackService{
				logger:           logger,
				controllerClient: fakeClient,
			}

			_, err := s.getOpenstackProject(&tc.organization)
			if err == nil {
				t.Fatalf("expected error, got nil")
			}

			if !errors.Is(err, tc.expected) {
				t.Errorf("expected error '%s', got '%s'", tc.expected, err)
			}
		})
	}
}
