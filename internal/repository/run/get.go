package run

import (
	"context"

	"github.com/mllbll/kosmohak_nn/internal/model"
	repoConverter "github.com/mllbll/kosmohak_nn/internal/repository/converter"
)

func (r *repository) Get(_ context.Context, runID string) (model.Run, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	run, ok := r.data[runID]
	if !ok {
		return model.Run{}, model.ErrRunNotFound
	}

	return repoConverter.RunToModel(run), nil
}
