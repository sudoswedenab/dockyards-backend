package handlers

import (
	"context"
	"net/http"

	"bitbucket.org/sudosweden/dockyards-backend/api/v1"
	"bitbucket.org/sudosweden/dockyards-backend/internal/util"
	"bitbucket.org/sudosweden/dockyards-backend/pkg/api/v1alpha1"
	"bitbucket.org/sudosweden/dockyards-backend/pkg/api/v1alpha1/index"
	"bitbucket.org/sudosweden/dockyards-backend/pkg/util/name"
	"github.com/gin-gonic/gin"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// +kubebuilder:rbac:groups=dockyards.io,resources=clusters,verbs=get;list;watch
// +kubebuilder:rbac:groups=dockyards.io,resources=nodepools,verbs=create;get;list;watch
// +kubebuilder:rbac:groups=dockyards.io,resources=nodes,verbs=get;list;watch
// +kubebuilder:rbac:groups=dockyards.io,resources=organizations,verbs=get;list;watch

func (h *handler) toV1NodePool(nodePool *v1alpha1.NodePool, cluster *v1alpha1.Cluster, nodeList *v1alpha1.NodeList) *v1.NodePool {
	v1NodePool := v1.NodePool{
		Id:         string(nodePool.UID),
		ClusterId:  string(cluster.UID),
		Name:       nodePool.Name,
		DiskSizeGb: int(nodePool.Status.Resources.Storage().Value() / 1024 / 1024 / 1024),
		CpuCount:   int(nodePool.Status.Resources.Cpu().Value()),
		RamSizeMb:  int(nodePool.Status.Resources.Memory().Value()),
	}

	if nodePool.Spec.Replicas != nil {
		v1NodePool.Quantity = int(*nodePool.Spec.Replicas)
	}

	if nodeList != nil && len(nodeList.Items) > 0 {
		nodes := make([]v1.Node, len(nodeList.Items))
		for i, node := range nodeList.Items {
			nodes[i] = v1.Node{
				Id:   string(node.UID),
				Name: node.Name,
			}
		}

		v1NodePool.Nodes = nodes
	}

	if nodePool.Spec.ControlPlane {
		v1NodePool.ControlPlane = &nodePool.Spec.ControlPlane
	}

	if nodePool.Spec.LoadBalancer {
		v1NodePool.LoadBalancer = &nodePool.Spec.LoadBalancer
	}

	if nodePool.Spec.DedicatedRole {
		v1NodePool.ControlPlaneComponentsOnly = &nodePool.Spec.DedicatedRole
	}

	return &v1NodePool
}

func (h *handler) GetNodePool(c *gin.Context) {
	ctx := context.Background()

	nodePoolID := c.Param("nodePoolID")

	matchingFields := client.MatchingFields{
		index.UIDIndexKey: nodePoolID,
	}

	var nodePoolList v1alpha1.NodePoolList
	err := h.controllerClient.List(ctx, &nodePoolList, matchingFields)
	if err != nil {
		h.logger.Error("error listing node pools in kubernetes", "err", err)

		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	if len(nodePoolList.Items) != 1 {
		h.logger.Debug("expected exactly one node pool", "length", len(nodePoolList.Items))

		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	nodePool := nodePoolList.Items[0]

	cluster, err := h.getOwnerCluster(ctx, &nodePool)
	if err != nil {
		h.logger.Error("error getting owner cluster", "err", err)

		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	organization, err := h.getOwnerOrganization(ctx, cluster)
	if err != nil {
		h.logger.Error("error getting owner cluster owner organization", "err", err)

		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	subject, err := h.getSubjectFromContext(c)
	if err != nil {
		h.logger.Error("error getting subject from context", "err", err)

		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	isMember := h.isMember(subject, organization)
	if !isMember {
		h.logger.Debug("user is not a member of organization", "subject", subject)

		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	matchingFields = client.MatchingFields{
		index.OwnerRefsIndexKey: nodePoolID,
	}

	var nodeList v1alpha1.NodeList
	err = h.controllerClient.List(ctx, &nodeList, matchingFields)
	if err != nil {
		h.logger.Error("error listing nodes", "err", err)

		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	v1NodePool := h.toV1NodePool(&nodePool, cluster, &nodeList)

	c.JSON(http.StatusOK, v1NodePool)
}

func (h *handler) PostClusterNodePools(c *gin.Context) {
	ctx := context.Background()

	clusterID := c.Param("clusterID")

	var nodePoolOptions v1.NodePoolOptions
	err := c.BindJSON(&nodePoolOptions)
	if err != nil {
		h.logger.Error("failed bind node pool options from request body", "err", err)

		c.AbortWithStatus(http.StatusUnprocessableEntity)
		return
	}

	details, isValidName := name.IsValidName(nodePoolOptions.Name)
	if !isValidName {
		h.logger.Error("node pool has invalid name", "name", nodePoolOptions.Name)

		c.AbortWithStatusJSON(http.StatusUnprocessableEntity, gin.H{
			"details": details,
		})
		return
	}

	if nodePoolOptions.Quantity > 9 {
		h.logger.Debug("quantity too large", "quantity", nodePoolOptions.Quantity)

		c.AbortWithStatusJSON(http.StatusUnprocessableEntity, gin.H{
			"error":    "node pool quota exceeded",
			"quantity": nodePoolOptions.Quantity,
			"details":  "quantity must be lower than 9",
		})
		return
	}

	matchingFields := client.MatchingFields{
		index.UIDIndexKey: clusterID,
	}

	var clusterList v1alpha1.ClusterList
	err = h.controllerClient.List(ctx, &clusterList, matchingFields)
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

		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	if organization == nil {
		h.logger.Debug("node pool has no organization owner")

		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	subject, err := h.getSubjectFromContext(c)
	if err != nil {
		h.logger.Error("error getting subject from context", "err", err)

		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	isMember := h.isMember(subject, organization)
	if !isMember {
		h.logger.Error("subject is not a member of organization")

		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	name := cluster.Name + "-" + nodePoolOptions.Name

	nodePool := v1alpha1.NodePool{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
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
		Spec: v1alpha1.NodePoolSpec{
			Replicas:  util.Ptr(int32(nodePoolOptions.Quantity)),
			Resources: corev1.ResourceList{},
		},
	}

	if nodePoolOptions.ControlPlane != nil {
		nodePool.Spec.ControlPlane = *nodePoolOptions.ControlPlane
	}

	if nodePoolOptions.LoadBalancer != nil {
		nodePool.Spec.LoadBalancer = *nodePoolOptions.LoadBalancer
	}

	if nodePoolOptions.ControlPlaneComponentsOnly != nil {
		nodePool.Spec.DedicatedRole = *nodePoolOptions.ControlPlaneComponentsOnly
	}

	err = h.controllerClient.Create(ctx, &nodePool)
	if err != nil {
		h.logger.Error("error creating node pool", "err", err)

		if apierrors.IsAlreadyExists(err) {
			c.AbortWithStatus(http.StatusConflict)
			return
		}

		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	v1NodePool := h.toV1NodePool(&nodePool, &cluster, nil)

	c.JSON(http.StatusCreated, v1NodePool)
}

func (h *handler) DeleteNodePool(c *gin.Context) {
	ctx := context.Background()

	nodePoolID := c.Param("nodePoolID")

	matchingFields := client.MatchingFields{
		index.UIDIndexKey: nodePoolID,
	}

	var nodePoolList v1alpha1.NodePoolList
	err := h.controllerClient.List(ctx, &nodePoolList, matchingFields)
	if err != nil {
		h.logger.Error("error listing node pools", "err", err)

		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	if len(nodePoolList.Items) != 1 {
		h.logger.Debug("expected exactly one node pool", "count", len(nodePoolList.Items))

		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	nodePool := nodePoolList.Items[0]

	cluster, err := h.getOwnerCluster(ctx, &nodePool)
	if err != nil {
		h.logger.Error("error getting owner cluster", "err", err)

		if apierrors.IsNotFound(err) {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	if cluster == nil {
		h.logger.Debug("node pool has no owner cluster")

		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	organization, err := h.getOwnerOrganization(ctx, cluster)
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
		h.logger.Debug("node pool has no owner organization")

		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	subject, err := h.getSubjectFromContext(c)
	if err != nil {
		h.logger.Error("error getting subject from context", "err", err)

		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	isMember := h.isMember(subject, organization)
	if !isMember {
		h.logger.Debug("subject is not a member of organization")

		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	deleteOptions := client.DeleteOptions{
		PropagationPolicy: util.Ptr(metav1.DeletePropagationBackground),
	}

	err = h.controllerClient.Delete(ctx, &nodePool, &deleteOptions)

	c.JSON(http.StatusNoContent, nil)
}
