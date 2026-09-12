package converter

import (
	"github.com/mllbll/kosmohak_nn/internal/model"
	repoModel "github.com/mllbll/kosmohak_nn/internal/repository/model"
)

func ProjectToRepoModel(req model.Project) repoModel.Project {
	return repoModel.Project{
		ID:        req.ID,
		Base:      model.CloneScenario(req.Base),
		Effective: model.CloneScenario(req.Effective),
	}
}

func ProjectToModel(req repoModel.Project) model.Project {
	return model.Project{
		ID:        req.ID,
		Base:      model.CloneScenario(req.Base),
		Effective: model.CloneScenario(req.Effective),
	}
}

func RunToRepoModel(req model.Run) repoModel.Run {
	return repoModel.Run{
		ID:                req.ID,
		ProjectID:         req.ProjectID,
		EffectiveScenario: model.CloneScenario(req.EffectiveScenario),
		Routes:            cloneRoutes(req.Routes),
		Metrics:           cloneMetrics(req.Metrics),
		Summary:           req.Summary,
	}
}

func RunToModel(req repoModel.Run) model.Run {
	return model.Run{
		ID:                req.ID,
		ProjectID:         req.ProjectID,
		EffectiveScenario: model.CloneScenario(req.EffectiveScenario),
		Routes:            cloneRoutes(req.Routes),
		Metrics:           cloneMetrics(req.Metrics),
		Summary:           req.Summary,
	}
}

func cloneRoutes(in []model.RouteRecord) []model.RouteRecord {
	out := make([]model.RouteRecord, len(in))
	for i, rec := range in {
		rec.Path = append([]string{}, rec.Path...)
		out[i] = rec
	}
	return out
}

func cloneMetrics(in []model.ClientMetrics) []model.ClientMetrics {
	out := make([]model.ClientMetrics, len(in))
	for i, m := range in {
		m.Gaps = append([]model.GapInterval{}, m.Gaps...)
		out[i] = m
	}
	return out
}
