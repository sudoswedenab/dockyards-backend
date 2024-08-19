package jwt

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"log/slog"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// +kubebuilder:rbac:groups=core,resources=configmaps,verbs=create;get;list;patch;watch
// +kubebuilder:rbac:groups=core,resources=secrets,verbs=create;get;list;patch;watch

const (
	defaultDockyardsNamespace = "dockyards"
	defaultJWTSecretName      = "jwt-tokens"
	AccessTokenPrivateKeyKey  = "accessTokenPrivateKey"
	RefreshTokenPrivateKeyKey = "refreshTokenPrivateKey"
	AccessTokenPublicKeyKey   = "accessTokenPublicKey"
)

func GetOrGenerateTokens(ctx context.Context, controllerClient client.Client, logger *slog.Logger) ([]byte, []byte, error) {
	empty := []byte{}

	objectKey := client.ObjectKey{
		Namespace: defaultDockyardsNamespace,
		Name:      defaultJWTSecretName,
	}

	var secret corev1.Secret
	err := controllerClient.Get(ctx, objectKey, &secret)
	if client.IgnoreNotFound(err) != nil {
		return empty, empty, err
	}

	if apierrors.IsNotFound(err) {
		logger.Debug("generating private secrets")

		b := make([]byte, 32)
		_, err := rand.Read(b)
		if err != nil {
			return empty, empty, err
		}
		accessToken := base64.StdEncoding.EncodeToString(b)

		logger.Debug("generated access token")

		b = make([]byte, 32)
		_, err = rand.Read(b)
		if err != nil {
			return empty, empty, err
		}
		refreshToken := base64.StdEncoding.EncodeToString(b)

		logger.Debug("generated refresh token")

		secret = corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: defaultDockyardsNamespace,
				Name:      defaultJWTSecretName,
			},
			StringData: map[string]string{
				"accessToken":  accessToken,
				"refreshToken": refreshToken,
			},
		}

		err = controllerClient.Create(ctx, &secret)
		if err != nil {
			return empty, empty, err
		}

		logger.Debug("created jwt tokens secret in kubernetes", "uid", secret.UID)
	}

	accessToken, hasToken := secret.Data["accessToken"]
	if !hasToken {
		return empty, empty, errors.New("jwt tokens secret has no access token in data")
	}

	refreshToken, hasToken := secret.Data["refreshToken"]
	if !hasToken {
		return empty, empty, errors.New("jwt tokens secret has no refresh token in data")
	}

	return accessToken, refreshToken, nil
}

func GetOrGenerateKeys(ctx context.Context, controllerClient client.Client, logger *slog.Logger) (*ecdsa.PrivateKey, *ecdsa.PrivateKey, error) {
	var (
		accessTokenPrivateKeyPEM  []byte
		refreshTokenPrivateKeyPEM []byte
		has                       bool
	)

	secret := corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "dockyards-backend-jwt",
			Namespace: defaultDockyardsNamespace,
		},
	}

	_, err := controllerutil.CreateOrPatch(ctx, controllerClient, &secret, func() error {
		if secret.Data == nil {
			secret.Data = make(map[string][]byte)
		}

		accessTokenPrivateKeyPEM, has = secret.Data[AccessTokenPrivateKeyKey]
		if !has {
			accessTokenPrivateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
			if err != nil {
				return err
			}

			b, err := x509.MarshalECPrivateKey(accessTokenPrivateKey)
			if err != nil {
				return err
			}

			block := pem.Block{
				Type:  "EC PRIVATE KEY",
				Bytes: b,
			}

			accessTokenPrivateKeyPEM = pem.EncodeToMemory(&block)

			secret.Data[AccessTokenPrivateKeyKey] = accessTokenPrivateKeyPEM
		}

		refreshTokenPrivateKeyPEM, has = secret.Data[RefreshTokenPrivateKeyKey]
		if !has {
			refreshTokenPrivateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
			if err != nil {
				return err
			}

			b, err := x509.MarshalECPrivateKey(refreshTokenPrivateKey)
			if err != nil {
				return err
			}

			block := pem.Block{
				Type:  "EC PRIVATE KEY",
				Bytes: b,
			}

			refreshTokenPrivateKeyPEM = pem.EncodeToMemory(&block)

			secret.Data[RefreshTokenPrivateKeyKey] = refreshTokenPrivateKeyPEM
		}

		return nil
	})
	if err != nil {
		return nil, nil, err
	}

	accessTokenPrivateKeyDER, _ := pem.Decode(accessTokenPrivateKeyPEM)
	if accessTokenPrivateKeyDER == nil || accessTokenPrivateKeyDER.Type != "EC PRIVATE KEY" {
		return nil, nil, errors.New("invalid access private key")
	}

	accessTokenPrivateKey, err := x509.ParseECPrivateKey(accessTokenPrivateKeyDER.Bytes)
	if err != nil {
		return nil, nil, err
	}

	refreshTokenPrivateKeyDER, _ := pem.Decode(refreshTokenPrivateKeyPEM)
	if refreshTokenPrivateKeyDER == nil || refreshTokenPrivateKeyDER.Type != "EC PRIVATE KEY" {
		return nil, nil, errors.New("invalid refresh private key")
	}

	refreshTokenPrivateKey, err := x509.ParseECPrivateKey(refreshTokenPrivateKeyDER.Bytes)
	if err != nil {
		return nil, nil, err
	}

	configMap := corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "dockyards-backend-jwt",
			Namespace: defaultDockyardsNamespace,
		},
	}

	_, err = controllerutil.CreateOrPatch(ctx, controllerClient, &configMap, func() error {
		if configMap.Data == nil {
			configMap.Data = make(map[string]string)
		}

		accessTokenPublicKey := accessTokenPrivateKey.PublicKey

		b, err := x509.MarshalPKIXPublicKey(&accessTokenPublicKey)
		if err != nil {
			return err
		}

		block := pem.Block{
			Type:  "PUBLIC KEY",
			Bytes: b,
		}

		accessTokenPublicKeyPEM := pem.EncodeToMemory(&block)

		configMap.Data[AccessTokenPublicKeyKey] = string(accessTokenPublicKeyPEM)

		return nil
	})
	if err != nil {
		return nil, nil, err
	}

	return accessTokenPrivateKey, refreshTokenPrivateKey, nil
}
