package handlers

import (
	"encoding/json"
	"net/http"

	"bitbucket.org/sudosweden/dockyards-api/pkg/types"
	"bitbucket.org/sudosweden/dockyards-backend/internal/api/v1/middleware"
	"bitbucket.org/sudosweden/dockyards-backend/pkg/api/v1alpha1"
	"bitbucket.org/sudosweden/dockyards-backend/pkg/api/v1alpha1/index"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// +kubebuilder:rbac:groups=dockyards.io,resources=apps,verbs=get;list;watch

func (h *handler) GetApps(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	logger := middleware.LoggerFrom(ctx)

	var appList v1alpha1.AppList
	err := h.List(ctx, &appList)
	if err != nil {
		logger.Error("error listing apps", "err", err)
		w.WriteHeader(http.StatusInternalServerError)

		return
	}

	apps := make([]types.App, len(appList.Items))
	for i, app := range appList.Items {
		apps[i] = types.App{
			ID:   string(app.UID),
			Name: app.Name,
		}

		if app.Spec.Description != "" {
			apps[i].Description = &appList.Items[i].Spec.Description
		}

		if app.Spec.Icon != "" {
			apps[i].Icon = &appList.Items[i].Spec.Icon
		}
	}

	b, err := json.Marshal(&apps)
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

func (h *handler) GetApp(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	logger := middleware.LoggerFrom(ctx)

	appID := r.PathValue("appID")
	if appID == "" {
		w.WriteHeader(http.StatusBadRequest)

		return
	}

	matchingFields := client.MatchingFields{
		index.UIDIndexKey: appID,
	}

	var appList v1alpha1.AppList
	err := h.List(ctx, &appList, matchingFields)
	if err != nil {
		logger.Error("error listing apps", "err", err)
		w.WriteHeader(http.StatusInternalServerError)

		return
	}

	if len(appList.Items) != 1 {
		logger.Debug("expected exactly one app", "count", len(appList.Items))
		w.WriteHeader(http.StatusInternalServerError)

		return
	}

	app := appList.Items[0]

	v1App := types.App{
		ID:   string(app.UID),
		Name: app.Name,
	}

	if app.Spec.Description != "" {
		v1App.Description = &app.Spec.Description
	}

	if app.Spec.Icon != "" {
		v1App.Icon = &app.Spec.Icon
	}

	appSteps := make([]types.AppStep, len(app.Spec.Steps))
	for i, step := range app.Spec.Steps {
		stepOptions := make([]types.StepOption, len(step.Options))

		for j, option := range step.Options {
			stepOptions[j] = types.StepOption{
				Default: &step.Options[j].Default,
			}

			if option.DisplayName != "" {
				stepOptions[j].DisplayName = &step.Options[j].DisplayName
			}

			if option.JSONPointer != "" {
				stepOptions[j].JSONPointer = &step.Options[j].JSONPointer
			}

			if option.Type != "" {
				stepOptions[j].Type = &step.Options[j].Type
			}

			if option.Hidden {
				stepOptions[j].Hidden = &step.Options[j].Hidden
			}

			if option.Managed {
				stepOptions[j].Managed = &step.Options[j].Managed
			}

			if len(option.Toggle) > 0 {
				stepOptions[j].Toggle = &step.Options[j].Toggle
			}

			if len(option.Tags) > 0 {
				stepOptions[j].Tags = &step.Options[j].Tags
			}
		}

		appSteps[i] = types.AppStep{
			Name:        step.Name,
			StepOptions: &stepOptions,
		}
	}

	v1App.AppSteps = &appSteps

	b, err := json.Marshal(&v1App)
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
