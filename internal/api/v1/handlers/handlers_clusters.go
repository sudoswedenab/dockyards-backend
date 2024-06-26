package handlers

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math"
	"math/big"
	"net/http"
	"time"

	"bitbucket.org/sudosweden/dockyards-backend/internal/api/v1"
	"bitbucket.org/sudosweden/dockyards-backend/internal/util"
	"bitbucket.org/sudosweden/dockyards-backend/pkg/api/apiutil"
	dockyardsv1 "bitbucket.org/sudosweden/dockyards-backend/pkg/api/v1alpha2"
	"bitbucket.org/sudosweden/dockyards-backend/pkg/api/v1alpha2/index"
	"bitbucket.org/sudosweden/dockyards-backend/pkg/util/name"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	authorizationv1 "k8s.io/api/authorization/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// +kubebuilder:rbac:groups=authorization.k8s.io,resources=subjectaccessreviews,verbs=create
// +kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch
// +kubebuilder:rbac:groups=dockyards.io,resources=clusters,verbs=get;list;watch;create;delete
// +kubebuilder:rbac:groups=dockyards.io,resources=nodepools,verbs=create
// +kubebuilder:rbac:groups=dockyards.io,resources=organizations,verbs=get;list;watch

func (h *handler) toV1Cluster(organization *dockyardsv1.Organization, cluster *dockyardsv1.Cluster, nodePoolList *dockyardsv1.NodePoolList) *v1.Cluster {
	v1Cluster := v1.Cluster{
		Id:           string(cluster.UID),
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
		nodePools := make([]v1.NodePool, len(nodePoolList.Items))
		for i, nodePool := range nodePoolList.Items {
			nodePools[i] = *h.toV1NodePool(&nodePool, cluster, nil)
		}

		v1Cluster.NodePools = nodePools
	}

	return &v1Cluster
}

