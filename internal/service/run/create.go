package run

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/mllbll/kosmohak_nn/internal/model"
)

func (s *service) Create(ctx context.Context, req model.CreateRunRequest) (model.CreateRunResponse, error) {
	project, err := s.projectRepository.Get(ctx, req.ProjectID)
	if err != nil {
		return model.CreateRunResponse{}, err
	}

	sc := project.Effective
	clients := sc.ClientIDs()
	if len(clients) == 0 {
		return model.CreateRunResponse{}, fmt.Errorf("%w: no clients in scenario", model.ErrInvalidArgument)
	}

	var routes []model.RouteRecord
	visible := map[int]map[string]bool{}

	err = s.geometryClient.WalkSnapshots(ctx, sc, func(snap model.Snapshot) error {
		t := int(snap.TS)
		visible[t] = map[string]bool{}
		for _, clientID := range clients {
			res := Route(sc, snap, clientID)
			path := res.Path
			if path == nil {
				path = []string{}
			}
			routes = append(routes, model.RouteRecord{
				TS:       t,
				ClientID: clientID,
				Path:     path,
				Reason:   res.Reason,
				Hops:     res.Hops,
			})
			visible[t][clientID] = hasUplink(snap, clientID)
		}
		return nil
	})
	if err != nil {
		return model.CreateRunResponse{}, err
	}

	metrics := Aggregate(sc, routes, visible)
	run := model.Run{
		ID:                uuid.NewString(),
		ProjectID:         project.ID,
		EffectiveScenario: sc,
		Routes:            routes,
		Metrics:           metrics,
		Summary:           buildSummary(metrics, sc.Environment.TargetAvailability),
	}

	if err := s.runRepository.Create(ctx, run); err != nil {
		return model.CreateRunResponse{}, err
	}

	return model.CreateRunResponse{
		RunID:     run.ID,
		ProjectID: run.ProjectID,
		Metrics:   run.Metrics,
	}, nil
}

func hasUplink(snap model.Snapshot, clientID string) bool {
	for _, e := range snap.Edges {
		if e.A == clientID || e.B == clientID {
			return true
		}
	}
	return false
}

func buildSummary(metrics []model.ClientMetrics, target float64) string {
	failed := make([]string, 0)
	for _, m := range metrics {
		if !m.MeetsTarget {
			failed = append(failed, m.ClientID)
		}
	}
	if len(failed) == 0 {
		return fmt.Sprintf("Все пункты достигают целевой доступности %.0f%%", target*100)
	}
	return fmt.Sprintf("Ниже цели %.0f%%: %v", target*100, failed)
}
