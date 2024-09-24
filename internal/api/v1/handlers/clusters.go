package handlers

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"io"
	"math"
	"math/big"
	"net/http"
	"time"

	"bitbucket.org/sudosweden/dockyards-api/pkg/types"
	"bitbucket.org/sudosweden/dockyards-backend/internal/api/v1/middleware"
	"bitbucket.org/sudosweden/dockyards-backend/pkg/api/apiutil"
	dockyardsv1 "bitbucket.org/sudosweden/dockyards-backend/pkg/api/v1alpha2"
	"bitbucket.org/sudosweden/dockyards-backend/pkg/api/v1alpha2/index"
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
			nodePools[i] = *h.toV1NodePool(&nodePool, cluster, nil)
		}

		v1Cluster.NodePools = nodePools
	}

	if cluster.Spec.AllocateInternalIP {
		v1Cluster.AllocateInternalIP = &cluster.Spec.AllocateInternalIP
	}

	return &v1Cluster
}

func (h *handler) nodePoolOptionsToNodePool(nodePoolOptions *types.NodePoolOptions, cluster *dockyardsv1.Cluster) (*dockyardsv1.NodePool, error) {
	nodePool := dockyardsv1.NodePool{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cluster.Name + "-" + nodePoolOptions.Name,
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
			Replicas: ptr.To(int32(nodePoolOptions.Quantity)),
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

	return &nodePool, nil
}

func (h *handler) PostOrgClusters(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	logger := middleware.LoggerFrom(ctx)

	organizationID := r.PathValue("organizationID")
	if organizationID == "" {
		w.WriteHeader(http.StatusBadRequest)

		return
	}

	objectKey := client.ObjectKey{
		Name: organizationID,
	}

	var organization dockyardsv1.Organization
	err := h.Get(ctx, objectKey, &organization)
	if err != nil {
		logger.Error("error getting organization from kubernetes", "err", err)
		w.WriteHeader(http.StatusUnauthorized)

		return
	}

	subject, err := middleware.SubjectFrom(ctx)
	if err != nil {
		logger.Debug("error fetching subject from context", "err", err)
		w.WriteHeader(http.StatusInternalServerError)

		return
	}

	isMember := h.isMember(subject, &organization)
	if !isMember {
		logger.Debug("subject is not a member of organization", "subject", subject, "organization", organization.Name)
		w.WriteHeader(http.StatusUnauthorized)

		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)

		return
	}

	r.Body.Close()

	var clusterOptions types.ClusterOptions
	err = json.Unmarshal(body, &clusterOptions)
	if err != nil {
		logger.Debug("error unmarshalling body", "err", err)
		w.WriteHeader(http.StatusUnprocessableEntity)

		return
	}

	logger.Debug("create cluster", "organization", organization.Name, "name", clusterOptions.Name, "version", clusterOptions.Version)

	_, validName := name.IsValidName(clusterOptions.Name)
	if !validName {
		w.WriteHeader(http.StatusUnprocessableEntity)

		return
	}

	if clusterOptions.NodePoolOptions != nil && clusterOptions.ClusterTemplate != nil {
		logger.Debug("both node pool options and cluster template set")
		w.WriteHeader(http.StatusUnprocessableEntity)

		return
	}

	if clusterOptions.NodePoolOptions != nil {
		for _, nodePoolOptions := range *clusterOptions.NodePoolOptions {
			_, validName := name.IsValidName(nodePoolOptions.Name)
			if !validName {
				w.WriteHeader(http.StatusUnprocessableEntity)

				return
			}

			if nodePoolOptions.Quantity > 9 {
				w.WriteHeader(http.StatusUnprocessableEntity)

				return
			}
		}
	}

	cluster := dockyardsv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      clusterOptions.Name,
			Namespace: organization.Status.NamespaceRef,
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

	if clusterOptions.Version != nil {
		cluster.Spec.Version = *clusterOptions.Version
	} else {
		var release dockyardsv1.Release
		err := h.Get(ctx, client.ObjectKey{Name: dockyardsv1.ReleaseNameSupportedKubernetesVersions, Namespace: h.namespace}, &release)
		if err != nil {
			logger.Error("error getting supported kubernetes versions release", "err", err)
			w.WriteHeader(http.StatusInternalServerError)

			return
		}

		cluster.Spec.Version = release.Status.LatestVersion
	}

	if clusterOptions.AllocateInternalIP != nil {
		cluster.Spec.AllocateInternalIP = *clusterOptions.AllocateInternalIP
	}

	err = h.Create(ctx, &cluster)
	if client.IgnoreAlreadyExists(err) != nil {
		logger.Error("error creating cluster", "err", err)
		w.WriteHeader(http.StatusInternalServerError)

		return
	}

	if apierrors.IsAlreadyExists(err) {
		w.WriteHeader(http.StatusConflict)

		return
	}

	hasErrors := false

	if clusterOptions.NodePoolOptions == nil {
		objectKey := client.ObjectKey{
			Name:      dockyardsv1.ClusterTemplateNameRecommended,
			Namespace: h.namespace,
		}

		if clusterOptions.ClusterTemplate != nil {
			objectKey.Name = *clusterOptions.ClusterTemplate
		}

		logger.Debug("using node pool options", "clusterTemplate", objectKey.Name)

		var clusterTemplate dockyardsv1.ClusterTemplate
		err := h.Get(ctx, objectKey, &clusterTemplate)
		if err != nil {
			logger.Error("error getting cluster template", "err", err)
			w.WriteHeader(http.StatusInternalServerError)

			return
		}

		for _, nodePoolTemplate := range clusterTemplate.Spec.NodePoolTemplates {
			nodePool := nodePoolTemplate.DeepCopy()

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

			err = h.Create(ctx, nodePool)
			if err != nil {
				hasErrors = true
				logger.Debug("error creating node pool", "err", err)

				break
			}

			logger.Debug("created node pool", "uid", nodePool.UID, "name", nodePool.Name)
		}
	}

	if clusterOptions.NodePoolOptions != nil {
		for _, nodePoolOptions := range *clusterOptions.NodePoolOptions {
			nodePool, err := h.nodePoolOptionsToNodePool(&nodePoolOptions, &cluster)
			if err != nil {
				logger.Error("error converting to node pool", "err", err)
				hasErrors = true

				break
			}

			err = h.Create(ctx, nodePool)
			if err != nil {
				logger.Error("error creating node pool", "err", err)
				hasErrors = true

				break
			}

			logger.Debug("created cluster node pool", "id", nodePool.UID)
		}
	}

	if hasErrors {
		logger.Error("deleting cluster due to node pool error", "id", cluster.UID)
		w.WriteHeader(http.StatusInternalServerError)

		return
	}

	v1Cluster := h.toV1Cluster(&organization, &cluster, nil)

	b, err := json.Marshal(&v1Cluster)
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
		Verb:      "get",
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
		logger.Debug("subject is not allowed to get cluster", "subject", subject, "cluster", cluster.Name, "namespace", cluster.Namespace)
		w.WriteHeader(http.StatusUnauthorized)

		return
	}

	if !cluster.Status.APIEndpoint.IsValid() {
		logger.Info("cluster does not have a valid api endpoint", "uid", cluster.UID)
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
		logger.Info("cluster certificate authority has no tls certificate")
		w.WriteHeader(http.StatusInternalServerError)

		return
	}

	signingKeyPEM, has := secret.Data[corev1.TLSPrivateKeyKey]
	if !has {
		logger.Info("cluster certificate authority has to tls private key")
		w.WriteHeader(http.StatusInternalServerError)

		return
	}

	block, _ := pem.Decode(caCertificatePEM)
	if block == nil {
		logger.Info("unable to decode ca certificate as pem")
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
		logger.Info("unexpected users count", "count", len(userList.Items))
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

func (h *handler) DeleteCluster(w http.ResponseWriter, r *http.Request) {
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
		w.WriteHeader(http.StatusUnauthorized)

		return
	}

	if len(clusterList.Items) != 1 {
		logger.Debug("expected exactly one cluster", "count", len(clusterList.Items))
		w.WriteHeader(http.StatusInternalServerError)

		return
	}

	cluster := clusterList.Items[0]

	organization, err := apiutil.GetOwnerOrganization(ctx, h.Client, &cluster)
	if err != nil {
		logger.Error("error getting owner organization", "err", err)
	}

	subject, err := middleware.SubjectFrom(ctx)
	if err != nil {
		logger.Debug("error fetching user from context", "err", err)
		w.WriteHeader(http.StatusInternalServerError)

		return
	}

	isMember := h.isMember(subject, organization)
	if !isMember {
		logger.Debug("subject is not a member of organization", "subject", subject, "organization", organization.Name)
		w.WriteHeader(http.StatusUnauthorized)

		return
	}

	err = h.Delete(ctx, &cluster, client.PropagationPolicy(metav1.DeletePropagationForeground))
	if err != nil {
		logger.Error("error deleting cluster", "err", err)
		w.WriteHeader(http.StatusInternalServerError)

		return
	}

	logger.Debug("deleted cluster", "id", cluster.UID)

	w.WriteHeader(http.StatusAccepted)
}

