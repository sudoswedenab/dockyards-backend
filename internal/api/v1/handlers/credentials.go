package handlers

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"bitbucket.org/sudosweden/dockyards-backend/internal/api/v1"
	"bitbucket.org/sudosweden/dockyards-backend/internal/api/v1/middleware"
	dockyardsv1 "bitbucket.org/sudosweden/dockyards-backend/pkg/api/v1alpha2"
	"bitbucket.org/sudosweden/dockyards-backend/pkg/api/v1alpha2/index"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// +kubebuilder:rbac:groups=core,resources=secrets,verbs=create;delete;get;list;patch;watch

func (h *handler) GetOrganizationCredentials(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	logger := middleware.LoggerFrom(ctx)

	organizationName := r.PathValue("organizationName")
	if organizationName == "" {
		w.WriteHeader(http.StatusBadRequest)

		return
	}

	var organization dockyardsv1.Organization
	err := h.Get(ctx, client.ObjectKey{Name: organizationName}, &organization)
	if client.IgnoreNotFound(err) != nil {
		logger.Error("error getting organization", "err", err)
		w.WriteHeader(http.StatusInternalServerError)

		return
	}

	if apierrors.IsNotFound(err) {
		w.WriteHeader(http.StatusUnauthorized)

		return
	}

	subject, err := middleware.SubjectFrom(ctx)
	if err != nil {
		logger.Error("error getting subject from context", "err", err)
		w.WriteHeader(http.StatusInternalServerError)

		return
	}

	if !h.isMember(subject, &organization) {
		w.WriteHeader(http.StatusUnauthorized)

		return
	}

	var secretList corev1.SecretList
	err = h.List(ctx, &secretList, client.InNamespace(organization.Status.NamespaceRef))
	if err != nil {
		logger.Error("error listing secrets", "err", err)
		w.WriteHeader(http.StatusInternalServerError)

		return
	}

	credentials := []v1.Credential{}

	for _, secret := range secretList.Items {
		if secret.Type != dockyardsv1.SecretTypeCredential {
			continue
		}

		credential := v1.Credential{
			ID:           string(secret.UID),
			Name:         strings.TrimPrefix(secret.Name, "credential-"),
			Organization: organization.Name,
		}

		credentialTemplate, has := secret.Labels[dockyardsv1.LabelCredentialTemplateName]
		if has {
			credential.CredentialTemplate = &credentialTemplate
		}

		credentials = append(credentials, credential)
	}

	b, err := json.Marshal(&credentials)
	if err != nil {
		logger.Error("error marhalling credentials", "err", err)
		w.WriteHeader(http.StatusInternalServerError)

		return
	}

	w.WriteHeader(http.StatusOK)
	_, err = w.Write(b)
	if err != nil {
		logger.Error("error writing response data", "err", err)
	}
}

func (h *handler) PostOrganizationCredentials(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	logger := middleware.LoggerFrom(ctx)

	organizationName := r.PathValue("organizationName")
	if organizationName == "" {
		w.WriteHeader(http.StatusBadRequest)

		return
	}

	var organization dockyardsv1.Organization
	err := h.Get(ctx, client.ObjectKey{Name: organizationName}, &organization)
	if client.IgnoreNotFound(err) != nil {
		logger.Error("error getting organization", "err", err)
		w.WriteHeader(http.StatusInternalServerError)

		return
	}

	if apierrors.IsNotFound(err) {
		w.WriteHeader(http.StatusUnauthorized)

		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)

		return
	}

	r.Body.Close()

	var credential v1.Credential
	err = json.Unmarshal(body, &credential)
	if err != nil {
		logger.Debug("error unmashalling body", "err", err)
		w.WriteHeader(http.StatusUnprocessableEntity)

		return
	}

	subject, err := middleware.SubjectFrom(ctx)
	if err != nil {
		logger.Error("error getting subject from context", "err", err)
		w.WriteHeader(http.StatusInternalServerError)

		return
	}

	if !h.isMember(subject, &organization) {
		w.WriteHeader(http.StatusUnauthorized)

		return
	}

	secret := corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "credential-" + credential.Name,
			Namespace: organization.Status.NamespaceRef,
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: dockyardsv1.GroupVersion.String(),
					Kind:       dockyardsv1.OrganizationKind,
					Name:       organization.Name,
					UID:        organization.UID,
				},
			},
		},
		Type: dockyardsv1.SecretTypeCredential,
	}

	if credential.Data != nil {
		secret.Data = make(map[string][]byte)

		for key, value := range *credential.Data {
			secret.Data[key] = value
		}
	}

	err = h.Create(ctx, &secret)
	if err != nil {
		logger.Error("error creating secret", "err", err)
		w.WriteHeader(http.StatusInternalServerError)

		return
	}

	createdCredential := v1.Credential{
		ID:   string(secret.UID),
		Name: secret.Name,
	}

	b, err := json.Marshal(&createdCredential)
	if err != nil {
		logger.Debug("error marshalling credential", "err", err)
		w.WriteHeader(http.StatusInternalServerError)
	}

	w.WriteHeader(http.StatusCreated)
	_, err = w.Write(b)
	if err != nil {
		logger.Error("error writing response data", "err", err)
	}
}

