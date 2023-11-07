package handlers

import (
	"context"
	"net/http"

	"bitbucket.org/sudosweden/dockyards-backend/api/v1"
	"bitbucket.org/sudosweden/dockyards-backend/pkg/api/v1alpha1"
	"bitbucket.org/sudosweden/dockyards-backend/pkg/api/v1alpha1/index"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (h *handler) Login(c *gin.Context) {
	ctx := context.Background()

	var body v1.Login
	if c.BindJSON(&body) != nil {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	matchingFields := client.MatchingFields{
		index.EmailIndexKey: body.Email,
	}

	var userList v1alpha1.UserList
	err := h.controllerClient.List(ctx, &userList, matchingFields)
	if err != nil {
		h.logger.Error("error getting user from kubernetes", "err", err)

		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	if len(userList.Items) != 1 {
		h.logger.Error("expected exactly one user from kubernetes", "users", len(userList.Items))

		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	user := userList.Items[0]

	condition := meta.FindStatusCondition(user.Status.Conditions, v1alpha1.VerifiedCondition)
	if condition == nil || condition.Status != metav1.ConditionTrue {
		h.logger.Error("user is not verified")

		c.AbortWithStatus(http.StatusForbidden)
		return
	}

	//Compare sent in pass with saved user pass hash
	err = bcrypt.CompareHashAndPassword([]byte(user.Spec.Password), []byte(body.Password))
	if err != nil {
		h.logger.Error("error comparing password", "err", err)

		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	tokens, err := h.generateTokens(user)
	if err != nil {
		h.logger.Error("error generating tokens", "err", err)

		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	c.JSON(http.StatusOK, tokens)
}
