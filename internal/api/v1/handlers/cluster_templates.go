package handlers

import (
	"context"

	"github.com/sudoswedenab/dockyards-api/pkg/types"
	dockyardsv1 "github.com/sudoswedenab/dockyards-backend/api/v1alpha3"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (h *handler) ListGlobalClusterTemplates(ctx context.Context) (*[]types.ClusterTemplate, error) {
	var clusterTemplateList dockyardsv1.ClusterTemplateList
	err := h.List(ctx, &clusterTemplateList, client.InNamespace(h.namespace))
	if err != nil {
		return nil, err
	}

	response := []types.ClusterTemplate{}

	for _, item := range clusterTemplateList.Items {
		clusterTemplate := types.ClusterTemplate{
			Name: item.Name,
			ClusterOptions: types.ClusterOptions{
				NodePoolOptions: &[]types.NodePoolOptions{},
			},
		}

		defaultAnnotation := item.Annotations[dockyardsv1.AnnotationDefaultTemplate]
		if defaultAnnotation == "true" {
			clusterTemplate.IsDefault = true
		}

		for _, nodePoolTemplate := range item.Spec.NodePoolTemplates {
			nodePoolOptions := types.NodePoolOptions{
				Name: &nodePoolTemplate.Name,
			}

			if nodePoolTemplate.Spec.Replicas != nil {
				nodePoolOptions.Quantity = ptr.To(int(*nodePoolTemplate.Spec.Replicas))
			}

			if nodePoolTemplate.Spec.ControlPlane {
				nodePoolOptions.ControlPlane = &nodePoolTemplate.Spec.ControlPlane
			}

			cpu := nodePoolTemplate.Spec.Resources.Cpu()
			if cpu != nil {
				nodePoolOptions.CPUCount = ptr.To(int(cpu.Value()))
			}

			memory := nodePoolTemplate.Spec.Resources.Memory()
			if memory != nil {
				nodePoolOptions.RAMSize = ptr.To(memory.String())
			}

			storage := nodePoolTemplate.Spec.Resources.Storage()
			if storage != nil {
				nodePoolOptions.DiskSize = ptr.To(storage.String())
			}

			*clusterTemplate.ClusterOptions.NodePoolOptions = append(*clusterTemplate.ClusterOptions.NodePoolOptions, nodePoolOptions)
		}

		response = append(response, clusterTemplate)
	}

	return &response, nil
}
