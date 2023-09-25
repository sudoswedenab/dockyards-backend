package handlers

import (
	"context"
	"net/http"

	"bitbucket.org/sudosweden/dockyards-backend/api/v1"
	"bitbucket.org/sudosweden/dockyards-backend/pkg/api/v1alpha1"
	"github.com/gin-gonic/gin"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (h *handler) GetWhoami(c *gin.Context) {
	sub, err := h.getSubjectFromContext(c)
	if err != nil {
		h.logger.Error("error getting subject from context", "err", err)

		c.AbortWithStatus(http.StatusInternalServerError)
	}

	ctx := context.Background()

	matchingFields := client.MatchingFields{
		"metadata.uid": sub,
	}

	var userList v1alpha1.UserList
	err = h.controllerClient.List(ctx, &userList, matchingFields)
	if err != nil {
		h.logger.Error("error getting user from kubernetes", "err", err)

		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	if len(userList.Items) != 1 {
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	user := userList.Items[0]

	v1User := v1.User{
		ID:    string(user.UID),
		Name:  user.Name,
		Email: user.Spec.Email,
	}

	c.JSON(http.StatusOK, v1User)
}
