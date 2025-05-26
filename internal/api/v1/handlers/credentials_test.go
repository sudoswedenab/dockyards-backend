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
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path"
	"strings"
	"testing"

	dockyardsv1 "bitbucket.org/sudosweden/dockyards-backend/pkg/api/v1alpha3"
	"bitbucket.org/sudosweden/dockyards-backend/pkg/testing/testingutil"
	"github.com/google/go-cmp/cmp"
	"github.com/sudoswedenab/dockyards-api/pkg/types"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestOrganizationCredentials_List(t *testing.T) {
	if os.Getenv("KUBEBUILDER_ASSETS") == "" {
		t.Skip("no kubebuilder assets configured")
	}

	mgr := testEnvironment.GetManager()
	c := testEnvironment.GetClient()

	organization := testEnvironment.MustCreateOrganization(t)
	superUser := testEnvironment.MustGetOrganizationUser(t, organization, dockyardsv1.OrganizationMemberRoleSuperUser)
	superUserToken := MustSignToken(t, string(superUser.UID))

	u := url.URL{
		Path: path.Join("/v1/orgs", organization.Name, "credentials"),
	}

	t.Run("test single credential", func(t *testing.T) {
		secret := corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "credential-test",
				Namespace: organization.Spec.NamespaceRef.Name,
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

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, u.Path, nil)

		r.Header.Add("Authorization", "Bearer "+superUserToken)

		mux.ServeHTTP(w, r)

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
				ID:        string(secret.UID),
				Name:      "test",
				CreatedAt: &secret.CreationTimestamp.Time,
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
					Namespace: organization.Spec.NamespaceRef.Name,
				},
				Type: dockyardsv1.SecretTypeCredential,
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "credential-kubernetes-io-ssh-auth",
					Namespace: organization.Spec.NamespaceRef.Name,
				},
				Data: map[string][]byte{
					corev1.SSHAuthPrivateKey: []byte("ssh-privatekey"),
				},
				Type: corev1.SecretTypeSSHAuth,
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "credential-kubernetes-io-tls",
					Namespace: organization.Spec.NamespaceRef.Name,
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
					Namespace: organization.Spec.NamespaceRef.Name,
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
				ID:        string(secret.UID),
				Name:      strings.TrimPrefix(secret.Name, "credential-"),
				CreatedAt: &secret.CreationTimestamp.Time,
			})
		}

		err := testingutil.RetryUntilFound(ctx, mgr.GetClient(), &secrets[len(secrets)-1])
		if err != nil {
			t.Fatal(err)
		}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, u.Path, nil)

		r.Header.Add("Authorization", "Bearer "+superUserToken)

		mux.ServeHTTP(w, r)

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
				Namespace: organization.Spec.NamespaceRef.Name,
				Labels: map[string]string{
					dockyardsv1.LabelCredentialTemplateName: "test",
				},
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

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, u.Path, nil)

		r.Header.Add("Authorization", "Bearer "+superUserToken)

		mux.ServeHTTP(w, r)

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
				ID:                     string(secret.UID),
				Name:                   "test",
				CredentialTemplateName: ptr.To("test"),
				CreatedAt:              &secret.CreationTimestamp.Time,
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

