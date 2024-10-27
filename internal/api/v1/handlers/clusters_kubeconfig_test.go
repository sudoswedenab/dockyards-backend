package handlers

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path"
	"testing"

	"bitbucket.org/sudosweden/dockyards-backend/internal/api/v1/middleware"
	dockyardsv1 "bitbucket.org/sudosweden/dockyards-backend/pkg/api/v1alpha3"
	"bitbucket.org/sudosweden/dockyards-backend/pkg/api/v1alpha3/index"
	"bitbucket.org/sudosweden/dockyards-backend/pkg/testing/testingutil"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientcmdapiv1 "k8s.io/client-go/tools/clientcmd/api/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
)

func TestGetClusterKubeconfig(t *testing.T) {
	if os.Getenv("KUBEBUILDER_ASSETS") == "" {
		t.Skip("no kubebuilder assets configured")
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	logr := logr.FromSlogHandler(logger.Handler())

	ctrl.SetLogger(logr)

	ctx, cancel := context.WithCancel(context.TODO())

	testEnvironment, err := testingutil.NewTestEnvironment(ctx, []string{path.Join("../../../../config/crd")})
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		cancel()
		err := testEnvironment.GetEnvironment().Stop()
		if err != nil {
			panic(err)
		}
	})

	mgr := testEnvironment.GetManager()
	c := testEnvironment.GetClient()

	err = mgr.GetFieldIndexer().IndexField(ctx, &dockyardsv1.Cluster{}, index.UIDField, index.ByUID)
	if err != nil {
		t.Fatal(err)
	}

	err = mgr.GetFieldIndexer().IndexField(ctx, &dockyardsv1.User{}, index.UIDField, index.ByUID)
	if err != nil {
		t.Fatal(err)
	}

	h := handler{
		Client: mgr.GetClient(),
	}

	organization := testEnvironment.GetOrganization()
	superUser := testEnvironment.GetSuperUser()
	user := testEnvironment.GetUser()
	reader := testEnvironment.GetReader()

	cluster := dockyardsv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "test-",
			Namespace:    organization.Status.NamespaceRef.Name,
		},
	}

	err = c.Create(ctx, &cluster)
	if err != nil {
		t.Fatal(err)
	}

	patch := client.MergeFrom(cluster.DeepCopy())

	cluster.Status.APIEndpoint = dockyardsv1.ClusterAPIEndpoint{
		Host: "localhost",
		Port: 6443,
	}

	err = c.Status().Patch(ctx, &cluster, patch)
	if err != nil {
		t.Fatal(err)
	}

	ca := corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cluster.Name + "-ca",
			Namespace: cluster.Namespace,
		},
		Data: map[string][]byte{
			corev1.TLSCertKey: []byte(`-----BEGIN CERTIFICATE-----
MIIBhTCCASugAwIBAgIQIRi6zePL6mKjOipn+dNuaTAKBggqhkjOPQQDAjASMRAw
DgYDVQQKEwdBY21lIENvMB4XDTE3MTAyMDE5NDMwNloXDTE4MTAyMDE5NDMwNlow
EjEQMA4GA1UEChMHQWNtZSBDbzBZMBMGByqGSM49AgEGCCqGSM49AwEHA0IABD0d
7VNhbWvZLWPuj/RtHFjvtJBEwOkhbN/BnnE8rnZR8+sbwnc/KhCk3FhnpHZnQz7B
5aETbbIgmuvewdjvSBSjYzBhMA4GA1UdDwEB/wQEAwICpDATBgNVHSUEDDAKBggr
BgEFBQcDATAPBgNVHRMBAf8EBTADAQH/MCkGA1UdEQQiMCCCDmxvY2FsaG9zdDo1
NDUzgg4xMjcuMC4wLjE6NTQ1MzAKBggqhkjOPQQDAgNIADBFAiEA2zpJEPQyz6/l
Wf86aX6PepsntZv2GYlA5UpabfT2EZICICpJ5h/iI+i341gBmLiAFQOyTDT+/wQc
6MF9+Yw1Yy0t
-----END CERTIFICATE-----`),
			corev1.TLSPrivateKeyKey: []byte(`-----BEGIN EC PRIVATE KEY-----
MHcCAQEEIIrYSSNQFaA2Hwf1duRSxKtLYX5CB04fSeQ6tF1aY/PuoAoGCCqGSM49
AwEHoUQDQgAEPR3tU2Fta9ktY+6P9G0cWO+0kETA6SFs38GecTyudlHz6xvCdz8q
EKTcWGekdmdDPsHloRNtsiCa697B2O9IFA==
-----END EC PRIVATE KEY-----`),
		},
	}

	err = c.Create(ctx, &ca)
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
		t.Log("could not sync cache")
	}

	t.Run("test as super user", func(t *testing.T) {
		h := handler{
			Client: mgr.GetClient(),
		}

		u := url.URL{
			Path: path.Join("/v1/clusters", string(cluster.UID), "kubeconfig"),
		}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, u.Path, nil)

		r.SetPathValue("clusterID", string(cluster.UID))

		ctx := middleware.ContextWithSubject(context.Background(), string(superUser.UID))
		ctx = middleware.ContextWithLogger(ctx, logger)

		h.GetClusterKubeconfig(w, r.Clone(ctx))

		statusCode := w.Result().StatusCode
		if statusCode != http.StatusOK {
			t.Fatalf("expected status code %d, got %d", http.StatusOK, statusCode)
		}

		b, err := io.ReadAll(w.Result().Body)
		if err != nil {
			t.Fatal(err)
		}

		var actual clientcmdapiv1.Config
		err = yaml.Unmarshal(b, &actual)
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("test as user", func(t *testing.T) {
		h := handler{
			Client: mgr.GetClient(),
		}

		u := url.URL{
			Path: path.Join("/v1/clusters", string(cluster.UID), "kubeconfig"),
		}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, u.Path, nil)

		r.SetPathValue("clusterID", string(cluster.UID))

		ctx := middleware.ContextWithSubject(context.Background(), string(user.UID))
		ctx = middleware.ContextWithLogger(ctx, logger)

		h.GetClusterKubeconfig(w, r.Clone(ctx))

		statusCode := w.Result().StatusCode
		if statusCode != http.StatusOK {
			t.Fatalf("expected status code %d, got %d", http.StatusOK, statusCode)
		}

		b, err := io.ReadAll(w.Result().Body)
		if err != nil {
			t.Fatal(err)
		}

		var actual clientcmdapiv1.Config
		err = yaml.Unmarshal(b, &actual)
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("test as reader", func(t *testing.T) {
		h := handler{
			Client: mgr.GetClient(),
		}

		u := url.URL{
			Path: path.Join("/v1/clusters", string(cluster.UID), "kubeconfig"),
		}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, u.Path, nil)

		r.SetPathValue("clusterID", string(cluster.UID))

		ctx := middleware.ContextWithSubject(context.Background(), string(reader.UID))
		ctx = middleware.ContextWithLogger(ctx, logger)

		h.GetClusterKubeconfig(w, r.Clone(ctx))

		statusCode := w.Result().StatusCode
		if statusCode != http.StatusUnauthorized {
			t.Fatalf("expected status code %d, got %d", http.StatusUnauthorized, statusCode)
		}
	})

	t.Run("test empty cluster id", func(t *testing.T) {
		u := url.URL{
			Path: path.Join("/v1/clusters", "", "kubeconfig"),
		}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, u.Path, nil)

		ctx := middleware.ContextWithSubject(context.Background(), string(superUser.UID))
		ctx = middleware.ContextWithLogger(ctx, logger)

		h.GetClusterKubeconfig(w, r.Clone(ctx))

		statusCode := w.Result().StatusCode
		if statusCode != http.StatusBadRequest {
			t.Fatalf("expected status code %d, got %d", http.StatusBadRequest, statusCode)
		}
	})

	t.Run("test non-existing cluster", func(t *testing.T) {
		h := handler{
			Client: mgr.GetClient(),
		}

		u := url.URL{
			Path: path.Join("/v1/clusters", "9228f00c-5141-4383-b68c-5873826c0351", "kubeconfig"),
		}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, u.Path, nil)

		r.SetPathValue("clusterID", "9228f00c-5141-4383-b68c-5873826c0351")

		ctx := middleware.ContextWithSubject(context.Background(), string(superUser.UID))
		ctx = middleware.ContextWithLogger(ctx, logger)

		h.GetClusterKubeconfig(w, r.Clone(ctx))

		statusCode := w.Result().StatusCode
		if statusCode != http.StatusUnauthorized {
			t.Fatalf("expected status code %d, got %d", http.StatusForbidden, statusCode)
		}
	})
}
