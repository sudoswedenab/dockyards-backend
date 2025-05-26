// Copyright 2025 Sudo Sweden AB
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

package index

import (
	"context"

	dockyardsv1 "github.com/sudoswedenab/dockyards-backend/api/v1alpha3"
	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func AddDefaultIndexes(ctx context.Context, mgr ctrl.Manager) error {
	err := ByEmail(ctx, mgr)
	if err != nil {
		return err
	}

	for _, object := range []client.Object{&dockyardsv1.User{}, &dockyardsv1.Cluster{}, &dockyardsv1.NodePool{}, &dockyardsv1.Node{}, &dockyardsv1.Organization{}} {
		err := mgr.GetFieldIndexer().IndexField(ctx, object, UIDField, ByUID)
		if err != nil {
			return err
		}
	}

	err = mgr.GetFieldIndexer().IndexField(ctx, &dockyardsv1.Organization{}, MemberReferencesField, ByMemberReferences)
	if err != nil {
		return err
	}

	for _, object := range []client.Object{&dockyardsv1.NodePool{}, &dockyardsv1.Node{}, &dockyardsv1.Cluster{}} {
		err := mgr.GetFieldIndexer().IndexField(ctx, object, OwnerReferencesField, ByOwnerReferences)
		if err != nil {
			return err
		}
	}

	err = mgr.GetFieldIndexer().IndexField(ctx, &corev1.Secret{}, SecretTypeField, BySecretType)
	if err != nil {
		return err
	}

	err = mgr.GetFieldIndexer().IndexField(ctx, &dockyardsv1.OrganizationVoucher{}, CodeField, ByCode)
	if err != nil {
		return err
	}

	return nil
}
