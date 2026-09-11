package project

import (
	"context"

	"github.com/google/uuid"
	"github.com/mllbll/kosmohak_nn/internal/model"
)

func (s *service) Copy(ctx context.Context, req model.CopyProjectRequest) (model.CopyProjectResponse, error) {
	project, err := s.projectRepository.Get(ctx, req.ProjectID)
	if err != nil {
		return model.CopyProjectResponse{}, err
	}

	copied := model.Project{
		ID:        uuid.NewString(),
		Base:      model.CloneScenario(project.Base),
		Effective: model.CloneScenario(project.Effective),
	}
	if err := s.projectRepository.Create(ctx, copied); err != nil {
		return model.CopyProjectResponse{}, err
	}

	return model.CopyProjectResponse{Project: copied}, nil
}
