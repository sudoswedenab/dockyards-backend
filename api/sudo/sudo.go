package sudo

import (
	"log/slog"

	"bitbucket.org/sudosweden/dockyards-backend/internal/clusterservices"
	"gorm.io/gorm"
)

type sudoAPI struct {
	logger         *slog.Logger
	db             *gorm.DB
	clusterService clusterservices.ClusterService
	gitProjectRoot string
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

func WithClusterService(clusterService clusterservices.ClusterService) SudoOption {
	return func(a *sudoAPI) {
		a.clusterService = clusterService
	}
}

func WithGitProjectRoot(gitProjectRoot string) SudoOption {
	return func(a *sudoAPI) {
		a.gitProjectRoot = gitProjectRoot
	}
}

func NewSudoAPI(sudoOptions ...SudoOption) (*sudoAPI, error) {
	a := sudoAPI{}

	for _, sudoOption := range sudoOptions {
		sudoOption(&a)
	}

	return &a, nil
}
