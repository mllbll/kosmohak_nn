package repository

import (
	"context"

	"github.com/mllbll/kosmohak_nn/internal/model"
)

type ProjectRepository interface {
	Create(ctx context.Context, project model.Project) error
	Get(ctx context.Context, projectID string) (model.Project, error)
	Update(ctx context.Context, project model.Project) error
}

type RunRepository interface {
	Create(ctx context.Context, run model.Run) error
	Get(ctx context.Context, runID string) (model.Run, error)
}
