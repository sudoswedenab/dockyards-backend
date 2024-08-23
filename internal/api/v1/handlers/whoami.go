package handlers

import (
	"encoding/json"
	"net/http"

	"bitbucket.org/sudosweden/dockyards-backend/internal/api/v1"
	"bitbucket.org/sudosweden/dockyards-backend/internal/api/v1/middleware"
	dockyardsv1 "bitbucket.org/sudosweden/dockyards-backend/pkg/api/v1alpha2"
	"bitbucket.org/sudosweden/dockyards-backend/pkg/api/v1alpha2/index"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (h *handler) GetWhoami(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	logger := middleware.LoggerFrom(ctx)

	sub, err := middleware.SubjectFrom(ctx)
	if err != nil {
		logger.Error("error getting subject from context", "err", err)
		w.WriteHeader(http.StatusInternalServerError)

		return
	}

	matchingFields := client.MatchingFields{
		index.UIDField: sub,
	}

	var userList dockyardsv1.UserList
	err = h.List(ctx, &userList, matchingFields)
	if err != nil {
		logger.Error("error getting user from kubernetes", "err", err)
		w.WriteHeader(http.StatusInternalServerError)

		return
	}

	if len(userList.Items) != 1 {
		w.WriteHeader(http.StatusInternalServerError)

		return
	}

	user := userList.Items[0]

	v1User := v1.User{
		Id:    string(user.UID),
		Name:  user.Name,
		Email: user.Spec.Email,
	}

	b, err := json.Marshal(&v1User)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)

		return
	}

	w.WriteHeader(http.StatusOK)
	_, err = w.Write(b)
	if err != nil {
		logger.Error("error writing response data", "err", err)
	}
}
