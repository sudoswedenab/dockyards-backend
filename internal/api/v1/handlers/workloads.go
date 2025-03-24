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
	"encoding/json"
	"strings"

	"bitbucket.org/sudosweden/dockyards-api/pkg/types"
	dockyardsv1 "bitbucket.org/sudosweden/dockyards-backend/pkg/api/v1alpha3"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// +kubebuilder:rbac:groups=dockyards.io,resources=clusters,verbs=get;list;watch
// +kubebuilder:rbac:groups=dockyards.io,resources=organizations,verbs=get;list;watch
// +kubebuilder:rbac:groups=dockyards.io,resources=workloads,verbs=create;delete;get;list;patch;watch

func (h *handler) CreateClusterWorkload(ctx context.Context, cluster *dockyardsv1.Cluster, request *types.WorkloadOptions) (*types.Workload, error) {
	if request.WorkloadTemplateName == nil || request.Name == nil {
		statusError := apierrors.NewInvalid(dockyardsv1.GroupVersion.WithKind(dockyardsv1.WorkloadKind).GroupKind(), "", nil)

		return nil, statusError
	}

	if request.Namespace == nil {
		request.Namespace = request.Name
	}

	workload := dockyardsv1.Workload{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cluster.Name + "-" + *request.Name,
			Namespace: cluster.Namespace,
			Labels: map[string]string{
				dockyardsv1.LabelClusterName: cluster.Name,
			},
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: dockyardsv1.GroupVersion.String(),
					Kind:       dockyardsv1.ClusterKind,
					Name:       cluster.Name,
					UID:        cluster.UID,
				},
			},
		},
		Spec: dockyardsv1.WorkloadSpec{
			Provenience:     dockyardsv1.ProvenienceUser,
			TargetNamespace: *request.Namespace,
			WorkloadTemplateRef: &corev1.TypedObjectReference{
				Kind:      dockyardsv1.WorkloadTemplateKind,
				Name:      *request.WorkloadTemplateName,
				Namespace: &h.namespace,
			},
		},
	}

	if request.Input != nil {
		raw, err := json.Marshal(*request.Input)
		if err != nil {
			return nil, err
		}

		workload.Spec.Input = &apiextensionsv1.JSON{
			Raw: raw,
		}
	}

	err := h.Create(ctx, &workload)
	if err != nil {
		return nil, err
	}

	response := types.Workload{
		CreatedAt: workload.CreationTimestamp.Time,
		ID:        string(workload.UID),
		Name:      workload.Name,
	}

	return &response, nil
}

func (h *handler) DeleteClusterWorkload(ctx context.Context, cluster *dockyardsv1.Cluster, workloadName string) error {
	objectKey := client.ObjectKey{
		Name:      cluster.Name + "-" + workloadName,
		Namespace: cluster.Namespace,
	}

	var workload dockyardsv1.Workload
	err := h.Get(ctx, objectKey, &workload)
	if err != nil {
		return err
	}

	err = h.Delete(ctx, &workload, client.PropagationPolicy(metav1.DeletePropagationForeground))
	if err != nil {
		return err
	}

	return nil
}

func (h *handler) UpdateClusterWorkload(ctx context.Context, cluster *dockyardsv1.Cluster, workloadName string, request *types.Workload) error {
	objectKey := client.ObjectKey{
		Name:      cluster.Name + "-" + workloadName,
		Namespace: cluster.Namespace,
	}

	var workload dockyardsv1.Workload
	err := h.Get(ctx, objectKey, &workload)
	if err != nil {
		return err
	}

	if workload.Spec.Provenience != dockyardsv1.ProvenienceUser {
		return apierrors.NewForbidden(dockyardsv1.GroupVersion.WithResource("workloads").GroupResource(), workload.Name, nil)
	}

	if request.WorkloadTemplateName == nil || request.Namespace == nil {
		return apierrors.NewInvalid(dockyardsv1.GroupVersion.WithKind(dockyardsv1.WorkloadKind).GroupKind(), "", nil)
	}

	if *request.WorkloadTemplateName != workload.Spec.WorkloadTemplateRef.Name {
		return apierrors.NewInvalid(dockyardsv1.GroupVersion.WithKind(dockyardsv1.WorkloadKind).GroupKind(), "", nil)
	}

	patch := client.MergeFrom(workload.DeepCopy())

	workload.Spec.TargetNamespace = *request.Namespace

	if request.Input != nil {
		raw, err := json.Marshal(*request.Input)
		if err != nil {
			return err
		}

		workload.Spec.Input = &apiextensionsv1.JSON{
			Raw: raw,
		}
	} else {
		workload.Spec.Input = nil
	}

	err = h.Patch(ctx, &workload, patch)
	if err != nil {
		return err
	}

	return nil
}

func (h *handler) ListClusterWorkloads(ctx context.Context, cluster *dockyardsv1.Cluster) (*[]types.Workload, error) {
	matchingLabels := client.MatchingLabels{
		dockyardsv1.LabelClusterName: cluster.Name,
	}

	var workloadList dockyardsv1.WorkloadList
	err := h.List(ctx, &workloadList, matchingLabels, client.InNamespace(cluster.Namespace))
	if err != nil {
		return nil, err
	}

	response := make([]types.Workload, len(workloadList.Items))

	for i, workload := range workloadList.Items {
		response[i] = types.Workload{
			ID:        string(workload.UID),
			Name:      strings.TrimPrefix(workload.Name, cluster.Name+"-"),
			Namespace: ptr.To(workload.Spec.TargetNamespace),
		}

		if workload.Spec.WorkloadTemplateRef != nil {
			response[i].WorkloadTemplateName = &workload.Spec.WorkloadTemplateRef.Name
		}
	}

	return &response, err
}

func (h *handler) GetClusterWorkload(ctx context.Context, cluster *dockyardsv1.Cluster, workloadName string) (*types.Workload, error) {
	objectKey := client.ObjectKey{
		Name:      cluster.Name + "-" + workloadName,
		Namespace: cluster.Namespace,
	}

	var workload dockyardsv1.Workload
	err := h.Get(ctx, objectKey, &workload)
	if err != nil {
		return nil, err
	}

	response := types.Workload{
		Name:      strings.TrimPrefix(workload.Name, cluster.Name+"-"),
		Namespace: ptr.To(workload.Spec.TargetNamespace),
	}

	if workload.Spec.WorkloadTemplateRef != nil {
		response.WorkloadTemplateName = &workload.Spec.WorkloadTemplateRef.Name
	}

	if workload.Spec.Input != nil {
		var input map[string]any
		err := json.Unmarshal(workload.Spec.Input.Raw, &input)
		if err != nil {
			return nil, err
		}

		response.Input = &input
	}

	return &response, nil
}
