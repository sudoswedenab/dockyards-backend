package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path"
	"testing"
	"time"

	"bitbucket.org/sudosweden/dockyards-backend/internal/api/v1"
	"bitbucket.org/sudosweden/dockyards-backend/internal/api/v1/middleware"
	dockyardsv1 "bitbucket.org/sudosweden/dockyards-backend/pkg/api/v1alpha2"
	"bitbucket.org/sudosweden/dockyards-backend/pkg/api/v1alpha2/index"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
)

func TestGetOrgCredentials(t *testing.T) {
	tt := []struct {
		name           string
		sub            string
		organizationID string
		lists          []client.ObjectList
		expected       []v1.Credential
	}{
		{
			name:           "test single credential",
			sub:            "654202f2-44f6-4fa6-873b-0b9817d3957c",
			organizationID: "af2224ee-fd4b-4e6c-8ff6-21c2d1ddcc5c",
			lists: []client.ObjectList{
				&dockyardsv1.OrganizationList{
					Items: []dockyardsv1.Organization{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name: "test",
								UID:  "af2224ee-fd4b-4e6c-8ff6-21c2d1ddcc5c",
							},
							Spec: dockyardsv1.OrganizationSpec{
								MemberRefs: []dockyardsv1.MemberReference{
									{
										UID: "654202f2-44f6-4fa6-873b-0b9817d3957c",
									},
								},
							},
							Status: dockyardsv1.OrganizationStatus{
								NamespaceRef: "testing",
							},
						},
					},
				},
				&corev1.SecretList{
					Items: []corev1.Secret{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "test",
								Namespace: "testing",
								UID:       "54376668-876c-43d7-8d29-2ef37ccab831",
							},
							Type: dockyardsv1.SecretTypeCredential,
						},
					},
				},
			},
			expected: []v1.Credential{
				{
					ID:           "54376668-876c-43d7-8d29-2ef37ccab831",
					Name:         "test",
					Organization: "test",
				},
			},
		},
		{
			name:           "test several secret types",
			sub:            "41ae3267-da66-4be0-b2ac-57a60549ff57",
			organizationID: "8afac404-d43a-4253-a102-a90ff80fa13c",
			lists: []client.ObjectList{
				&dockyardsv1.OrganizationList{
					Items: []dockyardsv1.Organization{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name: "test",
								UID:  "8afac404-d43a-4253-a102-a90ff80fa13c",
							},
							Spec: dockyardsv1.OrganizationSpec{
								MemberRefs: []dockyardsv1.MemberReference{
									{
										UID: "41ae3267-da66-4be0-b2ac-57a60549ff57",
									},
								},
							},
							Status: dockyardsv1.OrganizationStatus{
								NamespaceRef: "testing",
							},
						},
					},
				},
				&corev1.SecretList{
					Items: []corev1.Secret{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "dockyards-io-credential",
								Namespace: "testing",
								UID:       "3cca83a8-7848-40ad-aa89-916a28f6016d",
							},
							Type: dockyardsv1.SecretTypeCredential,
						},
						{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "kubernetes-io-ssh-auth",
								Namespace: "testing",
								UID:       "bf8fc71c-3278-40fe-a452-ed0b1ee189b8",
							},
							Type: corev1.SecretTypeSSHAuth,
						},
						{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "kubernetes-io-tls",
								Namespace: "testing",
								UID:       "224a442e-515f-4042-9e03-10de6b827ecf",
							},
							Type: corev1.SecretTypeTLS,
						},
						{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "opaque",
								Namespace: "testing",
								UID:       "1efe00b7-9e6a-425d-88ea-99cc41eb6011",
							},
							Type: corev1.SecretTypeOpaque,
						},
					},
				},
			},
			expected: []v1.Credential{
				{
					ID:           "3cca83a8-7848-40ad-aa89-916a28f6016d",
					Name:         "dockyards-io-credential",
					Organization: "test",
				},
			},
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

			scheme := scheme.Scheme
			dockyardsv1.AddToScheme(scheme)
			fakeClient := fake.NewClientBuilder().
				WithScheme(scheme).
				WithLists(tc.lists...).
				WithIndex(&dockyardsv1.Organization{}, index.UIDField, index.ByUID).
				WithIndex(&corev1.Secret{}, index.SecretTypeField, index.BySecretType).
				Build()

			h := handler{
				Client: fakeClient,
			}

			u := url.URL{
				Path: path.Join("/v1/orgs", tc.organizationID, "credentials"),
			}

			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, u.Path, nil)

			r.SetPathValue("organizationID", tc.organizationID)

			ctx := middleware.ContextWithSubject(context.Background(), tc.sub)
			ctx = middleware.ContextWithLogger(ctx, logger)

			h.GetOrgCredentials(w, r.Clone(ctx))

			statusCode := w.Result().StatusCode
			if statusCode != http.StatusOK {
				t.Fatalf("expected status code %d, got %d", http.StatusOK, statusCode)
			}

			body, err := io.ReadAll(w.Result().Body)
			if err != nil {
				t.Fatalf("error reading result body: %s", err)
			}

			var actual []v1.Credential
			err = json.Unmarshal(body, &actual)
			if err != nil {
				t.Fatalf("error unmarshalling result body json: %s", err)
			}

			if !cmp.Equal(actual, tc.expected) {
				t.Errorf("diff: %s", cmp.Diff(tc.expected, actual))
			}
		})
	}
}