func (h *handler) DeleteOrganizationCredential(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	logger := middleware.LoggerFrom(ctx)

	organizationName := r.PathValue("organizationName")
	credentialName := r.PathValue("credentialName")
	if organizationName == "" || credentialName == "" {
		w.WriteHeader(http.StatusBadRequest)

		return
	}

	var organization dockyardsv1.Organization
	err := h.Get(ctx, client.ObjectKey{Name: organizationName}, &organization)
	if client.IgnoreNotFound(err) != nil {
		logger.Error("eror getting organization", "err", err)
		w.WriteHeader(http.StatusInternalServerError)

		return
	}

	if apierrors.IsNotFound(err) {
		w.WriteHeader(http.StatusUnauthorized)

		return
	}

	subject, err := middleware.SubjectFrom(ctx)
	if err != nil {
		logger.Error("error getting subject from context", "err", err)
		w.WriteHeader(http.StatusInternalServerError)

		return
	}

	if !h.isMember(subject, &organization) {
		w.WriteHeader(http.StatusUnauthorized)

		return
	}

	var secret corev1.Secret
	err = h.Get(ctx, client.ObjectKey{Name: "credential-" + credentialName, Namespace: organization.Status.NamespaceRef}, &secret)
	if client.IgnoreNotFound(err) != nil {
		logger.Error("error getting secret", "err", err)
		w.WriteHeader(http.StatusInternalServerError)

		return
	}

	if apierrors.IsNotFound(err) {
		w.WriteHeader(http.StatusNotFound)

		return
	}

	if secret.Type != dockyardsv1.SecretTypeCredential {
		w.WriteHeader(http.StatusUnauthorized)

		return
	}

	err = h.Delete(ctx, &secret)
	if err != nil {
		logger.Error("error deleting secret", "err", err)
		w.WriteHeader(http.StatusInternalServerError)

		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *handler) GetOrganizationCredential(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	logger := middleware.LoggerFrom(ctx)

	organizationName := r.PathValue("organizationName")
	credentialName := r.PathValue("credentialName")

	if credentialName == "" || organizationName == "" {
		w.WriteHeader(http.StatusBadRequest)

		return
	}

	var organization dockyardsv1.Organization
	err := h.Get(ctx, client.ObjectKey{Name: organizationName}, &organization)
	if client.IgnoreNotFound(err) != nil {
		logger.Error("error getting organization", "err", err)
		w.WriteHeader(http.StatusInternalServerError)

		return
	}

	subject, err := middleware.SubjectFrom(ctx)
	if err != nil {
		logger.Error("error getting subject from context", "err", err)
		w.WriteHeader(http.StatusInternalServerError)

		return
	}

	if !h.isMember(subject, &organization) {
		w.WriteHeader(http.StatusUnauthorized)

		return
	}

	var secret corev1.Secret
	err = h.Get(ctx, client.ObjectKey{Name: "credential-" + credentialName, Namespace: organization.Status.NamespaceRef}, &secret)
	if client.IgnoreNotFound(err) != nil {
		logger.Error("error getting secret", "err", err)
		w.WriteHeader(http.StatusInternalServerError)

		return
	}

	if apierrors.IsNotFound(err) {
		w.WriteHeader(http.StatusNotFound)

		return
	}

	if secret.Type != dockyardsv1.SecretTypeCredential {
		w.WriteHeader(http.StatusUnauthorized)

		return
	}

	plaintextKeys := make(map[string]bool)

	v1Credential := v1.Credential{
		ID:           string(secret.UID),
		Name:         strings.TrimPrefix(secret.Name, "credential-"),
		Organization: organization.Name,
	}

	credentialTemplateName, has := secret.Labels[dockyardsv1.LabelCredentialTemplateName]
	if has {
		var credentialTemplate dockyardsv1.CredentialTemplate
		err := h.Get(ctx, client.ObjectKey{Name: credentialTemplateName, Namespace: h.namespace}, &credentialTemplate)
		if client.IgnoreNotFound(err) != nil {
			logger.Error("error getting credential template", "err", err)
			w.WriteHeader(http.StatusInternalServerError)

			return
		}

		if !apierrors.IsNotFound(err) {
			for _, option := range credentialTemplate.Spec.Options {
				if option.Plaintext {
					plaintextKeys[option.Key] = true
				}
			}
		}

		v1Credential.CredentialTemplate = &credentialTemplateName
	}

	if secret.Data != nil {
		data := make(map[string][]byte)

		for key, value := range secret.Data {
			if plaintextKeys[key] {
				data[key] = value

				continue
			}

			data[key] = nil
		}

		v1Credential.Data = &data
	}

	b, err := json.Marshal(&v1Credential)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)

		return
	}

	w.WriteHeader(http.StatusOK)
	_, err = w.Write(b)
	if err != nil {
		logger.Error("error writing response data", "err", err)
	}
}

