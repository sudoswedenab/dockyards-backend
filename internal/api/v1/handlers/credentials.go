package handlers

import (
	"encoding/json"
	"io"
	"net/http"

	"bitbucket.org/sudosweden/dockyards-backend/internal/api/v1"
	"bitbucket.org/sudosweden/dockyards-backend/internal/api/v1/middleware"
	"bitbucket.org/sudosweden/dockyards-backend/pkg/api/apiutil"
	dockyardsv1 "bitbucket.org/sudosweden/dockyards-backend/pkg/api/v1alpha2"
	"bitbucket.org/sudosweden/dockyards-backend/pkg/api/v1alpha2/index"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	DockyardsSecretTypeCredential = "dockyards.io/credential"
)

// +kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch;create;delete

func (h *handler) GetOrgCredentials(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	logger := middleware.LoggerFrom(ctx)

	subject, err := middleware.SubjectFrom(ctx)
	if err != nil {
		logger.Error("error getting subject from context", "err", err)
		w.WriteHeader(http.StatusInternalServerError)

		return
	}

	organizationID := r.PathValue("organizationID")
	if organizationID == "" {
		logger.Error("empty organizationID")
		w.WriteHeader(http.StatusBadRequest)

		return
	}

	matchingFields := client.MatchingFields{
		index.UIDField: organizationID,
	}

	var organizationList dockyardsv1.OrganizationList
	err = h.List(ctx, &organizationList, matchingFields)
	if err != nil {
		logger.Error("error getting organizations from kubernetes", "err", err)
		w.WriteHeader(http.StatusInternalServerError)

		return
	}

	if len(organizationList.Items) != 1 {
		logger.Debug("expected exactly one organization", "count", len(organizationList.Items))
		w.WriteHeader(http.StatusInternalServerError)

		return
	}

	organization := organizationList.Items[0]

	if !h.isMember(subject, &organization) {
		w.WriteHeader(http.StatusUnauthorized)

		return
	}

	matchingFields = client.MatchingFields{
		index.SecretTypeField: DockyardsSecretTypeCredential,
	}

	var secretList corev1.SecretList
	err = h.List(ctx, &secretList, matchingFields)
	if err != nil {
		logger.Error("error listing secrets", "err", err)
		w.WriteHeader(http.StatusInternalServerError)

		return
	}

	v1Credentials := make([]v1.Credential, len(secretList.Items))

	for i, secret := range secretList.Items {
		v1Credentials[i] = v1.Credential{
			ID:           string(secret.UID),
			Name:         secret.Name,
			Organization: organization.Name,
		}
	}

	b, err := json.Marshal(&v1Credentials)
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

func (h *handler) PostOrgCredentials(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	logger := middleware.LoggerFrom(ctx)

	organizationID := r.PathValue("organizationID")
	if organizationID == "" {
		w.WriteHeader(http.StatusBadRequest)

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

	matchingFields := client.MatchingFields{
		index.UIDField: organizationID,
	}

	var organizationList dockyardsv1.OrganizationList
	err = h.List(ctx, &organizationList, matchingFields)
	if err != nil {
		logger.Error("error listing organizations", "err", err)
		w.WriteHeader(http.StatusInternalServerError)

		return
	}

	if len(organizationList.Items) != 1 {
		logger.Debug("expected exactly one organization", "count", len(organizationList.Items))
		w.WriteHeader(http.StatusInternalServerError)

		return
	}

	organization := organizationList.Items[0]

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
			Name:      credential.Name,
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

func (h *handler) DeleteCredential(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	logger := middleware.LoggerFrom(ctx)

	credentialID := r.PathValue("credentialID")
	if credentialID == "" {
		w.WriteHeader(http.StatusBadRequest)

		return
	}

	matchingFields := client.MatchingFields{
		index.UIDField: credentialID,
	}

	var secretList corev1.SecretList
	err := h.List(ctx, &secretList, matchingFields)
	if err != nil {
		logger.Error("error listing secrets", "err", err)
		w.WriteHeader(http.StatusInternalServerError)

		return
	}

	if len(secretList.Items) != 1 {
		logger.Debug("expected exactly one secret", "count", len(secretList.Items))
		w.WriteHeader(http.StatusInternalServerError)

		return
	}

	secret := secretList.Items[0]

	organization, err := apiutil.GetOwnerOrganization(ctx, h.Client, &secret)
	if err != nil {
		logger.Error("error getting owner organization", "err", err)
		w.WriteHeader(http.StatusInternalServerError)

		return
	}

	if organization == nil {
		logger.Debug("secret is not owned by organization")
		w.WriteHeader(http.StatusUnauthorized)

		return
	}

	subject, err := middleware.SubjectFrom(ctx)
	if err != nil {
		logger.Error("error getting subject from context", "err", err)
		w.WriteHeader(http.StatusInternalServerError)

		return
	}

	if !h.isMember(subject, organization) {
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

func (h *handler) GetCredential(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	logger := middleware.LoggerFrom(ctx)

	credentialID := r.PathValue("credentialID")
	if credentialID == "" {
		w.WriteHeader(http.StatusBadRequest)

		return
	}

	matchingFields := client.MatchingFields{
		index.UIDField: credentialID,
	}

	var secretList corev1.SecretList
	err := h.List(ctx, &secretList, matchingFields)
	if err != nil {
		logger.Error("error listing secrets", "err", err)
		w.WriteHeader(http.StatusInternalServerError)

		return
	}

	if len(secretList.Items) != 1 {
		logger.Debug("expected exactly one secret", "count", len(secretList.Items))
		w.WriteHeader(http.StatusInternalServerError)

		return
	}

	secret := secretList.Items[0]

	organization, err := apiutil.GetOwnerOrganization(ctx, h.Client, &secret)
	if err != nil {
		logger.Error("error getting owner organization", "err", err)
		w.WriteHeader(http.StatusInternalServerError)

		return
	}

	if organization == nil {
		logger.Debug("secret is not owned by organization")
		w.WriteHeader(http.StatusUnauthorized)

		return
	}

	subject, err := middleware.SubjectFrom(ctx)
	if err != nil {
		logger.Error("error getting subject from context", "err", err)
		w.WriteHeader(http.StatusInternalServerError)

		return
	}

	if !h.isMember(subject, organization) {
		w.WriteHeader(http.StatusUnauthorized)

		return
	}

	v1Credential := v1.Credential{
		ID:           string(secret.UID),
		Name:         secret.Name,
		Organization: organization.Name,
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
