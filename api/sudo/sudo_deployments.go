package sudo

import (
	"context"

	"bitbucket.org/sudosweden/dockyards-backend/api/v1"
	utildeployment "bitbucket.org/sudosweden/dockyards-backend/pkg/util/deployment"
	"github.com/google/uuid"
)

func (a *sudoAPI) GetDeployments(ctx context.Context, req GetDeploymentsRequestObject) (GetDeploymentsResponseObject, error) {
	var deployments []v1.Deployment
	err := a.db.Find(&deployments).Error
	if err != nil {
		a.logger.Error("error finding deployments in database", "err", err)

		return GetDeployments500Response{}, nil
	}

	return GetDeployments200JSONResponse(deployments), nil
}

func (a *sudoAPI) GetDeployment(ctx context.Context, req GetDeploymentRequestObject) (GetDeploymentResponseObject, error) {
	var deployment v1.Deployment
	err := a.db.Take(&deployment, "id = ?", req.DeploymentID).Error
	if err != nil {
		a.logger.Error("error taking deployment from database", "id", req.DeploymentID, "err", err)

		return GetDeployment500Response{}, nil
	}

	return GetDeployment200JSONResponse(deployment), nil
}

func (a *sudoAPI) CreateDeployment(ctx context.Context, req CreateDeploymentRequestObject) (CreateDeploymentResponseObject, error) {
	deployment := *req.Body
	deployment.ID = uuid.New()

	err := utildeployment.CreateRepository(&deployment, a.gitProjectRoot)
	if err != nil {
		a.logger.Error("error creating repository for deployment", "err", err)

		return CreateDeployment500Response{}, nil
	}

	err = a.db.Create(&deployment).Error
	if err != nil {
		a.logger.Error("error creating deployment in database", "err", err)

		return CreateDeployment500Response{}, nil
	}

	return CreateDeployment200JSONResponse(deployment), nil
}

func (a *sudoAPI) CreateDeploymentStatus(ctx context.Context, req CreateDeploymentStatusRequestObject) (CreateDeploymentStatusResponseObject, error) {
	deploymentStatus := *req.Body
	deploymentStatus.ID = uuid.New()

	var lastDeploymentStatus v1.DeploymentStatus
	err := a.db.Order("created_at desc").First(&lastDeploymentStatus, "deployment_id = ?", deploymentStatus.DeploymentID).Error
	if *lastDeploymentStatus.State == *deploymentStatus.State && *lastDeploymentStatus.Health == *deploymentStatus.Health {
		a.logger.Error("deployment status same as last")

		return CreateDeploymentStatus208Response{}, nil
	}

	err = a.db.Create(&deploymentStatus).Error
	if err != nil {
		a.logger.Error("error creating deployment status in database", "err", err)

		return CreateDeploymentStatus500Response{}, nil
	}

	return CreateDeploymentStatus201Response{}, nil
}
