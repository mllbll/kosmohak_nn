package project

import (
	"context"

	"github.com/mllbll/kosmohak_nn/internal/model"
)

func (s *service) Get(ctx context.Context, req model.GetProjectRequest) (model.GetProjectResponse, error) {
	project, err := s.projectRepository.Get(ctx, req.ProjectID)
	if err != nil {
		return model.GetProjectResponse{}, err
	}
	return model.GetProjectResponse{Project: project}, nil
}
