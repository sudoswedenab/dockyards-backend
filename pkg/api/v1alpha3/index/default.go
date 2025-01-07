package index

import (
	"context"

	dockyardsv1 "bitbucket.org/sudosweden/dockyards-backend/pkg/api/v1alpha3"
	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func AddDefaultIndexes(ctx context.Context, mgr ctrl.Manager) error {
	err := mgr.GetFieldIndexer().IndexField(ctx, &dockyardsv1.User{}, EmailField, ByEmail)
	if err != nil {
		return err
	}

	for _, object := range []client.Object{&dockyardsv1.User{}, &dockyardsv1.Cluster{}, &dockyardsv1.NodePool{}, &dockyardsv1.Node{}, &dockyardsv1.Organization{}} {
		err = mgr.GetFieldIndexer().IndexField(ctx, object, UIDField, ByUID)
		if err != nil {
			return err
		}
	}

	err = mgr.GetFieldIndexer().IndexField(ctx, &dockyardsv1.Organization{}, MemberReferencesField, ByMemberReferences)
	if err != nil {
		return err
	}

	for _, object := range []client.Object{&dockyardsv1.NodePool{}, &dockyardsv1.Node{}, &dockyardsv1.Cluster{}} {
		err = mgr.GetFieldIndexer().IndexField(ctx, object, OwnerReferencesField, ByOwnerReferences)
		if err != nil {
			return err
		}
	}

	err = mgr.GetFieldIndexer().IndexField(ctx, &corev1.Secret{}, SecretTypeField, BySecretType)
	if err != nil {
		return err
	}

	return nil
}
