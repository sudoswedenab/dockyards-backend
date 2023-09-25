package jwt

import (
	"context"
	"crypto/rand"
	"encoding/base64"
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
)

func GetOrGenerateTokens(ctx context.Context, controllerClient client.Client, logger *slog.Logger) (string, string, error) {
	objectKey := client.ObjectKey{
		Namespace: defaultDockyardsNamespace,
		Name:      defaultJWTSecretName,
	}

	var secret corev1.Secret
	err := controllerClient.Get(ctx, objectKey, &secret)
	if client.IgnoreNotFound(err) != nil {
		return "", "", err
	}

	if apierrors.IsNotFound(err) {
		logger.Debug("generating private secrets")

		b := make([]byte, 32)
		_, err := rand.Read(b)
		if err != nil {
			return "", "", err
		}
		accessToken := base64.StdEncoding.EncodeToString(b)

		logger.Debug("generated access token")

		b = make([]byte, 32)
		_, err = rand.Read(b)
		if err != nil {
			return "", "", err
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
			return "", "", err
		}

		logger.Debug("created jwt tokens secret in kubernetes", "uid", secret.UID)
	}

	accessToken, hasToken := secret.Data["accessToken"]
	if !hasToken {
		return "", "", errors.New("jwt tokens secret has no access token in data")
	}

	refreshToken, hasToken := secret.Data["refreshToken"]
	if !hasToken {
		return "", "", errors.New("jwt tokens secret has no refresh token in data")
	}

	return string(accessToken), string(refreshToken), nil
}