func (h *handler) GetClusters(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	logger := middleware.LoggerFrom(ctx)

	subject, err := middleware.SubjectFrom(ctx)
	if err != nil {
		logger.Debug("error getting subject from context", "err", err)
		w.WriteHeader(http.StatusInternalServerError)

		return
	}

	matchingFields := client.MatchingFields{
		index.MemberReferencesField: subject,
	}

	var organizationList dockyardsv1.OrganizationList
	err = h.List(ctx, &organizationList, matchingFields)
	if err != nil {
		logger.Error("error listing organizations", "err", err)
		w.WriteHeader(http.StatusInternalServerError)

		return
	}

	clusters := []types.Cluster{}

	for _, organization := range organizationList.Items {
		var clusterList dockyardsv1.ClusterList
		err = h.List(ctx, &clusterList, client.InNamespace(organization.Status.NamespaceRef))
		if err != nil {
			logger.Error("error listing clusters", "err", err)
			w.WriteHeader(http.StatusInternalServerError)

			return
		}

		for _, cluster := range clusterList.Items {
			clusters = append(clusters, *h.toV1Cluster(&organization, &cluster, nil))
		}
	}

	b, err := json.Marshal(&clusters)
	if err != nil {
		logger.Debug("error marshalling response", "err", err)
		w.WriteHeader(http.StatusInternalServerError)

		return
	}

	w.WriteHeader(http.StatusOK)
	_, err = w.Write(b)
	if err != nil {
		logger.Error("error writing response data", "err", err)
	}
}