func TestOrganizationCredentials_Update(t *testing.T) {
	if os.Getenv("KUBEBUILDER_ASSETS") == "" {
		t.Skip("no kubebuilder assets configured")
	}

	mgr := testEnvironment.GetManager()
	c := testEnvironment.GetClient()

	organization := testEnvironment.MustCreateOrganization(t)

	superUser := testEnvironment.MustGetOrganizationUser(t, organization, dockyardsv1.OrganizationMemberRoleSuperUser)
	reader := testEnvironment.MustGetOrganizationUser(t, organization, dockyardsv1.OrganizationMemberRoleReader)

	superUserToken := MustSignToken(t, string(superUser.UID))
	readerToken := MustSignToken(t, string(reader.UID))

	t.Run("test update as reader", func(t *testing.T) {
		credentialName := "test-update-reader"

		secret := corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "credential-" + credentialName,
				Namespace: organization.Spec.NamespaceRef.Name,
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
			Path: path.Join("/v1/orgs", organization.Name, "credentials", credentialName),
		}

		update := types.CredentialOptions{
			Data: &map[string][]byte{
				"test": []byte("secret"),
			},
		}

		b, err := json.Marshal(update)
		if err != nil {
			t.Fatal(err)
		}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPatch, u.Path, bytes.NewBuffer(b))

		r.Header.Add("Authorization", "Bearer "+readerToken)

		mux.ServeHTTP(w, r)

		if w.Result().StatusCode != http.StatusUnauthorized {
			t.Fatalf("expected status code %d, got %d", http.StatusUnauthorized, w.Result().StatusCode)
		}
	})

	t.Run("test update empty credential", func(t *testing.T) {
		credentialName := "test-update-empty"

		secret := corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "credential-" + credentialName,
				Namespace: organization.Spec.NamespaceRef.Name,
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
			Path: path.Join("/v1/orgs", organization.Name, "credentials", credentialName),
		}

		update := types.CredentialOptions{
			Data: &map[string][]byte{
				"test": []byte("secret"),
			},
		}

		b, err := json.Marshal(update)
		if err != nil {
			t.Fatal(err)
		}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPatch, u.Path, bytes.NewBuffer(b))

		r.Header.Add("Authorization", "Bearer "+superUserToken)

		mux.ServeHTTP(w, r)

		if w.Result().StatusCode != http.StatusAccepted {
			t.Fatalf("expected status code %d, got %d", http.StatusAccepted, w.Result().StatusCode)
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
				Namespace: organization.Spec.NamespaceRef.Name,
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
			Path: path.Join("/v1/orgs", organization.Name, "credentials", credentialName),
		}

		update := types.CredentialOptions{
			Data: &map[string][]byte{
				"test": []byte("secret"),
			},
		}

		b, err := json.Marshal(update)
		if err != nil {
			t.Fatal(err)
		}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPatch, u.Path, bytes.NewBuffer(b))

		r.Header.Add("Authorization", "Bearer "+superUserToken)

		mux.ServeHTTP(w, r)

		if w.Result().StatusCode != http.StatusAccepted {
			t.Fatalf("expected status code %d, got %d", http.StatusAccepted, w.Result().StatusCode)
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
				Namespace: organization.Spec.NamespaceRef.Name,
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
			Path: path.Join("/v1/orgs", organization.Name, "credentials", credentialName),
		}

		update := types.CredentialOptions{
			Data: &map[string][]byte{
				"test": nil,
			},
		}

		b, err := json.Marshal(update)
		if err != nil {
			t.Fatal(err)
		}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPatch, u.Path, bytes.NewBuffer(b))

		r.Header.Add("Authorization", "Bearer "+superUserToken)

		mux.ServeHTTP(w, r)

		if w.Result().StatusCode != http.StatusAccepted {
			t.Fatalf("expected status code %d, got %d", http.StatusAccepted, w.Result().StatusCode)
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
				Namespace: organization.Spec.NamespaceRef.Name,
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
			Path: path.Join("/v1/orgs", organization.Name, "credentials", credentialName),
		}

		update := types.CredentialOptions{
			Data: &map[string][]byte{
				"test": []byte(""),
			},
		}

		b, err := json.Marshal(update)
		if err != nil {
			t.Fatal(err)
		}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPatch, u.Path, bytes.NewBuffer(b))

		r.Header.Add("Authorization", "Bearer "+superUserToken)

		mux.ServeHTTP(w, r)

		if w.Result().StatusCode != http.StatusAccepted {
			t.Fatalf("expected status code %d, got %d", http.StatusAccepted, w.Result().StatusCode)
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

	t.Run("test update name", func(t *testing.T) {
		credentialName := "test-update-name"

		secret := corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "credential-" + credentialName,
				Namespace: organization.Spec.NamespaceRef.Name,
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
			Path: path.Join("/v1/orgs", organization.Name, "credentials", credentialName),
		}

		update := types.CredentialOptions{
			Name: ptr.To("new-name"),
		}

		b, err := json.Marshal(update)
		if err != nil {
			t.Fatal(err)
		}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPatch, u.Path, bytes.NewBuffer(b))

		r.Header.Add("Authorization", "Bearer "+superUserToken)

		mux.ServeHTTP(w, r)

		if w.Result().StatusCode != http.StatusUnprocessableEntity {
			t.Fatalf("expected status code %d, got %d", http.StatusUnprocessableEntity, w.Result().StatusCode)
		}
	})
}

