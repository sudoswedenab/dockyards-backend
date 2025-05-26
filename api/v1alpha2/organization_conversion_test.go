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

package v1alpha2_test

import (
	"testing"
	"time"

	"github.com/sudoswedenab/dockyards-backend/api/v1alpha2"
	"github.com/sudoswedenab/dockyards-backend/api/v1alpha3"
	"github.com/google/go-cmp/cmp"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
)

func TestOrganizationConvertTo(t *testing.T) {
	tt := []struct {
		name     string
		src      v1alpha2.Organization
		expected v1alpha3.Organization
	}{
		{
			name: "test spec",
			src: v1alpha2.Organization{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-empty-spec",
				},
				Spec: v1alpha2.OrganizationSpec{
					DisplayName:    "test",
					SkipAutoAssign: true,
				},
			},
			expected: v1alpha3.Organization{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-empty-spec",
				},
				Spec: v1alpha3.OrganizationSpec{
					DisplayName:    "test",
					SkipAutoAssign: true,
				},
			},
		},
		{
			name: "test namespace reference",
			src: v1alpha2.Organization{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-namespace-reference",
				},
				Status: v1alpha2.OrganizationStatus{
					NamespaceRef: "testing",
				},
			},
			expected: v1alpha3.Organization{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-namespace-reference",
				},
				Status: v1alpha3.OrganizationStatus{
					NamespaceRef: &corev1.LocalObjectReference{
						Name: "testing",
					},
				},
			},
		},
		{
			name: "test member references",
			src: v1alpha2.Organization{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-member-references",
				},
				Spec: v1alpha2.OrganizationSpec{
					MemberRefs: []v1alpha2.MemberReference{
						{
							Group: v1alpha2.GroupVersion.Group,
							Kind:  v1alpha2.UserKind,
							Name:  "test",
							Role:  v1alpha2.MemberRoleSuperUser,
							UID:   "f8133851-706b-4bfb-b947-7d7af92bb7fd",
						},
					},
				},
			},
			expected: v1alpha3.Organization{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-member-references",
				},
				Spec: v1alpha3.OrganizationSpec{
					MemberRefs: []v1alpha3.OrganizationMemberReference{
						{
							TypedLocalObjectReference: corev1.TypedLocalObjectReference{
								APIGroup: &v1alpha3.GroupVersion.Group,
								Kind:     v1alpha3.UserKind,
								Name:     "test",
							},
							Role: v1alpha3.OrganizationMemberRoleSuperUser,
							UID:  "f8133851-706b-4bfb-b947-7d7af92bb7fd",
						},
					},
				},
			},
		},
		{
			name: "test cloud project reference",
			src: v1alpha2.Organization{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-cloud-project-reference",
				},
				Spec: v1alpha2.OrganizationSpec{
					Cloud: v1alpha2.Cloud{
						ProjectRef: &v1alpha2.NamespacedObjectReference{
							APIVersion: "cloud.dockyards.io/v1alpha1",
							Kind:       "CloudProject",
							Name:       "test",
							Namespace:  "testing",
						},
					},
				},
			},
			expected: v1alpha3.Organization{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-cloud-project-reference",
				},
				Spec: v1alpha3.OrganizationSpec{
					ProjectRef: &corev1.TypedObjectReference{
						APIGroup:  ptr.To("cloud.dockyards.io"),
						Kind:      "CloudProject",
						Name:      "test",
						Namespace: ptr.To("testing"),
					},
				},
			},
		},
		{
			name: "test cloud secret reference",
			src: v1alpha2.Organization{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-cloud-secret-reference",
				},
				Spec: v1alpha2.OrganizationSpec{
					Cloud: v1alpha2.Cloud{
						SecretRef: &v1alpha2.NamespacedSecretReference{
							Name:      "test",
							Namespace: "testing",
						},
					},
				},
			},
			expected: v1alpha3.Organization{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-cloud-secret-reference",
				},
				Spec: v1alpha3.OrganizationSpec{
					CredentialRef: &corev1.TypedObjectReference{
						Kind:      "Secret",
						Name:      "test",
						Namespace: ptr.To("testing"),
					},
				},
			},
		},
		{
			name: "test duration",
			src: v1alpha2.Organization{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-duration",
				},
				Spec: v1alpha2.OrganizationSpec{
					Duration: &metav1.Duration{
						Duration: time.Hour + time.Minute + time.Second,
					},
				},
			},
			expected: v1alpha3.Organization{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-duration",
				},
				Spec: v1alpha3.OrganizationSpec{
					Duration: &metav1.Duration{
						Duration: time.Hour + time.Minute + time.Second,
					},
				},
			},
		},
		{
			name: "test conditions",
			src: v1alpha2.Organization{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-conditions",
				},
				Status: v1alpha2.OrganizationStatus{
					Conditions: []metav1.Condition{
						{
							Type: v1alpha2.ReadyCondition,
						},
					},
				},
			},
			expected: v1alpha3.Organization{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-conditions",
				},
				Status: v1alpha3.OrganizationStatus{
					Conditions: []metav1.Condition{
						{
							Type: v1alpha3.ReadyCondition,
						},
					},
				},
			},
		},
		{
			name: "test expiration timestamp",
			src: v1alpha2.Organization{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-expiration-timestamp",
				},
				Status: v1alpha2.OrganizationStatus{
					ExpirationTimestamp: &metav1.Time{
						Time: time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC),
					},
				},
			},
			expected: v1alpha3.Organization{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-expiration-timestamp",
				},
				Status: v1alpha3.OrganizationStatus{
					ExpirationTimestamp: &metav1.Time{
						Time: time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC),
					},
				},
			},
		},
		{
			name: "test resource quotas",
			src: v1alpha2.Organization{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-resource-quotas",
				},
				Status: v1alpha2.OrganizationStatus{
					ResourceQuotas: corev1.ResourceList{
						corev1.ResourceMemory: resource.MustParse("123Mi"),
					},
				},
			},
			expected: v1alpha3.Organization{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-resource-quotas",
				},
				Status: v1alpha3.OrganizationStatus{
					ResourceQuotas: corev1.ResourceList{
						corev1.ResourceMemory: resource.MustParse("123Mi"),
					},
				},
			},
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			var actual v1alpha3.Organization
			err := tc.src.ConvertTo(&actual)
			if err != nil {
				t.Fatalf("error converting to: %s", err)
			}

			if !cmp.Equal(actual, tc.expected) {
				t.Errorf("diff: %s", cmp.Diff(tc.expected, actual))
			}
		})
	}
}