func TestPutOrganizationCredential(t *testing.T) {
	tt := []struct {
		name             string
		subject          string
		organizationName string
		credentialName   string
		organization     dockyardsv1.Organization
		credential       v1.Credential
		secret           corev1.Secret
		expected         corev1.Secret
	}{
		{
			name:             "test update empty",
			subject:          "92b0aabc-96a4-40ef-987d-5daa412f4f0d",
			organizationName: "test",
			credentialName:   "test-update-empty",
			organization: dockyardsv1.Organization{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test",
				},
				Spec: dockyardsv1.OrganizationSpec{
					MemberRefs: []dockyardsv1.MemberReference{
						{
							Role: dockyardsv1.MemberRoleSuperUser,
							UID:  "92b0aabc-96a4-40ef-987d-5daa412f4f0d",
						},
					},
				},
				Status: dockyardsv1.OrganizationStatus{
					NamespaceRef: "testing",
				},
			},
			credential: v1.Credential{
				Data: &map[string][]byte{
					"test": []byte("secret"),
				},
			},
			secret: corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "credential-test-update-empty",
					Namespace: "testing",
				},
				Type: dockyardsv1.SecretTypeCredential,
			},
			expected: corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "credential-test-update-empty",
					Namespace: "testing",
				},
				Data: map[string][]byte{
					"test": []byte("secret"),
				},
				Type: dockyardsv1.SecretTypeCredential,
			},
		},
		{
			name:             "test update existing key",
			subject:          "ea6a1fa3-56c7-40d3-90cb-4d1c8249576e",
			organizationName: "test",
			credentialName:   "test-update-existing-key",
			organization: dockyardsv1.Organization{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test",
				},
				Spec: dockyardsv1.OrganizationSpec{
					MemberRefs: []dockyardsv1.MemberReference{
						{
							Role: dockyardsv1.MemberRoleSuperUser,
							UID:  "ea6a1fa3-56c7-40d3-90cb-4d1c8249576e",
						},
					},
				},
				Status: dockyardsv1.OrganizationStatus{
					NamespaceRef: "testing",
				},
			},
			credential: v1.Credential{
				Data: &map[string][]byte{
					"test": []byte("secret"),
				},
			},
			secret: corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "credential-test-update-existing-key",
					Namespace: "testing",
				},
				Data: map[string][]byte{
					"test": []byte("qwfp"),
					"hjkl": []byte("arst"),
					"zxcv": []byte("neio"),
				},
				Type: dockyardsv1.SecretTypeCredential,
			},
			expected: corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "credential-test-update-existing-key",
					Namespace: "testing",
				},
				Data: map[string][]byte{
					"test": []byte("secret"),
					"hjkl": []byte("arst"),
					"zxcv": []byte("neio"),
				},
				Type: dockyardsv1.SecretTypeCredential,
			},
		},
		{
			name:             "test remove existing key",
			organizationName: "test",
			credentialName:   "test-remove-existing-key",
			subject:          "8f129312-5639-4110-8415-b0cd3c66f58f",
			organization: dockyardsv1.Organization{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test",
				},
				Spec: dockyardsv1.OrganizationSpec{
					MemberRefs: []dockyardsv1.MemberReference{
						{
							Role: dockyardsv1.MemberRoleSuperUser,
							UID:  "8f129312-5639-4110-8415-b0cd3c66f58f",
						},
					},
				},
				Status: dockyardsv1.OrganizationStatus{
					NamespaceRef: "testing",
				},
			},
			credential: v1.Credential{
				Data: &map[string][]byte{
					"test": nil,
				},
			},
			secret: corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "credential-test-remove-existing-key",
					Namespace: "testing",
				},
				Data: map[string][]byte{
					"test": []byte("secret"),
					"arst": []byte("qwfp"),
				},
				Type: dockyardsv1.SecretTypeCredential,
			},
			expected: corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "credential-test-remove-existing-key",
					Namespace: "testing",
				},
				Data: map[string][]byte{
					"arst": []byte("qwfp"),
				},
				Type: dockyardsv1.SecretTypeCredential,
			},
		},
		{
			name:             "test update existing key to empty string",
			organizationName: "test",
			credentialName:   "test-update-empty-string",
			subject:          "0b24eb27-2aac-4a00-b64a-3eaf3f301194",
			organization: dockyardsv1.Organization{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test",
				},
				Spec: dockyardsv1.OrganizationSpec{
					MemberRefs: []dockyardsv1.MemberReference{
						{
							Role: dockyardsv1.MemberRoleSuperUser,
							UID:  "0b24eb27-2aac-4a00-b64a-3eaf3f301194",
						},
					},
				},
				Status: dockyardsv1.OrganizationStatus{
					NamespaceRef: "testing",
				},
			},
			secret: corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "credential-test-update-empty-string",
					Namespace: "testing",
				},
				Data: map[string][]byte{
					"test": []byte("secret"),
					"zxcv": []byte("neio"),
				},
				Type: dockyardsv1.SecretTypeCredential,
			},
			credential: v1.Credential{
				Data: &map[string][]byte{
					"test": []byte(""),
				},
			},
			expected: corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "credential-test-update-empty-string",
					Namespace: "testing",
				},
				Data: map[string][]byte{
					"test": []byte(""),
					"zxcv": []byte("neio"),
				},
				Type: dockyardsv1.SecretTypeCredential,
			},
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			if os.Getenv("KUBEBUILDER_ASSETS") == "" {
				t.Skip("no kubebuilder assets configured")
			}

			logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

			ctx, cancel := context.WithCancel(context.TODO())

			environment := envtest.Environment{
				CRDDirectoryPaths: []string{
					"../../../../config/crd",
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

			scheme := scheme.Scheme
			_ = dockyardsv1.AddToScheme(scheme)

			c, err := client.New(cfg, client.Options{Scheme: scheme})
			if err != nil {
				t.Fatalf("error creating test client: %s", err)
			}

			namespace := corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "testing",
				},
			}

			err = c.Create(ctx, &namespace)
			if err != nil {
				t.Fatalf("error creating test namespace: %s", err)
			}

			err = c.Create(ctx, &tc.organization)
			if err != nil {
				t.Fatalf("error creating test organization: %s", err)
			}

			patch := client.MergeFrom(tc.organization.DeepCopy())

			tc.organization.Status.NamespaceRef = "testing"

			err = c.Status().Patch(ctx, &tc.organization, patch)
			if err != nil {
				t.Fatalf("error patching test organization: %s", err)
			}

			err = c.Create(ctx, &tc.secret)
			if err != nil {
				t.Fatalf("error creating test secret: %s", err)
			}

			h := handler{
				Client: c,
			}

			u := url.URL{
				Path: path.Join("/v1/organizations", tc.organizationName, "credentials", tc.credentialName),
			}

			b, err := json.Marshal(tc.credential)
			if err != nil {
				t.Fatalf("error marshalling test credential: %s", err)
			}

			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, u.Path, bytes.NewBuffer(b))

			r.SetPathValue("organizationName", tc.organizationName)
			r.SetPathValue("credentialName", tc.credentialName)

			ctx = middleware.ContextWithSubject(ctx, tc.subject)
			ctx = middleware.ContextWithLogger(ctx, logger)

			h.PutOrganizationCredential(w, r.Clone(ctx))

			if w.Result().StatusCode != http.StatusNoContent {
				t.Fatalf("expected status code %d, got %d", http.StatusNoContent, w.Result().StatusCode)
			}

			var actual corev1.Secret
			err = c.Get(ctx, client.ObjectKeyFromObject(&tc.expected), &actual)
			if err != nil {
				t.Fatalf("error getting expected secret: %s", err)
			}

			options := cmp.Options{
				cmpopts.IgnoreTypes(time.Time{}),
				cmpopts.IgnoreFields(metav1.ObjectMeta{}, "UID", "ResourceVersion", "ManagedFields"),
			}

			if !cmp.Equal(tc.expected, actual, options) {
				t.Errorf("diff: %s", cmp.Diff(actual, tc.expected, options))
			}
		})
	}
}