func TestOrganizationCredentials_Create(t *testing.T) {
	if os.Getenv("KUBEBUILDER_ASSETS") == "" {
		t.Skip("no kubebuilder assets configured")
	}

	mgr := testEnvironment.GetManager()
	c := testEnvironment.GetClient()

	organization := testEnvironment.MustCreateOrganization(t)

	superUser := testEnvironment.MustGetOrganizationUser(t, organization, dockyardsv1.OrganizationMemberRoleSuperUser)
	reader := testEnvironment.MustGetOrganizationUser(t, organization, dockyardsv1.OrganizationMemberRoleReader)

	superUserToken := MustSignToken(t, string(superUser.UID))
	readerToken := MustSignToken(t, string(reader.UID))

	u := url.URL{
		Path: path.Join("/v1/orgs", organization.Name, "credentials"),
	}

	mgr.GetCache().WaitForCacheSync(ctx)

	t.Run("test create empty credential", func(t *testing.T) {
		credential := types.CredentialOptions{
			Name: ptr.To("test-create-empty"),
			Data: &map[string][]byte{},
		}

		b, err := json.Marshal(credential)
		if err != nil {
			t.Fatal(err)
		}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, u.Path, bytes.NewBuffer(b))

		r.Header.Add("Authorization", "Bearer "+superUserToken)

		mux.ServeHTTP(w, r)

		if w.Result().StatusCode != http.StatusCreated {
			t.Fatalf("expected status code %d, got %d", http.StatusCreated, w.Result().StatusCode)
		}

		objectKey := client.ObjectKey{
			Name:      "credential-" + *credential.Name,
			Namespace: organization.Spec.NamespaceRef.Name,
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
		credential := types.CredentialOptions{
			Name: ptr.To("test-create-empty-credential"),
			Data: &map[string][]byte{
				"test": []byte("secret"),
			},
		}

		b, err := json.Marshal(credential)
		if err != nil {
			t.Fatal(err)
		}

		t.Logf("b: %s", b)

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, u.Path, bytes.NewBuffer(b))

		r.Header.Add("Authorization", "Bearer "+superUserToken)

		mux.ServeHTTP(w, r)

		if w.Result().StatusCode != http.StatusCreated {
			t.Fatalf("expected status code %d, got %d", http.StatusCreated, w.Result().StatusCode)
		}

		objectKey := client.ObjectKey{
			Name:      "credential-" + *credential.Name,
			Namespace: organization.Spec.NamespaceRef.Name,
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
		credential := types.CredentialOptions{
			Name: ptr.To("test-credential-with-multiple-keys"),
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

		r.Header.Add("Authorization", "Bearer "+superUserToken)

		mux.ServeHTTP(w, r)

		if w.Result().StatusCode != http.StatusCreated {
			t.Fatalf("expected status code %d, got %d", http.StatusCreated, w.Result().StatusCode)
		}

		objectKey := client.ObjectKey{
			Name:      "credential-" + *credential.Name,
			Namespace: organization.Spec.NamespaceRef.Name,
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
		credential := types.CredentialOptions{
			Name: ptr.To("test-reader"),
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

		r.Header.Add("Authorization", "Bearer "+readerToken)

		mux.ServeHTTP(w, r)

		if w.Result().StatusCode != http.StatusUnauthorized {
			t.Fatalf("expected status code %d, got %d", http.StatusUnauthorized, w.Result().StatusCode)
		}
	})

	t.Run("test credential template name", func(t *testing.T) {
		options := types.CredentialOptions{
			Name:                   ptr.To("test-credential-template-name"),
			CredentialTemplateName: ptr.To("testing"),
		}

		b, err := json.Marshal(&options)
		if err != nil {
			t.Fatal(err)
		}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, u.Path, bytes.NewBuffer(b))

		r.Header.Add("Authorization", "Bearer "+superUserToken)

		mux.ServeHTTP(w, r)

		if w.Result().StatusCode != http.StatusCreated {
			t.Fatalf("expected status code %d, got %d", http.StatusCreated, w.Result().StatusCode)
		}

		objectKey := client.ObjectKey{
			Name:      "credential-" + *options.Name,
			Namespace: organization.Spec.NamespaceRef.Name,
		}

		var actual corev1.Secret
		err = c.Get(ctx, objectKey, &actual)
		if err != nil {
			t.Fatal(err)
		}

		expected := corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Labels: map[string]string{
					dockyardsv1.LabelCredentialTemplateName: *options.CredentialTemplateName,
				},
				//
				CreationTimestamp: actual.CreationTimestamp,
				ManagedFields:     actual.ManagedFields,
				Name:              actual.Name,
				Namespace:         actual.Namespace,
				OwnerReferences:   actual.OwnerReferences,
				ResourceVersion:   actual.ResourceVersion,
				UID:               actual.UID,
			},
			Type: dockyardsv1.SecretTypeCredential,
		}

		if !cmp.Equal(actual, expected) {
			t.Errorf("diff: %s", cmp.Diff(expected, actual))
		}
	})
}

