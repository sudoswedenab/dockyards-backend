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
	"errors"

	"github.com/sudoswedenab/dockyards-api/pkg/types"
	dockyardsv1 "github.com/sudoswedenab/dockyards-backend/api/v1alpha3"
	"github.com/sudoswedenab/dockyards-backend/internal/api/v1/middleware"
	"github.com/sudoswedenab/dockyards-backend/pkg/util/name"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// +kubebuilder:rbac:groups=dockyards.io,resources=clusters,verbs=get;list;watch
// +kubebuilder:rbac:groups=dockyards.io,resources=nodepools,verbs=create;get;list;watch
// +kubebuilder:rbac:groups=dockyards.io,resources=nodes,verbs=get;list;watch
// +kubebuilder:rbac:groups=dockyards.io,resources=organizations,verbs=get;list;watch

const maxReplicas = 9

func (h *handler) toV1NodePool(nodePool *dockyardsv1.NodePool, nodeList *dockyardsv1.NodeList) *types.NodePool {
	v1NodePool := types.NodePool{
		ID:   string(nodePool.UID),
		Name: nodePool.Name,
	}

	resourceCPU := nodePool.Spec.Resources.Cpu()
	if !resourceCPU.IsZero() {
		value := resourceCPU.Value()
		v1NodePool.CPUCount = ptr.To(int(value))
	}

	resourceStorage := nodePool.Spec.Resources.Storage()
	if !resourceStorage.IsZero() {
		v1NodePool.DiskSize = ptr.To(resourceStorage.String())
	}

	resourceMemory := nodePool.Spec.Resources.Memory()
	if !resourceMemory.IsZero() {
		v1NodePool.RAMSize = ptr.To(resourceMemory.String())
	}

	if nodePool.Spec.Replicas != nil {
		replicas := *nodePool.Spec.Replicas
		v1NodePool.Quantity = ptr.To(int(replicas))
	}

	if nodeList != nil && len(nodeList.Items) > 0 {
		nodes := make([]types.Node, len(nodeList.Items))
		for i, node := range nodeList.Items {
			nodes[i] = types.Node{
				ID:   string(node.UID),
				Name: node.Name,
			}
		}

		v1NodePool.Nodes = &nodes
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

	if nodePool.Spec.StorageResources != nil {
		storageResources := make([]types.StorageResource, len(nodePool.Spec.StorageResources))

		for i, storageResource := range nodePool.Spec.StorageResources {
			storageResources[i] = types.StorageResource{
				Name:     storageResource.Name,
				Quantity: storageResource.Quantity.String(),
			}

			if storageResource.Type != "" {
				storageResources[i].Type = &storageResource.Type
			}
		}

		v1NodePool.StorageResources = &storageResources
	}

	return &v1NodePool
}

func (h *handler) GetClusterNodePool(ctx context.Context, cluster *dockyardsv1.Cluster, nodePoolName string) (*types.NodePool, error) {
	objectKey := client.ObjectKey{
		Name:      nodePoolName,
		Namespace: cluster.Namespace,
	}

	var nodePool dockyardsv1.NodePool
	err := h.Get(ctx, objectKey, &nodePool)
	if err != nil {
		return nil, err
	}

	matchingLabels := client.MatchingLabels{
		dockyardsv1.LabelNodePoolName: nodePool.Name,
	}

	var nodeList dockyardsv1.NodeList
	err = h.List(ctx, &nodeList, matchingLabels)
	if err != nil {
		return nil, err
	}

	v1NodePool := h.toV1NodePool(&nodePool, &nodeList)

	return v1NodePool, nil
}

func (h *handler) DeleteClusterNodePool(ctx context.Context, cluster *dockyardsv1.Cluster, nodePoolName string) error {
	objectKey := client.ObjectKey{
		Name:      nodePoolName,
		Namespace: cluster.Namespace,
	}

	var nodePool dockyardsv1.NodePool
	err := h.Get(ctx, objectKey, &nodePool)
	if err != nil {
		return err
	}

	deleteOptions := client.DeleteOptions{
		PropagationPolicy: ptr.To(metav1.DeletePropagationBackground),
	}

	err = h.Delete(ctx, &nodePool, &deleteOptions)
	if err != nil {
		return err
	}

	return nil
}

func (h *handler) UpdateClusterNodePool(ctx context.Context, cluster *dockyardsv1.Cluster, nodePoolName string, patchRequest *types.NodePoolOptions) error {
	logger := middleware.LoggerFrom(ctx)

	objectKey := client.ObjectKey{
		Name:      nodePoolName,
		Namespace: cluster.Namespace,
	}

	var nodePool dockyardsv1.NodePool
	err := h.Get(ctx, objectKey, &nodePool)
	if err != nil {
		return err
	}

	patch := client.MergeFrom(nodePool.DeepCopy())

	replicas := patchRequest.Quantity
	if replicas != nil {
		if *replicas <= 0 || *replicas > maxReplicas {
			logger.Debug("invalid amount of replicas", "replicas", *replicas)

			return apierrors.NewInvalid(dockyardsv1.GroupVersion.WithKind(dockyardsv1.NodePoolKind).GroupKind(), "", nil)
		}
		nodePool.Spec.Replicas = ptr.To(int32(*replicas))
	}

	if patchRequest.Name != nil {
		if nodePool.Name != *patchRequest.Name {
			logger.Debug("cannot change name of node pool")

			return apierrors.NewInvalid(dockyardsv1.GroupVersion.WithKind(dockyardsv1.NodePoolKind).GroupKind(), "", nil)
		}
	}

	if patchRequest.ControlPlane != nil {
		if nodePool.Spec.ControlPlane != *patchRequest.ControlPlane {
			logger.Error("ControlPlane field may not be changed")

			return apierrors.NewInvalid(dockyardsv1.GroupVersion.WithKind(dockyardsv1.NodePoolKind).GroupKind(), "", nil)
		}
	}

	if patchRequest.LoadBalancer != nil {
		nodePool.Spec.LoadBalancer = *patchRequest.LoadBalancer
	}

	if patchRequest.ControlPlaneComponentsOnly != nil {
		nodePool.Spec.DedicatedRole = *patchRequest.ControlPlaneComponentsOnly
	}

	if patchRequest.StorageResources != nil {
		storageResource, err := nodePoolStorageResourcesFromStorageResources(*patchRequest.StorageResources)
		if err != nil {
			logger.Debug("could not create convert node pool resources", "storageResources", patchRequest.StorageResources)

			return apierrors.NewInvalid(dockyardsv1.GroupVersion.WithKind(dockyardsv1.NodePoolKind).GroupKind(), "", nil)
		}
		nodePool.Spec.StorageResources = storageResource
	}

	if patchRequest.CPUCount != nil {
		if *patchRequest.CPUCount <= 0 {
			logger.Debug("invalid cpu count", "count", *patchRequest.CPUCount)

			return apierrors.NewInvalid(dockyardsv1.GroupVersion.WithKind(dockyardsv1.NodePoolKind).GroupKind(), "", nil)
		}

		cpu := resource.NewQuantity(int64(*patchRequest.CPUCount), resource.DecimalSI)
		nodePool.Spec.Resources[corev1.ResourceCPU] = *cpu
	}

	if patchRequest.DiskSize != nil {
		size, err := resource.ParseQuantity(*patchRequest.DiskSize)
		if err != nil {
			logger.Debug("error parsing disk size quantity", "err", err)

			return apierrors.NewInvalid(dockyardsv1.GroupVersion.WithKind(dockyardsv1.NodePoolKind).GroupKind(), "", nil)
		}

		if !nodePool.Spec.Resources[corev1.ResourceStorage].Equal(size) {
			logger.Debug("disk size may not be changed")

			return apierrors.NewInvalid(dockyardsv1.GroupVersion.WithKind(dockyardsv1.NodePoolKind).GroupKind(), "", nil)
		}
	}

	if patchRequest.RAMSize != nil {
		size, err := resource.ParseQuantity(*patchRequest.RAMSize)
		if err != nil {
			return apierrors.NewInvalid(dockyardsv1.GroupVersion.WithKind(dockyardsv1.NodePoolKind).GroupKind(), "", nil)
		}

		nodePool.Spec.Resources[corev1.ResourceMemory] = size
	}

	if patchRequest.Quantity != nil {
		if *patchRequest.Quantity > maxReplicas {
			logger.Debug("invalid amount of replicas", "quantity", patchRequest.Quantity)

			return apierrors.NewInvalid(dockyardsv1.GroupVersion.WithKind(dockyardsv1.NodePoolKind).GroupKind(), "", nil)
		}

		nodePool.Spec.Replicas = ptr.To(int32(*patchRequest.Quantity))
	}

	err = h.Patch(ctx, &nodePool, patch)
	if err != nil {
		logger.Error("error patching node pool", "err", err)

		return err
	}

	return nil
}

func nodePoolStorageResourcesFromStorageResources(storageResources []types.StorageResource) ([]dockyardsv1.NodePoolStorageResource, error) {
	result := make([]dockyardsv1.NodePoolStorageResource, len(storageResources))

	for i, item := range storageResources {
		quantity, err := resource.ParseQuantity(item.Quantity)
		if err != nil {
			return nil, err
		}

		resourceType := dockyardsv1.StorageResourceTypeHostPath
		if item.Type != nil {
			resourceType = *item.Type
		}
		if resourceType != dockyardsv1.StorageResourceTypeHostPath {
			return nil, errors.New("invalid storage type")
		}

		if reason, isValid := name.IsValidName(item.Name); !isValid {
			return nil, errors.New(reason)
		}

		result[i] = dockyardsv1.NodePoolStorageResource{
			Name:     item.Name,
			Quantity: quantity,
			Type:     resourceType,
		}
	}

	return result, nil
}

func (h *handler) CreateClusterNodePool(ctx context.Context, cluster *dockyardsv1.Cluster, request *types.NodePoolOptions) (*types.NodePool, error) {
	if request.Name == nil {
		return nil, nil
	}

	nodePoolQuantity := *request.Quantity
	if nodePoolQuantity > maxReplicas {
		statusError := apierrors.NewInvalid(dockyardsv1.GroupVersion.WithKind(dockyardsv1.WorkloadKind).GroupKind(), "", nil)

		return nil, statusError
	}

	resources := make(corev1.ResourceList)

	if request.RAMSize != nil {
		memory, err := resource.ParseQuantity(*request.RAMSize)
		if err != nil {
			return nil, err
		}

		resources[corev1.ResourceMemory] = memory
	}

	if request.CPUCount != nil {
		cpu := resource.NewQuantity(int64(*request.CPUCount), resource.DecimalSI)
		resources[corev1.ResourceCPU] = *cpu
	}

	if request.DiskSize != nil {
		storage, err := resource.ParseQuantity(*request.DiskSize)
		if err != nil {
			return nil, err
		}

		resources[corev1.ResourceStorage] = storage
	}

	nodePool := dockyardsv1.NodePool{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cluster.Name + "-" + *request.Name,
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
		Spec: dockyardsv1.NodePoolSpec{
			Replicas:  ptr.To(int32(nodePoolQuantity)),
			Resources: resources,
		},
	}

	if request.ControlPlane != nil {
		nodePool.Spec.ControlPlane = *request.ControlPlane
	}

	if request.LoadBalancer != nil {
		nodePool.Spec.LoadBalancer = *request.LoadBalancer
	}

	if request.ControlPlaneComponentsOnly != nil {
		nodePool.Spec.DedicatedRole = *request.ControlPlaneComponentsOnly
	}

	if request.StorageResources != nil {
		for _, storageResource := range *request.StorageResources {
			quantity, err := resource.ParseQuantity(storageResource.Quantity)
			if err != nil {
				return nil, err
			}

			nodePoolStorageResource := dockyardsv1.NodePoolStorageResource{
				Name:     storageResource.Name,
				Quantity: quantity,
			}

			if storageResource.Type != nil {
				nodePoolStorageResource.Type = *storageResource.Type
			}

			nodePool.Spec.StorageResources = append(nodePool.Spec.StorageResources, nodePoolStorageResource)
		}
	}

	err := h.Create(ctx, &nodePool)
	if err != nil {
		return nil, err
	}

	v1NodePool := h.toV1NodePool(&nodePool, nil)

	return v1NodePool, nil
}

func (h *handler) ListClusterNodePools(ctx context.Context, cluster *dockyardsv1.Cluster) (*[]types.NodePool, error) {
	matchingLabels := client.MatchingLabels{
		dockyardsv1.LabelClusterName: cluster.Name,
	}

	var nodePoolList dockyardsv1.NodePoolList
	err := h.List(ctx, &nodePoolList, matchingLabels, client.InNamespace(cluster.Namespace))
	if err != nil {
		return nil, err
	}

	nodePools := []types.NodePool{}

	for _, item := range nodePoolList.Items {
		nodePool := types.NodePool{
			CreatedAt: &item.CreationTimestamp.Time,
			ID:        string(item.UID),
			Name:      item.Name,
		}

		if item.Spec.Replicas != nil {
			quantity := int(*item.Spec.Replicas)
			nodePool.Quantity = &quantity
		}

		if !item.DeletionTimestamp.IsZero() {
			nodePool.DeletedAt = &item.DeletionTimestamp.Time
		}

		nodePools = append(nodePools, nodePool)
	}

	return &nodePools, nil
}
