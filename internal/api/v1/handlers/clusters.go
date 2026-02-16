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
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/sudoswedenab/dockyards-api/pkg/types"
	"github.com/sudoswedenab/dockyards-backend/api/apiutil"
	"github.com/sudoswedenab/dockyards-backend/api/config"
	dockyardsv1 "github.com/sudoswedenab/dockyards-backend/api/v1alpha3"
	"github.com/sudoswedenab/dockyards-backend/pkg/util/name"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

	if len(cluster.Spec.PodSubnets) > 0 {
		v1Cluster.PodSubnets = &cluster.Spec.PodSubnets
	}

	if len(cluster.Spec.ServiceSubnets) > 0 {
		v1Cluster.ServiceSubnets = &cluster.Spec.ServiceSubnets
	}

	v1Cluster.AuthenticationConfig = toAuthenticationConfiguration(cluster.Spec.AuthenticationConfig)

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

	if request.PodSubnets != nil {
		cluster.Spec.PodSubnets = *request.PodSubnets
	}

	if request.ServiceSubnets != nil {
		cluster.Spec.ServiceSubnets = *request.ServiceSubnets
	}

	var errs []error
	cluster.Spec.AuthenticationConfig = parseAuthenticationConfiguration(request.AuthenticationConfig, ".AuthenticationConfig", &errs)
	if len(errs) != 0 {
		return nil, apierrors.NewBadRequest(errors.Join(errs...).Error())
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

func parseAuthenticationConfiguration(value *types.AuthenticationConfiguration, path string, errs *[]error) *apiserverv1.AuthenticationConfiguration {
	if value == nil {
		return nil
	}

	return &apiserverv1.AuthenticationConfiguration{
		TypeMeta: metav1.TypeMeta{
			Kind: "AuthenticationConfiguration",
			APIVersion: "apiserver.config.k8s.io/v1",
		},
		JWT: parseJWTAuthenticator(value.Jwt, path + ".JWT", errs),
		Anonymous: parseAnonymousAuthConfig(value.Anonymous, path + ".Anonymous", errs),
	}
}

func parseJWTAuthenticator(value []types.JwtAuthenticator, path string, errs *[]error) []apiserverv1.JWTAuthenticator {
	if value == nil {
		return nil
	}

	urls := make(map[string]bool, len(value))

	result := make([]apiserverv1.JWTAuthenticator, len(value))
	for i, e := range value {
		p := fmt.Sprintf("%s[%d]", path, i)
		if urls[e.Issuer.URL] {
			add(errs, fmt.Errorf("%s: .Issuer.URLs must be unique", p))
		}
		urls[e.Issuer.URL] = true
		result[i] = apiserverv1.JWTAuthenticator{
			Issuer: parseIssuer(e.Issuer, p + ".Issuer", errs),
			ClaimValidationRules: parseClaimValidationRules(e.ClaimValidationRules, p + ".ClaimValidationRules", errs),
			ClaimMappings: parseClaimMappings(e.ClaimMappings, p + ".ClaimMappings", errs),
			UserValidationRules: parseUserValidationRules(e.UserValidationRules, p + ".UserValidationRules", errs),
		}
	}

	return result
}

func parseIssuer(value types.Issuer, path string, errs *[]error) apiserverv1.Issuer {
	if value.DiscoveryURL != nil {
		if value.URL == *value.DiscoveryURL {
			add(errs, fmt.Errorf("%s: .DiscoveryURL must be different from .URL", path))
		}
	}

	for i, e := range value.Audiences {
		if e == "" {
			add(errs, fmt.Errorf("%s.Audiences[%d]: Required to be non-empty", path, i))
		}
	}

	return apiserverv1.Issuer{
		URL: value.URL,
		DiscoveryURL: value.DiscoveryURL,
		CertificateAuthority: deref(value.CertificateAuthority),
		Audiences: value.Audiences,
		AudienceMatchPolicy: parseAudienceMatchPolicy(value.AudienceMatchPolicy, path + ".AudienceMatchPolicy", errs),
		EgressSelectorType: parseEgressSelectorType(value.EgressSelectorType, path + ".EgressSelectorType", errs),
	}
}

func parseClaimValidationRules(value *[]types.ClaimValidationRule, path string, errs *[]error) []apiserverv1.ClaimValidationRule {
	if value == nil {
		return nil
	}

	if *value == nil {
		return nil
	}
	result := make([]apiserverv1.ClaimValidationRule, len(*value))
	for i, e := range *value {
		p := fmt.Sprintf("%s[%d]", path, i)

		claim := deref(e.Claim)
		requiredValue := deref(e.RequiredValue)
		expression := deref(e.RequiredValue)
		message := deref(e.Message)

		if popcount(claim, expression, message) > 1 {
			add(errs, fmt.Errorf("%s: .Claim, .Expression and .Message are mutually exclusive", p))
		}

		if popcount(expression, claim, requiredValue) > 1 {
			add(errs, fmt.Errorf("%s: .Expression, .Claim and .RequiredValue are mutually exclusive", p))
		}

		if popcount(message, claim, requiredValue) > 1 {
			add(errs, fmt.Errorf("%s: .Message, .Claim and .RequiredValue are mutually exclusive", p))
		}

		if popcount(requiredValue, expression, message) > 1 {
			add(errs, fmt.Errorf("%s: .RequiredValue, .Expression and .Message are mutually exclusive", p))
		}

		if requiredValue == "" && claim != "" {
			add(errs, fmt.Errorf("%s: If claim is set and required is not set, the claim must be present with a value set to the empty string", p))
		}

		result[i] = apiserverv1.ClaimValidationRule{
			Claim: claim,
			RequiredValue: requiredValue,
			Expression: expression,
			Message: message,
		}
	}

	return result
}

func popcount[T comparable](v... T) int {
	c := 0

	zeros := *new(T)

	for _, e := range v {
		if e != zeros {
			continue
		}
		c++
	}

	return c
}

func parseClaimMappings(value types.ClaimMappings, path string, errs *[]error) apiserverv1.ClaimMappings {
	return apiserverv1.ClaimMappings{
		Username: parsePrefixedClaimOrExpression(value.Username, path + ".Username", errs),
		Groups: parsePrefixedClaimOrExpression(deref(value.Groups), path + ".Groups", errs),
		UID: parseClaimOrExpression(deref(value.UID), path + ".UID", errs),
		Extra: parseExtraMapping(value.Extra, path + ".Extra", errs),
	}
}

func parsePrefixedClaimOrExpression(value types.PrefixedClaimOrExpression, path string, errs *[]error) apiserverv1.PrefixedClaimOrExpression {
	claim := deref(value.Claim)
	expression := deref(value.Expression)
	prefix := value.Prefix

	if popcount(claim, expression) > 1 {
		add(errs, fmt.Errorf("%s: .Claim and .Expression are mutually exclusive", path))
	}

	if claim != "" && prefix == nil {
		add(errs, fmt.Errorf("%s: If .Claim is set, .Prefix must be set (can be the empty string)", path))
	}

	return apiserverv1.PrefixedClaimOrExpression{
		Claim: deref(value.Claim),
		Prefix: value.Prefix,
		Expression: deref(value.Expression),
	}
}

func parseClaimOrExpression(value types.ClaimOrExpression, path string, errs *[]error) apiserverv1.ClaimOrExpression {
	claim := deref(value.Claim)
	expression := deref(value.Expression)

	if popcount(claim, expression) > 1 {
		add(errs, fmt.Errorf("%s: .Claim and .Expression are mutually exclusive", path))
	}

	return apiserverv1.ClaimOrExpression{
		Claim: deref(value.Claim),
		Expression: deref(value.Expression),
	}
}

func parseExtraMapping(value *[]types.ExtraMapping, path string, errs *[]error) []apiserverv1.ExtraMapping {
	if value == nil {
		return nil
	}

	if *value == nil {
		return nil
	}

	seenBefore := make(map[string]bool, len(*value))

	result := make([]apiserverv1.ExtraMapping, len(*value))
	for i, e := range result {
		p := fmt.Sprintf("%s[%d]", path, i)

		if e.Key != strings.ToLower(e.Key) {
			add(errs, fmt.Errorf("%s: .Key must be lowercase", p))
		}

		if seenBefore[e.Key] {
			add(errs, fmt.Errorf("%s: .Key must be unique", p))
		}
		seenBefore[e.Key] = true

		result[i] = apiserverv1.ExtraMapping{
			Key: e.Key,
			ValueExpression: e.ValueExpression,
		}
	}

	return result
}

func parseUserValidationRules(value *[]types.UserValidationRule, path string, errs *[]error) []apiserverv1.UserValidationRule {
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
			Message: deref(e.Message),
		}
	}

	return result
}

