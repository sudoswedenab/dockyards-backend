package handlers

import (
	"context"
	"net/http"

	"bitbucket.org/sudosweden/dockyards-backend/internal/api/v1"
	"bitbucket.org/sudosweden/dockyards-backend/pkg/api/v1alpha1"
	"bitbucket.org/sudosweden/dockyards-backend/pkg/api/v1alpha1/index"
	"github.com/gin-gonic/gin"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// +kubebuilder:rbac:groups=dockyards.io,resources=apps,verbs=get;list;watch

func (h *handler) GetApps(c *gin.Context) {
	ctx := context.Background()

	var appList v1alpha1.AppList
	err := h.controllerClient.List(ctx, &appList)
	if err != nil {
		h.logger.Error("error listing apps", "err", err)

		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	apps := make([]v1.App, len(appList.Items))
	for i, app := range appList.Items {
		apps[i] = v1.App{
			Id:   string(app.UID),
			Name: app.Name,
		}

		if app.Spec.Description != "" {
			apps[i].Description = &appList.Items[i].Spec.Description
		}

		if app.Spec.Icon != "" {
			apps[i].Icon = &appList.Items[i].Spec.Icon
		}
	}

	c.JSON(http.StatusOK, apps)
}

func (h *handler) GetApp(c *gin.Context) {
	ctx := context.Background()

	appID := c.Param("appID")
	if appID == "" {
		h.logger.Error("empty app id")

		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	matchingFields := client.MatchingFields{
		index.UIDIndexKey: appID,
	}

	var appList v1alpha1.AppList
	err := h.controllerClient.List(ctx, &appList, matchingFields)
	if err != nil {
		h.logger.Error("error listing apps", "err", err)

		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	if len(appList.Items) != 1 {
		h.logger.Debug("expected exactly one app", "count", len(appList.Items))

		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	app := appList.Items[0]

	v1App := v1.App{
		Id:   string(app.UID),
		Name: app.Name,
	}

	if app.Spec.Description != "" {
		v1App.Description = &app.Spec.Description
	}

	if app.Spec.Icon != "" {
		v1App.Icon = &app.Spec.Icon
	}

	appSteps := make([]v1.AppStep, len(app.Spec.Steps))
	for i, step := range app.Spec.Steps {
		stepOptions := make([]v1.StepOption, len(step.Options))

		for j, option := range step.Options {
			stepOptions[j] = v1.StepOption{
				Default: &step.Options[j].Default,
			}

			if option.DisplayName != "" {
				stepOptions[j].DisplayName = &step.Options[j].DisplayName
			}

			if option.JSONPointer != "" {
				stepOptions[j].JsonPointer = &step.Options[j].JSONPointer
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

		appSteps[i] = v1.AppStep{
			Name:        step.Name,
			StepOptions: &stepOptions,
		}
	}

	v1App.AppSteps = &appSteps

	c.JSON(http.StatusOK, v1App)
}
