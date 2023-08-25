package controller

import (
	"context"
	"errors"
	"log/slog"

	"bitbucket.org/sudosweden/dockyards-backend/api/v1"
	"bitbucket.org/sudosweden/dockyards-backend/pkg/api/v1alpha1"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type appController struct {
	client.Client
	scheme *runtime.Scheme
	logger *slog.Logger
	mgr    ctrl.Manager
	db     *gorm.DB
}

func (c *appController) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var app v1alpha1.App
	err := c.Get(ctx, req.NamespacedName, &app)
	if err != nil {
		c.logger.Error("error getting app to reconcile", "err", err)

		return ctrl.Result{}, err
	}

	c.logger.Debug("app to reconcile", "name", app.ObjectMeta.Name)

	var dockyardsApp v1.App
	err = c.db.Take(&dockyardsApp, "name = ?", app.ObjectMeta.Name).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			dockyardsApp := v1.App{
				ID:   uuid.New(),
				Name: app.ObjectMeta.Name,
			}

			if app.Spec.Description != "" {
				dockyardsApp.Description = &app.Spec.Description
			}

			if app.Spec.Icon != "" {
				dockyardsApp.Icon = &app.Spec.Icon
			}

			err := c.db.Create(&dockyardsApp).Error
			if err != nil {
				c.logger.Error("error creating dockyards app in database", "err", err)

				return ctrl.Result{}, err
			}

			c.logger.Debug("created dockyards app", "id", dockyardsApp.ID)

			for _, appStep := range app.Spec.Steps {
				dockyardsAppStep := v1.AppStep{
					ID:    uuid.New(),
					AppID: dockyardsApp.ID,
					Name:  appStep.Name,
				}

				err := c.db.Create(&dockyardsAppStep).Error
				if err != nil {
					c.logger.Error("error creating dockyards app step in database", "err", err)

					return ctrl.Result{}, err
				}

				for _, stepOption := range appStep.Options {
					dockyardsAppOption := v1.StepOption{
						AppStepID:   dockyardsAppStep.ID,
						JSONPointer: stepOption.JSONPointer,
						DisplayName: stepOption.DisplayName,
					}

					if stepOption.Default != "" {
						dockyardsAppOption.Default = &stepOption.Default
					}

					if stepOption.Hidden {
						dockyardsAppOption.Hidden = &stepOption.Hidden
					}

					if stepOption.Type != "" {
						dockyardsAppOption.Type = &stepOption.Type
					}

					if len(stepOption.Selection) != 0 {
						dockyardsAppOption.Selection = &stepOption.Selection
					}

					err := c.db.Create(&dockyardsAppOption).Error
					if err != nil {
						c.logger.Error("error creating dockyards app option in database", "err", err)

						return ctrl.Result{}, err
					}
				}
			}
		}

		c.logger.Error("error taking dockyards app from database", "err", err)

		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

type ControllerOption func(*appController)

func WithLogger(logger *slog.Logger) ControllerOption {
	return func(c *appController) {
		c.logger = logger
	}
}

func WithManager(mgr ctrl.Manager) ControllerOption {
	return func(c *appController) {
		c.mgr = mgr
	}
}

func WithDatabase(db *gorm.DB) ControllerOption {
	return func(c *appController) {
		c.db = db
	}
}

func NewAppController(controllerOptions ...ControllerOption) error {
	c := appController{}

	for _, controllerOption := range controllerOptions {
		controllerOption(&c)
	}

	c.Client = c.mgr.GetClient()
	c.scheme = c.mgr.GetScheme()

	err := ctrl.NewControllerManagedBy(c.mgr).For(&v1alpha1.App{}).Complete(&c)
	if err != nil {
		return err
	}

	return nil
}
