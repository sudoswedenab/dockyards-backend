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
	"strings"
	"testing"

	"bitbucket.org/sudosweden/dockyards-api/pkg/types"
	"bitbucket.org/sudosweden/dockyards-backend/internal/api/v1/middleware"
	dockyardsv1 "bitbucket.org/sudosweden/dockyards-backend/pkg/api/v1alpha3"
	"bitbucket.org/sudosweden/dockyards-backend/pkg/api/v1alpha3/index"
	"bitbucket.org/sudosweden/dockyards-backend/pkg/testing/testingutil"
	"github.com/google/go-cmp/cmp"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestCredential_GetOrganizationCredentials(t *testing.T) {
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

	organization := testEnvironment.GetOrganization()

	c := testEnvironment.GetClient()

	superUser := testEnvironment.GetSuperUser()

	h := handler{
		Client: mgr.GetClient(),
	}

	u := url.URL{
		Path: path.Join("/v1/organizations", organization.Name, "credentials"),
	}

	t.Run("test single credential", func(t *testing.T) {
		secret := corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "credential-test",
				Namespace: organization.Status.NamespaceRef.Name,
			},
			Type: dockyardsv1.SecretTypeCredential,
		}

		err = c.Create(ctx, &secret)
		if err != nil {
			t.Fatal(err)
		}

		err = testingutil.RetryUntilFound(ctx, mgr.GetClient(), &secret)
		if err != nil {
			t.Fatal(err)
		}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, u.Path, nil)

		r.SetPathValue("organizationName", organization.Name)

		ctx = middleware.ContextWithSubject(ctx, string(superUser.UID))
		ctx = middleware.ContextWithLogger(ctx, logger)

		h.GetOrganizationCredentials(w, r.Clone(ctx))

		statusCode := w.Result().StatusCode
		if statusCode != http.StatusOK {
			t.Fatalf("expected status code %d, got %d", http.StatusOK, statusCode)
		}

		body, err := io.ReadAll(w.Result().Body)
		if err != nil {
			t.Fatal(err)
		}

		var actual []types.Credential
		err = json.Unmarshal(body, &actual)
		if err != nil {
			t.Fatal(err)
		}

		expected := []types.Credential{
			{
				ID:           string(secret.UID),
				Name:         "test",
				Organization: organization.Name,
			},
		}

		if !cmp.Equal(actual, expected) {
			t.Errorf("diff: %s", cmp.Diff(expected, actual))
		}

		err = c.Delete(ctx, &secret)
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("test several secret types", func(t *testing.T) {
		secrets := []corev1.Secret{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "credential-dockyards-io-credential",
					Namespace: organization.Status.NamespaceRef.Name,
				},
				Type: dockyardsv1.SecretTypeCredential,
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "credential-kubernetes-io-ssh-auth",
					Namespace: organization.Status.NamespaceRef.Name,
				},
				Data: map[string][]byte{
					corev1.SSHAuthPrivateKey: []byte("ssh-privatekey"),
				},
				Type: corev1.SecretTypeSSHAuth,
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "credential-kubernetes-io-tls",
					Namespace: organization.Status.NamespaceRef.Name,
				},
				Data: map[string][]byte{
					corev1.TLSCertKey:       []byte("tls.crt"),
					corev1.TLSPrivateKeyKey: []byte("tls.key"),
				},
				Type: corev1.SecretTypeTLS,
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "credential-opaque",
					Namespace: organization.Status.NamespaceRef.Name,
				},
				Type: corev1.SecretTypeOpaque,
			},
		}

		expected := []types.Credential{}

		for _, secret := range secrets {
			err := c.Create(ctx, &secret)
			if err != nil {
				t.Fatal(err)
			}

			if secret.Type != dockyardsv1.SecretTypeCredential {
				continue
			}

			expected = append(expected, types.Credential{
				ID:           string(secret.UID),
				Name:         strings.TrimPrefix(secret.Name, "credential-"),
				Organization: organization.Name,
			})
		}

		err = testingutil.RetryUntilFound(ctx, mgr.GetClient(), &secrets[len(secrets)-1])
		if err != nil {
			t.Fatal(err)
		}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, u.Path, nil)

		r.SetPathValue("organizationName", organization.Name)

		ctx = middleware.ContextWithSubject(ctx, string(superUser.UID))
		ctx = middleware.ContextWithLogger(ctx, logger)

		h.GetOrganizationCredentials(w, r.Clone(ctx))

		statusCode := w.Result().StatusCode
		if statusCode != http.StatusOK {
			t.Fatalf("expected status code %d, got %d", http.StatusOK, statusCode)
		}

		body, err := io.ReadAll(w.Result().Body)
		if err != nil {
			t.Fatal(err)
		}

		var actual []types.Credential
		err = json.Unmarshal(body, &actual)
		if err != nil {
			t.Fatal(err)
		}

		if !cmp.Equal(actual, expected) {
			t.Errorf("diff: %s", cmp.Diff(expected, actual))
		}

		for _, secret := range secrets {
			err := c.Delete(ctx, &secret)
			if err != nil {
				t.Fatal(err)
			}
		}
	})

	t.Run("test secret from credential template", func(t *testing.T) {
		secret := corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "credential-test",
				Namespace: organization.Status.NamespaceRef.Name,
				Labels: map[string]string{
					dockyardsv1.LabelCredentialTemplateName: "test",
				},
			},
			Type: dockyardsv1.SecretTypeCredential,
		}

		err = c.Create(ctx, &secret)
		if err != nil {
			t.Fatal(err)
		}

		err = testingutil.RetryUntilFound(ctx, mgr.GetClient(), &secret)
		if err != nil {
			t.Fatal(err)
		}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, u.Path, nil)

		r.SetPathValue("organizationName", organization.Name)

		ctx = middleware.ContextWithSubject(ctx, string(superUser.UID))
		ctx = middleware.ContextWithLogger(ctx, logger)

		h.GetOrganizationCredentials(w, r.Clone(ctx))

		statusCode := w.Result().StatusCode
		if statusCode != http.StatusOK {
			t.Fatalf("expected status code %d, got %d", http.StatusOK, statusCode)
		}

		body, err := io.ReadAll(w.Result().Body)
		if err != nil {
			t.Fatal(err)
		}

		var actual []types.Credential
		err = json.Unmarshal(body, &actual)
		if err != nil {
			t.Fatal(err)
		}

		expected := []types.Credential{
			{
				ID:                 string(secret.UID),
				Name:               "test",
				Organization:       organization.Name,
				CredentialTemplate: ptr.To("test"),
			},
		}

		if !cmp.Equal(actual, expected) {
			t.Errorf("diff: %s", cmp.Diff(expected, actual))
		}

		err = c.Delete(ctx, &secret)
		if err != nil {
			t.Fatal(err)
		}
	})
}

