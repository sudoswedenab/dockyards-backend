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

package testingutil

import (
	"context"
	"errors"
	"testing"
	"time"

	dockyardsv1 "github.com/sudoswedenab/dockyards-backend/api/v1alpha3"
	"github.com/sudoswedenab/dockyards-backend/pkg/authorization"
	authorizationv1 "k8s.io/api/authorization/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/util/retry"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/metrics/server"
)

type TestEnvironment struct {
	mgr           manager.Manager
	environment   *envtest.Environment
	c             client.Client
	namespaceName string
}

func (e *TestEnvironment) GetManager() manager.Manager {
	return e.mgr
}

func (e *TestEnvironment) GetEnvironment() *envtest.Environment {
	return e.environment
}

func (e *TestEnvironment) GetClient() client.Client {
	return e.c
}

func (e *TestEnvironment) GetDockyardsNamespace() string {
	return e.namespaceName
}

func (e *TestEnvironment) CreateOrganization(ctx context.Context) (*dockyardsv1.Organization, error) {
	c := e.GetClient()

	superUser := dockyardsv1.User{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "superuser-",
		},
		Spec: dockyardsv1.UserSpec{
			Email: "superuser@dockyards.dev",
		},
	}

	err := c.Create(ctx, &superUser)
	if err != nil {
		return nil, err
	}

	user := dockyardsv1.User{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "user-",
		},
		Spec: dockyardsv1.UserSpec{
			Email: "user@dockyards.dev",
		},
	}

	err = c.Create(ctx, &user)
	if err != nil {
		return nil, err
	}

	reader := dockyardsv1.User{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "reader-",
		},
		Spec: dockyardsv1.UserSpec{
			Email: "reader@dockyards.dev",
		},
	}

	err = c.Create(ctx, &reader)
	if err != nil {
		return nil, err
	}

	namespace := corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "test-",
		},
	}

	err = c.Create(ctx, &namespace)
	if err != nil {
		return nil, err
	}

	organization := dockyardsv1.Organization{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "test-",
		},
		Spec: dockyardsv1.OrganizationSpec{
			DisplayName: "test",
			MemberRefs: []dockyardsv1.OrganizationMemberReference{
				{
					TypedLocalObjectReference: corev1.TypedLocalObjectReference{
						APIGroup: &dockyardsv1.GroupVersion.Group,
						Kind:     dockyardsv1.UserKind,
						Name:     superUser.Name,
					},
					Role: dockyardsv1.OrganizationMemberRoleSuperUser,
					UID:  superUser.UID,
				},
				{
					TypedLocalObjectReference: corev1.TypedLocalObjectReference{
						APIGroup: &dockyardsv1.GroupVersion.Group,
						Kind:     dockyardsv1.UserKind,
						Name:     user.Name,
					},
					Role: dockyardsv1.OrganizationMemberRoleUser,
					UID:  user.UID,
				},
				{
					TypedLocalObjectReference: corev1.TypedLocalObjectReference{
						APIGroup: &dockyardsv1.GroupVersion.Group,
						Kind:     dockyardsv1.UserKind,
						Name:     reader.Name,
					},
					Role: dockyardsv1.OrganizationMemberRoleReader,
					UID:  reader.UID,
				},
			},
			NamespaceRef: &corev1.LocalObjectReference{
				Name: namespace.Name,
			},
			ProviderID: ptr.To(dockyardsv1.ProviderPrefixDockyards),
		},
	}

	err = c.Create(ctx, &organization)
	if err != nil {
		return nil, err
	}

	err = authorization.ReconcileOrganizationAuthorization(ctx, c, &organization)
	if err != nil {
		return nil, err
	}

	return &organization, nil
}

func (e *TestEnvironment) MustCreateOrganization(t *testing.T) *dockyardsv1.Organization {
	ctx := t.Context()

	organization, err := e.CreateOrganization(ctx)
	if err != nil {
		t.Fatal(err)
	}

	return organization
}

func (e *TestEnvironment) GetOrganizationUser(ctx context.Context, organization *dockyardsv1.Organization, role dockyardsv1.OrganizationMemberRole) (*dockyardsv1.User, error) {
	for _, memberRef := range organization.Spec.MemberRefs {
		if memberRef.Role != role {
			continue
		}

		var user dockyardsv1.User
		err := e.c.Get(ctx, client.ObjectKey{Name: memberRef.Name}, &user)
		if err != nil {
			return nil, err
		}

		return &user, nil
	}

	return nil, errors.New("no such user")
}

func (e *TestEnvironment) MustGetOrganizationUser(t *testing.T, organization *dockyardsv1.Organization, role dockyardsv1.OrganizationMemberRole) *dockyardsv1.User {
	ctx := t.Context()

	user, err := e.GetOrganizationUser(ctx, organization, role)
	if err != nil {
		t.Fatal(err)
	}

	return user
}

func NewTestEnvironment(ctx context.Context, crdDirectoryPaths []string) (*TestEnvironment, error) {
	environment := envtest.Environment{
		CRDDirectoryPaths: crdDirectoryPaths,
	}

	cfg, err := environment.Start()
	if err != nil {
		return nil, err
	}

	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)
	_ = dockyardsv1.AddToScheme(scheme)
	_ = authorizationv1.AddToScheme(scheme)
	_ = rbacv1.AddToScheme(scheme)

	c, err := client.New(cfg, client.Options{Scheme: scheme})
	if err != nil {
		return nil, err
	}

	opts := manager.Options{
		Scheme: scheme,
		Metrics: server.Options{
			BindAddress: "0",
		},
	}

	mgr, err := ctrl.NewManager(cfg, opts)
	if err != nil {
		return nil, err
	}

	namespace := corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "dockyards-",
		},
	}

	err = c.Create(ctx, &namespace)
	if err != nil {
		return nil, err
	}

	err = authorization.ReconcileGlobalAuthorization(ctx, c)
	if err != nil {
		return nil, err
	}

	t := TestEnvironment{
		mgr:           mgr,
		environment:   &environment,
		c:             c,
		namespaceName: namespace.Name,
	}

	return &t, nil
}

func RetryUntilFound(ctx context.Context, reader client.Reader, obj client.Object) error {
	backoff := wait.Backoff{
		Steps:    5,
		Duration: 100 * time.Millisecond,
		Factor:   2.0,
	}

	err := retry.OnError(backoff, apierrors.IsNotFound, func() error {
		err := reader.Get(ctx, client.ObjectKeyFromObject(obj), obj)

		return err
	})
	if err != nil {
		return err
	}

	return nil
}
