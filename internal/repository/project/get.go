package project

import (
	"context"

	"github.com/mllbll/kosmohak_nn/internal/model"
	repoConverter "github.com/mllbll/kosmohak_nn/internal/repository/converter"
)

func (r *repository) Get(_ context.Context, projectID string) (model.Project, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	project, ok := r.data[projectID]
	if !ok {
		return model.Project{}, model.ErrProjectNotFound
	}

	return repoConverter.ProjectToModel(project), nil
}
