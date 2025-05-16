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

package handlers_test

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path"
	"testing"
	"time"

	"bitbucket.org/sudosweden/dockyards-api/pkg/types"
	"bitbucket.org/sudosweden/dockyards-backend/pkg/api/apiutil"
	dockyardsv1 "bitbucket.org/sudosweden/dockyards-backend/pkg/api/v1alpha3"
	"bitbucket.org/sudosweden/dockyards-backend/pkg/testing/testingutil"
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

	c := testEnvironment.GetClient()

	organization := testEnvironment.MustCreateOrganization(t)

	superUser := testEnvironment.MustGetOrganizationUser(t, organization, dockyardsv1.OrganizationMemberRoleSuperUser)
	reader := testEnvironment.MustGetOrganizationUser(t, organization, dockyardsv1.OrganizationMemberRoleReader)

	superUserToken := MustSignToken(t, string(superUser.UID))
	readerToken := MustSignToken(t, string(reader.UID))

	cluster := dockyardsv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "test-",
			Namespace:    organization.Spec.NamespaceRef.Name,
			OwnerReferences: []metav1.OwnerReference{
				{
					Kind:       dockyardsv1.OrganizationKind,
					APIVersion: dockyardsv1.GroupVersion.String(),
					Name:       organization.Name,
					UID:        organization.UID,
				},
			},
		},
	}

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

	mgr := testEnvironment.GetManager()

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

		u := url.URL{
			Path: path.Join("/v1/orgs/", organization.Name, "clusters", cluster.Name, "kubeconfig"),
		}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, u.Path, bytes.NewBuffer(b))

		r.Header.Add("Authorization", "Bearer "+superUserToken)

		mux.ServeHTTP(w, r)

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

		ownerOrganization, err := apiutil.GetOwnerOrganization(ctx, c, &cluster)
		if err != nil {
			t.Fatal("could not get cluster oranization owner")
		}

		if ownerOrganization.Name == "" {
			t.Fatal("could not get organization name")
		}

		superUserAlias := superUser.Name + "/" + cluster.Name
		clusterAlias := cluster.Name + "/" + ownerOrganization.Name

		expected := &clientcmdapi.Config{
			CurrentContext: superUserAlias + "@" + clusterAlias,
			Clusters: map[string]*clientcmdapi.Cluster{
				clusterAlias: {
					Server:                   "https://localhost:6443",
					CertificateAuthorityData: crt,
					Extensions:               map[string]runtime.Object{},
				},
			},
			Contexts: map[string]*clientcmdapi.Context{
				superUserAlias + "@" + clusterAlias: {
					Cluster:    clusterAlias,
					AuthInfo:   superUserAlias,
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

		block, _ := pem.Decode(actual.AuthInfos[superUserAlias].ClientCertificateData)

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

		u := url.URL{
			Path: path.Join("/v1/orgs/", organization.Name, "clusters", cluster.Name, "kubeconfig"),
		}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, u.Path, bytes.NewBuffer(b))

		r.Header.Add("Authorization", "Bearer "+readerToken)

		mux.ServeHTTP(w, r)

		statusCode := w.Result().StatusCode
		if statusCode != http.StatusUnauthorized {
			t.Fatalf("expected status code %d, got %d", http.StatusUnauthorized, statusCode)
		}
	})
}
