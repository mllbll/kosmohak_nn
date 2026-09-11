package converter

import (
	"github.com/mllbll/kosmohak_nn/internal/model"
	repoModel "github.com/mllbll/kosmohak_nn/internal/repository/model"
)

func ProjectToRepoModel(req model.Project) repoModel.Project {
	return repoModel.Project{
		ID:        req.ID,
		Base:      req.Base,
		Effective: req.Effective,
	}
}

func ProjectToModel(req repoModel.Project) model.Project {
	return model.Project{
		ID:        req.ID,
		Base:      req.Base,
		Effective: req.Effective,
	}
}

func RunToRepoModel(req model.Run) repoModel.Run {
	return repoModel.Run{
		ID:                req.ID,
		ProjectID:         req.ProjectID,
		EffectiveScenario: req.EffectiveScenario,
		Routes:            req.Routes,
		Metrics:           req.Metrics,
		Summary:           req.Summary,
	}
}

func RunToModel(req repoModel.Run) model.Run {
	return model.Run{
		ID:                req.ID,
		ProjectID:         req.ProjectID,
		EffectiveScenario: req.EffectiveScenario,
		Routes:            req.Routes,
		Metrics:           req.Metrics,
		Summary:           req.Summary,
	}
}
