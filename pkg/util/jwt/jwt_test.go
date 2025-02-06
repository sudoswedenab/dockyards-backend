package jwt_test

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"testing"

	"bitbucket.org/sudosweden/dockyards-backend/pkg/util/jwt"
	"github.com/google/go-cmp/cmp"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
)

func TestGetOrGenerateKeys(t *testing.T) {
	environment := envtest.Environment{}

	cfg, err := environment.Start()
	if err != nil {
		t.Fatal(err)
	}

	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)

	c, err := client.New(cfg, client.Options{Scheme: scheme})
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()

	encodeKey := func(privateKey *ecdsa.PrivateKey) ([]byte, error) {
		b, err := x509.MarshalECPrivateKey(privateKey)
		if err != nil {
			return nil, err
		}

		block := pem.Block{
			Type:  "EC PRIVATE KEY",
			Bytes: b,
		}

		return pem.EncodeToMemory(&block), nil
	}

	generateKey := func() (*ecdsa.PrivateKey, error) {
		privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		if err != nil {
			return nil, err
		}

		return privateKey, nil
	}

	t.Run("test missing secret", func(t *testing.T) {
		namespace := corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: "test-jwt-missing-",
			},
		}

		err := c.Create(ctx, &namespace)
		if err != nil {
			t.Fatal(err)
		}

		accessTokenPrivateKey, refreshTokenPrivateKey, err := jwt.GetOrGenerateKeys(ctx, c, namespace.Name)
		if err != nil {
			t.Fatal(err)
		}

		accessTokenPEM, err := encodeKey(accessTokenPrivateKey)
		if err != nil {
			t.Fatal(err)
		}

		refreshTokenPEM, err := encodeKey(refreshTokenPrivateKey)
		if err != nil {
			t.Fatal(err)
		}

		var actual corev1.Secret
		err = c.Get(ctx, client.ObjectKey{Name: "dockyards-backend-jwt", Namespace: namespace.Name}, &actual)
		if err != nil {
			t.Fatal(err)
		}

		expected := corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "dockyards-backend-jwt",
				Namespace:         namespace.Name,
				CreationTimestamp: actual.CreationTimestamp,
				ManagedFields:     actual.ManagedFields,
				ResourceVersion:   actual.ResourceVersion,
				UID:               actual.UID,
			},
			Data: map[string][]byte{
				jwt.AccessTokenPrivateKeyKey:  accessTokenPEM,
				jwt.RefreshTokenPrivateKeyKey: refreshTokenPEM,
			},
			Type: corev1.SecretTypeOpaque,
		}

		if !cmp.Equal(actual, expected) {
			t.Errorf("diff: %s", cmp.Diff(expected, actual))
		}
	})

	t.Run("test existing secret", func(t *testing.T) {
		namespace := corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: "test-jwt-existing-",
			},
		}

		err := c.Create(ctx, &namespace)
		if err != nil {
			t.Fatal(err)
		}

		accessTokenKey, err := generateKey()
		if err != nil {
			t.Fatal(err)
		}

		accessTokenPEM, err := encodeKey(accessTokenKey)
		if err != nil {
			t.Fatal(err)
		}

		refreshTokenKey, err := generateKey()
		if err != nil {
			t.Fatal(err)
		}

		refreshTokenPEM, err := encodeKey(refreshTokenKey)
		if err != nil {
			t.Fatal(err)
		}

		secret := corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "dockyards-backend-jwt",
				Namespace: namespace.Name,
			},
			Data: map[string][]byte{
				jwt.AccessTokenPrivateKeyKey:  accessTokenPEM,
				jwt.RefreshTokenPrivateKeyKey: refreshTokenPEM,
			},
			Type: corev1.SecretTypeOpaque,
		}

		err = c.Create(ctx, &secret)
		if err != nil {
			t.Fatal(err)
		}

		actualAccessTokenKey, actualRefreshTokenKey, err := jwt.GetOrGenerateKeys(ctx, c, namespace.Name)
		if err != nil {
			t.Fatal(err)
		}

		if !actualAccessTokenKey.Equal(accessTokenKey) {
			t.Error("access token keys not equal")
		}

		if !actualRefreshTokenKey.Equal(refreshTokenKey) {
			t.Error("refresh token keys not equal")
		}
	})
}
