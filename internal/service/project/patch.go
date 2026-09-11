package project

import (
	"context"

	"github.com/mllbll/kosmohak_nn/internal/model"
)

func (s *service) Patch(ctx context.Context, req model.PatchProjectRequest) (model.PatchProjectResponse, error) {
	project, err := s.projectRepository.Get(ctx, req.ProjectID)
	if err != nil {
		return model.PatchProjectResponse{}, err
	}

	effective, err := ApplyPatch(project.Effective, req.Patch)
	if err != nil {
		return model.PatchProjectResponse{}, err
	}

	if _, err := s.geometryClient.Snapshot(ctx, effective, 0); err != nil {
		return model.PatchProjectResponse{}, err
	}

	project.Effective = effective
	if err := s.projectRepository.Update(ctx, project); err != nil {
		return model.PatchProjectResponse{}, err
	}

	return model.PatchProjectResponse{Project: project}, nil
}
