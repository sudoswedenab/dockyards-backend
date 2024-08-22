package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"bitbucket.org/sudosweden/dockyards-backend/internal/api/v1"
	"bitbucket.org/sudosweden/dockyards-backend/internal/util"
	"bitbucket.org/sudosweden/dockyards-backend/pkg/api/apiutil"
	dockyardsv1 "bitbucket.org/sudosweden/dockyards-backend/pkg/api/v1alpha2"
	"bitbucket.org/sudosweden/dockyards-backend/pkg/api/v1alpha2/index"
	utildeployment "bitbucket.org/sudosweden/dockyards-backend/pkg/util/deployment"
	"bitbucket.org/sudosweden/dockyards-backend/pkg/util/name"
	"github.com/gin-gonic/gin"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// +kubebuilder:rbac:groups=dockyards.io,resources=deployments,verbs=create;delete;get;list;watch;patch
// +kubebuilder:rbac:groups=dockyards.io,resources=containerimagedeployments,verbs=get;list;watch;create;update;patch
// +kubebuilder:rbac:groups=dockyards.io,resources=helmdeployments,verbs=get;list;watch;create;update;patch
// +kubebuilder:rbac:groups=dockyards.io,resources=kustomizedeployments,verbs=get;list;watch;create;update;patch
// +kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch

