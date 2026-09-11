package run

import (
	"context"

	"github.com/mllbll/kosmohak_nn/internal/model"
)

func (s *service) Get(ctx context.Context, req model.GetRunRequest) (model.GetRunResponse, error) {
	run, err := s.runRepository.Get(ctx, req.RunID)
	if err != nil {
		return model.GetRunResponse{}, err
	}
	return model.GetRunResponse{Run: run}, nil
}