func TestCredential_PutOrganizationCredential(t *testing.T) {
	if os.Getenv("KUBEBUILDER_ASSETS") == "" {
		t.Skip("no kubebuilder assets configured")
	}

	ctx, cancel := context.WithCancel(context.TODO())

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

	testEnvironment, err := testingutil.NewTestEnvironment(ctx, []string{path.Join("../../../../config/crd")})
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		cancel()
		testEnvironment.GetEnvironment().Stop()
	})

	mgr := testEnvironment.GetManager()

	go func() {
		err := mgr.Start(ctx)
		if err != nil {
			panic(err)
		}
	}()

	organization := testEnvironment.GetOrganization()

	c := testEnvironment.GetClient()

	superUser := testEnvironment.GetSuperUser()
	reader := testEnvironment.GetReader()

	h := handler{
		Client: mgr.GetClient(),
	}

	t.Run("test update as reader", func(t *testing.T) {
		credentialName := "test-update-reader"

		secret := corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "credential-" + credentialName,
				Namespace: organization.Status.NamespaceRef.Name,
			},
			Type: dockyardsv1.SecretTypeCredential,
		}

		err := c.Create(ctx, &secret)
		if err != nil {
			t.Fatal(err)
		}

		err = testingutil.RetryUntilFound(ctx, mgr.GetClient(), &secret)
		if err != nil {
			t.Fatal(err)
		}

		u := url.URL{
			Path: path.Join("/v1/organizations", organization.Name, "credentials", credentialName),
		}

		update := types.Credential{
			Data: &map[string][]byte{
				"test": []byte("secret"),
			},
		}

		b, err := json.Marshal(update)
		if err != nil {
			t.Fatal(err)
		}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, u.Path, bytes.NewBuffer(b))

		r.SetPathValue("organizationName", organization.Name)
		r.SetPathValue("credentialName", credentialName)

		ctx = middleware.ContextWithSubject(ctx, string(reader.UID))
		ctx = middleware.ContextWithLogger(ctx, logger)

		h.PutOrganizationCredential(w, r.Clone(ctx))

		if w.Result().StatusCode != http.StatusUnauthorized {
			t.Fatalf("expected status code %d, got %d", http.StatusUnauthorized, w.Result().StatusCode)
		}
	})

	t.Run("test update empty credential", func(t *testing.T) {
		credentialName := "test-update-empty"

		secret := corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "credential-" + credentialName,
				Namespace: organization.Status.NamespaceRef.Name,
			},
			Type: dockyardsv1.SecretTypeCredential,
		}

		err := c.Create(ctx, &secret)
		if err != nil {
			t.Fatal(err)
		}

		err = testingutil.RetryUntilFound(ctx, mgr.GetClient(), &secret)
		if err != nil {
			t.Fatal(err)
		}

		u := url.URL{
			Path: path.Join("/v1/organizations", organization.Name, "credentials", credentialName),
		}

		update := types.Credential{
			Data: &map[string][]byte{
				"test": []byte("secret"),
			},
		}

		b, err := json.Marshal(update)
		if err != nil {
			t.Fatal(err)
		}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, u.Path, bytes.NewBuffer(b))

		r.SetPathValue("organizationName", organization.Name)
		r.SetPathValue("credentialName", credentialName)

		ctx = middleware.ContextWithSubject(ctx, string(superUser.UID))
		ctx = middleware.ContextWithLogger(ctx, logger)

		h.PutOrganizationCredential(w, r.Clone(ctx))

		if w.Result().StatusCode != http.StatusNoContent {
			t.Fatalf("expected status code %d, got %d", http.StatusNoContent, w.Result().StatusCode)
		}

		expected := map[string][]byte{
			"test": []byte("secret"),
		}

		var actual corev1.Secret
		err = c.Get(ctx, client.ObjectKeyFromObject(&secret), &actual)
		if err != nil {
			t.Fatal(err)
		}

		if !cmp.Equal(actual.Data, expected) {
			t.Errorf("diff: %s", cmp.Diff(expected, actual.Data))
		}
	})

	t.Run("test update existing key", func(t *testing.T) {
		credentialName := "test-update-existing-key"

		secret := corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "credential-" + credentialName,
				Namespace: organization.Status.NamespaceRef.Name,
			},
			Data: map[string][]byte{
				"test": []byte("qwfp"),
				"hjkl": []byte("arst"),
				"zxcv": []byte("neio"),
			},
			Type: dockyardsv1.SecretTypeCredential,
		}

		err := c.Create(ctx, &secret)
		if err != nil {
			t.Fatal(err)
		}

		err = testingutil.RetryUntilFound(ctx, mgr.GetClient(), &secret)
		if err != nil {
			t.Fatal(err)
		}

		u := url.URL{
			Path: path.Join("/v1/organizations", organization.Name, "credentials", credentialName),
		}

		update := types.Credential{
			Data: &map[string][]byte{
				"test": []byte("secret"),
			},
		}

		b, err := json.Marshal(update)
		if err != nil {
			t.Fatal(err)
		}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, u.Path, bytes.NewBuffer(b))

		r.SetPathValue("organizationName", organization.Name)
		r.SetPathValue("credentialName", credentialName)

		ctx = middleware.ContextWithSubject(ctx, string(superUser.UID))
		ctx = middleware.ContextWithLogger(ctx, logger)

		h.PutOrganizationCredential(w, r.Clone(ctx))

		if w.Result().StatusCode != http.StatusNoContent {
			t.Fatalf("expected status code %d, got %d", http.StatusNoContent, w.Result().StatusCode)
		}

		expected := map[string][]byte{
			"test": []byte("secret"),
			"hjkl": []byte("arst"),
			"zxcv": []byte("neio"),
		}

		var actual corev1.Secret
		err = c.Get(ctx, client.ObjectKeyFromObject(&secret), &actual)
		if err != nil {
			t.Fatal(err)
		}

		if !cmp.Equal(expected, actual.Data) {
			t.Errorf("diff: %s", cmp.Diff(actual.Data, expected))
		}
	})

	t.Run("test remove existing key", func(t *testing.T) {
		credentialName := "test-remove-existing-key"

		secret := corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "credential-" + credentialName,
				Namespace: organization.Status.NamespaceRef.Name,
			},
			Data: map[string][]byte{
				"test": []byte("secret"),
				"arst": []byte("qwfp"),
			},
			Type: dockyardsv1.SecretTypeCredential,
		}

		err := c.Create(ctx, &secret)
		if err != nil {
			t.Fatal(err)
		}

		err = testingutil.RetryUntilFound(ctx, mgr.GetClient(), &secret)
		if err != nil {
			t.Fatal(err)
		}

		u := url.URL{
			Path: path.Join("/v1/organizations", organization.Name, "credentials", credentialName),
		}

		update := types.Credential{
			Data: &map[string][]byte{
				"test": nil,
			},
		}

		b, err := json.Marshal(update)
		if err != nil {
			t.Fatal(err)
		}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, u.Path, bytes.NewBuffer(b))

		r.SetPathValue("organizationName", organization.Name)
		r.SetPathValue("credentialName", credentialName)

		ctx = middleware.ContextWithSubject(ctx, string(superUser.UID))
		ctx = middleware.ContextWithLogger(ctx, logger)

		h.PutOrganizationCredential(w, r.Clone(ctx))

		if w.Result().StatusCode != http.StatusNoContent {
			t.Fatalf("expected status code %d, got %d", http.StatusNoContent, w.Result().StatusCode)
		}

		expected := map[string][]byte{
			"arst": []byte("qwfp"),
		}

		var actual corev1.Secret
		err = c.Get(ctx, client.ObjectKeyFromObject(&secret), &actual)
		if err != nil {
			t.Fatal(err)
		}

		if !cmp.Equal(expected, actual.Data) {
			t.Errorf("diff: %s", cmp.Diff(actual.Data, expected))
		}
	})

	t.Run("test update existing key to empty string", func(t *testing.T) {
		credentialName := "test-update-empty-string"

		secret := corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "credential-" + credentialName,
				Namespace: organization.Status.NamespaceRef.Name,
			},
			Data: map[string][]byte{
				"test": []byte("secret"),
				"zxcv": []byte("neio"),
			},
			Type: dockyardsv1.SecretTypeCredential,
		}

		err := c.Create(ctx, &secret)
		if err != nil {
			t.Fatal(err)
		}

		err = testingutil.RetryUntilFound(ctx, mgr.GetClient(), &secret)
		if err != nil {
			t.Fatal(err)
		}

		u := url.URL{
			Path: path.Join("/v1/organizations", organization.Name, "credentials", credentialName),
		}

		update := types.Credential{
			Data: &map[string][]byte{
				"test": []byte(""),
			},
		}

		b, err := json.Marshal(update)
		if err != nil {
			t.Fatal(err)
		}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, u.Path, bytes.NewBuffer(b))

		r.SetPathValue("organizationName", organization.Name)
		r.SetPathValue("credentialName", credentialName)

		ctx = middleware.ContextWithSubject(ctx, string(superUser.UID))
		ctx = middleware.ContextWithLogger(ctx, logger)

		h.PutOrganizationCredential(w, r.Clone(ctx))

		if w.Result().StatusCode != http.StatusNoContent {
			t.Fatalf("expected status code %d, got %d", http.StatusNoContent, w.Result().StatusCode)
		}

		expected := map[string][]byte{
			"test": []byte(""),
			"zxcv": []byte("neio"),
		}

		var actual corev1.Secret
		err = c.Get(ctx, client.ObjectKeyFromObject(&secret), &actual)
		if err != nil {
			t.Fatal(err)
		}

		if !cmp.Equal(expected, actual.Data) {
			t.Errorf("diff: %s", cmp.Diff(actual.Data, expected))
		}
	})
}