func TestPostOrganizationCredentials(t *testing.T) {
	tt := []struct {
		name             string
		subject          string
		organizationName string
		organization     dockyardsv1.Organization
		credential       v1.Credential
		expected         corev1.Secret
	}{
		{
			name:             "test create empty credential",
			subject:          "755c43a6-09bb-485a-8826-23a582b70a98",
			organizationName: "test",
			organization: dockyardsv1.Organization{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test",
					UID:  "0b8a1fe3-90d7-4762-a3ab-929c8d4cae68",
				},
				Spec: dockyardsv1.OrganizationSpec{
					MemberRefs: []dockyardsv1.MemberReference{
						{
							Role: dockyardsv1.MemberRoleSuperUser,
							UID:  "755c43a6-09bb-485a-8826-23a582b70a98",
						},
					},
				},
				Status: dockyardsv1.OrganizationStatus{
					NamespaceRef: "testing",
				},
			},
			credential: v1.Credential{
				Name: "test-create-empty-credential",
				Data: &map[string][]byte{},
			},
			expected: corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "credential-test-create-empty-credential",
					Namespace: "testing",
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion: dockyardsv1.GroupVersion.String(),
							Kind:       dockyardsv1.OrganizationKind,
							Name:       "test",
							UID:        "0b8a1fe3-90d7-4762-a3ab-929c8d4cae68",
						},
					},
				},
				Type: dockyardsv1.SecretTypeCredential,
			},
		},
		{
			name:             "test credential with single key",
			subject:          "962d948f-a7c4-44b8-94e8-03f11d1ee1dc",
			organizationName: "test",
			organization: dockyardsv1.Organization{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test",
					UID:  "0b8a1fe3-90d7-4762-a3ab-929c8d4cae68",
				},
				Spec: dockyardsv1.OrganizationSpec{
					MemberRefs: []dockyardsv1.MemberReference{
						{
							Role: dockyardsv1.MemberRoleSuperUser,
							UID:  "962d948f-a7c4-44b8-94e8-03f11d1ee1dc",
						},
					},
				},
				Status: dockyardsv1.OrganizationStatus{
					NamespaceRef: "testing",
				},
			},
			credential: v1.Credential{
				Name: "test-create-empty-credential",
				Data: &map[string][]byte{
					"test": []byte("secret"),
				},
			},
			expected: corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "credential-test-create-empty-credential",
					Namespace: "testing",
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion: dockyardsv1.GroupVersion.String(),
							Kind:       dockyardsv1.OrganizationKind,
							Name:       "test",
							UID:        "0b8a1fe3-90d7-4762-a3ab-929c8d4cae68",
						},
					},
				},
				Data: map[string][]byte{
					"test": []byte("secret"),
				},
				Type: dockyardsv1.SecretTypeCredential,
			},
		},
		{
			name:             "test credential with multiple keys",
			subject:          "fdf5bb49-e430-4fb8-b846-575363224c76",
			organizationName: "test",
			organization: dockyardsv1.Organization{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test",
					UID:  "0b8a1fe3-90d7-4762-a3ab-929c8d4cae68",
				},
				Spec: dockyardsv1.OrganizationSpec{
					MemberRefs: []dockyardsv1.MemberReference{
						{
							Role: dockyardsv1.MemberRoleSuperUser,
							UID:  "fdf5bb49-e430-4fb8-b846-575363224c76",
						},
					},
				},
				Status: dockyardsv1.OrganizationStatus{
					NamespaceRef: "testing",
				},
			},
			credential: v1.Credential{
				Name: "test-credential-with-multiple-keys",
				Data: &map[string][]byte{
					"qwfp": []byte("arst"),
					"zxcv": []byte("neio"),
					"hjkl": []byte("wars"),
				},
			},
			expected: corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "credential-test-credential-with-multiple-keys",
					Namespace: "testing",
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion: dockyardsv1.GroupVersion.String(),
							Kind:       dockyardsv1.OrganizationKind,
							Name:       "test",
							UID:        "0b8a1fe3-90d7-4762-a3ab-929c8d4cae68",
						},
					},
				},
				Data: map[string][]byte{
					"qwfp": []byte("arst"),
					"zxcv": []byte("neio"),
					"hjkl": []byte("wars"),
				},
				Type: dockyardsv1.SecretTypeCredential,
			},
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			if os.Getenv("KUBEBUILDER_ASSETS") == "" {
				t.Skip("no kubebuilder assets configured")
			}

			logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

			ctx, cancel := context.WithCancel(context.TODO())

			environment := envtest.Environment{
				CRDDirectoryPaths: []string{
					"../../../../config/crd",
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

			scheme := scheme.Scheme
			_ = dockyardsv1.AddToScheme(scheme)

			c, err := client.New(cfg, client.Options{Scheme: scheme})
			if err != nil {
				t.Fatalf("error creating test client: %s", err)
			}

			namespace := corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "testing",
				},
			}

			err = c.Create(ctx, &namespace)
			if err != nil {
				t.Fatalf("error creating test namespace: %s", err)
			}

			err = c.Create(ctx, &tc.organization)
			if err != nil {
				t.Fatalf("error creating test organization: %s", err)
			}

			patch := client.MergeFrom(tc.organization.DeepCopy())

			tc.organization.Status.NamespaceRef = "testing"

			err = c.Status().Patch(ctx, &tc.organization, patch)
			if err != nil {
				t.Fatalf("error patching test organization: %s", err)
			}

			h := handler{
				Client: c,
			}

			u := url.URL{
				Path: path.Join("/v1/organizations", tc.organizationName, "credentials"),
			}

			b, err := json.Marshal(tc.credential)
			if err != nil {
				t.Fatalf("error marshalling test credential: %s", err)
			}

			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodPost, u.Path, bytes.NewBuffer(b))

			r.SetPathValue("organizationName", tc.organizationName)

			ctx = middleware.ContextWithSubject(ctx, tc.subject)
			ctx = middleware.ContextWithLogger(ctx, logger)

			h.PostOrganizationCredentials(w, r.Clone(ctx))

			if w.Result().StatusCode != http.StatusCreated {
				t.Fatalf("expected status code %d, got %d", http.StatusCreated, w.Result().StatusCode)
			}

			var actual corev1.Secret
			err = c.Get(ctx, client.ObjectKeyFromObject(&tc.expected), &actual)
			if err != nil {
				t.Fatalf("error getting expected secret: %s", err)
			}

			options := cmp.Options{
				cmpopts.IgnoreTypes(time.Time{}),
				cmpopts.IgnoreFields(metav1.ObjectMeta{}, "UID", "ResourceVersion", "ManagedFields"),
				cmpopts.IgnoreFields(metav1.OwnerReference{}, "UID"),
			}

			if !cmp.Equal(actual, tc.expected, options) {
				t.Errorf("diff: %s", cmp.Diff(tc.expected, actual, options))
			}
		})
	}
}

