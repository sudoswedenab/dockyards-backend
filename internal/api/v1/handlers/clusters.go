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
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"math"
	"math/big"
	"net/http"
	"time"

	"bitbucket.org/sudosweden/dockyards-api/pkg/types"
	"bitbucket.org/sudosweden/dockyards-backend/internal/api/v1/middleware"
	"bitbucket.org/sudosweden/dockyards-backend/pkg/api/apiutil"
	dockyardsv1 "bitbucket.org/sudosweden/dockyards-backend/pkg/api/v1alpha3"
	"bitbucket.org/sudosweden/dockyards-backend/pkg/api/v1alpha3/index"
	"bitbucket.org/sudosweden/dockyards-backend/pkg/util/name"
	authorizationv1 "k8s.io/api/authorization/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// +kubebuilder:rbac:groups=authorization.k8s.io,resources=subjectaccessreviews,verbs=create
// +kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch
// +kubebuilder:rbac:groups=dockyards.io,resources=clusters,verbs=get;list;watch;create;delete
// +kubebuilder:rbac:groups=dockyards.io,resources=nodepools,verbs=create
// +kubebuilder:rbac:groups=dockyards.io,resources=organizations,verbs=get;list;watch

func (h *handler) toV1Cluster(organization *dockyardsv1.Organization, cluster *dockyardsv1.Cluster, nodePoolList *dockyardsv1.NodePoolList) *types.Cluster {
	v1Cluster := types.Cluster{
		ID:           string(cluster.UID),
		Name:         cluster.Name,
		Organization: organization.Name,
		CreatedAt:    cluster.CreationTimestamp.Time,
		Version:      cluster.Status.Version,
	}

	condition := meta.FindStatusCondition(cluster.Status.Conditions, dockyardsv1.ReadyCondition)
	if condition != nil {
		v1Cluster.State = condition.Message
	}

	if nodePoolList != nil && len(nodePoolList.Items) > 0 {
		nodePools := make([]types.NodePool, len(nodePoolList.Items))
		for i, nodePool := range nodePoolList.Items {
			nodePools[i] = *h.toV1NodePool(&nodePool, nil)
		}

		v1Cluster.NodePools = nodePools
	}

	if cluster.Spec.AllocateInternalIP {
		v1Cluster.AllocateInternalIP = &cluster.Spec.AllocateInternalIP
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
			Namespace: organization.Status.NamespaceRef.Name,
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

	v1Cluster := h.toV1Cluster(organization, &cluster, nil)

	return v1Cluster, nil
}

func (h *handler) GetClusterKubeconfig(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	logger := middleware.LoggerFrom(ctx)

	clusterID := r.PathValue("clusterID")
	if clusterID == "" {
		w.WriteHeader(http.StatusBadRequest)

		return
	}

	matchingFields := client.MatchingFields{
		index.UIDField: clusterID,
	}

	var clusterList dockyardsv1.ClusterList
	err := h.List(ctx, &clusterList, matchingFields)
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

	subject, err := middleware.SubjectFrom(ctx)
	if err != nil {
		logger.Error("error fetching user from context", "err", err)
		w.WriteHeader(http.StatusInternalServerError)

		return
	}

	resourceAttributes := authorizationv1.ResourceAttributes{
		Verb:      "patch",
		Resource:  "clusters",
		Group:     "dockyards.io",
		Namespace: cluster.Namespace,
	}

	allowed, err := apiutil.IsSubjectAllowed(ctx, h.Client, subject, &resourceAttributes)
	if err != nil {
		logger.Error("error reviewing subject", "err", err)
		w.WriteHeader(http.StatusInternalServerError)

		return
	}

	if !allowed {
		logger.Debug("subject is not allowed to patch cluster", "subject", subject, "cluster", cluster.Name, "namespace", cluster.Namespace)
		w.WriteHeader(http.StatusUnauthorized)

		return
	}

	if !cluster.Status.APIEndpoint.IsValid() {
		logger.Error("cluster does not have a valid api endpoint", "uid", cluster.UID)
		w.WriteHeader(http.StatusInternalServerError)

		return
	}

	objectKey := client.ObjectKey{
		Name:      cluster.Name + "-ca",
		Namespace: cluster.Namespace,
	}

	var secret corev1.Secret
	err = h.Get(ctx, objectKey, &secret)
	if err != nil {
		logger.Error("error getting cluster certificate authority", "err", err)
		w.WriteHeader(http.StatusInternalServerError)

		return
	}

	caCertificatePEM, has := secret.Data[corev1.TLSCertKey]
	if !has {
		logger.Error("cluster certificate authority has no tls certificate")
		w.WriteHeader(http.StatusInternalServerError)

		return
	}

	signingKeyPEM, has := secret.Data[corev1.TLSPrivateKeyKey]
	if !has {
		logger.Error("cluster certificate authority has to tls private key")
		w.WriteHeader(http.StatusInternalServerError)

		return
	}

	block, _ := pem.Decode(caCertificatePEM)
	if block == nil {
		logger.Error("unable to decode ca certificate as pem")
		w.WriteHeader(http.StatusInternalServerError)

		return
	}

	caCertificate, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		logger.Error("error parsing ca certificate", "err", err)
		w.WriteHeader(http.StatusInternalServerError)

		return
	}

	block, _ = pem.Decode(signingKeyPEM)

	signingKey, err := x509.ParseECPrivateKey(block.Bytes)
	if err != nil {
		logger.Error("error parsing signing key", "err", err)
		w.WriteHeader(http.StatusInternalServerError)

		return
	}

	matchingFields = client.MatchingFields{
		index.UIDField: subject,
	}

	var userList dockyardsv1.UserList
	err = h.List(ctx, &userList, matchingFields)
	if err != nil {
		logger.Error("error listing users", "err", err)
		w.WriteHeader(http.StatusInternalServerError)

		return
	}

	if len(userList.Items) != 1 {
		logger.Error("unexpected users count", "count", len(userList.Items))
		w.WriteHeader(http.StatusInternalServerError)

		return
	}

	user := userList.Items[0]

	contextName := user.Name + "@" + cluster.Name

	serial, err := rand.Int(rand.Reader, big.NewInt(math.MaxInt64))
	if err != nil {
		logger.Error("error generating random serial number", "err", err)
		w.WriteHeader(http.StatusInternalServerError)

		return
	}

	tmpl := x509.Certificate{
		Subject: pkix.Name{
			CommonName: user.Name,
			Organization: []string{
				"system:masters",
			},
		},
		NotBefore:    caCertificate.NotBefore,
		NotAfter:     time.Now().Add(time.Hour * 12),
		SerialNumber: serial,
		KeyUsage:     x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage: []x509.ExtKeyUsage{
			x509.ExtKeyUsageClientAuth,
		},
	}

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		logger.Error("error generating private rsa key", "err", err)
		w.WriteHeader(http.StatusInternalServerError)

		return
	}

	certificate, err := x509.CreateCertificate(rand.Reader, &tmpl, caCertificate, privateKey.Public(), signingKey)
	if err != nil {
		logger.Error("error creating certificate", "err", err)
		w.WriteHeader(http.StatusInternalServerError)

		return
	}

	block = &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certificate,
	}

	certificatePEM := pem.EncodeToMemory(block)

	block = &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	}

	privateKeyPEM := pem.EncodeToMemory(block)

	cfg := api.Config{
		Clusters: map[string]*api.Cluster{
			cluster.Name: {
				Server:                   cluster.Status.APIEndpoint.String(),
				CertificateAuthorityData: caCertificatePEM,
			},
		},
		Contexts: map[string]*api.Context{
			contextName: {
				Cluster:  cluster.Name,
				AuthInfo: user.Name,
			},
		},
		AuthInfos: map[string]*api.AuthInfo{
			user.Name: {
				ClientCertificateData: certificatePEM,
				ClientKeyData:         privateKeyPEM,
			},
		},
		CurrentContext: contextName,
	}

	b, err := clientcmd.Write(cfg)
	if err != nil {
		logger.Error("error marshalling kubeconfig", "err", err)
		w.WriteHeader(http.StatusInternalServerError)

		return
	}

	w.WriteHeader(http.StatusOK)
	_, err = w.Write(b)
	if err != nil {
		logger.Error("error writing response data", "err", err)
	}
	//c.Data(http.StatusOK, binding.MIMEYAML, kubeconfig)
}

func (h *handler) DeleteOrganizationCluster(ctx context.Context, organization *dockyardsv1.Organization, clusterName string) error {
	objectKey := client.ObjectKey{
		Name:      clusterName,
		Namespace: organization.Status.NamespaceRef.Name,
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
	err := h.List(ctx, &clusterList, client.InNamespace(organization.Status.NamespaceRef.Name))
	if err != nil {
		return nil, err
	}

	response := make([]types.Cluster, len(clusterList.Items))

	for i, cluster := range clusterList.Items {
		response[i] = *h.toV1Cluster(organization, &cluster, nil)
	}

	return &response, nil
}

func (h *handler) GetOrganizationCluster(ctx context.Context, organization *dockyardsv1.Organization, clusterName string) (*types.Cluster, error) {
	objectKey := client.ObjectKey{
		Name:      clusterName,
		Namespace: organization.Status.NamespaceRef.Name,
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

	v1Cluster := h.toV1Cluster(organization, &cluster, &nodePoolList)

	return v1Cluster, nil
}
