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
	"time"

	"bitbucket.org/sudosweden/dockyards-backend/pkg/api/apiutil"
	dockyardsv1 "bitbucket.org/sudosweden/dockyards-backend/pkg/api/v1alpha3"
	"bitbucket.org/sudosweden/dockyards-backend/pkg/util/name"
	"github.com/sudoswedenab/dockyards-api/pkg/types"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// +kubebuilder:rbac:groups=authorization.k8s.io,resources=subjectaccessreviews,verbs=create
// +kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch
// +kubebuilder:rbac:groups=dockyards.io,resources=clusters,verbs=get;list;watch;create;delete
// +kubebuilder:rbac:groups=dockyards.io,resources=nodepools,verbs=create
// +kubebuilder:rbac:groups=dockyards.io,resources=organizations,verbs=get;list;watch

func (h *handler) toV1Cluster(cluster *dockyardsv1.Cluster, nodePoolList *dockyardsv1.NodePoolList) *types.Cluster {
	v1Cluster := types.Cluster{
		ID:        string(cluster.UID),
		Name:      cluster.Name,
		CreatedAt: cluster.CreationTimestamp.Time,
		Version:   &cluster.Status.Version,
	}

	if !cluster.DeletionTimestamp.IsZero() {
		v1Cluster.DeletedAt = &cluster.DeletionTimestamp.Time
	}

	condition := meta.FindStatusCondition(cluster.Status.Conditions, dockyardsv1.ReadyCondition)
	if condition != nil {
		v1Cluster.Condition = &condition.Reason

		v1Cluster.UpdatedAt = &condition.LastTransitionTime.Time
	}

	nodePoolsCount := 0
	if nodePoolList != nil {
		nodePoolsCount = len(nodePoolList.Items)
		v1Cluster.NodePoolsCount = &nodePoolsCount
	}

	if cluster.Spec.AllocateInternalIP {
		v1Cluster.AllocateInternalIP = &cluster.Spec.AllocateInternalIP
	}

	if cluster.Status.APIEndpoint.IsValid() {
		v1Cluster.APIEndpoint = ptr.To(cluster.Status.APIEndpoint.String())
	}

	if len(cluster.Status.DNSZones) > 0 {
		v1Cluster.DNSZones = &cluster.Status.DNSZones
	}

	return &v1Cluster
}