func TestOrganizationCredentials_Create(t *testing.T) {
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

	go func() {
		err := mgr.Start(ctx)
		if err != nil {
			panic(err)
		}
	}()

	organization := testEnvironment.GetOrganization()

	c := testEnvironment.GetClient()

	superUser := testEnvironment.GetSuperUser()
	reader := testEnvironment.GetReader()

	h := handler{
		Client: mgr.GetClient(),
	}

	handlerFunc := CreateOrganizationResource(&h, "clusters", h.CreateOrganizationCredential)

	u := url.URL{
		Path: path.Join("/v1/organizations", organization.Name, "credentials"),
	}

	mgr.GetCache().WaitForCacheSync(ctx)

	t.Run("test create empty credential", func(t *testing.T) {
		credential := types.Credential{
			Name: "test-create-empty",
			Data: &map[string][]byte{},
		}

		b, err := json.Marshal(credential)
		if err != nil {
			t.Fatal(err)
		}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, u.Path, bytes.NewBuffer(b))

		r.SetPathValue("organizationName", organization.Name)

		ctx = middleware.ContextWithSubject(ctx, string(superUser.UID))
		ctx = middleware.ContextWithLogger(ctx, logger)

		handlerFunc(w, r.Clone(ctx))

		if w.Result().StatusCode != http.StatusCreated {
			t.Fatalf("expected status code %d, got %d", http.StatusCreated, w.Result().StatusCode)
		}

		objectKey := client.ObjectKey{
			Name:      "credential-" + credential.Name,
			Namespace: organization.Status.NamespaceRef.Name,
		}

		expected := map[string][]byte(nil)

		var actual corev1.Secret
		err = c.Get(ctx, objectKey, &actual)
		if err != nil {
			t.Fatal(err)
		}

		if !cmp.Equal(actual.Data, expected) {
			t.Errorf("diff: %s", cmp.Diff(expected, actual.Data))
		}
	})

	t.Run("test single key", func(t *testing.T) {
		credential := types.Credential{
			Name: "test-create-empty-credential",
			Data: &map[string][]byte{
				"test": []byte("secret"),
			},
		}

		b, err := json.Marshal(credential)
		if err != nil {
			t.Fatal(err)
		}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, u.Path, bytes.NewBuffer(b))

		r.SetPathValue("organizationName", organization.Name)

		ctx = middleware.ContextWithSubject(ctx, string(superUser.UID))
		ctx = middleware.ContextWithLogger(ctx, logger)

		handlerFunc(w, r.Clone(ctx))

		if w.Result().StatusCode != http.StatusCreated {
			t.Fatalf("expected status code %d, got %d", http.StatusCreated, w.Result().StatusCode)
		}

		objectKey := client.ObjectKey{
			Name:      "credential-" + credential.Name,
			Namespace: organization.Status.NamespaceRef.Name,
		}

		var actual corev1.Secret
		err = c.Get(ctx, objectKey, &actual)
		if err != nil {
			t.Fatal(err)
		}

		expected := map[string][]byte{
			"test": []byte("secret"),
		}

		if !cmp.Equal(actual.Data, expected) {
			t.Errorf("diff: %s", cmp.Diff(expected, actual.Data))
		}
	})

	t.Run("test multiple keys", func(t *testing.T) {
		credential := types.Credential{
			Name: "test-credential-with-multiple-keys",
			Data: &map[string][]byte{
				"qwfp": []byte("arst"),
				"zxcv": []byte("neio"),
				"hjkl": []byte("wars"),
			},
		}

		b, err := json.Marshal(credential)
		if err != nil {
			t.Fatal(err)
		}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, u.Path, bytes.NewBuffer(b))

		r.SetPathValue("organizationName", organization.Name)

		ctx = middleware.ContextWithSubject(ctx, string(superUser.UID))
		ctx = middleware.ContextWithLogger(ctx, logger)

		handlerFunc(w, r.Clone(ctx))

		if w.Result().StatusCode != http.StatusCreated {
			t.Fatalf("expected status code %d, got %d", http.StatusCreated, w.Result().StatusCode)
		}

		objectKey := client.ObjectKey{
			Name:      "credential-" + credential.Name,
			Namespace: organization.Status.NamespaceRef.Name,
		}

		var actual corev1.Secret
		err = c.Get(ctx, objectKey, &actual)
		if err != nil {
			t.Fatal(err)
		}

		expected := map[string][]byte{
			"qwfp": []byte("arst"),
			"zxcv": []byte("neio"),
			"hjkl": []byte("wars"),
		}

		if !cmp.Equal(actual.Data, expected) {
			t.Errorf("diff: %s", cmp.Diff(expected, actual.Data))
		}
	})

	t.Run("test as reader", func(t *testing.T) {
		credential := types.Credential{
			Name: "test-reader",
			Data: &map[string][]byte{
				"test": []byte("secret"),
			},
		}

		b, err := json.Marshal(credential)
		if err != nil {
			t.Fatal(err)
		}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, u.Path, bytes.NewBuffer(b))

		r.SetPathValue("organizationName", organization.Name)

		ctx = middleware.ContextWithSubject(ctx, string(reader.UID))
		ctx = middleware.ContextWithLogger(ctx, logger)

		handlerFunc(w, r.Clone(ctx))

		if w.Result().StatusCode != http.StatusUnauthorized {
			t.Fatalf("expected status code %d, got %d", http.StatusUnauthorized, w.Result().StatusCode)
		}
	})
}

