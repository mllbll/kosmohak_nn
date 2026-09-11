package run

import (
	"context"

	"github.com/mllbll/kosmohak_nn/internal/model"
	repoConverter "github.com/mllbll/kosmohak_nn/internal/repository/converter"
)

func (r *repository) Create(_ context.Context, req model.Run) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.data[req.ID] = repoConverter.RunToRepoModel(req)
	return nil
}