func (h *handler) nodePoolOptionsToNodePool(nodePoolOptions *types.NodePoolOptions, cluster *dockyardsv1.Cluster) (*dockyardsv1.NodePool, error) {
	if nodePoolOptions.Name == nil {
		return nil, errors.New("name must not be nil")
	}

	if nodePoolOptions.Quantity == nil {
		return nil, errors.New("quantity must not be nil")
	}

	nodePool := dockyardsv1.NodePool{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cluster.Name + "-" + *nodePoolOptions.Name,
			Namespace: cluster.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion:         dockyardsv1.GroupVersion.String(),
					Kind:               dockyardsv1.ClusterKind,
					Name:               cluster.Name,
					UID:                cluster.UID,
					BlockOwnerDeletion: ptr.To(true),
				},
			},
			Labels: map[string]string{
				dockyardsv1.LabelClusterName: cluster.Name,
			},
		},
		Spec: dockyardsv1.NodePoolSpec{
			Replicas: ptr.To(int32(*nodePoolOptions.Quantity)),
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

	nodePool.Spec.Resources = corev1.ResourceList{}

	if nodePoolOptions.CPUCount != nil {
		quantity := resource.NewQuantity(int64(*nodePoolOptions.CPUCount), resource.BinarySI)

		nodePool.Spec.Resources[corev1.ResourceCPU] = *quantity
	}

	if nodePoolOptions.DiskSize != nil {
		quantity, err := resource.ParseQuantity(*nodePoolOptions.DiskSize)
		if err != nil {
			return nil, err
		}

		nodePool.Spec.Resources[corev1.ResourceStorage] = quantity
	}

	if nodePoolOptions.RAMSize != nil {
		quantity, err := resource.ParseQuantity(*nodePoolOptions.RAMSize)
		if err != nil {
			return nil, err
		}

		nodePool.Spec.Resources[corev1.ResourceMemory] = quantity
	}

	if nodePoolOptions.StorageResources != nil {
		for _, storageResource := range *nodePoolOptions.StorageResources {
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

	return &nodePool, nil
}

func (h *handler) CreateOrganizationCluster(ctx context.Context, organization *dockyardsv1.Organization, request *types.ClusterOptions) (*types.Cluster, error) {
	_, validName := name.IsValidName(request.Name)
	if !validName {
		statusError := apierrors.NewInvalid(dockyardsv1.GroupVersion.WithKind(dockyardsv1.ClusterKind).GroupKind(), "", nil)

		return nil, statusError
	}

	if request.NodePoolOptions != nil && request.ClusterTemplate != nil {
		statusError := apierrors.NewInvalid(dockyardsv1.GroupVersion.WithKind(dockyardsv1.ClusterKind).GroupKind(), "", nil)

		return nil, statusError
	}

	if request.NodePoolOptions != nil {
		for _, nodePoolOptions := range *request.NodePoolOptions {
			if nodePoolOptions.Name == nil {
				statusError := apierrors.NewInvalid(dockyardsv1.GroupVersion.WithKind(dockyardsv1.ClusterKind).GroupKind(), "", nil)

				return nil, statusError
			}
			_, validName := name.IsValidName(*nodePoolOptions.Name)
			if !validName {
				statusError := apierrors.NewInvalid(dockyardsv1.GroupVersion.WithKind(dockyardsv1.ClusterKind).GroupKind(), "", nil)

				return nil, statusError
			}

			if nodePoolOptions.Quantity == nil {
				statusError := apierrors.NewInvalid(dockyardsv1.GroupVersion.WithKind(dockyardsv1.ClusterKind).GroupKind(), "", nil)

				return nil, statusError
			}

			if *nodePoolOptions.Quantity > maxReplicas {
				statusError := apierrors.NewInvalid(dockyardsv1.GroupVersion.WithKind(dockyardsv1.ClusterKind).GroupKind(), "", nil)

				return nil, statusError
			}
		}
	}

	cluster := dockyardsv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      request.Name,
			Namespace: organization.Spec.NamespaceRef.Name,
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion:         dockyardsv1.GroupVersion.String(),
					Kind:               dockyardsv1.OrganizationKind,
					Name:               organization.Name,
					UID:                organization.UID,
					BlockOwnerDeletion: ptr.To(true),
				},
			},
		},
		Spec: dockyardsv1.ClusterSpec{},
	}

	if request.Version != nil {
		cluster.Spec.Version = *request.Version
	} else {
		release, err := apiutil.GetDefaultRelease(ctx, h.Client, dockyardsv1.ReleaseTypeKubernetes)
		if err != nil {
			return nil, err
		}

		if release == nil {
			return nil, nil
		}

		cluster.Spec.Version = release.Status.LatestVersion
	}

	if request.AllocateInternalIP != nil {
		cluster.Spec.AllocateInternalIP = *request.AllocateInternalIP
	}

	if request.Duration != nil {
		duration, err := time.ParseDuration(*request.Duration)
		if err != nil {
			return nil, err
		}

		cluster.Spec.Duration = &metav1.Duration{
			Duration: duration,
		}
	}

	err := h.Create(ctx, &cluster)
	if err != nil {
		return nil, err
	}

	var clusterTemplate *dockyardsv1.ClusterTemplate

	if request.ClusterTemplate != nil {
		objectKey := client.ObjectKey{
			Name:      *request.ClusterTemplate,
			Namespace: h.namespace,
		}

		var customTemplate dockyardsv1.ClusterTemplate
		err := h.Get(ctx, objectKey, &customTemplate)
		if err != nil {
			return nil, err
		}

		clusterTemplate = &customTemplate
	} else {
		clusterTemplate, err = apiutil.GetDefaultClusterTemplate(ctx, h.Client)
		if err != nil {
			return nil, err
		}

		if clusterTemplate == nil {
			return nil, nil
		}
	}

	if request.NodePoolOptions == nil {
		for _, nodePoolTemplate := range clusterTemplate.Spec.NodePoolTemplates {
			var nodePool dockyardsv1.NodePool

			nodePool.ObjectMeta = nodePoolTemplate.ObjectMeta
			nodePoolTemplate.Spec.DeepCopyInto(&nodePool.Spec)

			nodePool.OwnerReferences = []metav1.OwnerReference{
				{
					APIVersion:         dockyardsv1.GroupVersion.String(),
					Kind:               dockyardsv1.ClusterKind,
					Name:               cluster.Name,
					UID:                cluster.UID,
					BlockOwnerDeletion: ptr.To(true),
				},
			}

			if nodePool.Labels == nil {
				nodePool.Labels = make(map[string]string)
			}

			nodePool.Labels[dockyardsv1.LabelClusterName] = cluster.Name

			nodePool.Name = cluster.Name + "-" + nodePool.Name
			nodePool.Namespace = cluster.Namespace

			err = h.Create(ctx, &nodePool)
			if err != nil {
				return nil, err
			}
		}
	}

	if request.NodePoolOptions != nil {
		for _, nodePoolOptions := range *request.NodePoolOptions {
			nodePool, err := h.nodePoolOptionsToNodePool(&nodePoolOptions, &cluster)
			if err != nil {
				return nil, err
			}

			err = h.Create(ctx, nodePool)
			if err != nil {
				return nil, err
			}
		}
	}

	v1Cluster := h.toV1Cluster(&cluster, nil)

	return v1Cluster, nil
}