func TestGetOrganizationCredential(t *testing.T) {
	tt := []struct {
		name             string
		organizationName string
		credentialName   string
		subject          string
		organization     dockyardsv1.Organization
		secret           corev1.Secret
		expected         v1.Credential
	}{
		{
			name:             "test empty credential",
			organizationName: "test",
			credentialName:   "test-empty-credential",
			subject:          "19b704bd-4217-41ef-b86f-a3b73ce5b4c6",
			organization: dockyardsv1.Organization{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test",
				},
				Spec: dockyardsv1.OrganizationSpec{
					MemberRefs: []dockyardsv1.MemberReference{
						{
							Role: dockyardsv1.MemberRoleSuperUser,
							UID:  "19b704bd-4217-41ef-b86f-a3b73ce5b4c6",
						},
					},
				},
			},
			secret: corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "credential-test-empty-credential",
					Namespace: "testing",
					UID:       "7bf4a804-82eb-4a43-8d33-c017cd57fda5",
				},
				Type: dockyardsv1.SecretTypeCredential,
			},
			expected: v1.Credential{
				ID:           "7bf4a804-82eb-4a43-8d33-c017cd57fda5",
				Name:         "test-empty-credential",
				Organization: "test",
			},
		},
		{
			name:             "test credential with multiple keys",
			organizationName: "test",
			credentialName:   "test-multiple-keys",
			subject:          "dd886032-c690-4d7d-b1a1-c0f19fce1ea7",
			organization: dockyardsv1.Organization{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test",
				},
				Spec: dockyardsv1.OrganizationSpec{
					MemberRefs: []dockyardsv1.MemberReference{
						{
							Role: dockyardsv1.MemberRoleSuperUser,
							UID:  "dd886032-c690-4d7d-b1a1-c0f19fce1ea7",
						},
					},
				},
			},
			secret: corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "credential-test-multiple-keys",
					Namespace: "testing",
					UID:       "219070d3-8294-4cb5-8db7-c4486cff9730",
				},
				Data: map[string][]byte{
					"qwfp": []byte("arst"),
					"zxcv": []byte("neio"),
					"hjkl": []byte("wars"),
				},
				Type: dockyardsv1.SecretTypeCredential,
			},
			expected: v1.Credential{
				ID:           "219070d3-8294-4cb5-8db7-c4486cff9730",
				Name:         "test-multiple-keys",
				Organization: "test",
				Data: &map[string][]byte{
					"qwfp": nil,
					"zxcv": nil,
					"hjkl": nil,
				},
			},
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			if os.Getenv("KUBEBUILDER_ASSETS") == "" {
				t.Skip("no kubebuilder assets configured")
			}

			logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

			ctx, cancel := context.WithCancel(context.TODO())

			environment := envtest.Environment{
				CRDDirectoryPaths: []string{
					"../../../../config/crd",
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

			scheme := scheme.Scheme
			_ = dockyardsv1.AddToScheme(scheme)

			c, err := client.New(cfg, client.Options{Scheme: scheme})
			if err != nil {
				t.Fatalf("error creating test client: %s", err)
			}

			namespace := corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "testing",
				},
			}

			err = c.Create(ctx, &namespace)
			if err != nil {
				t.Fatalf("error creating test namespace: %s", err)
			}

			err = c.Create(ctx, &tc.organization)
			if err != nil {
				t.Fatalf("error creating test organization: %s", err)
			}

			patch := client.MergeFrom(tc.organization.DeepCopy())

			tc.organization.Status.NamespaceRef = "testing"

			err = c.Status().Patch(ctx, &tc.organization, patch)
			if err != nil {
				t.Fatalf("error patching test organization: %s", err)
			}

			err = c.Create(ctx, &tc.secret)
			if err != nil {
				t.Fatalf("error creating test secret: %s", err)
			}

			h := handler{
				Client: c,
			}

			u := url.URL{
				Path: path.Join("/v1/organizations", tc.organizationName, "credentials", tc.credentialName),
			}

			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, u.Path, nil)

			r.SetPathValue("organizationName", tc.organizationName)
			r.SetPathValue("credentialName", tc.credentialName)

			ctx = middleware.ContextWithSubject(ctx, tc.subject)
			ctx = middleware.ContextWithLogger(ctx, logger)

			h.GetOrganizationCredential(w, r.Clone(ctx))

			if w.Result().StatusCode != http.StatusOK {
				t.Fatalf("expected status code %d, got %d", http.StatusOK, w.Result().StatusCode)
			}

			body, err := io.ReadAll(w.Result().Body)
			if err != nil {
				t.Fatalf("error reading test body: %s", err)
			}

			t.Log(string(body))

			var actual v1.Credential
			err = json.Unmarshal(body, &actual)
			if err != nil {
				t.Fatalf("error unmarhalling body: %s", err)
			}

			opts := cmpopts.IgnoreFields(v1.Credential{}, "ID")

			if !cmp.Equal(actual, tc.expected, opts) {
				t.Errorf("diff: %s", cmp.Diff(tc.expected, actual, opts))
			}
		})
	}
}
