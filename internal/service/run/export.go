package run

import (
	"context"

	"github.com/mllbll/kosmohak_nn/internal/model"
)

func (s *service) Export(ctx context.Context, req model.ExportRunRequest) (model.ExportDocument, error) {
	run, err := s.runRepository.Get(ctx, req.RunID)
	if err != nil {
		return model.ExportDocument{}, err
	}

	routes := make([]model.ExportRoute, 0, len(run.Routes))
	for _, r := range run.Routes {
		path := r.Path
		if path == nil {
			path = []string{}
		}
		routes = append(routes, model.ExportRoute{
			TS:       r.TS,
			ClientID: r.ClientID,
			Path:     path,
			Reason:   r.Reason,
			Hops:     r.Hops,
		})
	}

	return model.ExportDocument{
		SchemaVersion:     model.ResultSchemaVersion,
		EffectiveScenario: model.CloneScenario(run.EffectiveScenario),
		Routes:            routes,
		Metrics:           cloneClientMetrics(run.Metrics),
		Summary:           run.Summary,
	}, nil
}
