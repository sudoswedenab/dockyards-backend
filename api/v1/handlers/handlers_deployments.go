package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"bitbucket.org/sudosweden/dockyards-backend/api/v1"
	"bitbucket.org/sudosweden/dockyards-backend/internal/util"
	"bitbucket.org/sudosweden/dockyards-backend/pkg/api/v1alpha1"
	"bitbucket.org/sudosweden/dockyards-backend/pkg/api/v1alpha1/index"
	utildeployment "bitbucket.org/sudosweden/dockyards-backend/pkg/util/deployment"
	"bitbucket.org/sudosweden/dockyards-backend/pkg/util/name"
	"github.com/gin-gonic/gin"
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

func (h *handler) PostClusterDeployments(c *gin.Context) {
	ctx := context.Background()

	clusterID := c.Param("clusterID")
	if clusterID == "" {
		h.logger.Debug("cluster empty")

		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	matchingFields := client.MatchingFields{
		index.UIDIndexKey: clusterID,
	}

	var clusterList v1alpha1.ClusterList
	err := h.controllerClient.List(ctx, &clusterList, matchingFields)
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

	deployment := v1alpha1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cluster.Name + "-" + *v1Deployment.Name,
			Namespace: cluster.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: v1alpha1.GroupVersion.String(),
					Kind:       v1alpha1.ClusterKind,
					Name:       cluster.Name,
					UID:        cluster.UID,
				},
			},
		},
		Spec: v1alpha1.DeploymentSpec{
			TargetNamespace: *v1Deployment.Namespace,
		},
	}

	err = h.controllerClient.Create(ctx, &deployment)
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
		containerImageDeployment := v1alpha1.ContainerImageDeployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      deployment.Name,
				Namespace: deployment.Namespace,
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion: v1alpha1.GroupVersion.String(),
						Kind:       v1alpha1.DeploymentKind,
						Name:       deployment.Name,
						UID:        deployment.UID,
					},
				},
			},
			Spec: v1alpha1.ContainerImageDeploymentSpec{
				Image: *v1Deployment.ContainerImage,
			},
		}

		if v1Deployment.Port != nil {
			containerImageDeployment.Spec.Port = int32(*v1Deployment.Port)
		}

		err := h.controllerClient.Create(ctx, &containerImageDeployment)
		if err != nil {
			h.logger.Error("error creating container image deployment", "err", err)

			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		deployment.Spec.DeploymentRef = v1alpha1.DeploymentReference{
			APIVersion: v1alpha1.GroupVersion.String(),
			Kind:       v1alpha1.ContainerImageDeploymentKind,
			Name:       containerImageDeployment.Name,
		}

		createdDeployment.Type = v1.DeploymentTypeContainerImage
		createdDeployment.ContainerImage = &containerImageDeployment.Spec.Image
	}

	if v1Deployment.Kustomize != nil {
		kustomizeDeployment := v1alpha1.KustomizeDeployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      deployment.Name,
				Namespace: deployment.Namespace,
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion: v1alpha1.GroupVersion.String(),
						Kind:       v1alpha1.DeploymentKind,
						Name:       deployment.Name,
						UID:        deployment.UID,
					},
				},
			},
			Spec: v1alpha1.KustomizeDeploymentSpec{
				Kustomize: *v1Deployment.Kustomize,
			},
		}

		err := h.controllerClient.Create(ctx, &kustomizeDeployment)
		if err != nil {
			h.logger.Error("error creating kustomize deployment", "err", err)

			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		deployment.Spec.DeploymentRef = v1alpha1.DeploymentReference{
			APIVersion: v1alpha1.GroupVersion.String(),
			Kind:       v1alpha1.KustomizeDeploymentKind,
			Name:       kustomizeDeployment.Name,
		}

		createdDeployment.Type = v1.DeploymentTypeKustomize
		createdDeployment.Kustomize = &kustomizeDeployment.Spec.Kustomize
	}

	if v1Deployment.HelmChart != nil {
		helmDeployment := v1alpha1.HelmDeployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      deployment.Name,
				Namespace: deployment.Namespace,
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion: v1alpha1.GroupVersion.String(),
						Kind:       v1alpha1.DeploymentKind,
						Name:       deployment.Name,
						UID:        deployment.UID,
					},
				},
			},
			Spec: v1alpha1.HelmDeploymentSpec{
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

		err := h.controllerClient.Create(ctx, &helmDeployment)
		if err != nil {
			h.logger.Error("error creating helm deployment", "err", err)

			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		deployment.Spec.DeploymentRef = v1alpha1.DeploymentReference{
			APIVersion: v1alpha1.GroupVersion.String(),
			Kind:       v1alpha1.HelmDeploymentKind,
			Name:       helmDeployment.Name,
		}

		createdDeployment.Type = v1.DeploymentTypeHelm
		createdDeployment.HelmChart = &helmDeployment.Spec.Chart
		createdDeployment.HelmRepository = &helmDeployment.Spec.Repository
		createdDeployment.HelmVersion = &helmDeployment.Spec.Version

	}

	err = h.controllerClient.Patch(ctx, &deployment, patch)
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
		index.UIDIndexKey: clusterID,
	}

	var clusterList v1alpha1.ClusterList
	err := h.controllerClient.List(ctx, &clusterList, matchingFields)
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

	organization, err := h.getOwnerOrganization(ctx, &cluster)
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
		index.OwnerRefsIndexKey: string(cluster.UID),
	}

	var deploymentList v1alpha1.DeploymentList
	err = h.controllerClient.List(ctx, &deploymentList, matchingFields)
	if err != nil {
		h.logger.Error("error listing deployments", "err", err)

		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	deployments := make([]v1.Deployment, len(deploymentList.Items))
	for i, deployment := range deploymentList.Items {
		name := strings.TrimPrefix(deployment.Name, cluster.Name+"-")

		var deploymentType v1.DeploymentType
		switch deployment.Spec.DeploymentRef.Kind {
		case v1alpha1.ContainerImageDeploymentKind:
			deploymentType = v1.DeploymentTypeContainerImage
		case v1alpha1.HelmDeploymentKind:
			deploymentType = v1.DeploymentTypeHelm
		case v1alpha1.KustomizeDeploymentKind:
			deploymentType = v1.DeploymentTypeKustomize
		}

		deployments[i] = v1.Deployment{
			Id:        string(deployment.UID),
			ClusterId: string(cluster.UID),
			Name:      &name,
			Type:      deploymentType,
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
		index.UIDIndexKey: deploymentID,
	}

	var deploymentList v1alpha1.DeploymentList
	err := h.controllerClient.List(ctx, &deploymentList, matchingFields)
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

	err = h.controllerClient.Delete(ctx, &deployment)
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
		index.UIDIndexKey: deploymentID,
	}

	var deploymentList v1alpha1.DeploymentList
	err := h.controllerClient.List(ctx, &deploymentList, matchingFields)
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
		Id:   string(deployment.UID),
		Name: &deployment.Name,
	}

	objectKey := client.ObjectKey{
		Name:      deployment.Spec.DeploymentRef.Name,
		Namespace: deployment.Namespace,
	}

	switch deployment.Spec.DeploymentRef.Kind {
	case v1alpha1.ContainerImageDeploymentKind:
		var containerImageDeployment v1alpha1.ContainerImageDeployment
		err := h.controllerClient.Get(ctx, objectKey, &containerImageDeployment)
		if err != nil {
			h.logger.Error("error getting container image deployment", "err", err)

			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		v1Deployment.ContainerImage = &containerImageDeployment.Spec.Image

		if containerImageDeployment.Spec.Port != 0 {
			v1Deployment.Port = util.Ptr(int(containerImageDeployment.Spec.Port))
		}
	case v1alpha1.HelmDeploymentKind:
		var helmDeployment v1alpha1.HelmDeployment
		err := h.controllerClient.Get(ctx, objectKey, &helmDeployment)
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
	case v1alpha1.KustomizeDeploymentKind:
		var kustomizeDeployment v1alpha1.KustomizeDeployment
		err := h.controllerClient.Get(ctx, objectKey, &kustomizeDeployment)
		if err != nil {
			h.logger.Error("error getting kustomize deployment", "err", err)

			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		v1Deployment.Kustomize = &kustomizeDeployment.Spec.Kustomize
	default:
		h.logger.Error("deployment has unsupported deployment kind", "kind", deployment.Spec.DeploymentRef.Kind)

		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	readyCondition := meta.FindStatusCondition(deployment.Status.Conditions, v1alpha1.ReadyCondition)
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
