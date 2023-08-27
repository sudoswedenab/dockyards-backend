package sudo

import (
	"context"

	"bitbucket.org/sudosweden/dockyards-backend/api/v1"
)

func (a *sudoAPI) GetCredential(ctx context.Context, req GetCredentialRequestObject) (GetCredentialResponseObject, error) {
	var credential v1.Credential
	err := a.db.Take(&credential, "id = ?", req.CredentialID).Error
	if err != nil {
		a.logger.Error("error taking credential from database", "err", err)

		return GetCredential500Response{}, nil
	}

	return GetCredential200JSONResponse(credential), nil
}
