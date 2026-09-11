package project

import (
	"context"

	"github.com/google/uuid"
	"github.com/mllbll/kosmohak_nn/internal/model"
)

func (s *service) Create(ctx context.Context, req model.CreateProjectRequest) (model.CreateProjectResponse, error) {
	if err := ValidateScenario(req.Scenario); err != nil {
		return model.CreateProjectResponse{}, err
	}

	if _, err := s.geometryClient.Snapshot(ctx, req.Scenario, 0); err != nil {
		return model.CreateProjectResponse{}, err
	}

	id := req.Scenario.Meta.ID
	if id == "" {
		id = uuid.NewString()
	}

	project := model.Project{
		ID:        id,
		Base:      req.Scenario,
		Effective: req.Scenario,
	}

	if err := s.projectRepository.Create(ctx, project); err != nil {
		return model.CreateProjectResponse{}, err
	}

	return model.CreateProjectResponse{Project: project}, nil
}
