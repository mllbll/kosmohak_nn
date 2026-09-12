package run

import (
	"context"
	"fmt"
	"math"

	"github.com/mllbll/kosmohak_nn/internal/model"
)

func (s *service) GetSnapshot(ctx context.Context, req model.GetSnapshotRequest) (model.GetSnapshotResponse, error) {
	run, err := s.runRepository.Get(ctx, req.RunID)
	if err != nil {
		return model.GetSnapshotResponse{}, err
	}

	horizon := float64(run.EffectiveScenario.Environment.HorizonS)
	if math.IsNaN(req.TS) || math.IsInf(req.TS, 0) || req.TS < 0 || req.TS > horizon {
		return model.GetSnapshotResponse{}, fmt.Errorf("%w: t_s must be within [0, horizon_s]", model.ErrInvalidArgument)
	}

	ids := run.EffectiveScenario.ClientIDs()
	clientID := req.ClientID
	if clientID == "" {
		if len(ids) == 0 {
			return model.GetSnapshotResponse{}, fmt.Errorf("%w: client_id required", model.ErrInvalidArgument)
		}
		clientID = ids[0]
	} else if !hasClient(ids, clientID) {
		return model.GetSnapshotResponse{}, fmt.Errorf("%w: unknown client %q", model.ErrInvalidArgument, clientID)
	}

	snap, err := s.geometryClient.Snapshot(ctx, run.EffectiveScenario, req.TS)
	if err != nil {
		return model.GetSnapshotResponse{}, err
	}

	route := Route(run.EffectiveScenario, snap, clientID)
	prev, prevFound := previousRoute(run, clientID, req.TS)
	return model.GetSnapshotResponse{
		Snapshot:     snap,
		Route:        route,
		NetworkDelta: networkDelta(run.EffectiveScenario, snap, prev, prevFound, route),
	}, nil
}

func hasClient(ids []string, id string) bool {
	for _, item := range ids {
		if item == id {
			return true
		}
	}
	return false
}

func previousRoute(run model.Run, clientID string, ts float64) ([]string, bool) {
	step := run.EffectiveScenario.Environment.StepS
	if step <= 0 {
		return nil, false
	}
	prevT := int(ts) - step
	if prevT < 0 {
		return nil, false
	}
	for _, rec := range run.Routes {
		if rec.ClientID == clientID && rec.TS == prevT {
			return rec.Path, true
		}
	}
	return nil, false
}
