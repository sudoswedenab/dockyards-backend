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

package apiutil

import (
	"context"
	"fmt"
	"net/http"
	"slices"

	"github.com/sudoswedenab/dockyards-backend/api/featurenames"
	dockyardsv1 "github.com/sudoswedenab/dockyards-backend/api/v1alpha3"
	authorizationv1 "k8s.io/api/authorization/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Expirable interface {
	GetExpiration() *metav1.Time
}

func GetOwnerOrganization(ctx context.Context, c client.Client, o client.Object) (dockyardsv1.Organization, error) {
	label := dockyardsv1.LabelOrganizationName
	owner := dockyardsv1.Organization{}
	owner.Kind = dockyardsv1.OrganizationKind
	err := getOwner(ctx, c, o, label, &owner)
	if err != nil {
		return owner, err
	}
	return owner, nil
}

func GetOwnerCluster(ctx context.Context, c client.Client, o client.Object) (dockyardsv1.Cluster, error) {
	label := dockyardsv1.LabelClusterName
	owner := dockyardsv1.Cluster{}
	owner.Kind = dockyardsv1.ClusterKind
	err := getOwner(ctx, c, o, label, &owner)
	if err != nil {
		return owner, err
	}
	return owner, nil
}

func GetOwnerNodePool(ctx context.Context, c client.Client, o client.Object) (dockyardsv1.NodePool, error) {
	label := dockyardsv1.LabelNodePoolName
	owner := dockyardsv1.NodePool{}
	owner.Kind = dockyardsv1.NodePoolKind
	err := getOwner(ctx, c, o, label, &owner)
	if err != nil {
		return owner, err
	}
	return owner, nil
}

func GetOwnerDeployment(ctx context.Context, c client.Client, o client.Object) (dockyardsv1.Deployment, error) {
	label := dockyardsv1.LabelDeploymentName
	owner := dockyardsv1.Deployment{}
	owner.Kind = dockyardsv1.DeploymentKind
	err := getOwner(ctx, c, o, label, &owner)
	if err != nil {
		return owner, err
	}
	return owner, nil
}

func GetOwnerHelmDeployment(ctx context.Context, c client.Client, o client.Object) (dockyardsv1.HelmDeployment, error) {
	label := dockyardsv1.LabelHelmDeploymentName
	owner := dockyardsv1.HelmDeployment{}
	owner.Kind = dockyardsv1.HelmDeploymentKind
	err := getOwner(ctx, c, o, label, &owner)
	if err != nil {
		return owner, err
	}
	return owner, nil
}

func GetOwnerKustomizeDeployment(ctx context.Context, c client.Client, o client.Object) (dockyardsv1.KustomizeDeployment, error) {
	label := dockyardsv1.LabelKustomizeDeploymentName
	owner := dockyardsv1.KustomizeDeployment{}
	owner.Kind = dockyardsv1.KustomizeDeploymentKind
	err := getOwner(ctx, c, o, label, &owner)
	if err != nil {
		return owner, err
	}
	return owner, nil
}

func GetOwnerContainerImageDeployment(ctx context.Context, c client.Client, o client.Object) (dockyardsv1.ContainerImageDeployment, error) {
	label := dockyardsv1.LabelContainerImageDeployment
	owner := dockyardsv1.ContainerImageDeployment{}
	owner.Kind = dockyardsv1.ContainerImageDeploymentKind
	err := getOwner(ctx, c, o, label, &owner)
	if err != nil {
		return owner, err
	}
	return owner, nil
}

func GetOwnerWorkload(ctx context.Context, c client.Client, o client.Object) (dockyardsv1.Workload, error) {
	label := dockyardsv1.LabelWorkloadName
	owner := dockyardsv1.Workload{}
	owner.Kind = dockyardsv1.WorkloadKind
	err := getOwner(ctx, c, o, label, &owner)
	if err != nil {
		return owner, err
	}
	return owner, nil
}

func IsFeatureEnabled(ctx context.Context, c client.Reader, featureName featurenames.FeatureName, inNamespace string) (bool, error) {
	var featureList dockyardsv1.FeatureList
	err := c.List(ctx, &featureList, client.InNamespace(inNamespace))
	if err != nil {
		return false, err
	}

	for _, feature := range featureList.Items {
		if feature.Name == string(featureName) {
			return true, nil
		}
	}

	return false, nil
}

func GetNamespaceOrganization(ctx context.Context, c client.Client, namespace string) (*dockyardsv1.Organization, error) {
	var organizationList dockyardsv1.OrganizationList
	err := c.List(ctx, &organizationList)
	if err != nil {
		return nil, err
	}

	for _, organization := range organizationList.Items {
		if organization.Spec.NamespaceRef == nil {
			continue
		}

		if organization.Spec.NamespaceRef.Name == namespace {
			return &organization, nil
		}
	}

	return nil, nil
}

func IsSubjectAllowed(ctx context.Context, c client.Client, subject string, resourceAttributes *authorizationv1.ResourceAttributes) (bool, error) {
	accessReview := authorizationv1.SubjectAccessReview{
		Spec: authorizationv1.SubjectAccessReviewSpec{
			Groups: []string{
				"dockyards:authenticated",
			},
			User:               subject,
			ResourceAttributes: resourceAttributes,
		},
	}

	err := c.Create(ctx, &accessReview)
	if err != nil {
		return false, err
	}

	return accessReview.Status.Allowed, nil
}

func IgnoreConflict(err error) error {
	if apierrors.IsConflict(err) {
		return nil
	}

	return err
}

func IgnoreForbidden(err error) error {
	if apierrors.IsForbidden(err) {
		return nil
	}

	return err
}