func (h *handler) PostClusterDeployments(c *gin.Context) {
	ctx := context.Background()

	clusterID := c.Param("clusterID")
	if clusterID == "" {
		h.logger.Debug("cluster empty")
		c.AbortWithStatus(http.StatusBadRequest)

		return
	}

	matchingFields := client.MatchingFields{
		index.UIDField: clusterID,
	}

	var clusterList dockyardsv1.ClusterList
	err := h.List(ctx, &clusterList, matchingFields)
	if err != nil {
		h.logger.Error("error listing clusters", "err", err)
		c.AbortWithStatus(http.StatusInternalServerError)

		return
	}

	if len(clusterList.Items) != 1 {
		h.logger.Debug("expected exactly one cluster", "count", len(clusterList.Items))

		if len(clusterList.Items) == 0 {
			c.AbortWithStatus(http.StatusUnauthorized)

			return
		}

		c.AbortWithStatus(http.StatusInternalServerError)

		return
	}

	cluster := clusterList.Items[0]

	var v1Deployment v1.Deployment
	err = c.BindJSON(&v1Deployment)
	if err != nil {
		h.logger.Error("failed to read body", "err", err)
		c.AbortWithStatus(http.StatusUnprocessableEntity)

		return
	}

	v1Deployment.ClusterId = string(cluster.UID)

	err = utildeployment.AddNormalizedName(&v1Deployment)
	if err != nil {
		h.logger.Error("error adding deployment name", "err", err)
		c.AbortWithStatus(http.StatusUnprocessableEntity)

		return
	}

	details, validName := name.IsValidName(*v1Deployment.Name)
	if !validName {
		c.JSON(http.StatusUnprocessableEntity, gin.H{
			"error":   "name is not valid",
			"name":    v1Deployment.Name,
			"details": details,
		})

		return
	}

	var credentialRef *corev1.LocalObjectReference

	if v1Deployment.CredentialName != nil {
		objectKey := client.ObjectKey{
			Name:      *v1Deployment.CredentialName,
			Namespace: cluster.Namespace,
		}

		var secret corev1.Secret
		err := h.Get(ctx, objectKey, &secret)
		if client.IgnoreNotFound(err) != nil {
			h.logger.Error("error listing secrets", "err", err)
			c.AbortWithStatus(http.StatusInternalServerError)

			return
		}

		if apierrors.IsNotFound(err) {
			c.AbortWithStatus(http.StatusForbidden)

			return
		}

		if secret.Type != DockyardsSecretTypeCredential {
			c.AbortWithStatus(http.StatusForbidden)

			return
		}

		credentialRef = &corev1.LocalObjectReference{
			Name: secret.Name,
		}
	}

	deployment := dockyardsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cluster.Name + "-" + *v1Deployment.Name,
			Namespace: cluster.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: dockyardsv1.GroupVersion.String(),
					Kind:       dockyardsv1.ClusterKind,
					Name:       cluster.Name,
					UID:        cluster.UID,
				},
			},
			Labels: map[string]string{
				dockyardsv1.LabelClusterName: cluster.Name,
			},
		},
		Spec: dockyardsv1.DeploymentSpec{
			Provenience:     dockyardsv1.ProvenienceUser,
			TargetNamespace: *v1Deployment.Namespace,
		},
	}

	err = h.Create(ctx, &deployment)
	if err != nil {
		h.logger.Error("error creating deployment", "err", err)

		if apierrors.IsAlreadyExists(err) {
			c.AbortWithStatus(http.StatusConflict)

			return
		}

		c.AbortWithStatus(http.StatusInternalServerError)

		return
	}

	patch := client.MergeFrom(deployment.DeepCopy())

	createdDeployment := v1.Deployment{
		Id:        string(deployment.UID),
		ClusterId: string(cluster.UID),
		Name:      util.Ptr(strings.TrimPrefix(deployment.Name, cluster.Name+"-")),
		Namespace: &deployment.Spec.TargetNamespace,
	}

	if v1Deployment.ContainerImage != nil {
		containerImageDeployment := dockyardsv1.ContainerImageDeployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      deployment.Name,
				Namespace: deployment.Namespace,
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion: dockyardsv1.GroupVersion.String(),
						Kind:       dockyardsv1.DeploymentKind,
						Name:       deployment.Name,
						UID:        deployment.UID,
					},
				},
				Labels: map[string]string{
					dockyardsv1.LabelClusterName: cluster.Name,
				},
			},
			Spec: dockyardsv1.ContainerImageDeploymentSpec{
				Image: *v1Deployment.ContainerImage,
			},
		}

		if v1Deployment.Port != nil {
			containerImageDeployment.Spec.Port = int32(*v1Deployment.Port)
		}

		if credentialRef != nil {
			containerImageDeployment.Spec.CredentialRef = credentialRef
		}

		err := h.Create(ctx, &containerImageDeployment)
		if err != nil {
			h.logger.Error("error creating container image deployment", "err", err)
			c.AbortWithStatus(http.StatusInternalServerError)

			return
		}

		deployment.Spec.DeploymentRefs = []corev1.TypedLocalObjectReference{
			{
				APIGroup: &dockyardsv1.GroupVersion.Group,
				Kind:     dockyardsv1.ContainerImageDeploymentKind,
				Name:     containerImageDeployment.Name,
			},
		}

		createdDeployment.Type = v1.DeploymentTypeContainerImage
		createdDeployment.ContainerImage = &containerImageDeployment.Spec.Image
	}

	if v1Deployment.Kustomize != nil {
		kustomizeDeployment := dockyardsv1.KustomizeDeployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      deployment.Name,
				Namespace: deployment.Namespace,
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion: dockyardsv1.GroupVersion.String(),
						Kind:       dockyardsv1.DeploymentKind,
						Name:       deployment.Name,
						UID:        deployment.UID,
					},
				},
				Labels: map[string]string{
					dockyardsv1.LabelClusterName: cluster.Name,
				},
			},
			Spec: dockyardsv1.KustomizeDeploymentSpec{
				Kustomize: *v1Deployment.Kustomize,
			},
		}

		err := h.Create(ctx, &kustomizeDeployment)
		if err != nil {
			h.logger.Error("error creating kustomize deployment", "err", err)
			c.AbortWithStatus(http.StatusInternalServerError)

			return
		}

		deployment.Spec.DeploymentRefs = []corev1.TypedLocalObjectReference{
			{
				APIGroup: &dockyardsv1.GroupVersion.Group,
				Kind:     dockyardsv1.KustomizeDeploymentKind,
				Name:     kustomizeDeployment.Name,
			},
		}

		createdDeployment.Type = v1.DeploymentTypeKustomize
		createdDeployment.Kustomize = &kustomizeDeployment.Spec.Kustomize
	}

	if v1Deployment.HelmChart != nil {
		helmDeployment := dockyardsv1.HelmDeployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      deployment.Name,
				Namespace: deployment.Namespace,
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion: dockyardsv1.GroupVersion.String(),
						Kind:       dockyardsv1.DeploymentKind,
						Name:       deployment.Name,
						UID:        deployment.UID,
					},
				},
				Labels: map[string]string{
					dockyardsv1.LabelClusterName: cluster.Name,
				},
			},
			Spec: dockyardsv1.HelmDeploymentSpec{
				Chart:      *v1Deployment.HelmChart,
				Repository: *v1Deployment.HelmRepository,
				Version:    *v1Deployment.HelmVersion,
			},
		}

		if v1Deployment.HelmValues != nil {
			b, err := json.Marshal(*v1Deployment.HelmValues)
			if err != nil {
				h.logger.Error("error marshalling helm values", "err", err)
				c.AbortWithStatus(http.StatusInternalServerError)

				return
			}

			helmDeployment.Spec.Values = &apiextensionsv1.JSON{
				Raw: b,
			}
		}

		err := h.Create(ctx, &helmDeployment)
		if err != nil {
			h.logger.Error("error creating helm deployment", "err", err)
			c.AbortWithStatus(http.StatusInternalServerError)

			return
		}

		deployment.Spec.DeploymentRefs = []corev1.TypedLocalObjectReference{
			{
				APIGroup: &dockyardsv1.GroupVersion.Group,
				Kind:     dockyardsv1.HelmDeploymentKind,
				Name:     helmDeployment.Name,
			},
		}

		createdDeployment.Type = v1.DeploymentTypeHelm
		createdDeployment.HelmChart = &helmDeployment.Spec.Chart
		createdDeployment.HelmRepository = &helmDeployment.Spec.Repository
		createdDeployment.HelmVersion = &helmDeployment.Spec.Version
	}

	err = h.Patch(ctx, &deployment, patch)
	if err != nil {
		h.logger.Error("error patching deployment", "err", err)
		c.AbortWithStatus(http.StatusInternalServerError)

		return
	}

	c.JSON(http.StatusCreated, createdDeployment)
}