func TestOrganizationConvertFrom(t *testing.T) {
	tt := []struct {
		name     string
		hub      v1alpha3.Organization
		expected v1alpha2.Organization
	}{
		{
			name: "test spec",
			hub: v1alpha3.Organization{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-empty-spec",
				},
				Spec: v1alpha3.OrganizationSpec{
					DisplayName:    "test",
					SkipAutoAssign: true,
				},
			},
			expected: v1alpha2.Organization{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-empty-spec",
				},
				Spec: v1alpha2.OrganizationSpec{
					DisplayName:    "test",
					SkipAutoAssign: true,
				},
			},
		},
		{
			name: "test namespace reference",
			hub: v1alpha3.Organization{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-namespace-reference",
				},
				Status: v1alpha3.OrganizationStatus{
					NamespaceRef: &corev1.LocalObjectReference{
						Name: "testing",
					},
				},
			},
			expected: v1alpha2.Organization{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-namespace-reference",
				},
				Status: v1alpha2.OrganizationStatus{
					NamespaceRef: "testing",
				},
			},
		},
		{
			name: "test member references",
			hub: v1alpha3.Organization{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-member-references",
				},
				Spec: v1alpha3.OrganizationSpec{
					MemberRefs: []v1alpha3.OrganizationMemberReference{
						{
							TypedLocalObjectReference: corev1.TypedLocalObjectReference{
								APIGroup: &v1alpha3.GroupVersion.Group,
								Kind:     v1alpha3.UserKind,
								Name:     "test",
							},
							Role: v1alpha3.OrganizationMemberRoleSuperUser,
							UID:  "f8133851-706b-4bfb-b947-7d7af92bb7fd",
						},
					},
				},
			},
			expected: v1alpha2.Organization{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-member-references",
				},
				Spec: v1alpha2.OrganizationSpec{
					MemberRefs: []v1alpha2.MemberReference{
						{
							Group: v1alpha2.GroupVersion.Group,
							Kind:  v1alpha2.UserKind,
							Name:  "test",
							Role:  v1alpha2.MemberRoleSuperUser,
							UID:   "f8133851-706b-4bfb-b947-7d7af92bb7fd",
						},
					},
				},
			},
		},
		{
			name: "test cloud project reference",
			hub: v1alpha3.Organization{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-cloud-project-reference",
				},
				Spec: v1alpha3.OrganizationSpec{
					ProjectRef: &corev1.TypedObjectReference{
						APIGroup:  ptr.To("cloud.dockyards.io"),
						Kind:      "CloudProject",
						Name:      "test",
						Namespace: ptr.To("testing"),
					},
				},
			},
			expected: v1alpha2.Organization{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-cloud-project-reference",
				},
				Spec: v1alpha2.OrganizationSpec{
					Cloud: v1alpha2.Cloud{
						ProjectRef: &v1alpha2.NamespacedObjectReference{
							APIVersion: "cloud.dockyards.io/v1alpha1",
							Kind:       "CloudProject",
							Name:       "test",
							Namespace:  "testing",
						},
					},
				},
			},
		},
		{
			name: "test cloud secret reference",
			hub: v1alpha3.Organization{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-cloud-secret-reference",
				},
				Spec: v1alpha3.OrganizationSpec{
					CredentialRef: &corev1.TypedObjectReference{
						Kind:      "Secret",
						Name:      "test",
						Namespace: ptr.To("testing"),
					},
				},
			},
			expected: v1alpha2.Organization{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-cloud-secret-reference",
				},
				Spec: v1alpha2.OrganizationSpec{
					Cloud: v1alpha2.Cloud{
						SecretRef: &v1alpha2.NamespacedSecretReference{
							Name:      "test",
							Namespace: "testing",
						},
					},
				},
			},
		},
		{
			name: "test duration",
			hub: v1alpha3.Organization{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-duration",
				},
				Spec: v1alpha3.OrganizationSpec{
					Duration: &metav1.Duration{
						Duration: time.Hour + time.Minute + time.Second,
					},
				},
			},
			expected: v1alpha2.Organization{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-duration",
				},
				Spec: v1alpha2.OrganizationSpec{
					Duration: &metav1.Duration{
						Duration: time.Hour + time.Minute + time.Second,
					},
				},
			},
		},
		{
			name: "test conditions",
			hub: v1alpha3.Organization{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-conditions",
				},
				Status: v1alpha3.OrganizationStatus{
					Conditions: []metav1.Condition{
						{
							Type: v1alpha3.ReadyCondition,
						},
					},
				},
			},
			expected: v1alpha2.Organization{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-conditions",
				},
				Status: v1alpha2.OrganizationStatus{
					Conditions: []metav1.Condition{
						{
							Type: v1alpha2.ReadyCondition,
						},
					},
				},
			},
		},
		{
			name: "test expiration timestamp",
			hub: v1alpha3.Organization{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-expiration-timestamp",
				},
				Status: v1alpha3.OrganizationStatus{
					ExpirationTimestamp: &metav1.Time{
						Time: time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC),
					},
				},
			},
			expected: v1alpha2.Organization{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-expiration-timestamp",
				},
				Status: v1alpha2.OrganizationStatus{
					ExpirationTimestamp: &metav1.Time{
						Time: time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC),
					},
				},
			},
		},
		{
			name: "test resource quotas",
			hub: v1alpha3.Organization{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-resource-quotas",
				},
				Status: v1alpha3.OrganizationStatus{
					ResourceQuotas: corev1.ResourceList{
						corev1.ResourceMemory: resource.MustParse("123Mi"),
					},
				},
			},
			expected: v1alpha2.Organization{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-resource-quotas",
				},
				Status: v1alpha2.OrganizationStatus{
					ResourceQuotas: corev1.ResourceList{
						corev1.ResourceMemory: resource.MustParse("123Mi"),
					},
				},
			},
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			var actual v1alpha2.Organization
			err := actual.ConvertFrom(&tc.hub)
			if err != nil {
				t.Fatalf("error converting to: %s", err)
			}

			if !cmp.Equal(actual, tc.expected) {
				t.Errorf("diff: %s", cmp.Diff(tc.expected, actual))
			}
		})
	}
}