func (h *handler) DeleteOrganizationCluster(ctx context.Context, organization *dockyardsv1.Organization, clusterName string) error {
	objectKey := client.ObjectKey{
		Name:      clusterName,
		Namespace: organization.Spec.NamespaceRef.Name,
	}

	var cluster dockyardsv1.Cluster
	err := h.Get(ctx, objectKey, &cluster)
	if err != nil {
		return err
	}

	err = h.Delete(ctx, &cluster, client.PropagationPolicy(metav1.DeletePropagationForeground))
	if err != nil {
		return err
	}

	return nil
}

func (h *handler) ListOrganizationClusters(ctx context.Context, organization *dockyardsv1.Organization) (*[]types.Cluster, error) {
	var clusterList dockyardsv1.ClusterList
	err := h.List(ctx, &clusterList, client.InNamespace(organization.Spec.NamespaceRef.Name))
	if err != nil {
		return nil, err
	}

	response := make([]types.Cluster, len(clusterList.Items))

	for i, item := range clusterList.Items {
		cluster := types.Cluster{
			CreatedAt: item.CreationTimestamp.Time,
			ID:        string(item.UID),
			Name:      item.Name,
		}

		readyCondition := meta.FindStatusCondition(item.Status.Conditions, dockyardsv1.ReadyCondition)
		if readyCondition != nil {
			cluster.UpdatedAt = &readyCondition.LastTransitionTime.Time
			cluster.Condition = &readyCondition.Reason
		}

		if !item.DeletionTimestamp.IsZero() {
			cluster.DeletedAt = &item.DeletionTimestamp.Time
		}

		response[i] = cluster
	}

	return &response, nil
}

func (h *handler) GetOrganizationCluster(ctx context.Context, organization *dockyardsv1.Organization, clusterName string) (*types.Cluster, error) {
	objectKey := client.ObjectKey{
		Name:      clusterName,
		Namespace: organization.Spec.NamespaceRef.Name,
	}

	var cluster dockyardsv1.Cluster
	err := h.Get(ctx, objectKey, &cluster)
	if err != nil {
		return nil, err
	}

	matchingLabels := client.MatchingLabels{
		dockyardsv1.LabelClusterName: cluster.Name,
	}

	var nodePoolList dockyardsv1.NodePoolList
	err = h.List(ctx, &nodePoolList, matchingLabels)
	if err != nil {
		return nil, err
	}

	v1Cluster := h.toV1Cluster(&cluster, &nodePoolList)

	return v1Cluster, nil
}
