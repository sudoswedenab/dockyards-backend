package handlers

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"bitbucket.org/sudosweden/dockyards-api/pkg/types"
	"bitbucket.org/sudosweden/dockyards-backend/internal/api/v1/middleware"
	"bitbucket.org/sudosweden/dockyards-backend/pkg/api/apiutil"
	dockyardsv1 "bitbucket.org/sudosweden/dockyards-backend/pkg/api/v1alpha3"
	"bitbucket.org/sudosweden/dockyards-backend/pkg/api/v1alpha3/index"
	"bitbucket.org/sudosweden/dockyards-backend/pkg/util/name"
	authorizationv1 "k8s.io/api/authorization/v1"
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

func (h *handler) toV1NodePool(nodePool *dockyardsv1.NodePool, cluster *dockyardsv1.Cluster, nodeList *dockyardsv1.NodeList) *types.NodePool {
	v1NodePool := types.NodePool{
		ID:        string(nodePool.UID),
		ClusterID: string(cluster.UID),
		Name:      nodePool.Name,
		CPUCount:  int(nodePool.Spec.Resources.Cpu().Value()),
	}

	resourceStorage := nodePool.Spec.Resources.Storage()
	if !resourceStorage.IsZero() {
		v1NodePool.DiskSize = resourceStorage.String()
	}

	resourceMemory := nodePool.Spec.Resources.Memory()
	if !resourceMemory.IsZero() {
		v1NodePool.RAMSize = resourceMemory.String()
	}

	if nodePool.Spec.Replicas != nil {
		v1NodePool.Quantity = int(*nodePool.Spec.Replicas)
	}

	if nodeList != nil && len(nodeList.Items) > 0 {
		nodes := make([]types.Node, len(nodeList.Items))
		for i, node := range nodeList.Items {
			nodes[i] = types.Node{
				ID:   string(node.UID),
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

func (h *handler) GetNodePool(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	logger := middleware.LoggerFrom(ctx)

	nodePoolID := r.PathValue("nodePoolID")
	if nodePoolID == "" {
		w.WriteHeader(http.StatusBadRequest)

		return
	}

	matchingFields := client.MatchingFields{
		index.UIDField: nodePoolID,
	}

	var nodePoolList dockyardsv1.NodePoolList
	err := h.List(ctx, &nodePoolList, matchingFields)
	if err != nil {
		logger.Error("error listing node pools in kubernetes", "err", err)
		w.WriteHeader(http.StatusUnauthorized)

		return
	}

	if len(nodePoolList.Items) != 1 {
		logger.Debug("expected exactly one node pool", "length", len(nodePoolList.Items))
		w.WriteHeader(http.StatusUnauthorized)

		return
	}

	nodePool := nodePoolList.Items[0]

	cluster, err := apiutil.GetOwnerCluster(ctx, h.Client, &nodePool)
	if err != nil {
		logger.Error("error getting owner cluster", "err", err)
		w.WriteHeader(http.StatusUnauthorized)

		return
	}

	organization, err := apiutil.GetOwnerOrganization(ctx, h.Client, cluster)
	if err != nil {
		logger.Error("error getting owner cluster owner organization", "err", err)
		w.WriteHeader(http.StatusUnauthorized)

		return
	}

	subject, err := middleware.SubjectFrom(ctx)
	if err != nil {
		logger.Error("error getting subject from context", "err", err)
		w.WriteHeader(http.StatusInternalServerError)

		return
	}

	isMember := h.isMember(subject, organization)
	if !isMember {
		logger.Debug("user is not a member of organization", "subject", subject)
		w.WriteHeader(http.StatusUnauthorized)

		return
	}

	matchingFields = client.MatchingFields{
		index.OwnerReferencesField: nodePoolID,
	}

	var nodeList dockyardsv1.NodeList
	err = h.List(ctx, &nodeList, matchingFields)
	if err != nil {
		logger.Error("error listing nodes", "err", err)
		w.WriteHeader(http.StatusInternalServerError)

		return
	}

	v1NodePool := h.toV1NodePool(&nodePool, cluster, &nodeList)

	b, err := json.Marshal(&v1NodePool)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)

		return
	}

	w.WriteHeader(http.StatusOK)
	_, err = w.Write(b)
	if err != nil {
		logger.Error("error writing response data", "err", err)
	}
}

func (h *handler) PostClusterNodePools(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	logger := middleware.LoggerFrom(ctx)

	clusterID := r.PathValue("clusterID")
	if clusterID == "" {
		w.WriteHeader(http.StatusBadRequest)

		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)

		return
	}

	r.Body.Close()

	var nodePoolOptions types.NodePoolOptions
	err = json.Unmarshal(body, &nodePoolOptions)
	if err != nil {
		logger.Debug("error unmashalling body", "err", err)
		w.WriteHeader(http.StatusUnprocessableEntity)

		return
	}

	if nodePoolOptions.Name == nil {
		logger.Debug("node pool has invalid name", "name", nodePoolOptions.Name)
		w.WriteHeader(http.StatusUnprocessableEntity)

		return
	}
	nodePoolName := *nodePoolOptions.Name
	_, isValidName := name.IsValidName(nodePoolName)
	if !isValidName {
		logger.Debug("node pool has invalid name", "name", nodePoolOptions.Name)
		w.WriteHeader(http.StatusUnprocessableEntity)

		return
	}

	if nodePoolOptions.Quantity == nil {
		logger.Debug("quantity may not be nil")
		w.WriteHeader(http.StatusUnprocessableEntity)

		return
	}
	nodePoolQuantity := *nodePoolOptions.Quantity

	if nodePoolQuantity > maxReplicas {
		logger.Debug("quantity too large", "quantity", nodePoolQuantity)
		w.WriteHeader(http.StatusUnprocessableEntity)

		return
	}

	matchingFields := client.MatchingFields{
		index.UIDField: clusterID,
	}

	var clusterList dockyardsv1.ClusterList
	err = h.List(ctx, &clusterList, matchingFields)
	if err != nil {
		logger.Error("error listing clusters", "err", err)
		w.WriteHeader(http.StatusInternalServerError)

		return
	}

	if len(clusterList.Items) != 1 {
		logger.Debug("expected exactly one cluster", "count", len(clusterList.Items))
		w.WriteHeader(http.StatusUnauthorized)

		return
	}

	cluster := clusterList.Items[0]

	organization, err := apiutil.GetOwnerOrganization(ctx, h.Client, &cluster)
	if err != nil {
		logger.Error("error getting owner organization", "err", err)
		w.WriteHeader(http.StatusInternalServerError)

		return
	}

	if organization == nil {
		logger.Debug("node pool has no organization owner")
		w.WriteHeader(http.StatusUnauthorized)

		return
	}

	subject, err := middleware.SubjectFrom(ctx)
	if err != nil {
		logger.Error("error getting subject from context", "err", err)
		w.WriteHeader(http.StatusInternalServerError)

		return
	}

	isMember := h.isMember(subject, organization)
	if !isMember {
		logger.Debug("subject is not a member of organization")
		w.WriteHeader(http.StatusUnauthorized)

		return
	}

	name := cluster.Name + "-" + nodePoolName

	resources := make(corev1.ResourceList)

	if nodePoolOptions.RAMSize != nil {
		memory, err := resource.ParseQuantity(*nodePoolOptions.RAMSize)
		if err != nil {
			logger.Debug("error parsing ram size quantity", "err", err)
			w.WriteHeader(http.StatusUnprocessableEntity)

			return
		}

		resources[corev1.ResourceMemory] = memory
	}

	if nodePoolOptions.CPUCount != nil {
		cpu := resource.NewQuantity(int64(*nodePoolOptions.CPUCount), resource.DecimalSI)
		resources[corev1.ResourceCPU] = *cpu
	}

	if nodePoolOptions.DiskSize != nil {
		storage, err := resource.ParseQuantity(*nodePoolOptions.DiskSize)
		if err != nil {
			logger.Debug("error parsing disk size quantity", "err", err)
			w.WriteHeader(http.StatusUnprocessableEntity)

			return
		}

		resources[corev1.ResourceStorage] = storage
	}

	nodePool := dockyardsv1.NodePool{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: cluster.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: dockyardsv1.GroupVersion.String(),
					Kind:       dockyardsv1.ClusterKind,
					Name:       cluster.Name,
					UID:        cluster.UID,
				},
			},
		},
		Spec: dockyardsv1.NodePoolSpec{
			Replicas:  ptr.To(int32(nodePoolQuantity)),
			Resources: resources,
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

	if nodePoolOptions.StorageResources != nil {
		for _, storageResource := range *nodePoolOptions.StorageResources {
			quantity, err := resource.ParseQuantity(storageResource.Quantity)
			if err != nil {
				logger.Debug("error parsing storage resource quantity", "err", err)
				w.WriteHeader(http.StatusUnprocessableEntity)

				return
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

	err = h.Create(ctx, &nodePool)
	if client.IgnoreAlreadyExists(err) != nil {
		logger.Error("error creating node pool", "err", err)
		w.WriteHeader(http.StatusInternalServerError)

		return
	}

	if apierrors.IsAlreadyExists(err) {
		w.WriteHeader(http.StatusConflict)

		return
	}

	v1NodePool := h.toV1NodePool(&nodePool, &cluster, nil)

	b, err := json.Marshal(&v1NodePool)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)

		return
	}

	w.WriteHeader(http.StatusCreated)
	_, err = w.Write(b)
	if err != nil {
		logger.Error("error writing response data", "err", err)
	}
}

func (h *handler) DeleteNodePool(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	logger := middleware.LoggerFrom(ctx)

	nodePoolID := r.PathValue("nodePoolID")
	if nodePoolID == "" {
		w.WriteHeader(http.StatusBadRequest)

		return
	}

	matchingFields := client.MatchingFields{
		index.UIDField: nodePoolID,
	}

	var nodePoolList dockyardsv1.NodePoolList
	err := h.List(ctx, &nodePoolList, matchingFields)
	if err != nil {
		logger.Error("error listing node pools", "err", err)
		w.WriteHeader(http.StatusInternalServerError)

		return
	}

	if len(nodePoolList.Items) != 1 {
		logger.Debug("expected exactly one node pool", "count", len(nodePoolList.Items))
		w.WriteHeader(http.StatusUnauthorized)

		return
	}

	nodePool := nodePoolList.Items[0]

	cluster, err := apiutil.GetOwnerCluster(ctx, h.Client, &nodePool)
	if client.IgnoreNotFound(err) != nil {
		logger.Error("error getting owner cluster", "err", err)
		w.WriteHeader(http.StatusInternalServerError)

		return
	}

	if apierrors.IsNotFound(err) {
		w.WriteHeader(http.StatusUnauthorized)

		return
	}

	if cluster == nil {
		logger.Debug("node pool has no owner cluster")
		w.WriteHeader(http.StatusUnauthorized)

		return
	}

	organization, err := apiutil.GetOwnerOrganization(ctx, h.Client, cluster)
	if client.IgnoreNotFound(err) != nil {
		logger.Error("error getting owner organization", "err", err)
		w.WriteHeader(http.StatusInternalServerError)

		return
	}

	if apierrors.IsNotFound(err) {
		w.WriteHeader(http.StatusUnauthorized)

		return
	}

	if organization == nil {
		logger.Debug("node pool has no owner organization")
		w.WriteHeader(http.StatusUnauthorized)

		return
	}

	subject, err := middleware.SubjectFrom(ctx)
	if err != nil {
		logger.Error("error getting subject from context", "err", err)
		w.WriteHeader(http.StatusInternalServerError)

		return
	}

	isMember := h.isMember(subject, organization)
	if !isMember {
		logger.Debug("subject is not a member of organization")
		w.WriteHeader(http.StatusUnauthorized)

		return
	}

	deleteOptions := client.DeleteOptions{
		PropagationPolicy: ptr.To(metav1.DeletePropagationBackground),
	}

	err = h.Delete(ctx, &nodePool, &deleteOptions)
	if err != nil {
		logger.Error("error deleting node pool", "err", err)
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *handler) UpdateNodePool(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	logger := middleware.LoggerFrom(ctx)

	nodePoolID := r.PathValue("nodePoolID")
	if nodePoolID == "" {
		w.WriteHeader(http.StatusBadRequest)

		return
	}

	var patchRequest types.NodePoolOptions
	err := json.NewDecoder(r.Body).Decode(&patchRequest)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)

		return
	}

	matchingFields := client.MatchingFields{
		index.UIDField: nodePoolID,
	}

	var nodePoolList dockyardsv1.NodePoolList
	err = h.List(ctx, &nodePoolList, matchingFields)
	if err != nil {
		logger.Error("error listing node pools", "err", err)
		w.WriteHeader(http.StatusInternalServerError)

		return
	}

	if len(nodePoolList.Items) != 1 {
		logger.Debug("expected exactly one node pool", "count", len(nodePoolList.Items))
		w.WriteHeader(http.StatusUnauthorized)

		return
	}

	nodePool := nodePoolList.Items[0]

	cluster, err := apiutil.GetOwnerCluster(ctx, h.Client, &nodePool)
	if client.IgnoreNotFound(err) != nil {
		logger.Error("error getting owner cluster", "err", err)
		w.WriteHeader(http.StatusInternalServerError)

		return
	}

	if apierrors.IsNotFound(err) {
		logger.Error("error getting owner cluster", "err", err)
		w.WriteHeader(http.StatusUnauthorized)

		return
	}

	if cluster == nil {
		logger.Debug("node pool has no owner cluster")
		w.WriteHeader(http.StatusUnauthorized)

		return
	}

	subject, err := middleware.SubjectFrom(ctx)
	if err != nil {
		logger.Error("error getting subject from context", "err", err)
		w.WriteHeader(http.StatusInternalServerError)

		return
	}

	resourceAttributes := authorizationv1.ResourceAttributes{
		Group:     dockyardsv1.GroupVersion.Group,
		Namespace: nodePool.Namespace,
		Resource:  "nodepools",
		Verb:      "patch",
	}

	allowed, err := apiutil.IsSubjectAllowed(ctx, h.Client, subject, &resourceAttributes)
	if err != nil {
		logger.Error("error reviewing subject", "err", err)
		w.WriteHeader(http.StatusInternalServerError)

		return
	}

	if !allowed {
		logger.Debug("subject is not allowed to patch node pools", "subject", subject, "namespace", nodePool.Namespace)
		w.WriteHeader(http.StatusUnauthorized)

		return
	}

	replicas := patchRequest.Quantity
	if replicas != nil {
		if *replicas <= 0 || *replicas > maxReplicas {
			logger.Debug("invalid amount of replicas", "replicas", *replicas)
			w.WriteHeader(http.StatusUnprocessableEntity)

			return
		}
		nodePool.Spec.Replicas = ptr.To(int32(*replicas))
	}

	if patchRequest.Name != nil {
		if nodePool.ObjectMeta.Name != *patchRequest.Name {
			logger.Debug("cannot change name of node pool")
			w.WriteHeader(http.StatusUnprocessableEntity)

			return
		}
	}

	patch := client.MergeFrom(nodePool.DeepCopy())

	if patchRequest.ControlPlane != nil {
		if nodePool.Spec.ControlPlane != *patchRequest.ControlPlane {
			logger.Error("ControlPlane field may not be changed")
			w.WriteHeader(http.StatusUnprocessableEntity)

			return
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
			w.WriteHeader(http.StatusInternalServerError)

			return
		}
		nodePool.Spec.StorageResources = storageResource
	}

	if patchRequest.CPUCount != nil {
		if *patchRequest.CPUCount <= 0 {
			logger.Debug("invalid cpu count", "count", *patchRequest.CPUCount)
			w.WriteHeader(http.StatusUnprocessableEntity)

			return
		}

		cpu := resource.NewQuantity(int64(*patchRequest.CPUCount), resource.DecimalSI)
		nodePool.Spec.Resources[corev1.ResourceCPU] = *cpu
	}

	if patchRequest.DiskSize != nil {
		size, err := resource.ParseQuantity(*patchRequest.DiskSize)

		if err != nil {
			logger.Debug("error parsing disk size quantity", "err", err)
			w.WriteHeader(http.StatusUnprocessableEntity)

			return
		}

		if !nodePool.Spec.Resources[corev1.ResourceStorage].Equal(size) {
			logger.Debug("disk size may not be changed")
			w.WriteHeader(http.StatusUnprocessableEntity)

			return
		}
	}

	if patchRequest.RAMSize != nil {
		size, err := resource.ParseQuantity(*patchRequest.RAMSize)

		if err != nil {
			logger.Debug("error parsing memory size quantity", "err", err)
			w.WriteHeader(http.StatusUnprocessableEntity)

			return
		}

		nodePool.Spec.Resources[corev1.ResourceMemory] = size
	}

	if patchRequest.Quantity != nil {
		if *patchRequest.Quantity > maxReplicas {
			logger.Debug("invalid amount of replicas", "quantity", patchRequest.Quantity)
			w.WriteHeader(http.StatusUnprocessableEntity)

			return
		}

		nodePool.Spec.Replicas = ptr.To(int32(*patchRequest.Quantity))
	}

	responseJSON, err := json.Marshal(h.toV1NodePool(&nodePool, cluster, nil))
	if err != nil {
		logger.Error("error creating JSON response", "err", err)
		w.WriteHeader(http.StatusInternalServerError)

		return
	}

	err = h.Patch(ctx, &nodePool, patch)
	if err != nil {
		logger.Error("error patching node pool", "err", err)
		w.WriteHeader(http.StatusInternalServerError)

		return
	}

	w.WriteHeader(http.StatusAccepted)
	_, err = w.Write(responseJSON)
	if err != nil {
		logger.Error("error writing response data", "err", err)
	}
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
