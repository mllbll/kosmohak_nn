package project

import (
	"context"

	"github.com/mllbll/kosmohak_nn/internal/model"
)

func (s *service) Reset(ctx context.Context, req model.ResetProjectRequest) (model.ResetProjectResponse, error) {
	project, err := s.projectRepository.Get(ctx, req.ProjectID)
	if err != nil {
		return model.ResetProjectResponse{}, err
	}

	project.Effective = model.CloneScenario(project.Base)
	if err := s.projectRepository.Update(ctx, project); err != nil {
		return model.ResetProjectResponse{}, err
	}

	return model.ResetProjectResponse{Project: project}, nil
}