func (h *handler) GetCluster(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	logger := middleware.LoggerFrom(ctx)

	clusterID := r.PathValue("clusterID")
	if clusterID == "" {
		logger.Error("empty cluster id")
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

	organization, err := apiutil.GetOwnerOrganization(ctx, h.Client, &cluster)
	if client.IgnoreNotFound(err) != nil {
		logger.Error("error getting owner organization", "err", err)
		w.WriteHeader(http.StatusInternalServerError)

		return
	}

	if apierrors.IsNotFound(err) {
		w.WriteHeader(http.StatusUnauthorized)

		return
	}

	subject, err := middleware.SubjectFrom(ctx)
	if err != nil {
		logger.Error("error fetching user from context", "err", err)
		w.WriteHeader(http.StatusInternalServerError)

		return
	}

	isMember := h.isMember(subject, organization)
	if !isMember {
		logger.Debug("subject is not a member of organization", "subject", subject, "organization", organization.Name)
		w.WriteHeader(http.StatusUnauthorized)

		return
	}

	matchingFields = client.MatchingFields{
		index.OwnerReferencesField: clusterID,
	}

	var nodePoolList dockyardsv1.NodePoolList
	err = h.List(ctx, &nodePoolList, matchingFields)
	if err != nil {
		logger.Error("error listing node pools", "err", err)
		w.WriteHeader(http.StatusInternalServerError)

		return
	}

	v1Cluster := h.toV1Cluster(organization, &cluster, &nodePoolList)

	b, err := json.Marshal(&v1Cluster)
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