func TestCredential_GetOrganizationCredential(t *testing.T) {
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

	organization := testEnvironment.GetOrganization()

	c := testEnvironment.GetClient()

	superUser := testEnvironment.GetSuperUser()

	h := handler{
		Client:    mgr.GetClient(),
		namespace: testEnvironment.GetDockyardsNamespace(),
	}

	t.Run("test empty", func(t *testing.T) {
		credentialName := "test-empty"

		secret := corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "credential-" + credentialName,
				Namespace: organization.Status.NamespaceRef.Name,
			},
			Type: dockyardsv1.SecretTypeCredential,
		}

		err := c.Create(ctx, &secret)
		if err != nil {
			t.Fatal(err)
		}

		err = testingutil.RetryUntilFound(ctx, mgr.GetClient(), &secret)
		if err != nil {
			t.Fatal(err)
		}

		u := url.URL{
			Path: path.Join("/v1/organizations", organization.Name, "credentials", credentialName),
		}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, u.Path, nil)

		r.SetPathValue("organizationName", organization.Name)
		r.SetPathValue("credentialName", credentialName)

		ctx = middleware.ContextWithSubject(ctx, string(superUser.UID))
		ctx = middleware.ContextWithLogger(ctx, logger)

		h.GetOrganizationCredential(w, r.Clone(ctx))

		if w.Result().StatusCode != http.StatusOK {
			t.Fatalf("expected status code %d, got %d", http.StatusOK, w.Result().StatusCode)
		}

		body, err := io.ReadAll(w.Result().Body)
		if err != nil {
			t.Fatal(err)
		}

		var actual types.Credential
		err = json.Unmarshal(body, &actual)
		if err != nil {
			t.Fatal(err)
		}

		expected := types.Credential{
			ID:           string(secret.UID),
			Name:         credentialName,
			Organization: organization.Name,
		}

		if !cmp.Equal(actual, expected) {
			t.Errorf("diff: %s", cmp.Diff(expected, actual))
		}
	})

	t.Run("test multiple keys", func(t *testing.T) {
		credentialName := "test-multiple-keys"

		secret := corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "credential-" + credentialName,
				Namespace: organization.Status.NamespaceRef.Name,
			},
			Data: map[string][]byte{
				"qwfp": []byte("arst"),
				"zxcv": []byte("neio"),
				"hjkl": []byte("wars"),
			},
			Type: dockyardsv1.SecretTypeCredential,
		}

		err := c.Create(ctx, &secret)
		if err != nil {
			t.Fatal(err)
		}

		err = testingutil.RetryUntilFound(ctx, mgr.GetClient(), &secret)
		if err != nil {
			t.Fatal(err)
		}

		u := url.URL{
			Path: path.Join("/v1/organizations", organization.Name, "credentials", credentialName),
		}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, u.Path, nil)

		r.SetPathValue("organizationName", organization.Name)
		r.SetPathValue("credentialName", credentialName)

		ctx = middleware.ContextWithSubject(ctx, string(superUser.UID))
		ctx = middleware.ContextWithLogger(ctx, logger)

		h.GetOrganizationCredential(w, r.Clone(ctx))

		if w.Result().StatusCode != http.StatusOK {
			t.Fatalf("expected status code %d, got %d", http.StatusOK, w.Result().StatusCode)
		}

		body, err := io.ReadAll(w.Result().Body)
		if err != nil {
			t.Fatal(err)
		}

		var actual types.Credential
		err = json.Unmarshal(body, &actual)
		if err != nil {
			t.Fatal(err)
		}

		expected := types.Credential{
			ID:           string(secret.UID),
			Name:         credentialName,
			Organization: organization.Name,
			Data: &map[string][]byte{
				"qwfp": nil,
				"zxcv": nil,
				"hjkl": nil,
			},
		}

		if !cmp.Equal(actual, expected) {
			t.Errorf("diff: %s", cmp.Diff(expected, actual))
		}
	})

	t.Run("test credential template", func(t *testing.T) {
		credentialTemplate := dockyardsv1.CredentialTemplate{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test",
				Namespace: testEnvironment.GetDockyardsNamespace(),
			},
			Spec: dockyardsv1.CredentialTemplateSpec{
				Options: []dockyardsv1.CredentialOption{
					{
						Key:       "qwfp",
						Plaintext: true,
					},
					{
						Key: "test",
					},
				},
			},
		}

		err := c.Create(ctx, &credentialTemplate)
		if err != nil {
			t.Fatal(err)
		}

		credentialName := "test-credential-template"

		secret := corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "credential-" + credentialName,
				Namespace: organization.Status.NamespaceRef.Name,
				Labels: map[string]string{
					dockyardsv1.LabelCredentialTemplateName: credentialTemplate.Name,
				},
			},
			Data: map[string][]byte{
				"qwfp": []byte("arst"),
				"test": []byte("secret"),
			},
			Type: dockyardsv1.SecretTypeCredential,
		}

		err = c.Create(ctx, &secret)
		if err != nil {
			t.Fatal(err)
		}

		err = testingutil.RetryUntilFound(ctx, mgr.GetClient(), &secret)
		if err != nil {
			t.Fatal(err)
		}

		u := url.URL{
			Path: path.Join("/v1/organizations", organization.Name, "credentials", credentialName),
		}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, u.Path, nil)

		r.SetPathValue("organizationName", organization.Name)
		r.SetPathValue("credentialName", credentialName)

		ctx = middleware.ContextWithSubject(ctx, string(superUser.UID))
		ctx = middleware.ContextWithLogger(ctx, logger)

		h.GetOrganizationCredential(w, r.Clone(ctx))

		if w.Result().StatusCode != http.StatusOK {
			t.Fatalf("expected status code %d, got %d", http.StatusOK, w.Result().StatusCode)
		}

		body, err := io.ReadAll(w.Result().Body)
		if err != nil {
			t.Fatal(err)
		}

		var actual types.Credential
		err = json.Unmarshal(body, &actual)
		if err != nil {
			t.Fatal(err)
		}

		expected := types.Credential{
			ID:                 string(secret.UID),
			Name:               credentialName,
			Organization:       organization.Name,
			CredentialTemplate: ptr.To(credentialTemplate.Name),
			Data: &map[string][]byte{
				"qwfp": []byte("arst"),
				"test": nil,
			},
		}

		if !cmp.Equal(actual, expected) {
			t.Errorf("diff: %s", cmp.Diff(expected, actual))
		}
	})
}

