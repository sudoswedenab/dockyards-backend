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
	"encoding/json"
	"net/http"

	"github.com/sudoswedenab/dockyards-api/pkg/types"
	"bitbucket.org/sudosweden/dockyards-backend/internal/api/v1/middleware"
	dockyardsv1 "bitbucket.org/sudosweden/dockyards-backend/api/v1alpha3"
	"bitbucket.org/sudosweden/dockyards-backend/api/v1alpha3/index"
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

	v1User := types.User{
		ID:    string(user.UID),
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
