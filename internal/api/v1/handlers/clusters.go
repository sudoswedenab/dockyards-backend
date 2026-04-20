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
	"errors"
	"slices"
	"strings"
	"time"

	"github.com/sudoswedenab/dockyards-api/pkg/types"
	"github.com/sudoswedenab/dockyards-backend/api/apiutil"
	"github.com/sudoswedenab/dockyards-backend/api/config"
	dockyardsv1 "github.com/sudoswedenab/dockyards-backend/api/v1alpha3"
	"github.com/sudoswedenab/dockyards-backend/pkg/util/name"
	"gopkg.in/yaml.v3"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation/field"
	apiserverv1 "k8s.io/apiserver/pkg/apis/apiserver/v1beta1"
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

	if cluster.Spec.NoDefaultNetworkPlugin {
		v1Cluster.NoDefaultNetworkPlugin = &cluster.Spec.NoDefaultNetworkPlugin
	}

	if cluster.Spec.NoDefaultIngressProvider {
		v1Cluster.NoDefaultIngressProvider = &cluster.Spec.NoDefaultIngressProvider
	}

	if len(cluster.Spec.PodSubnets) > 0 {
		v1Cluster.PodSubnets = &cluster.Spec.PodSubnets
	}

	if len(cluster.Spec.ServiceSubnets) > 0 {
		v1Cluster.ServiceSubnets = &cluster.Spec.ServiceSubnets
	}

	v1Cluster.AuthenticationConfig = toAuthenticationConfiguration(cluster.Spec.AuthenticationConfig)
	v1Cluster.Advanced = toClusterAdvancedOptions(cluster.Spec.Advanced)

	return &v1Cluster
}

