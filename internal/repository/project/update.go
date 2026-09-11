package project

import (
	"context"

	"github.com/mllbll/kosmohak_nn/internal/model"
	repoConverter "github.com/mllbll/kosmohak_nn/internal/repository/converter"
)

func (r *repository) Update(_ context.Context, req model.Project) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.data[req.ID]; !ok {
		return model.ErrProjectNotFound
	}

	r.data[req.ID] = repoConverter.ProjectToRepoModel(req)
	return nil
}
