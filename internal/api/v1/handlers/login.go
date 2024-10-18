package handlers

import (
	"encoding/json"
	"io"
	"net/http"

	"bitbucket.org/sudosweden/dockyards-api/pkg/types"
	"bitbucket.org/sudosweden/dockyards-backend/internal/api/v1/middleware"
	dockyardsv1 "bitbucket.org/sudosweden/dockyards-backend/pkg/api/v1alpha3"
	"bitbucket.org/sudosweden/dockyards-backend/pkg/api/v1alpha3/index"
	"golang.org/x/crypto/bcrypt"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (h *handler) UnmarshalBody(r *http.Request, v any) error {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return err
	}

	defer r.Body.Close()

	err = json.Unmarshal(body, v)
	if err != nil {
		return err
	}

	return nil
}

func (h *handler) Login(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	logger := middleware.LoggerFrom(ctx)

	var login types.Login
	if h.UnmarshalBody(r, &login) != nil {
		w.WriteHeader(http.StatusBadRequest)

		return
	}

	matchingFields := client.MatchingFields{
		index.EmailField: login.Email,
	}

	var userList dockyardsv1.UserList
	err := h.List(ctx, &userList, matchingFields)
	if err != nil {
		logger.Error("error getting user from kubernetes", "err", err)
		w.WriteHeader(http.StatusUnauthorized)

		return
	}

	if len(userList.Items) != 1 {
		logger.Error("expected exactly one user from kubernetes", "users", len(userList.Items))
		w.WriteHeader(http.StatusUnauthorized)

		return
	}

	user := userList.Items[0]

	condition := meta.FindStatusCondition(user.Status.Conditions, dockyardsv1.ReadyCondition)
	if condition == nil || condition.Status != metav1.ConditionTrue {
		logger.Error("user is not ready")
		w.WriteHeader(http.StatusForbidden)

		return
	}

	//Compare sent in pass with saved user pass hash
	err = bcrypt.CompareHashAndPassword([]byte(user.Spec.Password), []byte(login.Password))
	if err != nil {
		logger.Error("error comparing password", "err", err)
		w.WriteHeader(http.StatusUnauthorized)

		return
	}

	tokens, err := h.generateTokens(user)
	if err != nil {
		logger.Error("error generating tokens", "err", err)
		w.WriteHeader(http.StatusInternalServerError)

		return
	}

	b, err := json.Marshal(&tokens)
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
