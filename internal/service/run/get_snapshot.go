package run

import (
	"context"

	"github.com/mllbll/kosmohak_nn/internal/model"
)

func (s *service) GetSnapshot(ctx context.Context, req model.GetSnapshotRequest) (model.GetSnapshotResponse, error) {
	run, err := s.runRepository.Get(ctx, req.RunID)
	if err != nil {
		return model.GetSnapshotResponse{}, err
	}

	clientID := req.ClientID
	if clientID == "" {
		ids := run.EffectiveScenario.ClientIDs()
		if len(ids) == 0 {
			return model.GetSnapshotResponse{}, model.ErrInvalidArgument
		}
		clientID = ids[0]
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