func parseAudienceMatchPolicy(value *string, path string, errs *[]error) apiserverv1.AudienceMatchPolicyType {
	_ = path
	_ = errs

	if value == nil {
		return ""
	}

	return apiserverv1.AudienceMatchPolicyType(*value)
}

func parseEgressSelectorType(value *string, path string, errs *[]error) apiserverv1.EgressSelectorType {
	_ = path
	_ = errs

	if value == nil {
		return ""
	}

	return apiserverv1.EgressSelectorType(*value)
}

func parseAnonymousAuthConfig(value *types.AnonymousAuthConfig, path string, errs *[]error) *apiserverv1.AnonymousAuthConfig {
	if value == nil {
		return nil
	}

	return &apiserverv1.AnonymousAuthConfig{
		Enabled: deref(value.Enabled),
		Conditions: parseAnonymousAuthCondition(value.Conditions, path + ".Conditions", errs),
	}
}

func parseAnonymousAuthCondition(value *[]types.AnonymousAuthCondition, path string, errs *[]error) []apiserverv1.AnonymousAuthCondition {
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

func add(errs *[]error, err error) {
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
		Jwt: toJwtAuthenticator(value.JWT),
	}
}

func toJwtAuthenticator(value []apiserverv1.JWTAuthenticator) []types.JwtAuthenticator {
	if value == nil {
		return nil
	}

	result := make([]types.JwtAuthenticator, len(value))
	for i, e := range value {
		result[i] = types.JwtAuthenticator{
			ClaimMappings: toClaimMappings(e.ClaimMappings),
			ClaimValidationRules: toClaimValidationRules(e.ClaimValidationRules),
			Issuer: toIssuer(e.Issuer),
			UserValidationRules: toUserValidationRules(e.UserValidationRules),
		}
	}

	return result
}

