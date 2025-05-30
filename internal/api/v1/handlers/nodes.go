// Copyright 2024 Sudo Sweden AB
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package handlers

import (
	"context"

	"github.com/sudoswedenab/dockyards-api/pkg/types"
	dockyardsv1 "github.com/sudoswedenab/dockyards-backend/api/v1alpha3"
	"k8s.io/apimachinery/pkg/api/meta"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// +kubebuilder:rbac:groups=dockyards.io,resources=nodes,verbs=get;list;watch

func (h *handler) ListClusterNodes(ctx context.Context, cluster *dockyardsv1.Cluster) (*[]types.Node, error) {
	matchingLabels := client.MatchingLabels{
		dockyardsv1.LabelClusterName: cluster.Name,
	}

	var nodeList dockyardsv1.NodeList
	err := h.List(ctx, &nodeList, matchingLabels, client.InNamespace(cluster.Namespace))
	if err != nil {
		return nil, err
	}

	result := make([]types.Node, len(nodeList.Items))

	for i, item := range nodeList.Items {
		node := types.Node{
			ID:        string(item.UID),
			Name:      item.Name,
			CreatedAt: item.CreationTimestamp.Time,
		}

		readyCondition := meta.FindStatusCondition(item.Status.Conditions, dockyardsv1.ReadyCondition)
		if readyCondition != nil {
			node.UpdatedAt = &readyCondition.LastTransitionTime.Time
			node.Condition = &readyCondition.Reason
		}

		if !item.DeletionTimestamp.IsZero() {
			node.DeletedAt = &item.DeletionTimestamp.Time
		}

		result[i] = node
	}

	return &result, nil
}

func (h *handler) GetClusterNode(ctx context.Context, cluster *dockyardsv1.Cluster, nodeName string) (*types.Node, error) {
	objectKey := client.ObjectKey{
		Name:      nodeName,
		Namespace: cluster.Namespace,
	}

	var node dockyardsv1.Node
	err := h.Get(ctx, objectKey, &node)
	if err != nil {
		return nil, err
	}

	result := types.Node{
		CreatedAt: node.CreationTimestamp.Time,
		ID:        string(node.UID),
		Name:      node.Name,
	}

	readyCondition := meta.FindStatusCondition(node.Status.Conditions, dockyardsv1.ReadyCondition)
	if readyCondition != nil {
		result.UpdatedAt = &readyCondition.LastTransitionTime.Time
		result.Condition = &readyCondition.Reason
	}

	if !node.DeletionTimestamp.IsZero() {
		result.DeletedAt = &node.DeletionTimestamp.Time
	}

	if node.Spec.ProviderID != nil {
		result.ProviderID = node.Spec.ProviderID
	}

	if node.Status.SystemInfo != nil {
		result.SystemInfo = &types.SystemInfo{
			Architecture:            &node.Status.SystemInfo.Architecture,
			BootID:                  &node.Status.SystemInfo.BootID,
			ContainerRuntimeVersion: &node.Status.SystemInfo.ContainerRuntimeVersion,
			KernelVersion:           &node.Status.SystemInfo.KernelVersion,
			KubeletVersion:          &node.Status.SystemInfo.KubeletVersion,
			MachineID:               &node.Status.SystemInfo.MachineID,
			OperatingSystem:         &node.Status.SystemInfo.OperatingSystem,
			OsImage:                 &node.Status.SystemInfo.OSImage,
			SystemUUID:              &node.Status.SystemInfo.SystemUUID,
		}
	}

	return &result, nil
}
