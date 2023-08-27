package sudo

import (
	"log/slog"

	"bitbucket.org/sudosweden/dockyards-backend/internal/types"
	"gorm.io/gorm"
)

type sudoAPI struct {
	logger         *slog.Logger
	db             *gorm.DB
	clusterService types.ClusterService
}

type SudoOption func(*sudoAPI)

func WithLogger(logger *slog.Logger) SudoOption {
	return func(a *sudoAPI) {
		a.logger = logger
	}
}

func WithDatabase(db *gorm.DB) SudoOption {
	return func(a *sudoAPI) {
		a.db = db
	}
}

func WithClusterService(clusterService types.ClusterService) SudoOption {
	return func(a *sudoAPI) {
		a.clusterService = clusterService
	}
}

func NewSudoAPI(sudoOptions ...SudoOption) (*sudoAPI, error) {
	a := sudoAPI{}

	for _, sudoOption := range sudoOptions {
		sudoOption(&a)
	}

	return &a, nil
}