func HasExpired(expirable Expirable) bool {
	expiration := expirable.GetExpiration()

	if expiration == nil {
		return false
	}

	if metav1.Now().After(expiration.Time) {
		return true
	}

	return false
}

func IgnoreInternalError(err error) error {
	if apierrors.IsInternalError(err) {
		return nil
	}

	return err
}

func GetDefaultRelease(ctx context.Context, c client.Reader, releaseType dockyardsv1.ReleaseType) (*dockyardsv1.Release, error) {
	var releaseList dockyardsv1.ReleaseList
	err := c.List(ctx, &releaseList)
	if err != nil {
		return nil, err
	}

	for _, release := range releaseList.Items {
		if release.Spec.Type != releaseType {
			continue
		}

		_, isDefault := release.Annotations[dockyardsv1.AnnotationDefaultRelease]
		if isDefault {
			return &release, nil
		}
	}

	return nil, nil
}

func GetDefaultClusterTemplate(ctx context.Context, c client.Client) (*dockyardsv1.ClusterTemplate, error) {
	var clusterTemplateList dockyardsv1.ClusterTemplateList
	err := c.List(ctx, &clusterTemplateList)
	if err != nil {
		return nil, err
	}

	for _, clusterTemplate := range clusterTemplateList.Items {
		_, isDefault := clusterTemplate.Annotations[dockyardsv1.AnnotationDefaultTemplate]
		if isDefault {
			return &clusterTemplate, nil
		}
	}

	return nil, nil
}

func IgnoreClientError(err error) error {
	if apierrors.IsInvalid(err) {
		return nil
	}

	if apierrors.IsConflict(err) {
		return nil
	}

	if apierrors.IsAlreadyExists(err) {
		return nil
	}

	return err
}

func IgnoreIsInvalid(err error) error {
	if apierrors.IsInvalid(err) {
		return nil
	}

	return err
}

func FindWorkloadReference(references []dockyardsv1.WorkloadReference, reference dockyardsv1.WorkloadReference) *dockyardsv1.WorkloadReference {
	for i, existing := range references {
		if existing.TypedObjectReference.String() != reference.TypedObjectReference.String() {
			continue
		}

		return &references[i]
	}

	return nil
}

func SetWorkloadReference(references *[]dockyardsv1.WorkloadReference, newReference dockyardsv1.WorkloadReference) bool {
	existingReference := FindWorkloadReference(*references, newReference)
	if existingReference == nil {
		*references = append(*references, newReference)

		return true
	}

	if slices.Compare(existingReference.URLs, newReference.URLs) != 0 {
		existingReference.URLs = newReference.URLs

		return true
	}

	return false
}

type Conditionable interface {
	GetConditions() []metav1.Condition
}

func IsReady(conditionable Conditionable) bool {
	conditions := conditionable.GetConditions()

	return meta.IsStatusConditionTrue(conditions, dockyardsv1.ReadyCondition)
}

func GetOrganizationByNamespaceRef(ctx context.Context, c client.Reader, namespaceName string) (*dockyardsv1.Organization, error) {
	var organizationList dockyardsv1.OrganizationList
	err := c.List(ctx, &organizationList)
	if err != nil {
		return nil, err
	}

	for _, organization := range organizationList.Items {
		if organization.Spec.NamespaceRef == nil {
			continue
		}

		if organization.Spec.NamespaceRef.Name != namespaceName {
			continue
		}

		return &organization, nil
	}

	return nil, &apierrors.StatusError{
		ErrStatus: metav1.Status{
			Status: metav1.StatusFailure,
			Code:   http.StatusNotFound,
			Reason: metav1.StatusReasonNotFound,
			Details: &metav1.StatusDetails{
				Group: dockyardsv1.GroupVersion.Group,
				Kind:  dockyardsv1.OrganizationKind,
			},
			Message: fmt.Sprintf("could not find organization referencing namespace %s", namespaceName),
		},
	}
}

func getOwner[T client.Object](ctx context.Context, c client.Client, o client.Object, ownerLabel string, owner T) error {
	labels := o.GetLabels()
	ownerName := labels[ownerLabel]
	if ownerName == "" {
		return getOwnerSlow(ctx, c, o, owner)
	}

	err := c.Get(ctx, types.NamespacedName{Namespace: o.GetNamespace(), Name: ownerName}, owner)
	if err != nil {
		return getOwnerSlow(ctx, c, o, owner)
	}

	return nil
}

func getOwnerSlow[T client.Object](ctx context.Context, c client.Client, o client.Object, owner T) error {
	ownerRef, err := FindOwnerReference(o, owner.GetObjectKind().GroupVersionKind().Kind)
	if err != nil {
		return err
	}

	err = c.Get(ctx, client.ObjectKey{Name: ownerRef.Name}, owner)
	if err != nil {
		return err
	}

	return nil
}

func FindOwnerReference(o client.Object, kind string) (metav1.OwnerReference, error) {
	for _, ownerReference := range o.GetOwnerReferences() {
		if ownerReference.Kind != kind {
			continue
		}

		groupVersion, err := schema.ParseGroupVersion(ownerReference.APIVersion)
		if err != nil {
			return metav1.OwnerReference{}, err
		}

		if groupVersion.Group == dockyardsv1.GroupVersion.Group {
			return ownerReference, nil
		}
	}

	return metav1.OwnerReference{}, &apierrors.StatusError{ErrStatus: metav1.Status{
		Status: metav1.StatusFailure,
		Code:   http.StatusNotFound,
		Reason: metav1.StatusReasonNotFound,
		Details: &metav1.StatusDetails{
			Group: dockyardsv1.GroupVersion.Group,
			Kind:  kind,
		},
		Message: fmt.Sprintf("could not find owner for %s", o.GetName()),
	}}
}
