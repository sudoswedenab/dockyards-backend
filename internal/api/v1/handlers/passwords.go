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
	mathrand "math/rand/v2"
	"net/mail"
	"time"

	"github.com/sudoswedenab/dockyards-api/pkg/types"
	dockyardsv1 "github.com/sudoswedenab/dockyards-backend/api/v1alpha3"
	"github.com/sudoswedenab/dockyards-backend/api/v1alpha3/index"
	"github.com/sudoswedenab/dockyards-backend/internal/api/v1/middleware"
	"github.com/sudoswedenab/dockyards-backend/pkg/util/bubblebabble"
	"golang.org/x/crypto/bcrypt"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)


func (h *handler) UpdateUserPassword(ctx context.Context, userName string, request *types.PasswordOptions) error {
	subject, err := middleware.SubjectFrom(ctx)
	if err != nil {
		return err
	}

	if userName != subject {
		return apierrors.NewUnauthorized("subject must match user name")
	}

	var user dockyardsv1.User
	err = h.Get(ctx, client.ObjectKey{Name: userName}, &user)
	if err != nil {
		return err
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.Spec.Password), []byte(request.OldPassword))
	if err != nil {
		return apierrors.NewUnauthorized("error comparing hash to old password")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(request.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	patch := client.MergeFrom(user.DeepCopy())

	user.Spec.Password = string(hash)

	err = h.Patch(ctx, &user, patch)
	if err != nil {
		return err
	}

	return nil
}

func (h *handler) ResetPassword(ctx context.Context, request *types.ResetPasswordOptions) error {
	time.Sleep(time.Duration(mathrand.Float64() * float64(time.Second))) // Harden against brute force attacks.

	var verificationRequestList dockyardsv1.VerificationRequestList
	err := h.List(ctx, &verificationRequestList, client.MatchingFields{
		index.CodeField: request.ResetCode,
	})
	if err != nil {
		return err
	}
	if len(verificationRequestList.Items) == 0 {
		return errors.New("could not find verification request")
	}
	if len(verificationRequestList.Items) > 1 {
		// Reset requests should be reasonably unique, but could still collide.
		return errors.New("unexpected multiple verification requests with the same code")
	}
	verificationRequest := verificationRequestList.Items[0]

	// Deleting eagerly since the verification is one time use,
	// even in the failure cases below.
	err = h.Delete(ctx, &verificationRequest)
	if err != nil {
		return err
	}

	expiresAt := verificationRequest.GetExpiration()
	if expiresAt == nil {
		return errors.New("password reset request needs an expiration")
	}
	if time.Now().After(expiresAt.Time) {
		return apierrors.NewUnauthorized("verification request has expired")
	}

	var user dockyardsv1.User
	err = h.Get(ctx, client.ObjectKey{Name: verificationRequest.Spec.UserRef.Name}, &user)
	if err != nil {
		return err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(request.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	patch := client.MergeFrom(user.DeepCopy())
	user.Spec.Password = string(hash)
	err = h.Patch(ctx, &user, patch)
	if err != nil {
		return err
	}

	return nil
}

func (h *handler) CreateGlobalPasswordResetRequest(ctx context.Context, options *types.PasswordResetRequestOptions) error {
	time.Sleep(time.Duration(mathrand.Float64() * float64(time.Second))) // Harden against brute force attacks.

	email, err := mail.ParseAddress(options.Email)
	if err != nil {
		return err
	}

	var userList dockyardsv1.UserList
	err = h.List(ctx, &userList, client.MatchingFields{
		index.EmailField: email.Address,
	})
	if err != nil {
		return err
	}

	if len(userList.Items) == 0 {
		// Pretend everything was fine as to not give malicious
		// users information about emails of existing users.
		return nil
	}
	if len(userList.Items) > 1 {
		// User emails should be unique, so this should
		// not be possible.
		return errors.New("unexpected multiple users with the same email")
	}
	user := userList.Items[0]

	userProvider := user.Spec.ProviderID

	if userProvider != dockyardsv1.ProviderPrefixDockyards {
		return errors.New("cannot reset password of externally managed users")
	}

	resetCode, err := bubblebabble.RandomWithEntropyOfAtLeast(32)
	if err != nil {
		return err
	}

	passwordResetRequest := dockyardsv1.VerificationRequest{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "password-reset-",
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: dockyardsv1.GroupVersion.String(),
					Kind:       dockyardsv1.UserKind,
					Name:       user.Name,
					UID:        user.UID,
				},
			},
		},
		Spec: dockyardsv1.VerificationRequestSpec{
			Code:     resetCode,
			Subject:  "Password Reset",
			BodyText: fmt.Sprintf("Here is your password reset code: %s.\nIf you have not requested a password reset, you can ignore this email.", resetCode),
			// FIXME: We should have a reconciler look at this field
			//        and delete the resource, but can't since the Account
			//        verifications have their own reconciler and also
			//        dockyards-ses might still need the resource.
			Duration: &metav1.Duration{Duration: 10 * time.Minute},
			UserRef: corev1.TypedLocalObjectReference{
				APIGroup: &dockyardsv1.GroupVersion.Group,
				Kind:     dockyardsv1.UserKind,
				Name:     user.Name,
			},
		},
	}
	err = h.Create(ctx, &passwordResetRequest)
	if err != nil {
		return err
	}

	return nil
}