func (h *handler) GetClusterDeployments(c *gin.Context) {
	ctx := context.Background()

	clusterID := c.Param("clusterID")

	matchingFields := client.MatchingFields{
		index.UIDField: clusterID,
	}

	var clusterList dockyardsv1.ClusterList
	err := h.List(ctx, &clusterList, matchingFields)
	if err != nil {
		h.logger.Error("error listing clusters", "err", err)
		c.AbortWithStatus(http.StatusInternalServerError)

		return
	}

	if len(clusterList.Items) != 1 {
		h.logger.Debug("expected exactly one cluster", "count", len(clusterList.Items))
		c.AbortWithStatus(http.StatusUnauthorized)

		return
	}

	cluster := clusterList.Items[0]

	organization, err := apiutil.GetOwnerOrganization(ctx, h.Client, &cluster)
	if err != nil {
		h.logger.Error("error getting owner organization", "err", err)

		if apierrors.IsNotFound(err) {
			c.AbortWithStatus(http.StatusUnauthorized)

			return
		}

		c.AbortWithStatus(http.StatusInternalServerError)

		return
	}

	if organization == nil {
		h.logger.Debug("cluster has no owner organization")
		c.AbortWithStatus(http.StatusUnauthorized)

		return
	}

	matchingFields = client.MatchingFields{
		index.OwnerReferencesField: string(cluster.UID),
	}

	var deploymentList dockyardsv1.DeploymentList
	err = h.List(ctx, &deploymentList, matchingFields)
	if err != nil {
		h.logger.Error("error listing deployments", "err", err)
		c.AbortWithStatus(http.StatusInternalServerError)

		return
	}

	deployments := make([]v1.Deployment, len(deploymentList.Items))
	for i, deployment := range deploymentList.Items {
		name := strings.TrimPrefix(deployment.Name, cluster.Name+"-")

		var deploymentType v1.DeploymentType

		if len(deployment.Spec.DeploymentRefs) > 0 {
			switch deployment.Spec.DeploymentRefs[0].Kind {
			case dockyardsv1.ContainerImageDeploymentKind:
				deploymentType = v1.DeploymentTypeContainerImage
			case dockyardsv1.HelmDeploymentKind:
				deploymentType = v1.DeploymentTypeHelm
			case dockyardsv1.KustomizeDeploymentKind:
				deploymentType = v1.DeploymentTypeKustomize
			}
		}

		deployments[i] = v1.Deployment{
			Id:          string(deployment.UID),
			ClusterId:   string(cluster.UID),
			Provenience: &deployment.Spec.Provenience,
			Name:        &name,
			Type:        deploymentType,
		}
	}

	c.JSON(http.StatusOK, deployments)
}

func (h *handler) DeleteDeployment(c *gin.Context) {
	ctx := context.Background()

	deploymentID := c.Param("deploymentID")
	if deploymentID == "" {
		h.logger.Debug("deployment id empty")
		c.AbortWithStatus(http.StatusBadRequest)

		return
	}

	matchingFields := client.MatchingFields{
		index.UIDField: deploymentID,
	}

	var deploymentList dockyardsv1.DeploymentList
	err := h.List(ctx, &deploymentList, matchingFields)
	if err != nil {
		h.logger.Error("error listing deployments", "err", err)
		c.AbortWithStatus(http.StatusInternalServerError)

		return
	}

	if len(deploymentList.Items) != 1 {
		h.logger.Debug("expected exactly one deployment", "count", len(deploymentList.Items))

		if len(deploymentList.Items) == 0 {
			c.AbortWithStatus(http.StatusUnauthorized)

			return
		}

		c.AbortWithStatus(http.StatusInternalServerError)

		return
	}

	deployment := deploymentList.Items[0]

	err = h.Delete(ctx, &deployment)
	if err != nil {
		h.logger.Error("error deleting deployment", "err", err)
		c.AbortWithStatus(http.StatusInternalServerError)

		return
	}

	h.logger.Debug("deleted deployment", "id", deployment.UID)

	c.Status(http.StatusNoContent)
}

