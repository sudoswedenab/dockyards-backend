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

package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	"github.com/spf13/pflag"
	"github.com/sudoswedenab/dockyards-backend/api/apiutil"
	dockyardsv1 "github.com/sudoswedenab/dockyards-backend/api/v1alpha2"
	"golang.org/x/crypto/bcrypt"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

func main() {
	var voucherCode string
	var email string
	var password string
	pflag.StringVar(&voucherCode, "voucher-code", "", "voucher code")
	pflag.StringVar(&email, "email", "test@sudosweden.com", "email")
	pflag.StringVar(&password, "password", "test", "password")
	pflag.Parse()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	cfg, err := config.GetConfig()
	if err != nil {
		panic(err)
	}

	c, err := client.New(cfg, client.Options{})
	if err != nil {
		panic(err)
	}

	scheme := c.Scheme()

	_ = dockyardsv1.AddToScheme(scheme)

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		panic(err)
	}

	user := dockyardsv1.User{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "user-",
		},
		Spec: dockyardsv1.UserSpec{
			Email:    email,
			Password: string(hash),
		},
	}

	err = c.Create(ctx, &user)
	if apiutil.IgnoreForbidden(err) != nil {
		panic(err)
	}

	if apierrors.IsForbidden(err) {
		fmt.Println("forbidden:", err)

		os.Exit(12)
	}

	organization := dockyardsv1.Organization{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "test-",
		},
		Spec: dockyardsv1.OrganizationSpec{
			MemberRefs: []dockyardsv1.MemberReference{
				{
					Group: dockyardsv1.GroupVersion.Group,
					Kind:  dockyardsv1.UserKind,
					Role:  dockyardsv1.MemberRoleSuperUser,
					Name:  user.Name,
					UID:   user.UID,
				},
			},
			SkipAutoAssign: true,
		},
	}

	if voucherCode != "" {
		organization.Annotations = map[string]string{
			dockyardsv1.AnnotationVoucherCode: voucherCode,
		}

		organization.Spec.SkipAutoAssign = false
	}

	err = c.Create(ctx, &organization)
	if err != nil {
		panic(err)
	}
}
