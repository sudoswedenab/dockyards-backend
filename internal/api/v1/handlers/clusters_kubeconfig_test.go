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
	"bitbucket.org/sudosweden/dockyards-backend/pkg/authorization"
	authorizationv1 "k8s.io/api/authorization/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientcmdapiv1 "k8s.io/client-go/tools/clientcmd/api/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/yaml"
)

func TestGetClusterKubeconfig(t *testing.T) {
	if os.Getenv("KUBEBUILDER_ASSETS") == "" {
		t.Skip("no kubebuilder assets configured")
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

	ctx, cancel := context.WithCancel(context.TODO())

	environment := envtest.Environment{
		CRDDirectoryPaths: []string{
			"../../../../config/crd",
		},
	}

	cfg, err := environment.Start()
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		cancel()
		environment.Stop()
	})

	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)
	_ = dockyardsv1.AddToScheme(scheme)
	_ = authorizationv1.AddToScheme(scheme)
	_ = rbacv1.AddToScheme(scheme)

	c, err := client.New(cfg, client.Options{Scheme: scheme})
	if err != nil {
		t.Fatal(err)
	}

	mgr, err := ctrl.NewManager(cfg, manager.Options{Scheme: scheme})
	if err != nil {
		t.Fatal(err)
	}

	err = mgr.GetFieldIndexer().IndexField(ctx, &dockyardsv1.Cluster{}, index.UIDField, index.ByUID)
	if err != nil {
		t.Fatal(err)
	}

	err = mgr.GetFieldIndexer().IndexField(ctx, &dockyardsv1.User{}, index.UIDField, index.ByUID)
	if err != nil {
		t.Fatal(err)
	}

	go func() {
		err := mgr.Start(ctx)
		if err != nil {
			panic(err)
		}
	}()

	h := handler{
		Client: mgr.GetClient(),
	}

	superUser := dockyardsv1.User{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "superuser-",
		},
		Spec: dockyardsv1.UserSpec{
			Email: "superuser@dockyards.dev",
		},
	}

	user := dockyardsv1.User{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "user-",
		},
		Spec: dockyardsv1.UserSpec{
			Email: "user@dockyards.dev",
		},
	}

	reader := dockyardsv1.User{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "reader-",
		},
		Spec: dockyardsv1.UserSpec{
			Email: "reader@dockyards.dev",
		},
	}

	organization := dockyardsv1.Organization{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "test-",
		},
	}

	t.Run("create users and organization", func(t *testing.T) {
		err := c.Create(ctx, &superUser)
		if err != nil {
			t.Fatal(err)
		}

		err = c.Create(ctx, &user)
		if err != nil {
			t.Fatal(err)
		}

		err = c.Create(ctx, &reader)
		if err != nil {
			t.Fatal(err)
		}

		organization.Spec = dockyardsv1.OrganizationSpec{
			MemberRefs: []dockyardsv1.OrganizationMemberReference{
				{
					Role: dockyardsv1.OrganizationMemberRoleSuperUser,
					UID:  superUser.UID,
				},
				{
					Role: dockyardsv1.OrganizationMemberRoleUser,
					UID:  user.UID,
				},
				{
					Role: dockyardsv1.OrganizationMemberRoleReader,
					UID:  reader.UID,
				},
			},
		}

		err = c.Create(ctx, &organization)
		if err != nil {
			t.Fatal(err)
		}

		namespace := corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: "test-",
			},
		}

		err = c.Create(ctx, &namespace)
		if err != nil {
			t.Fatal(err)
		}

		patch := client.MergeFrom(organization.DeepCopy())

		organization.Status.NamespaceRef = &corev1.LocalObjectReference{
			Name: namespace.Name,
		}

		err = c.Status().Patch(ctx, &organization, patch)
		if err != nil {
			t.Fatal(err)
		}

		err = authorization.ReconcileSuperUserClusterRoleAndBinding(ctx, c, &organization)
		if err != nil {
			t.Fatal(err)
		}

		err = authorization.ReconcileUserRoleAndBindings(ctx, c, &organization)
		if err != nil {
			t.Fatal(err)
		}

		err = authorization.ReconcileReaderClusterRoleAndBinding(ctx, c, &organization)
		if err != nil {
			t.Fatal(err)
		}

		err = authorization.ReconcileReaderRoleAndBinding(ctx, c, &organization)
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Logf("organization: %s, namespace reference: %s", organization.Name, organization.Status.NamespaceRef.Name)

	cluster := dockyardsv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "test-",
			Namespace:    organization.Status.NamespaceRef.Name,
		},
	}

	t.Run("create cluster", func(t *testing.T) {
		err := c.Create(ctx, &cluster)
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
	})

	t.Run("get kubeconfig as super user", func(t *testing.T) {
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

		//t.Logf("b: %s", b)

		var actual clientcmdapiv1.Config
		err = yaml.Unmarshal(b, &actual)
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("get kubeconfig as user", func(t *testing.T) {
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

		//t.Logf("b: %s", b)

		var actual clientcmdapiv1.Config
		err = yaml.Unmarshal(b, &actual)
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("get kubeconfig as reader", func(t *testing.T) {
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

	t.Run("empty cluster id", func(t *testing.T) {
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

	t.Run("invalid cluster id", func(t *testing.T) {
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
