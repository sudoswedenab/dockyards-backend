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

package jwt

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"errors"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// +kubebuilder:rbac:groups=core,resources=configmaps,verbs=create;get;list;patch;watch
// +kubebuilder:rbac:groups=core,resources=secrets,verbs=create;get;list;patch;watch

const (
	AccessTokenPrivateKeyKey  = "accessTokenPrivateKey"
	RefreshTokenPrivateKeyKey = "refreshTokenPrivateKey"
	AccessTokenPublicKeyKey   = "accessTokenPublicKey"
)

func GetOrGenerateKeys(ctx context.Context, c client.Client, namespace string) (*ecdsa.PrivateKey, *ecdsa.PrivateKey, error) {
	var (
		accessTokenPrivateKeyPEM  []byte
		refreshTokenPrivateKeyPEM []byte
		has                       bool
	)

	secret := corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "dockyards-backend-jwt",
			Namespace: namespace,
		},
	}

	_, err := controllerutil.CreateOrPatch(ctx, c, &secret, func() error {
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
			Namespace: namespace,
		},
	}

	_, err = controllerutil.CreateOrPatch(ctx, c, &configMap, func() error {
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