func (h *handler) PostOrgClusters(c *gin.Context) {
	ctx := context.Background()

	org := c.Param("org")
	if org == "" {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	objectKey := client.ObjectKey{
		Name: org,
	}

	var organization dockyardsv1.Organization
	err := h.controllerClient.Get(ctx, objectKey, &organization)
	if err != nil {
		h.logger.Error("error getting organization from kubernetes", "err", err)

		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	subject, err := h.getSubjectFromContext(c)
	if err != nil {
		h.logger.Debug("error fetching subject from context", "err", err)

		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	isMember := h.isMember(subject, &organization)
	if !isMember {
		h.logger.Debug("subject is not a member of organization", "subject", subject, "organization", organization.Name)

		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	var clusterOptions v1.ClusterOptions
	if c.BindJSON(&clusterOptions) != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Failed to read Body",
		})
		return
	}

	h.logger.Debug("create cluster", "organization", organization.Name, "name", clusterOptions.Name, "version", clusterOptions.Version)

	details, validName := name.IsValidName(clusterOptions.Name)
	if !validName {
		c.JSON(http.StatusUnprocessableEntity, gin.H{
			"error":   "name is not valid",
			"name":    clusterOptions.Name,
			"details": details,
		})
		return
	}

	if clusterOptions.NodePoolOptions != nil {
		for _, nodePoolOptions := range *clusterOptions.NodePoolOptions {
			details, validName := name.IsValidName(nodePoolOptions.Name)
			if !validName {
				c.JSON(http.StatusUnprocessableEntity, gin.H{
					"error":   "node pool name is not valid",
					"name":    nodePoolOptions.Name,
					"details": details,
				})
				return
			}

			if nodePoolOptions.Quantity > 9 {
				h.logger.Debug("quantity too large", "quantity", nodePoolOptions.Quantity)

				c.AbortWithStatusJSON(http.StatusUnprocessableEntity, gin.H{
					"error":    "node pool quota exceeded",
					"quantity": nodePoolOptions.Quantity,
					"details":  "quantity must be lower than 9",
				})
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
					BlockOwnerDeletion: util.Ptr(true),
				},
			},
		},
		Spec: dockyardsv1.ClusterSpec{},
	}

	if clusterOptions.Version != nil {
		cluster.Spec.Version = *clusterOptions.Version
	}

	err = h.controllerClient.Create(ctx, &cluster)
	if err != nil {
		h.logger.Error("error creating cluster", "err", err)

		if apierrors.IsAlreadyExists(err) {
			c.AbortWithStatus(http.StatusConflict)
			return
		}

		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	nodePoolOptions := clusterOptions.NodePoolOptions
	if nodePoolOptions == nil || len(*nodePoolOptions) == 0 {
		h.logger.Debug("using recommended node pool options")

		objectKey := client.ObjectKey{
			Name:      "recommended",
			Namespace: h.namespace,
		}

		var clusterTemplate dockyardsv1.ClusterTemplate
		err := h.controllerClient.Get(ctx, objectKey, &clusterTemplate)
		if err != nil {
			h.logger.Error("error getting cluster template", "err", err)

			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		nodePoolOptions = util.Ptr(getRecommendedNodePools(&clusterTemplate))
	}

	if clusterOptions.SingleNode != nil && *clusterOptions.SingleNode {
		h.logger.Debug("using single node pool")

		nodePoolOptions = util.Ptr([]v1.NodePoolOptions{
			{
				Name:         "single-node",
				Quantity:     1,
				ControlPlane: util.Ptr(true),
				LoadBalancer: util.Ptr(true),
			},
		})
	}

	nodePoolList := dockyardsv1.NodePoolList{
		Items: make([]dockyardsv1.NodePool, len(*nodePoolOptions)),
	}

	hasErrors := false
	for i, nodePoolOption := range *nodePoolOptions {
		nodePool := dockyardsv1.NodePool{
			ObjectMeta: metav1.ObjectMeta{
				Name:      cluster.Name + "-" + nodePoolOption.Name,
				Namespace: cluster.Namespace,
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion:         dockyardsv1.GroupVersion.String(),
						Kind:               dockyardsv1.ClusterKind,
						Name:               cluster.Name,
						UID:                cluster.UID,
						BlockOwnerDeletion: util.Ptr(true),
					},
				},
			},
			Spec: dockyardsv1.NodePoolSpec{
				Replicas: util.Ptr(int32(nodePoolOption.Quantity)),
			},
		}

		if nodePoolOption.ControlPlane != nil {
			nodePool.Spec.ControlPlane = *nodePoolOption.ControlPlane
		}

		if nodePoolOption.LoadBalancer != nil {
			nodePool.Spec.LoadBalancer = *nodePoolOption.LoadBalancer
		}

		if nodePoolOption.ControlPlaneComponentsOnly != nil {
			nodePool.Spec.DedicatedRole = *nodePoolOption.ControlPlaneComponentsOnly
		}

		nodePool.Spec.Resources = corev1.ResourceList{}

		if nodePoolOption.CpuCount != nil {
			quantity := resource.NewQuantity(int64(*nodePoolOption.CpuCount), resource.BinarySI)

			nodePool.Spec.Resources[corev1.ResourceCPU] = *quantity
		}

		if nodePoolOption.DiskSize != nil {
			quantity, err := resource.ParseQuantity(*nodePoolOption.DiskSize)
			if err != nil {
				h.logger.Error("error parsing disk size quantity", "err", err)

				hasErrors = true
				break
			}

			nodePool.Spec.Resources[corev1.ResourceStorage] = quantity
		}

		if nodePoolOption.RamSize != nil {
			quantity, err := resource.ParseQuantity(*nodePoolOption.RamSize)
			if err != nil {
				h.logger.Error("error parsing ram size quantity", "err", err)

				hasErrors = true
				break
			}

			nodePool.Spec.Resources[corev1.ResourceMemory] = quantity
		}

		err := h.controllerClient.Create(ctx, &nodePool)
		if err != nil {
			h.logger.Error("error creating node pool", "err", err)

			hasErrors = true
			break
		}

		nodePoolList.Items[i] = nodePool

		h.logger.Debug("created cluster node pool", "id", nodePool.UID)
	}

	if hasErrors {
		h.logger.Error("deleting cluster", "id", cluster.UID)

		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	v1Cluster := v1.Cluster{
		Id: string(cluster.UID),
	}

	c.JSON(http.StatusCreated, v1Cluster)
}

func (h *handler) GetClusterKubeconfig(c *gin.Context) {
	ctx := context.Background()

	clusterID := c.Param("clusterID")
	if clusterID == "" {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	matchingFields := client.MatchingFields{
		index.UIDField: clusterID,
	}

	var clusterList dockyardsv1.ClusterList
	err := h.controllerClient.List(ctx, &clusterList, matchingFields)
	if err != nil {
		h.logger.Error("error listing clusters", "err", err)

		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	if len(clusterList.Items) != 1 {
		h.logger.Debug("expected exactly one cluster", "count", len(clusterList.Items))

		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	cluster := clusterList.Items[0]

	subject, err := h.getSubjectFromContext(c)
	if err != nil {
		h.logger.Error("error fetching user from context", "err", err)

		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	resourceAttributes := authorizationv1.ResourceAttributes{
		Verb:      "get",
		Resource:  "clusters",
		Group:     "dockyards.io",
		Namespace: cluster.Namespace,
	}

	allowed, err := apiutil.IsSubjectAllowed(ctx, h.controllerClient, subject, &resourceAttributes)
	if err != nil {
		h.logger.Error("error reviewing subject", "err", err)
		c.AbortWithStatus(http.StatusInternalServerError)

		return
	}

	if !allowed {
		h.logger.Debug("subject is not allowed to get cluster", "subject", subject, "cluster", cluster.Name, "namespace", cluster.Namespace)
		c.AbortWithStatus(http.StatusUnauthorized)

		return
	}

	if !cluster.Status.APIEndpoint.IsValid() {
		h.logger.Info("cluster does not have a valid api endpoint", "uid", cluster.UID)
		c.AbortWithStatus(http.StatusInternalServerError)

		return
	}

	objectKey := client.ObjectKey{
		Name:      cluster.Name + "-ca",
		Namespace: cluster.Namespace,
	}

	var secret corev1.Secret
	err = h.controllerClient.Get(ctx, objectKey, &secret)
	if err != nil {
		h.logger.Error("error getting cluster certificate authority", "err", err)
		c.AbortWithStatus(http.StatusInternalServerError)

		return
	}

	caCertificatePEM, has := secret.Data[corev1.TLSCertKey]
	if !has {
		h.logger.Info("cluster certificate authority has no tls certificate")
		c.AbortWithStatus(http.StatusInternalServerError)

		return
	}

	signingKeyPEM, has := secret.Data[corev1.TLSPrivateKeyKey]
	if !has {
		h.logger.Info("cluster certificate authority has to tls private key")
		c.AbortWithStatus(http.StatusInternalServerError)

		return
	}

	block, _ := pem.Decode(caCertificatePEM)
	if block == nil {
		h.logger.Info("unable to decode ca certificate as pem")
		c.AbortWithStatus(http.StatusInternalServerError)

		return
	}

	caCertificate, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		h.logger.Error("error parsing ca certificate", "err", err)
		c.AbortWithStatus(http.StatusInternalServerError)

		return
	}

	block, _ = pem.Decode(signingKeyPEM)

	signingKey, err := x509.ParseECPrivateKey(block.Bytes)
	if err != nil {
		h.logger.Error("error parsing signing key", "err", err)
		c.AbortWithStatus(http.StatusInternalServerError)

		return
	}

	matchingFields = client.MatchingFields{
		index.UIDField: subject,
	}

	var userList dockyardsv1.UserList
	err = h.controllerClient.List(ctx, &userList, matchingFields)
	if err != nil {
		h.logger.Error("error listing users", "err", err)
		c.AbortWithStatus(http.StatusInternalServerError)

		return
	}

	if len(userList.Items) != 1 {
		h.logger.Info("unexpected users count", "count", len(userList.Items))
		c.AbortWithStatus(http.StatusInternalServerError)

		return
	}

	user := userList.Items[0]

	contextName := user.Name + "@" + cluster.Name

	serial, err := rand.Int(rand.Reader, big.NewInt(math.MaxInt64))
	if err != nil {
		h.logger.Error("error generating random serial number", "err", err)
		c.AbortWithStatus(http.StatusInternalServerError)

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
		h.logger.Error("error generating private rsa key", "err", err)
		c.AbortWithStatus(http.StatusInternalServerError)

		return
	}

	certificate, err := x509.CreateCertificate(rand.Reader, &tmpl, caCertificate, privateKey.Public(), signingKey)
	if err != nil {
		h.logger.Error("error creating certificate", "err", err)
		c.AbortWithStatus(http.StatusInternalServerError)

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

	kubeconfig, err := clientcmd.Write(cfg)
	if err != nil {
		h.logger.Error("error marshalling kubeconfig", "err", err)
		c.AbortWithStatus(http.StatusInternalServerError)

		return
	}

	c.Data(http.StatusOK, binding.MIMEYAML, kubeconfig)
}

func (h *handler) DeleteCluster(c *gin.Context) {
	ctx := context.Background()

	clusterID := c.Param("clusterID")
	if clusterID == "" {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	matchingFields := client.MatchingFields{
		index.UIDField: clusterID,
	}

	var clusterList dockyardsv1.ClusterList
	err := h.controllerClient.List(ctx, &clusterList, matchingFields)
	if err != nil {
		h.logger.Error("error listing clusters", "err", err)

		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	if len(clusterList.Items) != 1 {
		h.logger.Debug("expected exactly one cluster", "count", len(clusterList.Items))

		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	cluster := clusterList.Items[0]

	organization, err := apiutil.GetOwnerOrganization(ctx, h.controllerClient, &cluster)
	if err != nil {
		h.logger.Error("error getting owner organization", "err", err)
	}

	subject, err := h.getSubjectFromContext(c)
	if err != nil {
		h.logger.Debug("error fetching user from context", "err", err)

		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	isMember := h.isMember(subject, organization)
	if !isMember {
		h.logger.Debug("subject is not a member of organization", "subject", subject, "organization", organization.Name)

		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	err = h.controllerClient.Delete(ctx, &cluster, client.PropagationPolicy(metav1.DeletePropagationForeground))
	if err != nil {
		h.logger.Error("error deleting cluster", "err", err)

		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	h.logger.Debug("deleted cluster", "id", cluster.UID)

	c.JSON(http.StatusAccepted, gin.H{})
}

func (h *handler) GetClusters(c *gin.Context) {
	ctx := context.Background()

	subject, err := h.getSubjectFromContext(c)
	if err != nil {
		h.logger.Debug("error getting subject from context", "err", err)

		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	matchingFields := client.MatchingFields{
		index.MemberRefsIndexKey: subject,
	}

	var organizationList dockyardsv1.OrganizationList
	err = h.controllerClient.List(ctx, &organizationList, matchingFields)
	if err != nil {
		h.logger.Error("error listing organizations", "err", err)

		c.Status(http.StatusInternalServerError)
		return
	}

	clusters := []v1.Cluster{}

	for _, organization := range organizationList.Items {
		var clusterList dockyardsv1.ClusterList
		err = h.controllerClient.List(ctx, &clusterList, client.InNamespace(organization.Status.NamespaceRef))
		if err != nil {
			h.logger.Error("error listing clusters", "err", err)

			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		for _, cluster := range clusterList.Items {
			clusters = append(clusters, *h.toV1Cluster(&organization, &cluster, nil))
		}
	}

	c.JSON(http.StatusOK, clusters)
}

func (h *handler) GetCluster(c *gin.Context) {
	ctx := context.Background()

	clusterID := c.Param("clusterID")
	if clusterID == "" {
		h.logger.Error("empty cluster id")

		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	matchingFields := client.MatchingFields{
		index.UIDField: clusterID,
	}

	var clusterList dockyardsv1.ClusterList
	err := h.controllerClient.List(ctx, &clusterList, matchingFields)
	if err != nil {
		h.logger.Error("error listing clusters", "err", err)

		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	if len(clusterList.Items) != 1 {
		h.logger.Debug("expected exactly one cluster", "count", len(clusterList.Items))

		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	cluster := clusterList.Items[0]

	organization, err := apiutil.GetOwnerOrganization(ctx, h.controllerClient, &cluster)
	if err != nil {
		h.logger.Error("error getting owner organization", "err", err)

		if apierrors.IsNotFound(err) {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	subject, err := h.getSubjectFromContext(c)
	if err != nil {
		h.logger.Error("error fetching user from context", "err", err)

		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	isMember := h.isMember(subject, organization)
	if !isMember {
		h.logger.Debug("subject is not a member of organization", "subject", subject, "organization", organization.Name)

		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	matchingFields = client.MatchingFields{
		index.OwnerReferencesField: clusterID,
	}

	var nodePoolList dockyardsv1.NodePoolList
	err = h.controllerClient.List(ctx, &nodePoolList, matchingFields)
	if err != nil {
		h.logger.Error("error listing node pools", "err", err)

		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	v1Cluster := h.toV1Cluster(organization, &cluster, &nodePoolList)

	c.JSON(http.StatusOK, v1Cluster)
}