func TestOrganizationCredentials_Get(t *testing.T) {
	if os.Getenv("KUBEBUILDER_ASSETS") == "" {
		t.Skip("no kubebuilder assets configured")
	}

	mgr := testEnvironment.GetManager()
	c := testEnvironment.GetClient()

	organization := testEnvironment.MustCreateOrganization(t)
	superUser := testEnvironment.MustGetOrganizationUser(t, organization, dockyardsv1.OrganizationMemberRoleSuperUser)
	superUserToken := MustSignToken(t, string(superUser.UID))

	t.Run("test empty", func(t *testing.T) {
		credentialName := "test-empty"

		secret := corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "credential-" + credentialName,
				Namespace: organization.Spec.NamespaceRef.Name,
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
			Path: path.Join("/v1/orgs", organization.Name, "credentials", credentialName),
		}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, u.Path, nil)

		r.Header.Add("Authorization", "Bearer "+superUserToken)

		mux.ServeHTTP(w, r)

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
			ID:        string(secret.UID),
			Name:      credentialName,
			CreatedAt: &secret.CreationTimestamp.Time,
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
				Namespace: organization.Spec.NamespaceRef.Name,
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
			Path: path.Join("/v1/orgs", organization.Name, "credentials", credentialName),
		}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, u.Path, nil)

		r.Header.Add("Authorization", "Bearer "+superUserToken)

		mux.ServeHTTP(w, r)

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
			ID:   string(secret.UID),
			Name: credentialName,
			Data: &map[string][]byte{
				"qwfp": nil,
				"zxcv": nil,
				"hjkl": nil,
			},
			CreatedAt: &secret.CreationTimestamp.Time,
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
				Namespace: organization.Spec.NamespaceRef.Name,
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
			Path: path.Join("/v1/orgs", organization.Name, "credentials", credentialName),
		}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, u.Path, nil)

		r.Header.Add("Authorization", "Bearer "+superUserToken)

		mux.ServeHTTP(w, r)

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
			ID:                     string(secret.UID),
			Name:                   credentialName,
			CredentialTemplateName: ptr.To(credentialTemplate.Name),
			Data: &map[string][]byte{
				"qwfp": []byte("arst"),
				"test": nil,
			},
			CreatedAt: &secret.CreationTimestamp.Time,
		}

		if !cmp.Equal(actual, expected) {
			t.Errorf("diff: %s", cmp.Diff(expected, actual))
		}
	})
}