func toClaimMappings(value apiserverv1.ClaimMappings) types.ClaimMappings {
	return types.ClaimMappings{
		Extra: toExtraMapping(value.Extra),
		Groups: toPrefixedClaimOrExpression(value.Groups),
		UID: toClaimOrExpression(value.UID),
		Username: deref(toPrefixedClaimOrExpression(value.Username)),
	}
}

func toPrefixedClaimOrExpression(value apiserverv1.PrefixedClaimOrExpression) *types.PrefixedClaimOrExpression {
	zeros := apiserverv1.PrefixedClaimOrExpression{}
	if value == zeros {
		return nil
	}

	return &types.PrefixedClaimOrExpression{
		Claim: toString(value.Claim),
		Expression: toString(value.Expression),
		Prefix: clone(value.Prefix),
	}
}

func toClaimOrExpression(value apiserverv1.ClaimOrExpression) *types.ClaimOrExpression {
	zeros := apiserverv1.ClaimOrExpression{}
	if value == zeros {
		return nil
	}

	return &types.ClaimOrExpression{
		Claim: toString(value.Claim),
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
			Key: e.Key,
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
			Claim: toString(e.Claim),
			Expression: toString(e.Expression),
			Message: toString(e.Message),
			RequiredValue: toString(e.RequiredValue),
		}
	}

	return &result
}

func toIssuer(value apiserverv1.Issuer) types.Issuer {
	return types.Issuer{
		AudienceMatchPolicy: toString(string(value.AudienceMatchPolicy)),
		Audiences: slices.Clone(value.Audiences),
		CertificateAuthority: toString(value.CertificateAuthority),
		DiscoveryURL: value.DiscoveryURL,
		EgressSelectorType: toString(string(value.EgressSelectorType)),
		URL: value.URL,
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
			Message: clone(e.Message),
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
		Enabled: ptr.To(value.Enabled),
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
