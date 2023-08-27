package sudo

import (
	"context"

	"bitbucket.org/sudosweden/dockyards-backend/api/v1"
)

func (a *sudoAPI) GetOrganizations(ctx context.Context, req GetOrganizationsRequestObject) (GetOrganizationsResponseObject, error) {
	var organizations []v1.Organization
	err := a.db.Find(&organizations).Error
	if err != nil {
		a.logger.Error("error finding organizations in database", "err", err)

		return GetOrganizations500Response{}, nil
	}

	return GetOrganizations200JSONResponse(organizations), nil
}