func TestOrganizationCredentials_Delete(t *testing.T) {
	if os.Getenv("KUBEBUILDER_ASSETS") == "" {
		t.Skip("no kubebuilder assets configured")
	}

	mgr := testEnvironment.GetManager()
	c := testEnvironment.GetClient()

	organization := testEnvironment.MustCreateOrganization(t)

	superUser := testEnvironment.MustGetOrganizationUser(t, organization, dockyardsv1.OrganizationMemberRoleSuperUser)
	user := testEnvironment.MustGetOrganizationUser(t, organization, dockyardsv1.OrganizationMemberRoleUser)
	reader := testEnvironment.MustGetOrganizationUser(t, organization, dockyardsv1.OrganizationMemberRoleReader)

	superUserToken := MustSignToken(t, string(superUser.UID))
	userToken := MustSignToken(t, string(user.UID))
	readerToken := MustSignToken(t, string(reader.UID))

	t.Run("test as super user", func(t *testing.T) {
		credentialName := "test-super-user"

		secret := corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "credential-" + credentialName,
				Namespace: organization.Spec.NamespaceRef.Name,
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
			Path: path.Join("/v1/orgs", organization.Name, "credentials", credentialName),
		}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodDelete, u.Path, nil)

		r.Header.Add("Authorization", "Bearer "+superUserToken)

		mux.ServeHTTP(w, r)

		if w.Result().StatusCode != http.StatusAccepted {
			t.Fatalf("expected status code %d, got %d", http.StatusAccepted, w.Result().StatusCode)
		}
	})

	t.Run("test as user", func(t *testing.T) {
		credentialName := "test-user"

		secret := corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "credential-" + credentialName,
				Namespace: organization.Spec.NamespaceRef.Name,
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
			Path: path.Join("/v1/orgs", organization.Name, "credentials", credentialName),
		}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodDelete, u.Path, nil)

		r.Header.Add("Authorization", "Bearer "+userToken)

		mux.ServeHTTP(w, r)

		if w.Result().StatusCode != http.StatusAccepted {
			t.Fatalf("expected status code %d, got %d", http.StatusAccepted, w.Result().StatusCode)
		}
	})

	t.Run("test as reader", func(t *testing.T) {
		credentialName := "test-user"

		secret := corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "credential-" + credentialName,
				Namespace: organization.Spec.NamespaceRef.Name,
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
			Path: path.Join("/v1/orgs", organization.Name, "credentials", credentialName),
		}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodDelete, u.Path, nil)

		r.Header.Add("Authorization", "Bearer "+readerToken)

		mux.ServeHTTP(w, r)

		if w.Result().StatusCode != http.StatusUnauthorized {
			t.Fatalf("expected status code %d, got %d", http.StatusUnauthorized, w.Result().StatusCode)
		}
	})

	t.Run("test secret type", func(t *testing.T) {
		credentialName := "test-secret-type"

		secret := corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "credential-" + credentialName,
				Namespace: organization.Spec.NamespaceRef.Name,
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
			Path: path.Join("/v1/orgs", organization.Name, "credentials", credentialName),
		}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodDelete, u.Path, nil)

		r.Header.Add("Authorization", "Bearer "+superUserToken)

		mux.ServeHTTP(w, r)

		if w.Result().StatusCode != http.StatusInternalServerError {
			t.Fatalf("expected status code %d, got %d", http.StatusInternalServerError, w.Result().StatusCode)
		}
	})
}