func TestCredential_DeleteOrganizationCredential(t *testing.T) {
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

	organization := testEnvironment.GetOrganization()

	c := testEnvironment.GetClient()

	superUser := testEnvironment.GetSuperUser()
	user := testEnvironment.GetUser()
	reader := testEnvironment.GetReader()

	h := handler{
		Client:    mgr.GetClient(),
		namespace: testEnvironment.GetDockyardsNamespace(),
	}

	handlerFunc := DeleteOrganizationResource(&h, "clusters", h.DeleteOrganizationCredential)

	t.Run("test as super user", func(t *testing.T) {
		credentialName := "test-super-user"

		secret := corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "credential-" + credentialName,
				Namespace: organization.Status.NamespaceRef.Name,
			},
			Type: dockyardsv1.SecretTypeCredential,
		}

		err := c.Create(ctx, &secret)
		if err != nil {
			t.Fatal(err)
		}

		err = testingutil.RetryUntilFound(ctx, mgr.GetClient(), &secret)
		if err != nil {
			t.Fatal(err)
		}

		u := url.URL{
			Path: path.Join("/v1/organizations", organization.Name, "credentials", credentialName),
		}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodDelete, u.Path, nil)

		r.SetPathValue("organizationName", organization.Name)
		r.SetPathValue("resourceName", credentialName)

		ctx = middleware.ContextWithSubject(ctx, string(superUser.UID))
		ctx = middleware.ContextWithLogger(ctx, logger)

		handlerFunc(w, r.Clone(ctx))

		if w.Result().StatusCode != http.StatusAccepted {
			t.Fatalf("expected status code %d, got %d", http.StatusAccepted, w.Result().StatusCode)
		}
	})

	t.Run("test as user", func(t *testing.T) {
		credentialName := "test-user"

		secret := corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "credential-" + credentialName,
				Namespace: organization.Status.NamespaceRef.Name,
			},
			Type: dockyardsv1.SecretTypeCredential,
		}

		err := c.Create(ctx, &secret)
		if err != nil {
			t.Fatal(err)
		}

		err = testingutil.RetryUntilFound(ctx, mgr.GetClient(), &secret)
		if err != nil {
			t.Fatal(err)
		}

		u := url.URL{
			Path: path.Join("/v1/organizations", organization.Name, "credentials", credentialName),
		}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodDelete, u.Path, nil)

		r.SetPathValue("organizationName", organization.Name)
		r.SetPathValue("resourceName", credentialName)

		ctx = middleware.ContextWithSubject(ctx, string(user.UID))
		ctx = middleware.ContextWithLogger(ctx, logger)

		handlerFunc(w, r.Clone(ctx))

		if w.Result().StatusCode != http.StatusAccepted {
			t.Fatalf("expected status code %d, got %d", http.StatusAccepted, w.Result().StatusCode)
		}
	})

	t.Run("test as reader", func(t *testing.T) {
		credentialName := "test-user"

		secret := corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "credential-" + credentialName,
				Namespace: organization.Status.NamespaceRef.Name,
			},
			Type: dockyardsv1.SecretTypeCredential,
		}

		err := c.Create(ctx, &secret)
		if err != nil {
			t.Fatal(err)
		}

		err = testingutil.RetryUntilFound(ctx, mgr.GetClient(), &secret)
		if err != nil {
			t.Fatal(err)
		}

		u := url.URL{
			Path: path.Join("/v1/organizations", organization.Name, "credentials", credentialName),
		}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodDelete, u.Path, nil)

		r.SetPathValue("organizationName", organization.Name)
		r.SetPathValue("resourceName", credentialName)

		ctx = middleware.ContextWithSubject(ctx, string(reader.UID))
		ctx = middleware.ContextWithLogger(ctx, logger)

		handlerFunc(w, r.Clone(ctx))

		if w.Result().StatusCode != http.StatusUnauthorized {
			t.Fatalf("expected status code %d, got %d", http.StatusUnauthorized, w.Result().StatusCode)
		}
	})

	t.Run("test secret type", func(t *testing.T) {
		credentialName := "test-secret-type"

		secret := corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "credential-" + credentialName,
				Namespace: organization.Status.NamespaceRef.Name,
			},
			Type: corev1.SecretTypeOpaque,
		}

		err := c.Create(ctx, &secret)
		if err != nil {
			t.Fatal(err)
		}

		err = testingutil.RetryUntilFound(ctx, mgr.GetClient(), &secret)
		if err != nil {
			t.Fatal(err)
		}

		u := url.URL{
			Path: path.Join("/v1/organizations", organization.Name, "credentials", credentialName),
		}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodDelete, u.Path, nil)

		r.SetPathValue("organizationName", organization.Name)
		r.SetPathValue("resourceName", credentialName)

		ctx = middleware.ContextWithSubject(ctx, string(superUser.UID))
		ctx = middleware.ContextWithLogger(ctx, logger)

		handlerFunc(w, r.Clone(ctx))

		if w.Result().StatusCode != http.StatusInternalServerError {
			t.Fatalf("expected status code %d, got %d", http.StatusInternalServerError, w.Result().StatusCode)
		}
	})
}