func (h *handler) PutOrganizationCredential(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	logger := middleware.LoggerFrom(ctx)

	organizationName := r.PathValue("organizationName")
	credentialName := r.PathValue("credentialName")

	if organizationName == "" || credentialName == "" {
		w.WriteHeader(http.StatusBadRequest)

		return
	}

	subject, err := middleware.SubjectFrom(ctx)
	if err != nil {
		logger.Error("error getting subject from context", "err", err)
		w.WriteHeader(http.StatusInternalServerError)

		return
	}

	var organization dockyardsv1.Organization
	err = h.Get(ctx, client.ObjectKey{Name: organizationName}, &organization)
	if client.IgnoreNotFound(err) != nil {
		logger.Error("error getting organization", "err", err)
		w.WriteHeader(http.StatusInternalServerError)

		return
	}

	if apierrors.IsNotFound(err) {
		w.WriteHeader(http.StatusUnauthorized)

		return
	}

	if !h.isMember(subject, &organization) {
		w.WriteHeader(http.StatusUnauthorized)

		return
	}

	var secret corev1.Secret
	err = h.Get(ctx, client.ObjectKey{Name: "credential-" + credentialName, Namespace: organization.Status.NamespaceRef}, &secret)
	if client.IgnoreNotFound(err) != nil {
		logger.Error("error getting secret", "err", err)
		w.WriteHeader(http.StatusInternalServerError)

		return
	}

	if apierrors.IsNotFound(err) || secret.Type != dockyardsv1.SecretTypeCredential {
		w.WriteHeader(http.StatusUnauthorized)

		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		logger.Error("error reading request body", "err", err)
		w.WriteHeader(http.StatusInternalServerError)

		return
	}

	defer r.Body.Close()

	var credential v1.Credential
	err = json.Unmarshal(body, &credential)
	if err != nil {
		logger.Error("error unmarshalling request body", "err", err, "body", body)
		w.WriteHeader(http.StatusUnprocessableEntity)

		return
	}

	if credential.Data == nil {
		w.WriteHeader(http.StatusNoContent)

		return
	}

	patch := client.MergeFrom(secret.DeepCopy())

	if secret.Data == nil {
		secret.Data = make(map[string][]byte)
	}

	for key, value := range *credential.Data {
		secret.Data[key] = value
	}

	err = h.Patch(ctx, &secret, patch)
	if err != nil {
		logger.Error("error patching secret", "err", err)
		w.WriteHeader(http.StatusInternalServerError)

		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *handler) GetCredentials(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	logger := middleware.LoggerFrom(ctx)

	subject, err := middleware.SubjectFrom(ctx)
	if err != nil {
		logger.Error("error getting subject", "err", err)
		w.WriteHeader(http.StatusInternalServerError)

		return
	}

	matchingFields := client.MatchingFields{
		index.MemberReferencesField: subject,
	}

	var organizationList dockyardsv1.OrganizationList
	err = h.List(ctx, &organizationList, matchingFields)
	if err != nil {
		logger.Error("error listing organizations", "err", err)
		w.WriteHeader(http.StatusInternalServerError)

		return
	}

	credentials := []v1.Credential{}

	for _, organization := range organizationList.Items {
		if organization.Status.NamespaceRef == "" {
			continue
		}

		var secretList corev1.SecretList
		err := h.List(ctx, &secretList, client.InNamespace(organization.Status.NamespaceRef))
		if err != nil {
			logger.Error("error listing secrets", "err", err)
			w.WriteHeader(http.StatusInternalServerError)

			return
		}

		for _, secret := range secretList.Items {
			if secret.Type != dockyardsv1.SecretTypeCredential {
				continue
			}

			credential := v1.Credential{
				ID:           string(secret.UID),
				Name:         strings.TrimPrefix(secret.Name, "credential-"),
				Organization: organization.Name,
			}

			credentialTemplate, has := secret.Labels[dockyardsv1.LabelCredentialTemplateName]
			if has {
				credential.CredentialTemplate = &credentialTemplate
			}

			credentials = append(credentials, credential)
		}
	}

	b, err := json.Marshal(&credentials)
	if err != nil {
		logger.Error("error marshalling credentials", "err", err)
		w.WriteHeader(http.StatusInternalServerError)

		return
	}

	w.WriteHeader(http.StatusOK)
	_, err = w.Write(b)
	if err != nil {
		logger.Error("error writing response data", "err", err)
	}
}
