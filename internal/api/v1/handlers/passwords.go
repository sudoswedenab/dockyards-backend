package handlers

import (
	"context"

	"github.com/sudoswedenab/dockyards-api/pkg/types"
	dockyardsv1 "github.com/sudoswedenab/dockyards-backend/api/v1alpha3"
	"github.com/sudoswedenab/dockyards-backend/internal/api/v1/middleware"
	"golang.org/x/crypto/bcrypt"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
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