func (h *handler) GetDeployment(c *gin.Context) {
	ctx := context.Background()

	deploymentID := c.Param("deploymentID")

	matchingFields := client.MatchingFields{
		index.UIDField: deploymentID,
	}

	var deploymentList dockyardsv1.DeploymentList
	err := h.List(ctx, &deploymentList, matchingFields)
	if err != nil {
		h.logger.Error("error listing deployment", "err", err)
		c.AbortWithStatus(http.StatusInternalServerError)

		return
	}

	if len(deploymentList.Items) != 1 {
		h.logger.Debug("expected exactly one deployment", "count", len(deploymentList.Items))

		if len(deploymentList.Items) == 0 {
			c.AbortWithStatus(http.StatusUnauthorized)

			return
		}

		c.AbortWithStatus(http.StatusInternalServerError)

		return
	}

	deployment := deploymentList.Items[0]

	v1Deployment := v1.Deployment{
		Id:          string(deployment.UID),
		Provenience: &deployment.Spec.Provenience,
		Name:        &deployment.Name,
	}

	objectKey := client.ObjectKey{
		Name:      deployment.Spec.DeploymentRefs[0].Name,
		Namespace: deployment.Namespace,
	}

	switch deployment.Spec.DeploymentRefs[0].Kind {
	case dockyardsv1.ContainerImageDeploymentKind:
		var containerImageDeployment dockyardsv1.ContainerImageDeployment
		err := h.Get(ctx, objectKey, &containerImageDeployment)
		if err != nil {
			h.logger.Error("error getting container image deployment", "err", err)
			c.AbortWithStatus(http.StatusInternalServerError)

			return
		}

		v1Deployment.ContainerImage = &containerImageDeployment.Spec.Image

		if containerImageDeployment.Spec.Port != 0 {
			v1Deployment.Port = util.Ptr(int(containerImageDeployment.Spec.Port))
		}
	case dockyardsv1.HelmDeploymentKind:
		var helmDeployment dockyardsv1.HelmDeployment
		err := h.Get(ctx, objectKey, &helmDeployment)
		if err != nil {
			h.logger.Error("error getting helm deployment", "err", err)
			c.AbortWithStatus(http.StatusInternalServerError)

			return
		}

		v1Deployment.HelmChart = &helmDeployment.Spec.Chart
		v1Deployment.HelmRepository = &helmDeployment.Spec.Repository
		v1Deployment.HelmVersion = &helmDeployment.Spec.Version

		if helmDeployment.Spec.Values != nil {
			var values map[string]any
			err = json.Unmarshal(helmDeployment.Spec.Values.Raw, &values)
			if err != nil {
				h.logger.Error("error unmarshalling helm values", "err", err)
				c.AbortWithStatus(http.StatusInternalServerError)

				return
			}

			v1Deployment.HelmValues = &values
		}
	case dockyardsv1.KustomizeDeploymentKind:
		var kustomizeDeployment dockyardsv1.KustomizeDeployment
		err := h.Get(ctx, objectKey, &kustomizeDeployment)
		if err != nil {
			h.logger.Error("error getting kustomize deployment", "err", err)
			c.AbortWithStatus(http.StatusInternalServerError)

			return
		}

		v1Deployment.Kustomize = &kustomizeDeployment.Spec.Kustomize
	default:
		h.logger.Error("deployment has unsupported deployment kind", "kind", deployment.Spec.DeploymentRefs[0].Kind)
		c.AbortWithStatus(http.StatusInternalServerError)

		return
	}

	readyCondition := meta.FindStatusCondition(deployment.Status.Conditions, dockyardsv1.ReadyCondition)
	if readyCondition != nil {
		health := v1.DeploymentStatusHealthWarning
		if readyCondition.Status == metav1.ConditionTrue {
			health = v1.DeploymentStatusHealthHealthy
		}

		v1Deployment.Status = &v1.DeploymentStatus{
			CreatedAt: readyCondition.LastTransitionTime.Time,
			Health:    &health,
			State:     &readyCondition.Message,
		}
	}

	c.JSON(http.StatusOK, v1Deployment)
}
