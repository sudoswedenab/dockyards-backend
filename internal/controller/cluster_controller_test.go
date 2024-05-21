package controller_test

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	"bitbucket.org/sudosweden/dockyards-backend/internal/controller"
	dockyardsv1 "bitbucket.org/sudosweden/dockyards-backend/pkg/api/v1alpha2"
	"github.com/go-logr/logr"
	"github.com/google/go-cmp/cmp"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
)

func TestClusterController(t *testing.T) {
	if os.Getenv("KUBEBUILDER_ASSETS") == "" {
		t.Skip("no kubebuilder assets configured")
	}

	versions := []string{
		"v1.30.1",
		"v1.29.5",
		"v1.28.10",
		"v1.27.14",
	}

	tt := []struct {
		name     string
		cluster  dockyardsv1.Cluster
		release  dockyardsv1.Release
		expected []dockyardsv1.ClusterUpgrade
	}{
		{
			name: "test cluster without upgrades",
			cluster: dockyardsv1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: "testing",
				},
				Spec: dockyardsv1.ClusterSpec{
					Version: "v1.30.1",
				},
			},
			release: dockyardsv1.Release{
				ObjectMeta: metav1.ObjectMeta{
					Name:      dockyardsv1.ReleaseNameSupportedKubernetesVersions,
					Namespace: "dockyards-testing",
				},
				Status: dockyardsv1.ReleaseStatus{
					Versions: versions,
				},
			},
		},
		{
			name: "test cluster with patch upgrade",
			cluster: dockyardsv1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: "testing",
				},
				Spec: dockyardsv1.ClusterSpec{
					Version: "v1.30.0",
				},
			},
			release: dockyardsv1.Release{
				ObjectMeta: metav1.ObjectMeta{
					Name:      dockyardsv1.ReleaseNameSupportedKubernetesVersions,
					Namespace: "dockyards-testing",
				},
				Status: dockyardsv1.ReleaseStatus{
					Versions: versions,
				},
			},
			expected: []dockyardsv1.ClusterUpgrade{
				{
					To: "v1.30.1",
				},
			},
		},
		{
			name: "test cluster with minor upgrade",
			cluster: dockyardsv1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: "testing",
				},
				Spec: dockyardsv1.ClusterSpec{
					Version: "v1.29.5",
				},
			},
			release: dockyardsv1.Release{
				ObjectMeta: metav1.ObjectMeta{
					Name:      dockyardsv1.ReleaseNameSupportedKubernetesVersions,
					Namespace: "dockyards-testing",
				},
				Status: dockyardsv1.ReleaseStatus{
					Versions: versions,
				},
			},
			expected: []dockyardsv1.ClusterUpgrade{
				{
					To: "v1.30.1",
				},
			},
		},
		{
			name: "test cluster with minor and patch upgrades",
			cluster: dockyardsv1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: "testing",
				},
				Spec: dockyardsv1.ClusterSpec{
					Version: "v1.29.4",
				},
			},
			release: dockyardsv1.Release{
				ObjectMeta: metav1.ObjectMeta{
					Name:      dockyardsv1.ReleaseNameSupportedKubernetesVersions,
					Namespace: "dockyards-testing",
				},
				Status: dockyardsv1.ReleaseStatus{
					Versions: versions,
				},
			},
			expected: []dockyardsv1.ClusterUpgrade{
				{
					To: "v1.30.1",
				},
				{
					To: "v1.29.5",
				},
			},
		},
		{
			name: "test cluster unable to skip to latest version",
			cluster: dockyardsv1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: "testing",
				},
				Spec: dockyardsv1.ClusterSpec{
					Version: "v1.28.10",
				},
			},
			release: dockyardsv1.Release{
				ObjectMeta: metav1.ObjectMeta{
					Name:      dockyardsv1.ReleaseNameSupportedKubernetesVersions,
					Namespace: "dockyards-testing",
				},
				Status: dockyardsv1.ReleaseStatus{
					Versions: versions,
				},
			},
			expected: []dockyardsv1.ClusterUpgrade{
				{
					To: "v1.29.5",
				},
			},
		},
		{
			name: "test cluster too old to upgrade to any version",
			cluster: dockyardsv1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: "testing",
				},
				Spec: dockyardsv1.ClusterSpec{
					Version: "v1.24.17",
				},
			},
			release: dockyardsv1.Release{
				ObjectMeta: metav1.ObjectMeta{
					Name:      dockyardsv1.ReleaseNameSupportedKubernetesVersions,
					Namespace: "dockyards-testing",
				},
				Status: dockyardsv1.ReleaseStatus{
					Versions: versions,
				},
			},
		},
		{
			name: "test cluster just outside of supported versions",
			cluster: dockyardsv1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: "testing",
				},
				Spec: dockyardsv1.ClusterSpec{
					Version: "v1.26.15",
				},
			},
			release: dockyardsv1.Release{
				ObjectMeta: metav1.ObjectMeta{
					Name:      dockyardsv1.ReleaseNameSupportedKubernetesVersions,
					Namespace: "dockyards-testing",
				},
				Status: dockyardsv1.ReleaseStatus{
					Versions: versions,
				},
			},
			expected: []dockyardsv1.ClusterUpgrade{
				{
					To: "v1.27.14",
				},
			},
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError})
			slogr := logr.FromSlogHandler(handler)
			ctrl.SetLogger(slogr)

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
			_ = corev1.AddToScheme(scheme)
			_ = dockyardsv1.AddToScheme(scheme)

			c, err := client.New(cfg, client.Options{Scheme: scheme})
			if err != nil {
				t.Fatalf("error creating test client: %s", err)
			}

			mgr, err := ctrl.NewManager(cfg, ctrl.Options{})
			if err != nil {
				t.Fatalf("error creating test manager: %s", err)
			}

			for _, name := range []string{"testing", "dockyards-testing"} {
				namespace := corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: name,
					},
				}

				err = c.Create(ctx, &namespace)
				if err != nil {
					t.Fatalf("error creating test namespace: %s", err)
				}
			}

			err = c.Create(ctx, &tc.cluster)
			if err != nil {
				t.Fatalf("error creating test cluster: %s", err)
			}

			err = c.Create(ctx, &tc.release)
			if err != nil {
				t.Fatalf("error creating test release: %s", err)
			}

			patch := client.MergeFrom(tc.release.DeepCopy())

			tc.release.Status.Versions = versions

			err = c.Status().Patch(ctx, &tc.release, patch)
			if err != nil {
				t.Fatalf("error patching test release: %s", err)
			}

			err = (&controller.ClusterReconciler{
				Client:             mgr.GetClient(),
				DockyardsNamespace: "dockyards-testing",
			}).SetupWithManager(mgr)
			if err != nil {
				t.Fatalf("error creating test reconciler: %s", err)
			}

			go func() {
				err := mgr.Start(ctx)
				if err != nil {
					panic(err)
				}
			}()

			var actual dockyardsv1.Cluster

			for i := 0; i < 5; i++ {
				err := c.Get(ctx, client.ObjectKeyFromObject(&tc.cluster), &actual)
				if err != nil {
					t.Fatalf("error getting test cluster: %s", err)
				}

				if actual.Spec.Upgrades != nil {
					break
				}

				time.Sleep(time.Second)
			}

			if !cmp.Equal(actual.Spec.Upgrades, tc.expected) {
				t.Errorf("diff: %s", cmp.Diff(tc.expected, actual.Spec.Upgrades))
			}
		})
	}
}
