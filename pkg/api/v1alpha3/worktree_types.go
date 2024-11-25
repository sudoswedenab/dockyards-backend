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

package v1alpha3

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	WorktreeKind = "Worktree"
)

type WorktreeSpec struct {
	Files map[string][]byte `json:"files,omitempty"`
}

type WorktreeStatus struct {
	Conditions []metav1.Condition `json:"conditions,omitempty"`
	URL        *string            `json:"url,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
type Worktree struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   WorktreeSpec   `json:"spec,omitempty"`
	Status WorktreeStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
type WorktreeList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []Worktree `json:"items,omitempty"`
}

func (w *Worktree) GetConditions() []metav1.Condition {
	return w.Status.Conditions
}

func (w *Worktree) SetConditions(conditions []metav1.Condition) {
	w.Status.Conditions = conditions
}

func init() {
	SchemeBuilder.Register(&Worktree{}, &WorktreeList{})
}
