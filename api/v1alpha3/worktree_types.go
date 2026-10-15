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

// WorktreeSpec defines the desired contents of a worktree.
type WorktreeSpec struct {
	// Files maps relative file paths to file contents.
	Files map[string][]byte `json:"files,omitempty"`
}

// WorktreeStatus defines the observed state of a worktree.
type WorktreeStatus struct {
	// Conditions represents the latest available observations of worktree state.
	Conditions    []metav1.Condition `json:"conditions,omitempty"`
	// URL is the remote URL where this worktree can be accessed.
	URL           *string            `json:"url,omitempty"`
	// ReferenceName is the branch or ref name currently associated with the worktree.
	ReferenceName *string            `json:"referenceName,omitempty"`
	// CommitHash is the current commit hash for the worktree.
	CommitHash    *string            `json:"commitHash,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// Worktree represents a managed source tree and its synchronization state.
type Worktree struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec defines the desired worktree contents.
	Spec   WorktreeSpec   `json:"spec,omitempty"`
	// Status represents the observed worktree state.
	Status WorktreeStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
// WorktreeList contains a list of Worktree resources.
type WorktreeList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	// Items contains the list of Worktree resources.
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