func (h *handler) nodePoolOptionsToNodePool(ctx context.Context, nodePoolOptions *types.NodePoolOptions, cluster *dockyardsv1.Cluster) (*dockyardsv1.NodePool, error) {
	if nodePoolOptions.Name == nil {
		return nil, errors.New("name must not be nil")
	}

	if nodePoolOptions.Quantity == nil {
		return nil, errors.New("quantity must not be nil")
	}

	organization, err := apiutil.GetOwnerOrganization(ctx, h.Client, cluster)
	if err != nil {
		return nil, err
	}

	name := cluster.Name + "-" + *nodePoolOptions.Name
	nodePool := dockyardsv1.NodePool{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
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
				dockyardsv1.LabelOrganizationName: organization.Name,
				dockyardsv1.LabelClusterName:      cluster.Name,
				dockyardsv1.LabelNodePoolName:     name,
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

	if nodePoolOptions.NodeLabels != nil {
		nodePool.Spec.NodeLabels = *nodePoolOptions.NodeLabels
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
	publicNamespace := h.Config.GetValueOrDefault(config.KeyPublicNamespace, "dockyards-public")

	_, validName := name.IsValidName(request.Name)
	if !validName {
		statusError := apierrors.NewInvalid(dockyardsv1.GroupVersion.WithKind(dockyardsv1.ClusterKind).GroupKind(), "", nil)

		return nil, statusError
	}

	if request.NodePoolOptions != nil && request.ClusterTemplateName != nil {
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

	organizationName := organization.Name

	clusterName := request.Name
	cluster := dockyardsv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      clusterName,
			Namespace: organization.Spec.NamespaceRef.Name,
			Labels: map[string]string{
				dockyardsv1.LabelOrganizationName: organizationName,
			},
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

	if request.NoDefaultNetworkPlugin != nil && *request.NoDefaultNetworkPlugin {
		cluster.Spec.NoDefaultNetworkPlugin = true
	}

	if request.NoDefaultIngressProvider != nil && *request.NoDefaultIngressProvider {
		cluster.Spec.NoDefaultIngressProvider = true
	}

	if request.PodSubnets != nil {
		cluster.Spec.PodSubnets = *request.PodSubnets
	}

	if request.ServiceSubnets != nil {
		cluster.Spec.ServiceSubnets = *request.ServiceSubnets
	}

	var errs field.ErrorList
	cluster.Spec.AuthenticationConfig = parseAuthenticationConfiguration(request.AuthenticationConfig, field.NewPath("authentication_config"), &errs)
	cluster.Spec.Advanced = parseAdvancedOptions(request.Advanced, field.NewPath("advanced"), &errs)
	if len(errs) != 0 {
		return nil, apierrors.NewInvalid(dockyardsv1.GroupVersion.WithKind(dockyardsv1.ClusterKind).GroupKind(), "", errs)
	}

	err := h.Create(ctx, &cluster)
	if err != nil {
		return nil, err
	}

	var clusterTemplate *dockyardsv1.ClusterTemplate

	if request.ClusterTemplateName != nil {
		objectKey := client.ObjectKey{
			Name:      *request.ClusterTemplateName,
			Namespace: publicNamespace,
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
			meta := nodePoolTemplate.ObjectMeta
			meta.Name = cluster.Name + "-" + meta.Name
			meta.Namespace = cluster.Namespace

			if meta.Labels == nil {
				meta.Labels = make(map[string]string)
			}
			meta.Labels[dockyardsv1.LabelOrganizationName] = organizationName
			meta.Labels[dockyardsv1.LabelClusterName] = clusterName
			meta.Labels[dockyardsv1.LabelNodePoolName] = meta.Name

			meta.OwnerReferences = []metav1.OwnerReference{
				{
					APIVersion:         dockyardsv1.GroupVersion.String(),
					Kind:               dockyardsv1.ClusterKind,
					Name:               cluster.Name,
					UID:                cluster.UID,
					BlockOwnerDeletion: ptr.To(true),
				},
			}

			nodePool := dockyardsv1.NodePool{
				ObjectMeta: meta,
			}
			nodePoolTemplate.Spec.DeepCopyInto(&nodePool.Spec)

			err = h.Create(ctx, &nodePool)
			if err != nil {
				return nil, err
			}
		}
	}

	if request.NodePoolOptions != nil {
		for _, nodePoolOptions := range *request.NodePoolOptions {
			nodePool, err := h.nodePoolOptionsToNodePool(ctx, &nodePoolOptions, &cluster)
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

func parsePatches(input *[]map[string]any, path *field.Path, errs *field.ErrorList) []dockyardsv1.Patch {
	if input == nil {
		return nil
	}

	if *input == nil {
		return nil
	}

	result := make([]dockyardsv1.Patch, 0, len(*input))

	for i, value := range *input {
		data, err := json.Marshal(value)
		if err != nil {
			add(errs, field.TypeInvalid(path.Index(i), value, "this value could not be encoded to JSON"))

			continue
		}
		result = append(result, dockyardsv1.Patch{Raw: data})
	}

	return result
}

func parseTalosOptions(value *types.ClusterTalosOptions, path *field.Path, errs *field.ErrorList) dockyardsv1.ClusterTalosOptions {
	if value == nil {
		return dockyardsv1.ClusterTalosOptions{}
	}

	return dockyardsv1.ClusterTalosOptions{
		AdditionalSharedConfigPatches:       parsePatches(value.AdditionalSharedConfigPatches, path.Child("additional_shared_config_patches"), errs),
		AdditionalControlPlaneConfigPatches: parsePatches(value.AdditionalControlPlaneConfigPatches, path.Child("additional_control_plane_config_patches"), errs),
		AdditionalWorkerConfigPatches:       parsePatches(value.AdditionalWorkerConfigPatches, path.Child("additional_worker_config_patches"), errs),
	}
}

func parseKubevirtConfig(value *types.ClusterKubevirtOptions, path *field.Path, errs *field.ErrorList) dockyardsv1.ClusterKubevirtOptions {
	if value == nil {
		return dockyardsv1.ClusterKubevirtOptions{}
	}

	return dockyardsv1.ClusterKubevirtOptions{
		Talos: parseTalosOptions(value.Talos, path.Child("talos"), errs),
	}
}

func parseAdvancedOptions(value *types.ClusterAdvancedOptions, path *field.Path, errs *field.ErrorList) dockyardsv1.ClusterAdvancedOptions {
	if value == nil {
		return dockyardsv1.ClusterAdvancedOptions{}
	}

	return dockyardsv1.ClusterAdvancedOptions{
		Kubevirt: parseKubevirtConfig(value.Kubevirt, path.Child("kubevirt"), errs),
	}
}

func parseAuthenticationConfiguration(value *types.AuthenticationConfiguration, path *field.Path, errs *field.ErrorList) *apiserverv1.AuthenticationConfiguration {
	if value == nil {
		return nil
	}

	return &apiserverv1.AuthenticationConfiguration{
		TypeMeta: metav1.TypeMeta{
			Kind:       "AuthenticationConfiguration",
			APIVersion: "apiserver.config.k8s.io/v1",
		},
		JWT:       parseJWTAuthenticator(value.Jwt, path.Child("jwt"), errs),
		Anonymous: parseAnonymousAuthConfig(value.Anonymous, path.Child("anonymous"), errs),
	}
}

func parseJWTAuthenticator(value []types.JwtAuthenticator, path *field.Path, errs *field.ErrorList) []apiserverv1.JWTAuthenticator {
	if value == nil {
		return nil
	}

	urls := make(map[string]bool, len(value))

	result := make([]apiserverv1.JWTAuthenticator, len(value))
	for i, e := range value {
		p := path.Index(i)
		if urls[e.Issuer.URL] {
			add(errs, field.Duplicate(p.Child("issuer").Child("url"), e.Issuer.URL))
		}
		urls[e.Issuer.URL] = true
		result[i] = apiserverv1.JWTAuthenticator{
			Issuer:               parseIssuer(e.Issuer, p.Child("issuer"), errs),
			ClaimValidationRules: parseClaimValidationRules(e.ClaimValidationRules, p.Child("claim_validation_rules"), errs),
			ClaimMappings:        parseClaimMappings(e.ClaimMappings, p.Child("claim_mappings"), errs),
			UserValidationRules:  parseUserValidationRules(e.UserValidationRules, p.Child("user_validation_rules"), errs),
		}
	}

	return result
}

func parseIssuer(value types.Issuer, path *field.Path, errs *field.ErrorList) apiserverv1.Issuer {
	if value.DiscoveryURL != nil {
		if value.URL == *value.DiscoveryURL {
			add(errs, field.Invalid(path.Child("discovery_url"), value.URL, "must be different from .URL"))
		}
	}

	for i, e := range value.Audiences {
		if e == "" {
			add(errs, field.Required(path.Child("audiences").Index(i), ""))
		}
	}

	return apiserverv1.Issuer{
		URL:                  value.URL,
		DiscoveryURL:         value.DiscoveryURL,
		CertificateAuthority: deref(value.CertificateAuthority),
		Audiences:            value.Audiences,
		AudienceMatchPolicy:  parseAudienceMatchPolicy(value.AudienceMatchPolicy, path.Child("audience_match_policy"), errs),
		EgressSelectorType:   parseEgressSelectorType(value.EgressSelectorType, path.Child("egress_selector_type"), errs),
	}
}

func parseClaimValidationRules(value *[]types.ClaimValidationRule, path *field.Path, errs *field.ErrorList) []apiserverv1.ClaimValidationRule {
	if value == nil {
		return nil
	}

	if *value == nil {
		return nil
	}

	result := make([]apiserverv1.ClaimValidationRule, len(*value))
	for i, e := range *value {
		p := path.Index(i)

		claim := deref(e.Claim)
		requiredValue := deref(e.RequiredValue)
		expression := deref(e.RequiredValue)
		message := deref(e.Message)

		if popcount(claim, expression, message) > 1 {
			add(errs, field.Invalid(p, e, ".Claim, .Expression and .Message are mutually exclusive"))
		}

		if popcount(expression, claim, requiredValue) > 1 {
			add(errs, field.Invalid(p, e, ".Expression, .Claim and .RequiredValue are mutually exclusive"))
		}

		if popcount(message, claim, requiredValue) > 1 {
			add(errs, field.Invalid(p, e, ".Message, .Claim and .RequiredValue are mutually exclusive"))
		}

		if popcount(requiredValue, expression, message) > 1 {
			add(errs, field.Invalid(p, e, ".RequiredValue, .Expression and .Message are mutually exclusive"))
		}

		if requiredValue == "" && claim != "" {
			add(errs, field.Invalid(p, e, "If .Claim is set and .RequiredValue is not set, the claim must be present with a value set to the empty string"))
		}

		result[i] = apiserverv1.ClaimValidationRule{
			Claim:         claim,
			RequiredValue: requiredValue,
			Expression:    expression,
			Message:       message,
		}
	}

	return result
}

func popcount[T comparable](v ...T) int {
	c := 0

	zeros := *new(T)

	for _, e := range v {
		if e == zeros {
			continue
		}
		c++
	}

	return c
}

func parseClaimMappings(value types.ClaimMappings, path *field.Path, errs *field.ErrorList) apiserverv1.ClaimMappings {
	return apiserverv1.ClaimMappings{
		Username: parsePrefixedClaimOrExpression(value.Username, path.Child("username"), errs),
		Groups:   parsePrefixedClaimOrExpression(deref(value.Groups), path.Child("groups"), errs),
		UID:      parseClaimOrExpression(deref(value.UID), path.Child("uid"), errs),
		Extra:    parseExtraMapping(value.Extra, path.Child("extra"), errs),
	}
}

func parsePrefixedClaimOrExpression(value types.PrefixedClaimOrExpression, path *field.Path, errs *field.ErrorList) apiserverv1.PrefixedClaimOrExpression {
	claim := deref(value.Claim)
	expression := deref(value.Expression)
	prefix := value.Prefix

	if popcount(claim, expression) > 1 {
		add(errs, field.Invalid(path, value, ".Claim and .Expression are mutually exclusive"))
	}

	if claim != "" && prefix == nil {
		add(errs, field.Invalid(path, value, "If .Claim is set, .Prefix must be set (can be the empty string)"))
	}

	return apiserverv1.PrefixedClaimOrExpression{
		Claim:      deref(value.Claim),
		Prefix:     value.Prefix,
		Expression: deref(value.Expression),
	}
}

func parseClaimOrExpression(value types.ClaimOrExpression, path *field.Path, errs *field.ErrorList) apiserverv1.ClaimOrExpression {
	claim := deref(value.Claim)
	expression := deref(value.Expression)

	if popcount(claim, expression) > 1 {
		add(errs, field.Invalid(path, value, ".Claim and .Expression are mutually exclusive"))
	}

	return apiserverv1.ClaimOrExpression{
		Claim:      deref(value.Claim),
		Expression: deref(value.Expression),
	}
}

func parseExtraMapping(value *[]types.ExtraMapping, path *field.Path, errs *field.ErrorList) []apiserverv1.ExtraMapping {
	if value == nil {
		return nil
	}

	if *value == nil {
		return nil
	}

	seenBefore := make(map[string]bool, len(*value))

	result := make([]apiserverv1.ExtraMapping, len(*value))
	for i, e := range result {
		p := path.Index(i)

		if e.Key != strings.ToLower(e.Key) {
			add(errs, field.Invalid(p.Child("key"), e.Key, "must be lowercase"))
		}

		if seenBefore[e.Key] {
			add(errs, field.Duplicate(p.Child("key"), e.Key))
		}
		seenBefore[e.Key] = true

		result[i] = apiserverv1.ExtraMapping{
			Key:             e.Key,
			ValueExpression: e.ValueExpression,
		}
	}

	return result
}

func parseUserValidationRules(value *[]types.UserValidationRule, path *field.Path, errs *field.ErrorList) []apiserverv1.UserValidationRule {
	_ = path
	_ = errs

	if value == nil {
		return nil
	}

	if *value == nil {
		return nil
	}

	result := make([]apiserverv1.UserValidationRule, len(*value))
	for i, e := range *value {
		result[i] = apiserverv1.UserValidationRule{
			Expression: e.Expression,
			Message:    deref(e.Message),
		}
	}

	return result
}

func parseAudienceMatchPolicy(value *string, path *field.Path, errs *field.ErrorList) apiserverv1.AudienceMatchPolicyType {
	_ = path
	_ = errs

	if value == nil {
		return ""
	}

	return apiserverv1.AudienceMatchPolicyType(*value)
}

func parseEgressSelectorType(value *string, path *field.Path, errs *field.ErrorList) apiserverv1.EgressSelectorType {
	_ = path
	_ = errs

	if value == nil {
		return ""
	}

	return apiserverv1.EgressSelectorType(*value)
}

func parseAnonymousAuthConfig(value *types.AnonymousAuthConfig, path *field.Path, errs *field.ErrorList) *apiserverv1.AnonymousAuthConfig {
	if value == nil {
		return nil
	}

	return &apiserverv1.AnonymousAuthConfig{
		Enabled:    deref(value.Enabled),
		Conditions: parseAnonymousAuthCondition(value.Conditions, path.Child("conditions"), errs),
	}
}

func parseAnonymousAuthCondition(value *[]types.AnonymousAuthCondition, path *field.Path, errs *field.ErrorList) []apiserverv1.AnonymousAuthCondition {
	_ = path
	_ = errs

	if value == nil {
		return nil
	}

	if *value == nil {
		return nil
	}

	result := make([]apiserverv1.AnonymousAuthCondition, len(*value))
	for i, e := range *value {
		result[i] = apiserverv1.AnonymousAuthCondition{
			Path: deref(e.Path),
		}
	}

	return result
}

func deref[T any](value *T) T {
	if value == nil {
		return *new(T)
	}

	return *value
}

func add(errs *field.ErrorList, err *field.Error) {
	if errs == nil {
		return
	}

	*errs = append(*errs, err)
}

func toAuthenticationConfiguration(value *apiserverv1.AuthenticationConfiguration) *types.AuthenticationConfiguration {
	if value == nil {
		return nil
	}

	return &types.AuthenticationConfiguration{
		Anonymous: toAnonymousAuthConfig(value.Anonymous),
		Jwt:       toJwtAuthenticator(value.JWT),
	}
}

func toJwtAuthenticator(value []apiserverv1.JWTAuthenticator) []types.JwtAuthenticator {
	if value == nil {
		return nil
	}

	result := make([]types.JwtAuthenticator, len(value))
	for i, e := range value {
		result[i] = types.JwtAuthenticator{
			ClaimMappings:        toClaimMappings(e.ClaimMappings),
			ClaimValidationRules: toClaimValidationRules(e.ClaimValidationRules),
			Issuer:               toIssuer(e.Issuer),
			UserValidationRules:  toUserValidationRules(e.UserValidationRules),
		}
	}

	return result
}

func toClaimMappings(value apiserverv1.ClaimMappings) types.ClaimMappings {
	return types.ClaimMappings{
		Extra:    toExtraMapping(value.Extra),
		Groups:   toPrefixedClaimOrExpression(value.Groups),
		UID:      toClaimOrExpression(value.UID),
		Username: deref(toPrefixedClaimOrExpression(value.Username)),
	}
}

func toPrefixedClaimOrExpression(value apiserverv1.PrefixedClaimOrExpression) *types.PrefixedClaimOrExpression {
	zeros := apiserverv1.PrefixedClaimOrExpression{}
	if value == zeros {
		return nil
	}

	return &types.PrefixedClaimOrExpression{
		Claim:      toString(value.Claim),
		Expression: toString(value.Expression),
		Prefix:     clone(value.Prefix),
	}
}

func toClaimOrExpression(value apiserverv1.ClaimOrExpression) *types.ClaimOrExpression {
	zeros := apiserverv1.ClaimOrExpression{}
	if value == zeros {
		return nil
	}

	return &types.ClaimOrExpression{
		Claim:      toString(value.Claim),
		Expression: toString(value.Expression),
	}
}

func toExtraMapping(value []apiserverv1.ExtraMapping) *[]types.ExtraMapping {
	if value == nil {
		return nil
	}

	result := make([]types.ExtraMapping, len(value))
	for i, e := range result {
		result[i] = types.ExtraMapping{
			Key:             e.Key,
			ValueExpression: e.ValueExpression,
		}
	}

	return &result
}

func toClaimValidationRules(value []apiserverv1.ClaimValidationRule) *[]types.ClaimValidationRule {
	if value == nil {
		return nil
	}

	result := make([]types.ClaimValidationRule, len(value))
	for i, e := range value {
		result[i] = types.ClaimValidationRule{
			Claim:         toString(e.Claim),
			Expression:    toString(e.Expression),
			Message:       toString(e.Message),
			RequiredValue: toString(e.RequiredValue),
		}
	}

	return &result
}

func toIssuer(value apiserverv1.Issuer) types.Issuer {
	return types.Issuer{
		AudienceMatchPolicy:  toString(string(value.AudienceMatchPolicy)),
		Audiences:            slices.Clone(value.Audiences),
		CertificateAuthority: toString(value.CertificateAuthority),
		DiscoveryURL:         value.DiscoveryURL,
		EgressSelectorType:   toString(string(value.EgressSelectorType)),
		URL:                  value.URL,
	}
}

func clone(value *string) *string {
	if value == nil {
		return nil
	}

	return ptr.To(*value)
}

func toUserValidationRules(value []apiserverv1.UserValidationRule) *[]types.UserValidationRule {
	if value == nil {
		return nil
	}

	result := make([]types.UserValidationRule, len(value))
	for i, e := range result {
		result[i] = types.UserValidationRule{
			Expression: e.Expression,
			Message:    clone(e.Message),
		}
	}

	return &result
}

func toAnonymousAuthConfig(value *apiserverv1.AnonymousAuthConfig) *types.AnonymousAuthConfig {
	if value == nil {
		return nil
	}

	return &types.AnonymousAuthConfig{
		Conditions: toAnonymousAuthConditions(value.Conditions),
		Enabled:    ptr.To(value.Enabled),
	}
}

func toAnonymousAuthConditions(value []apiserverv1.AnonymousAuthCondition) *[]types.AnonymousAuthCondition {
	if value == nil {
		return nil
	}

	result := make([]types.AnonymousAuthCondition, len(value))
	for i, e := range value {
		result[i] = types.AnonymousAuthCondition{
			Path: toString(e.Path),
		}
	}

	return &result
}

func toString(value string) *string {
	if value == "" {
		return nil
	}

	return &value
}

func toPatches(value []dockyardsv1.Patch) *[]map[string]any {
	if len(value) == 0 {
		return nil
	}

	result := make([]map[string]any, 0, len(value))
	for _, v := range value {
		patch := map[string]any{}

		err := json.Unmarshal(v.Raw, &patch)
		if err == nil {
			result = append(result, patch)

			continue
		}

		// Failed to decode JSON, so maybe someone has poked some YAML inside instead.
		err = yaml.Unmarshal(v.Raw, &patch)
		if err == nil {
			result = append(result, patch)

			continue
		}

		// RawExtension is documented to only be JSON or YAML, but in case we fail anyway,
		// let's just strip it from the response since the response was otherwise successful.
		continue
	}

	return &result
}

func toClusterTalosOptions(value dockyardsv1.ClusterTalosOptions) *types.ClusterTalosOptions {
	if value.IsZero() {
		return nil
	}

	return &types.ClusterTalosOptions{
		ExternalNodeInterface:               ptr.To(value.ExternalNodeInterface),
		ExternalNodeIpv4Subnet:              ptr.To(value.ExternalNodeIPv4Subnet),
		AdditionalSharedConfigPatches:       toPatches(value.AdditionalSharedConfigPatches),
		AdditionalControlPlaneConfigPatches: toPatches(value.AdditionalControlPlaneConfigPatches),
		AdditionalWorkerConfigPatches:       toPatches(value.AdditionalWorkerConfigPatches),
	}
}

func toClusterKubevirtOptions(value dockyardsv1.ClusterKubevirtOptions) *types.ClusterKubevirtOptions {
	if value.IsZero() {
		return nil
	}

	return &types.ClusterKubevirtOptions{
		Talos: toClusterTalosOptions(value.Talos),
	}
}

func toClusterAdvancedOptions(value dockyardsv1.ClusterAdvancedOptions) *types.ClusterAdvancedOptions {
	if value.IsZero() {
		return nil
	}

	return &types.ClusterAdvancedOptions{
		Kubevirt: toClusterKubevirtOptions(value.Kubevirt),
	}
}
