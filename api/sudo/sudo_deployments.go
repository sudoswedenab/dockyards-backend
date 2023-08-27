package sudo

import (
	"context"

	"bitbucket.org/sudosweden/dockyards-backend/api/v1"
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

	err := a.db.Create(&deployment).Error
	if err != nil {
		a.logger.Error("error creating deployment in database", "err", err)

		return CreateDeployment500Response{}, nil
	}

	return CreateDeployment200JSONResponse(deployment), nil
}
