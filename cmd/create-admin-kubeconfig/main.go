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

package main

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"os"
	"os/signal"
	"time"

	"github.com/spf13/pflag"
	certificatesv1 "k8s.io/api/certificates/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

func main() {
	var clusterRoleBindingName string
	var commonName string
	var expirationSeconds int32
	pflag.StringVar(&clusterRoleBindingName, "cluster-role-name", "sudo-admin", "cluster role name")
	pflag.StringVar(&commonName, "common-name", "test", "common name")
	pflag.Int32Var(&expirationSeconds, "expiration-seconds", 1209600, "expiration seconds")
	pflag.Parse()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	cfg, err := config.GetConfig()
	if err != nil {
		panic(err)
	}

	c, err := client.NewWithWatch(cfg, client.Options{})
	if err != nil {
		panic(err)
	}

	var clusterRoleBinding rbacv1.ClusterRoleBinding

	err = c.Get(ctx, client.ObjectKey{Name: clusterRoleBindingName}, &clusterRoleBinding)
	if client.IgnoreNotFound(err) != nil {
		panic(err)
	}

	if apierrors.IsNotFound(err) {
		fmt.Println("cluster-role not found")

		os.Exit(1)
	}

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		panic(err)
	}

	certificateRequest := x509.CertificateRequest{
		Subject: pkix.Name{
			Organization: []string{
				clusterRoleBinding.Name,
			},
			CommonName: commonName,
		},
		PublicKey: privateKey.PublicKey,
	}

	csr, err := x509.CreateCertificateRequest(rand.Reader, &certificateRequest, privateKey)
	if err != nil {
		panic(err)
	}

	block := pem.Block{
		Type:  "CERTIFICATE REQUEST",
		Bytes: csr,
	}

	request := pem.EncodeToMemory(&block)

	certificateSigningRequest := certificatesv1.CertificateSigningRequest{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: commonName + "-",
		},
		Spec: certificatesv1.CertificateSigningRequestSpec{
			Request:           request,
			SignerName:        certificatesv1.KubeAPIServerClientSignerName,
			ExpirationSeconds: &expirationSeconds,
			Usages: []certificatesv1.KeyUsage{
				certificatesv1.UsageKeyEncipherment,
				certificatesv1.UsageDigitalSignature,
				certificatesv1.UsageClientAuth,
			},
		},
	}

	err = c.Create(ctx, &certificateSigningRequest)
	if err != nil {
		panic(err)
	}

	certificateSigningRequest.Status.Conditions = append(certificateSigningRequest.Status.Conditions, certificatesv1.CertificateSigningRequestCondition{
		Type:   certificatesv1.CertificateApproved,
		Status: corev1.ConditionStatus(metav1.ConditionTrue),
	})

	err = c.SubResource("approval").Update(ctx, &certificateSigningRequest)
	if err != nil {
		panic(err)
	}

	var certificateSigningRequestList certificatesv1.CertificateSigningRequestList

	w, err := c.Watch(ctx, &certificateSigningRequestList)
	if err != nil {
		panic(err)
	}

loop:
	for {
		select {
		case event, open := <-w.ResultChan():
			if !open {
				break loop
			}

			csr, ok := event.Object.(*certificatesv1.CertificateSigningRequest)
			if !ok {
				fmt.Printf("incorrect type: %T\n", event.Object)
			}

			if csr.Name != certificateSigningRequest.Name {
				continue
			}

			if csr.Status.Certificate == nil {
				continue
			}

			certificateSigningRequest = *csr

			break loop
		case <-time.After(time.Second * 5):
			fmt.Println("timeout")
			os.Exit(1)
		}
	}

	block = pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	}

	privateKeyPEM := pem.EncodeToMemory(&block)

	clientConfig := clientcmdapi.Config{
		AuthInfos: map[string]*clientcmdapi.AuthInfo{
			commonName: {
				ClientCertificateData: certificateSigningRequest.Status.Certificate,
				ClientKeyData:         privateKeyPEM,
			},
		},
		Clusters: map[string]*clientcmdapi.Cluster{
			commonName: {
				CertificateAuthorityData: cfg.CAData,
				Server:                   cfg.Host,
			},
		},
		Contexts: map[string]*clientcmdapi.Context{
			commonName: {
				Cluster:  commonName,
				AuthInfo: commonName,
			},
		},
		CurrentContext: commonName,
	}

	kubeconfig, err := clientcmd.Write(clientConfig)
	if err != nil {
		panic(err)
	}

	fmt.Println(string(kubeconfig))
}
