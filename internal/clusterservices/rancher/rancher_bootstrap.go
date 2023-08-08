package rancher

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/rancher/norman/clientbase"
	managementv3 "github.com/rancher/rancher/pkg/client/generated/management/v3"
	v3public "github.com/rancher/rancher/pkg/client/generated/management/v3public"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	defaultDockyardsNamespace    = "dockyards"
	defaultRancherSecretName     = "rancher-credentials"
	defaultRancherNamespace      = "cattle-system"
	defaultBootstrapSecretName   = "bootstrap-secret"
	defaultAdminUsername         = "admin"
	defaultRancherInternalCAName = "tls-rancher-internal-ca"
	defaultTokenDescription      = "dockyards"
)

func (r *rancher) getInternalCACerts() (string, error) {
	ctx := context.Background()

	objectKey := client.ObjectKey{
		Namespace: defaultRancherNamespace,
		Name:      defaultRancherInternalCAName,
	}

	var secret corev1.Secret
	err := r.controllerClient.Get(ctx, objectKey, &secret)
	if err != nil {
		r.logger.Error("error getting internal ca secret from kubernetes", "err", err)
		return "", err
	}

	tlsCert, hasTLSCert := secret.Data[corev1.TLSCertKey]
	if !hasTLSCert {
		r.logger.Error("internal ca secret has no tls cert key in data")
		return "", errors.New("internal ca secret has no tls cert key in data")
	}

	return string(tlsCert), nil
}

func (r *rancher) bootstrapLogin(bootstrapPassword string) (*v3public.Token, error) {
	basicLogin := v3public.BasicLogin{
		Username: defaultAdminUsername,
		Password: bootstrapPassword,
	}

	b, err := json.Marshal(basicLogin)
	if err != nil {
		r.logger.Error("error marshalling basic login", "err", err)
		return nil, err
	}

	buffer := bytes.NewBuffer(b)

	url := r.clientOpts.URL + "-public/localProviders/local?action=login"

	transport := http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: r.clientOpts.Insecure,
		},
	}

	if !r.clientOpts.Insecure {
		certPool := x509.NewCertPool()

		pemCerts := []byte(r.clientOpts.CACerts)
		ok := certPool.AppendCertsFromPEM(pemCerts)
		if !ok {
			r.logger.Warn("no certs were added to cert pool")
		}

		transport.TLSClientConfig.RootCAs = certPool
	}

	httpClient := http.Client{
		Transport: &transport,
	}

	request, err := http.NewRequest(http.MethodPost, url, buffer)
	if err != nil {
		r.logger.Error("error creating request", "err", err)
		return nil, err
	}

	r.logger.Debug("login action", "url", url)

	response, err := httpClient.Do(request)
	if err != nil {
		r.logger.Error("error doing http request", "err", err)
		return nil, err
	}

	if response.StatusCode != http.StatusCreated {
		r.logger.Error("unexpected status code", "code", response.StatusCode)
		return nil, errors.New("unexpected status code")
	}

	body, err := io.ReadAll(response.Body)
	if err != nil {
		r.logger.Error("error reading response body", "err", err)
		return nil, err
	}

	var token v3public.Token
	err = json.Unmarshal(body, &token)
	if err != nil {
		r.logger.Error("error unmarshalling temporary token", "err", err)
		return nil, err
	}

	return &token, nil
}

func (r *rancher) getTokenKeyOrBootstrap() (string, error) {
	ctx := context.Background()

	objectKey := client.ObjectKey{
		Namespace: defaultDockyardsNamespace,
		Name:      defaultRancherSecretName,
	}

	r.logger.Debug("getting rancher secret", "namespace", objectKey.Namespace, "name", objectKey.Name)

	var rancherSecret corev1.Secret
	err := r.controllerClient.Get(ctx, objectKey, &rancherSecret)
	if client.IgnoreNotFound(err) != nil {
		r.logger.Error("error getting rancher secret", "err", err)
		return "", err
	}

	if apierrors.IsNotFound(err) {
		r.logger.Debug("rancher secret not found, trying bootstrap secret")

		objectKey := client.ObjectKey{
			Namespace: defaultRancherNamespace,
			Name:      defaultBootstrapSecretName,
		}

		r.logger.Debug("getting bootstrap secret", "namespace", objectKey.Namespace, "name", objectKey.Name)

		var bootstrapSecret corev1.Secret
		err := r.controllerClient.Get(ctx, objectKey, &bootstrapSecret)
		if err != nil {
			r.logger.Error("error getting bootstrap secret", "err", err)
			return "", err
		}

		bootstrapPassword, hasBootstrapPassword := bootstrapSecret.Data["bootstrapPassword"]
		if !hasBootstrapPassword {
			err := errors.New("bootstrap secret has no bootstrap password in data")
			return "", err
		}

		r.logger.Debug("got bootstrap password", "password", bootstrapPassword)

		temporaryToken, err := r.bootstrapLogin(string(bootstrapPassword))
		if err != nil {
			r.logger.Error("error during bootstrap login", "err", err)
			return "", err
		}

		clientOpts := clientbase.ClientOpts{
			URL:      r.clientOpts.URL,
			TokenKey: temporaryToken.Token,
			CACerts:  r.clientOpts.CACerts,
			Insecure: r.clientOpts.Insecure,
		}

		temporaryManagementClient, err := managementv3.NewClient(&clientOpts)
		if err != nil {
			r.logger.Error("error creating temporary management client", "err", err)
			return "", err
		}

		adminUser, err := temporaryManagementClient.User.ByID(temporaryToken.UserID)
		if err != nil {
			r.logger.Error("error getting admin user", "err", err)
			return "", err
		}

		tokenInput := managementv3.Token{
			Description: defaultTokenDescription,
			UserID:      adminUser.ID,
		}

		createdToken, err := temporaryManagementClient.Token.Create(&tokenInput)
		if err != nil {
			r.logger.Error("error creating dockyards token", "err", err)
			return "", err
		}

		b := make([]byte, 32)
		_, err = rand.Read(b)
		if err != nil {
			r.logger.Error("error reading random bytes", "err", err)
			return "", err
		}

		newPassword := base64.StdEncoding.EncodeToString(b)

		passwordInput := managementv3.SetPasswordInput{
			NewPassword: newPassword,
		}

		_, err = temporaryManagementClient.User.ActionSetpassword(adminUser, &passwordInput)
		if err != nil {
			r.logger.Error("error setting new password", "err", err)
			return "", err
		}

		rancherSecret = corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: defaultDockyardsNamespace,
				Name:      defaultRancherSecretName,
			},
			StringData: map[string]string{
				"token":    createdToken.Token,
				"password": newPassword,
			},
		}

		r.logger.Debug("creating rancher secret", "namespace", rancherSecret.Namespace, "name", rancherSecret.Name)

		err = r.controllerClient.Create(ctx, &rancherSecret)
		if err != nil {
			r.logger.Error("error creating racher secret", "err", err)
			return "", err
		}
	}

	token, hasToken := rancherSecret.Data["token"]
	if !hasToken {
		return "", errors.New("rancher secret has no token in data")
	}

	return string(token), nil
}
