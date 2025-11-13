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
	"fmt"
	"testing"
	"time"

	dockyardsv1 "github.com/sudoswedenab/dockyards-backend/api/v1alpha3"
	"github.com/sudoswedenab/dockyards-backend/pkg/authorization"
	authorizationv1 "k8s.io/api/authorization/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/util/retry"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/config"
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
		return nil, fmt.Errorf("error creating super user: %w", err)
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
		return nil, fmt.Errorf("error creating user: %w", err)
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
		return nil, fmt.Errorf("error creating reader: %w", err)
	}

	for _, u := range []dockyardsv1.User{
		superUser,
		user,
		reader,
	} {
		err := authorization.ReconcileUserAuthorization(ctx, c, &u)
		if err != nil {
			return nil, fmt.Errorf("error reconciling user authorization: %w", err)
		}
	}

	namespace := corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "test-",
		},
	}

	err = c.Create(ctx, &namespace)
	if err != nil {
		return nil, fmt.Errorf("error creating namespace: %w", err)
	}

	organization := dockyardsv1.Organization{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "test-",
		},
		Spec: dockyardsv1.OrganizationSpec{
			NamespaceRef: &corev1.LocalObjectReference{
				Name: namespace.Name,
			},
			ProviderID: ptr.To(dockyardsv1.ProviderPrefixDockyards),
		},
	}

	err = c.Create(ctx, &organization)
	if err != nil {
		return nil, fmt.Errorf("error creating organization: %w", err)
	}

	for userName, role := range map[string]dockyardsv1.Role{
		superUser.Name: dockyardsv1.RoleSuperUser,
		user.Name:      dockyardsv1.RoleUser,
		reader.Name:    dockyardsv1.RoleReader,
	} {
		member := dockyardsv1.Member{
			ObjectMeta: metav1.ObjectMeta{
				Labels: map[string]string{
					dockyardsv1.LabelRoleName:         string(role),
					dockyardsv1.LabelUserName:         userName,
					dockyardsv1.LabelOrganizationName: organization.Name,
				},
				Name:      userName,
				Namespace: namespace.Name,
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion: dockyardsv1.GroupVersion.String(),
						Kind:       dockyardsv1.OrganizationKind,
						Name:       organization.Name,
						UID:        organization.UID,
					},
				},
			},
			Spec: dockyardsv1.MemberSpec{
				Role: role,
				UserRef: corev1.TypedLocalObjectReference{
					APIGroup: &dockyardsv1.GroupVersion.Group,
					Kind:     dockyardsv1.UserKind,
					Name:     userName,
				},
			},
		}

		err = c.Create(ctx, &member)
		if err != nil {
			return nil, fmt.Errorf("error creating member: %w", err)
		}

		err = authorization.ReconcileMemberAuthorization(ctx, c, &member)
		if err != nil {
			return nil, fmt.Errorf("error reconciling member: %w", err)
		}
	}

	err = authorization.ReconcileOrganizationAuthorization(ctx, c, &organization)
	if err != nil {
		return nil, fmt.Errorf("error reconciling organization authorization: %w", err)
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

func (e *TestEnvironment) GetOrganizationUser(ctx context.Context, organization *dockyardsv1.Organization, role dockyardsv1.Role) (*dockyardsv1.User, error) {
	var memberList dockyardsv1.MemberList
	err := e.c.List(ctx, &memberList, client.InNamespace(organization.Spec.NamespaceRef.Name))
	if err != nil {
		return nil, err
	}

	for _, member := range memberList.Items {
		if member.Spec.Role != role {
			continue
		}

		var user dockyardsv1.User
		err := e.c.Get(ctx, client.ObjectKey{Name: member.Spec.UserRef.Name}, &user)
		if err != nil {
			return nil, err
		}

		return &user, nil
	}

	return nil, errors.New("no such user")
}

func (e *TestEnvironment) MustGetOrganizationUser(t *testing.T, organization *dockyardsv1.Organization, role dockyardsv1.Role) *dockyardsv1.User {
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
	_ = apiextensionsv1.AddToScheme(scheme)

	c, err := client.New(cfg, client.Options{Scheme: scheme})
	if err != nil {
		return nil, err
	}

	opts := manager.Options{
		Scheme: scheme,
		Metrics: server.Options{
			BindAddress: "0",
		},
		Controller: config.Controller{
			SkipNameValidation: ptr.To(true),
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

	err = authorization.ReconcileClusterAuthorization(ctx, c)
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
