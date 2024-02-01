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
)

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
	objectKey := client.ObjectKey{
		Name:      "dockyards-backend-jwt",
		Namespace: defaultDockyardsNamespace,
	}

	var secret corev1.Secret
	err := controllerClient.Get(ctx, objectKey, &secret)
	if client.IgnoreNotFound(err) != nil {
		return nil, nil, err
	}

	if apierrors.IsNotFound(err) {
		logger.Debug("generating keys due to missing secret", "name", objectKey.Name)

		accessTokenPrivateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		if err != nil {
			return nil, nil, err

		}

		b, err := x509.MarshalECPrivateKey(accessTokenPrivateKey)
		if err != nil {
			return nil, nil, err
		}

		block := pem.Block{
			Type:  "EC PRIVATE KEY",
			Bytes: b,
		}

		accessTokenPrivateKeyPEM := pem.EncodeToMemory(&block)

		refreshTokenPrivateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		if err != nil {
			return nil, nil, err
		}

		b, err = x509.MarshalECPrivateKey(refreshTokenPrivateKey)
		if err != nil {
			return nil, nil, err
		}

		block = pem.Block{
			Type:  "EC PRIVATE KEY",
			Bytes: b,
		}

		refreshTokenPrivateKeyPEM := pem.EncodeToMemory(&block)

		secret = corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "dockyards-backend-jwt",
				Namespace: defaultDockyardsNamespace,
			},
			Data: map[string][]byte{
				AccessTokenPrivateKeyKey:  accessTokenPrivateKeyPEM,
				RefreshTokenPrivateKeyKey: refreshTokenPrivateKeyPEM,
			},
		}

		err = controllerClient.Create(ctx, &secret)
		if err != nil {
			return nil, nil, err
		}

		accessTokenPublicKey := accessTokenPrivateKey.PublicKey

		b, err = x509.MarshalPKIXPublicKey(&accessTokenPublicKey)
		if err != nil {
			return nil, nil, err
		}

		block = pem.Block{
			Type:  "PUBLIC KEY",
			Bytes: b,
		}

		accessTokenPublicKeyPEM := pem.EncodeToMemory(&block)

		configMap := corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "dockyards-backend-jwt",
				Namespace: defaultDockyardsNamespace,
			},
			Data: map[string]string{
				AccessTokenPublicKeyKey: string(accessTokenPublicKeyPEM),
			},
		}

		err = controllerClient.Create(ctx, &configMap)
		if err != nil {
			return nil, nil, err
		}
	}

	accessTokenPrivateKeyPEM, hasPrivateKey := secret.Data[AccessTokenPrivateKeyKey]
	if !hasPrivateKey {
		return nil, nil, errors.New("secret has no private accesss key")
	}

	accessTokenPrivateKeyDER, _ := pem.Decode(accessTokenPrivateKeyPEM)
	if accessTokenPrivateKeyDER == nil || accessTokenPrivateKeyDER.Type != "EC PRIVATE KEY" {
		return nil, nil, errors.New("invalid access private key")
	}

	accessTokenPrivateKey, err := x509.ParseECPrivateKey(accessTokenPrivateKeyDER.Bytes)
	if err != nil {
		return nil, nil, err
	}

	refreshTokenPrivateKeyPEM, hasPrivateKey := secret.Data[RefreshTokenPrivateKeyKey]
	if !hasPrivateKey {
		return nil, nil, errors.New("secret has no private refresh key")
	}

	refreshTokenPrivateKeyDER, _ := pem.Decode(refreshTokenPrivateKeyPEM)
	if refreshTokenPrivateKeyDER == nil || refreshTokenPrivateKeyDER.Type != "EC PRIVATE KEY" {
		return nil, nil, errors.New("invalid refresh private key")
	}

	refreshTokenPrivateKey, err := x509.ParseECPrivateKey(refreshTokenPrivateKeyDER.Bytes)
	if err != nil {
		return nil, nil, err
	}

	return accessTokenPrivateKey, refreshTokenPrivateKey, nil
}
