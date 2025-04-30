package handlers

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path"
	"testing"
	"time"

	"bitbucket.org/sudosweden/dockyards-api/pkg/types"
	dockyardsv1 "bitbucket.org/sudosweden/dockyards-backend/pkg/api/v1alpha3"
	"bitbucket.org/sudosweden/dockyards-backend/pkg/api/v1alpha3/index"
	"bitbucket.org/sudosweden/dockyards-backend/pkg/testing/testingutil"
	utiljwt "bitbucket.org/sudosweden/dockyards-backend/pkg/util/jwt"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestClusterKubeconfig_Create(t *testing.T) {
	if os.Getenv("KUBEBUILDER_ASSETS") == "" {
		t.Skip("no kubebuilder assets configured")
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

	ctx, cancel := context.WithCancel(context.TODO())

	testEnvironment, err := testingutil.NewTestEnvironment(ctx, []string{path.Join("../../../../config/crd")})
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		cancel()
		testEnvironment.GetEnvironment().Stop()
	})

	mgr := testEnvironment.GetManager()
	c := testEnvironment.GetClient()

	organization := testEnvironment.GetOrganization()
	superUser := testEnvironment.GetSuperUser()
	reader := testEnvironment.GetReader()

	err = index.AddDefaultIndexes(ctx, mgr)
	if err != nil {
		t.Fatal(err)
	}

	go func() {
		err := mgr.Start(ctx)
		if err != nil {
			panic(err)
		}
	}()

	cluster := dockyardsv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "test-",
			Namespace:    organization.Spec.NamespaceRef.Name,
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

	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	der, err := x509.MarshalECPrivateKey(privateKey)
	if err != nil {
		t.Fatal(err)
	}

	block := pem.Block{
		Type:  "EC PRIVATE KEY",
		Bytes: der,
	}

	key := pem.EncodeToMemory(&block)

	tmpl := x509.Certificate{
		Issuer: pkix.Name{
			Organization: []string{
				"testing",
			},
		},
		Subject: pkix.Name{
			Organization: []string{
				"testing",
			},
		},
		BasicConstraintsValid: true,
		IsCA:                  true,
		NotBefore:             time.Now().Add(-time.Minute * 15),
		NotAfter:              time.Now().Add(time.Minute * 15),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtKeyUsage: []x509.ExtKeyUsage{
			x509.ExtKeyUsageServerAuth,
			x509.ExtKeyUsageClientAuth,
		},
	}

	der, err = x509.CreateCertificate(rand.Reader, &tmpl, &tmpl, &privateKey.PublicKey, privateKey)
	if err != nil {
		t.Fatal(err)
	}

	block = pem.Block{
		Type:  "CERTIFICATE",
		Bytes: der,
	}

	crt := pem.EncodeToMemory(&block)

	secret := corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cluster.Name + "-ca",
			Namespace: cluster.Namespace,
		},
		Data: map[string][]byte{
			corev1.TLSCertKey:       crt,
			corev1.TLSPrivateKeyKey: key,
		},
	}

	err = c.Create(ctx, &secret)
	if err != nil {
		t.Fatal(err)
	}

	accessKey, refreshKey, err := utiljwt.GetOrGenerateKeys(ctx, c, testEnvironment.GetDockyardsNamespace())
	if err != nil {
		t.Fatal(err)
	}

	m := http.NewServeMux()

	handlerOptions := []HandlerOption{
		WithManager(mgr),
		WithNamespace(testEnvironment.GetDockyardsNamespace()),
		WithLogger(logger),
		WithJWTPrivateKeys(accessKey, refreshKey),
	}

	err = RegisterRoutes(m, handlerOptions...)
	if err != nil {
		t.Fatal(err)
	}

	err = testingutil.RetryUntilFound(ctx, mgr.GetClient(), &cluster)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("test as super user", func(t *testing.T) {
		options := types.KubeconfigOptions{}

		b, err := json.Marshal(&options)
		if err != nil {
			t.Fatal(err)
		}

		claims := jwt.RegisteredClaims{
			Subject:   string(superUser.UID),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Minute * 30)),
		}

		token := jwt.NewWithClaims(jwt.SigningMethodES256, claims)
		signedToken, err := token.SignedString(accessKey)
		if err != nil {
			t.Fatal(err)
		}

		u := url.URL{
			Path: path.Join("/v1/orgs/", organization.Name, "clusters", cluster.Name, "kubeconfig"),
		}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, u.Path, bytes.NewBuffer(b))

		r.Header.Add("Authorization", "Bearer "+signedToken)

		m.ServeHTTP(w, r)

		statusCode := w.Result().StatusCode
		if statusCode != http.StatusCreated {
			t.Fatalf("expected status code %d, got %d", http.StatusCreated, statusCode)
		}

		b, err = io.ReadAll(w.Result().Body)
		if err != nil {
			t.Fatalf("unexpected error reading result body: %s", err)
		}

		actual, err := clientcmd.Load(b)
		if err != nil {
			t.Fatal(err)
		}

		expected := &clientcmdapi.Config{
			CurrentContext: superUser.Name + "@" + cluster.Name,
			Clusters: map[string]*clientcmdapi.Cluster{
				cluster.Name: {
					Server:                   "https://localhost:6443",
					CertificateAuthorityData: crt,
					Extensions:               map[string]runtime.Object{},
				},
			},
			Contexts: map[string]*clientcmdapi.Context{
				superUser.Name + "@" + cluster.Name: {
					Cluster:    cluster.Name,
					AuthInfo:   superUser.Name,
					Extensions: map[string]runtime.Object{},
				},
			},
			Preferences: clientcmdapi.Preferences{
				Extensions: map[string]runtime.Object{},
			},
			Extensions: map[string]runtime.Object{},
		}

		opts := cmpopts.IgnoreFields(clientcmdapi.Config{}, "AuthInfos")

		if !cmp.Equal(actual, expected, opts) {
			t.Errorf("diff: %s", cmp.Diff(expected, actual, opts))
		}

		block, _ := pem.Decode(actual.AuthInfos[superUser.Name].ClientCertificateData)

		actualCertificate, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			t.Fatal(err)
		}

		expectedCertificate := x509.Certificate{
			Subject: pkix.Name{
				CommonName: superUser.Name,
				Organization: []string{
					"system:masters",
				},
				Names: actualCertificate.Subject.Names,
			},
			SignatureAlgorithm: x509.ECDSAWithSHA256,
			Issuer: pkix.Name{
				Organization: []string{
					"testing",
				},
				Names: actualCertificate.Issuer.Names,
			},
			KeyUsage: x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
			ExtKeyUsage: []x509.ExtKeyUsage{
				x509.ExtKeyUsageClientAuth,
			},
			PublicKeyAlgorithm: x509.RSA,
			Version:            3,
			//
			AuthorityKeyId:          actualCertificate.AuthorityKeyId,
			Extensions:              actualCertificate.Extensions,
			NotAfter:                actualCertificate.NotAfter,
			NotBefore:               actualCertificate.NotBefore,
			PublicKey:               actualCertificate.PublicKey,
			Raw:                     actualCertificate.Raw,
			RawIssuer:               actualCertificate.RawIssuer,
			RawSubject:              actualCertificate.RawSubject,
			RawSubjectPublicKeyInfo: actualCertificate.RawSubjectPublicKeyInfo,
			RawTBSCertificate:       actualCertificate.RawTBSCertificate,
			Signature:               actualCertificate.Signature,
		}

		ignoreFields := cmpopts.IgnoreFields(x509.Certificate{}, "SerialNumber")

		if !cmp.Equal(*actualCertificate, expectedCertificate, ignoreFields) {
			t.Errorf("diff: %s", cmp.Diff(expectedCertificate, *actualCertificate, ignoreFields))
		}

		roots := x509.NewCertPool()
		roots.AppendCertsFromPEM(crt)

		verifyOptions := x509.VerifyOptions{
			Roots: roots,
			KeyUsages: []x509.ExtKeyUsage{
				x509.ExtKeyUsageClientAuth,
			},
		}

		_, err = actualCertificate.Verify(verifyOptions)
		if err != nil {
			t.Errorf("expected certificate to verify without errors, got %s", err)
		}
	})

	t.Run("test as reader", func(t *testing.T) {
		options := types.KubeconfigOptions{}

		b, err := json.Marshal(&options)
		if err != nil {
			t.Fatal(err)
		}

		claims := jwt.RegisteredClaims{
			Subject:   string(reader.UID),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Minute * 30)),
		}

		token := jwt.NewWithClaims(jwt.SigningMethodES256, claims)
		signedToken, err := token.SignedString(accessKey)
		if err != nil {
			t.Fatal(err)
		}

		u := url.URL{
			Path: path.Join("/v1/orgs/", organization.Name, "clusters", cluster.Name, "kubeconfig"),
		}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, u.Path, bytes.NewBuffer(b))

		r.Header.Add("Authorization", "Bearer "+signedToken)

		m.ServeHTTP(w, r)

		statusCode := w.Result().StatusCode
		if statusCode != http.StatusUnauthorized {
			t.Fatalf("expected status code %d, got %d", http.StatusUnauthorized, statusCode)
		}
	})
}
