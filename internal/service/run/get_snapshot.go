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

	return model.GetSnapshotResponse{
		Snapshot: snap,
		Route:    Route(run.EffectiveScenario, snap, clientID),
	}, nil
}
