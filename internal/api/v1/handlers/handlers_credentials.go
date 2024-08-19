package handlers

import (
	"context"
	"net/http"

	"bitbucket.org/sudosweden/dockyards-backend/internal/api/v1"
	"bitbucket.org/sudosweden/dockyards-backend/pkg/api/apiutil"
	dockyardsv1 "bitbucket.org/sudosweden/dockyards-backend/pkg/api/v1alpha2"
	"bitbucket.org/sudosweden/dockyards-backend/pkg/api/v1alpha2/index"
	"github.com/gin-gonic/gin"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	DockyardsSecretTypeCredential = "dockyards.io/credential"
)

// +kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch;create;delete

func (h *handler) GetOrgCredentials(c *gin.Context) {
	ctx := context.Background()

	subject, err := h.getSubjectFromContext(c)
	if err != nil {
		h.logger.Error("error getting subject from context", "err", err)
		c.AbortWithStatus(http.StatusInternalServerError)

		return
	}

	organizationID := c.Param("org")
	matchingFields := client.MatchingFields{
		index.UIDField: organizationID,
	}

	var organizationList dockyardsv1.OrganizationList
	err = h.controllerClient.List(ctx, &organizationList, matchingFields)
	if err != nil {
		h.logger.Error("error getting organizations from kubernetes", "err", err)
		c.AbortWithStatus(http.StatusInternalServerError)

		return
	}

	if len(organizationList.Items) != 1 {
		h.logger.Debug("expected exactly one organization", "count", len(organizationList.Items))
		c.AbortWithStatus(http.StatusInternalServerError)

		return
	}

	organization := organizationList.Items[0]

	if !h.isMember(subject, &organization) {
		c.AbortWithStatus(http.StatusUnauthorized)

		return
	}

	matchingFields = client.MatchingFields{
		index.SecretTypeField: DockyardsSecretTypeCredential,
	}

	var secretList corev1.SecretList
	err = h.controllerClient.List(ctx, &secretList, matchingFields)
	if err != nil {
		h.logger.Error("error listing secrets", "err", err)
		c.AbortWithStatus(http.StatusInternalServerError)

		return
	}

	v1Credentials := make([]v1.Credential, len(secretList.Items))

	for i, secret := range secretList.Items {
		v1Credentials[i] = v1.Credential{
			Id:           string(secret.UID),
			Name:         secret.Name,
			Organization: organization.Name,
		}
	}

	c.JSON(http.StatusOK, v1Credentials)
}

func (h *handler) PostOrgCredentials(c *gin.Context) {
	ctx := context.Background()

	organizationID := c.Param("org")
	if organizationID == "" {
		c.AbortWithStatus(http.StatusBadRequest)

		return
	}

	var credential v1.Credential
	err := c.BindJSON(&credential)
	if err != nil {
		h.logger.Error("error binding request json to credential", "err", err)
		c.AbortWithStatus(http.StatusUnprocessableEntity)

		return
	}

	matchingFields := client.MatchingFields{
		index.UIDField: organizationID,
	}

	var organizationList dockyardsv1.OrganizationList
	err = h.controllerClient.List(ctx, &organizationList, matchingFields)
	if err != nil {
		h.logger.Error("error listing organizations", "err", err)
		c.AbortWithStatus(http.StatusInternalServerError)

		return
	}

	if len(organizationList.Items) != 1 {
		h.logger.Debug("expected exactly one organization", "count", len(organizationList.Items))
		c.AbortWithStatus(http.StatusInternalServerError)

		return
	}

	organization := organizationList.Items[0]

	subject, err := h.getSubjectFromContext(c)
	if err != nil {
		h.logger.Error("error getting subject from context", "err", err)
		c.AbortWithStatus(http.StatusInternalServerError)

		return
	}

	if !h.isMember(subject, &organization) {
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

	err = h.controllerClient.Create(ctx, &secret)
	if err != nil {
		h.logger.Error("error creating secret", "err", err)
		c.AbortWithStatus(http.StatusInternalServerError)

		return
	}

	createdCredential := v1.Credential{
		Id:   string(secret.UID),
		Name: secret.Name,
	}

	c.JSON(http.StatusCreated, createdCredential)
}

func (h *handler) DeleteCredential(c *gin.Context) {
	ctx := context.Background()

	credentialID := c.Param("credentialID")

	matchingFields := client.MatchingFields{
		index.UIDField: credentialID,
	}

	var secretList corev1.SecretList
	err := h.controllerClient.List(ctx, &secretList, matchingFields)
	if err != nil {
		h.logger.Error("error listing secrets", "err", err)
		c.AbortWithStatus(http.StatusInternalServerError)

		return
	}

	if len(secretList.Items) != 1 {
		h.logger.Debug("expected exactly one secret", "count", len(secretList.Items))
		c.AbortWithStatus(http.StatusInternalServerError)

		return
	}

	secret := secretList.Items[0]

	organization, err := apiutil.GetOwnerOrganization(ctx, h.controllerClient, &secret)
	if err != nil {
		h.logger.Error("error getting owner organization", "err", err)
		c.AbortWithStatus(http.StatusInternalServerError)

		return
	}

	if organization == nil {
		h.logger.Debug("secret is not owned by organization")
		c.AbortWithStatus(http.StatusUnauthorized)

		return
	}

	subject, err := h.getSubjectFromContext(c)
	if err != nil {
		h.logger.Error("error getting subject from context", "err", err)
		c.AbortWithStatus(http.StatusInternalServerError)

		return
	}

	if !h.isMember(subject, organization) {
		c.AbortWithStatus(http.StatusUnauthorized)

		return
	}

	err = h.controllerClient.Delete(ctx, &secret)
	if err != nil {
		h.logger.Error("error deleting secret", "err", err)
		c.AbortWithStatus(http.StatusInternalServerError)

		return
	}

	c.Status(http.StatusNoContent)
}

func (h *handler) GetCredential(c *gin.Context) {
	ctx := context.Background()

	credentialID := c.Param("credentialID")

	matchingFields := client.MatchingFields{
		index.UIDField: credentialID,
	}

	var secretList corev1.SecretList
	err := h.controllerClient.List(ctx, &secretList, matchingFields)
	if err != nil {
		h.logger.Error("error listing secrets", "err", err)
		c.AbortWithStatus(http.StatusInternalServerError)

		return
	}

	if len(secretList.Items) != 1 {
		h.logger.Debug("expected exactly one secret", "count", len(secretList.Items))
		c.AbortWithStatus(http.StatusInternalServerError)

		return
	}

	secret := secretList.Items[0]

	organization, err := apiutil.GetOwnerOrganization(ctx, h.controllerClient, &secret)
	if err != nil {
		h.logger.Error("error getting owner organization", "err", err)
		c.AbortWithStatus(http.StatusInternalServerError)

		return
	}

	if organization == nil {
		h.logger.Debug("secret is not owned by organization")
		c.AbortWithStatus(http.StatusUnauthorized)

		return
	}

	subject, err := h.getSubjectFromContext(c)
	if err != nil {
		h.logger.Error("error getting subject from context", "err", err)
		c.AbortWithStatus(http.StatusInternalServerError)

		return
	}

	if !h.isMember(subject, organization) {
		c.AbortWithStatus(http.StatusUnauthorized)

		return
	}

	v1Credential := v1.Credential{
		Id:           string(secret.UID),
		Name:         secret.Name,
		Organization: organization.Name,
	}

	c.JSON(http.StatusOK, v1Credential)
}
