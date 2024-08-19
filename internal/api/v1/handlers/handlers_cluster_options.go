package handlers

import (
	"context"
	"net/http"

	"bitbucket.org/sudosweden/dockyards-backend/internal/api/v1"
	"bitbucket.org/sudosweden/dockyards-backend/internal/util"
	dockyardsv1 "bitbucket.org/sudosweden/dockyards-backend/pkg/api/v1alpha2"
	"github.com/gin-gonic/gin"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// +kubebuilder:rbac:groups=dockyards.io,resources=clustertemplates,verbs=get;list;watch
// +kubebuilder:rbac:groups=dockyards.io,resources=releases,verbs=get;list;watch

func getRecommendedNodePools(clusterTemplate *dockyardsv1.ClusterTemplate) []v1.NodePoolOptions {
	if clusterTemplate == nil {
		return []v1.NodePoolOptions{}
	}

	nodePoolOptions := make([]v1.NodePoolOptions, len(clusterTemplate.Spec.NodePoolTemplates))

	for i, nodePoolTemplate := range clusterTemplate.Spec.NodePoolTemplates {
		nodePoolTemplate := nodePoolTemplate

		quantity := 1
		if nodePoolTemplate.Spec.Replicas != nil {
			quantity = int(*nodePoolTemplate.Spec.Replicas)
		}

		nodePoolOptions[i] = v1.NodePoolOptions{
			Name:     nodePoolTemplate.Name,
			Quantity: quantity,
		}

		if nodePoolTemplate.Spec.ControlPlane {
			nodePoolOptions[i].ControlPlane = &nodePoolTemplate.Spec.ControlPlane
		}

		if nodePoolTemplate.Spec.LoadBalancer {
			nodePoolOptions[i].LoadBalancer = &nodePoolTemplate.Spec.LoadBalancer
		}

		if nodePoolTemplate.Spec.DedicatedRole {
			nodePoolOptions[i].ControlPlaneComponentsOnly = &nodePoolTemplate.Spec.DedicatedRole
		}

		resourceCPU := nodePoolTemplate.Spec.Resources.Cpu()
		if resourceCPU != nil && resourceCPU.Value() != 0 {
			nodePoolOptions[i].CpuCount = util.Ptr(int(resourceCPU.Value()))
		}

		resourceMemory := nodePoolTemplate.Spec.Resources.Memory()
		if !resourceMemory.IsZero() {
			nodePoolOptions[i].RamSize = util.Ptr(resourceMemory.String())
		}

		resourceStorage := nodePoolTemplate.Spec.Resources.Storage()
		if !resourceStorage.IsZero() {
			nodePoolOptions[i].DiskSize = util.Ptr(resourceStorage.String())
		}
	}

	return nodePoolOptions
}

func (h *handler) GetClusterOptions(c *gin.Context) {
	ctx := context.Background()

	objectKey := client.ObjectKey{
		Name:      dockyardsv1.ReleaseNameSupportedKubernetesVersions,
		Namespace: h.namespace,
	}

	var release dockyardsv1.Release
	err := h.controllerClient.Get(ctx, objectKey, &release)
	if err != nil {
		h.logger.Error("error getting release", "err", err)
		c.AbortWithStatus(http.StatusInternalServerError)

		return
	}

	objectKey = client.ObjectKey{
		Name:      "recommended",
		Namespace: h.namespace,
	}

	var clusterTemplate dockyardsv1.ClusterTemplate
	err = h.controllerClient.Get(ctx, objectKey, &clusterTemplate)
	if err != nil {
		h.logger.Error("error getting cluster template", "err", err)
		c.AbortWithStatus(http.StatusInternalServerError)

		return
	}

	recommendedNodePools := getRecommendedNodePools(&clusterTemplate)

	c.JSON(http.StatusOK, v1.Options{
		SingleNode:      false,
		Version:         release.Status.Versions,
		NodePoolOptions: recommendedNodePools,
	})
}
