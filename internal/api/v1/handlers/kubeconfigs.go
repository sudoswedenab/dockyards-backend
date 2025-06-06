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
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"time"

	"github.com/sudoswedenab/dockyards-api/pkg/types"
	"github.com/sudoswedenab/dockyards-backend/api/apiutil"
	dockyardsv1 "github.com/sudoswedenab/dockyards-backend/api/v1alpha3"
	"github.com/sudoswedenab/dockyards-backend/internal/api/v1/middleware"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (h *handler) CreateClusterKubeconfig(ctx context.Context, cluster *dockyardsv1.Cluster, request *types.KubeconfigOptions) (*[]byte, error) {
	if !cluster.Status.APIEndpoint.IsValid() {
		return nil, errors.New("cluster does not have a valid api endpoint")
	}

	duration := time.Hour * 12

	if request.Duration != nil {
		requestDuration, err := time.ParseDuration(*request.Duration)
		if err != nil {
			return nil, err
		}

		if requestDuration > duration {
			invalid := field.Invalid(field.NewPath("duration"), requestDuration, "must not be greater than 12 hours")
			statusError := apierrors.NewInvalid(dockyardsv1.GroupVersion.WithKind(dockyardsv1.ClusterKind).GroupKind(), "kubeconfig", field.ErrorList{invalid})

			return nil, statusError
		}

		if requestDuration < 0 {
			invalid := field.Invalid(field.NewPath("duration"), requestDuration, "must not be smaller than 0")
			statusError := apierrors.NewInvalid(dockyardsv1.GroupVersion.WithKind(dockyardsv1.ClusterKind).GroupKind(), "kubeconfig", field.ErrorList{invalid})

			return nil, statusError
		}

		duration = requestDuration
	}

	objectKey := client.ObjectKey{
		Name:      cluster.Name + "-ca",
		Namespace: cluster.Namespace,
	}

	var secret corev1.Secret
	err := h.Get(ctx, objectKey, &secret)
	if err != nil {
		return nil, err
	}

	caCertificatePEM, has := secret.Data[corev1.TLSCertKey]
	if !has {
		return nil, errors.New("cluster certificate authority has no tls certificate")
	}

	signingKeyPEM, has := secret.Data[corev1.TLSPrivateKeyKey]
	if !has {
		return nil, errors.New("cluster certificate authority has no tls private key")
	}

	block, _ := pem.Decode(caCertificatePEM)
	if block == nil {
		return nil, errors.New("unable to decode ca certificate as pem")
	}

	caCertificate, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, err
	}

	block, _ = pem.Decode(signingKeyPEM)

	signingKey, err := x509.ParseECPrivateKey(block.Bytes)
	if err != nil {
		return nil, err
	}

	subject, err := middleware.SubjectFrom(ctx)
	if err != nil {
		return nil, err
	}

	var user dockyardsv1.User
	err = h.Get(ctx, client.ObjectKey{Name: subject}, &user)
	if err != nil {
		return nil, err
	}

	ownerOrganization, err := apiutil.GetOwnerOrganization(ctx, h.Client, cluster)
	if err != nil {
		return nil, err
	}

	if ownerOrganization == nil {
		return nil, errors.New("the cluster has no owner organization")
	}

	userAlias := user.Name + "/" + cluster.Name
	clusterAlias := cluster.Name + "/" + ownerOrganization.Name

	contextName := userAlias + "@" + clusterAlias

	tmpl := x509.Certificate{
		Subject: pkix.Name{
			CommonName: user.Name,
			Organization: []string{
				"system:masters",
			},
		},
		NotBefore: caCertificate.NotBefore,
		NotAfter:  time.Now().Add(duration),
		KeyUsage:  x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage: []x509.ExtKeyUsage{
			x509.ExtKeyUsageClientAuth,
		},
	}

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}

	certificate, err := x509.CreateCertificate(rand.Reader, &tmpl, caCertificate, privateKey.Public(), signingKey)
	if err != nil {
		return nil, err
	}

	block = &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certificate,
	}

	certificatePEM := pem.EncodeToMemory(block)

	block = &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	}

	privateKeyPEM := pem.EncodeToMemory(block)

	cfg := api.Config{
		Clusters: map[string]*api.Cluster{
			clusterAlias: {
				Server:                   cluster.Status.APIEndpoint.String(),
				CertificateAuthorityData: caCertificatePEM,
			},
		},
		Contexts: map[string]*api.Context{
			contextName: {
				Cluster:  clusterAlias,
				AuthInfo: userAlias,
			},
		},
		AuthInfos: map[string]*api.AuthInfo{
			userAlias: {
				ClientCertificateData: certificatePEM,
				ClientKeyData:         privateKeyPEM,
			},
		},
		CurrentContext: contextName,
	}

	b, err := clientcmd.Write(cfg)
	if err != nil {
		return nil, err
	}

	return &b, nil
}
