package handlers

import (
	"context"
	"net/http"

	"bitbucket.org/sudosweden/dockyards-backend/api/v1"
	"bitbucket.org/sudosweden/dockyards-backend/internal/util"
	"bitbucket.org/sudosweden/dockyards-backend/pkg/api/v1alpha1"
	"github.com/gin-gonic/gin"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// +kubebuilder:rbac:groups=dockyards.io,resources=releases,verbs=get;list;watch

func getRecommendedNodePools() []v1.NodePoolOptions {
	return []v1.NodePoolOptions{
		{
			Name:                       "control-plane",
			Quantity:                   3,
			ControlPlane:               util.Ptr(true),
			Etcd:                       util.Ptr(true),
			ControlPlaneComponentsOnly: util.Ptr(true),
			CpuCount:                   util.Ptr(2),
			RamSizeMb:                  util.Ptr(4096),
			DiskSizeGb:                 util.Ptr(100),
		},
		{
			Name:                       "load-balancer",
			Quantity:                   2,
			LoadBalancer:               util.Ptr(true),
			ControlPlaneComponentsOnly: util.Ptr(true),
			CpuCount:                   util.Ptr(2),
			RamSizeMb:                  util.Ptr(4096),
			DiskSizeGb:                 util.Ptr(100),
		},
		{
			Name:       "worker",
			Quantity:   2,
			CpuCount:   util.Ptr(4),
			RamSizeMb:  util.Ptr(8192),
			DiskSizeGb: util.Ptr(100),
		},
	}
}

func (h *handler) GetClusterOptions(c *gin.Context) {
	ctx := context.Background()

	objectKey := client.ObjectKey{
		Name:      "supported-versions",
		Namespace: h.namespace,
	}

	var release v1alpha1.Release
	err := h.controllerClient.Get(ctx, objectKey, &release)
	if err != nil {
		h.logger.Error("error getting release", "err", err)

		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	recommendedNodePools := getRecommendedNodePools()

	c.JSON(http.StatusOK, v1.Options{
		SingleNode:      false,
		Version:         release.Status.Versions,
		NodePoolOptions: recommendedNodePools,
	})
}
