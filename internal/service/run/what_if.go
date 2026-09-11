package run

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/mllbll/kosmohak_nn/internal/model"
)

func (s *service) WhatIf(ctx context.Context, req model.WhatIfRequest) (model.WhatIfResponse, error) {
	original, err := s.runRepository.Get(ctx, req.RunID)
	if err != nil {
		return model.WhatIfResponse{}, err
	}

	satID, err := resolveFailedSatellite(original, req)
	if err != nil {
		return model.WhatIfResponse{}, err
	}

	startS := req.TS
	if req.StartS != nil {
		startS = *req.StartS
	}
	endS := float64(original.EffectiveScenario.Environment.HorizonS)
	if req.EndS != nil {
		endS = *req.EndS
	}
	if startS < 0 || endS <= startS || endS > float64(original.EffectiveScenario.Environment.HorizonS) {
		return model.WhatIfResponse{}, fmt.Errorf("%w: invalid failure interval", model.ErrInvalidArgument)
	}

	base := model.CloneScenario(original.EffectiveScenario)
	effective := model.CloneScenario(original.EffectiveScenario)
	effective.Failures = append(append([]model.Failure{}, effective.Failures...), model.Failure{
		SatelliteID: satID,
		StartS:      startS,
		EndS:        endS,
	})

	if _, err := s.geometryClient.Snapshot(ctx, effective, 0); err != nil {
		return model.WhatIfResponse{}, err
	}

	project := model.Project{
		ID:        uuid.NewString(),
		Base:      base,
		Effective: effective,
	}
	if err := s.projectRepository.Create(ctx, project); err != nil {
		return model.WhatIfResponse{}, err
	}

	newRun, err := s.execute(ctx, project.ID, effective)
	if err != nil {
		return model.WhatIfResponse{}, err
	}
	if err := s.runRepository.Create(ctx, newRun); err != nil {
		return model.WhatIfResponse{}, err
	}

	cmp, err := s.Compare(ctx, model.CompareRunsRequest{
		RunAID: original.ID,
		RunBID: newRun.ID,
	})
	if err != nil {
		return model.WhatIfResponse{}, err
	}

	return model.WhatIfResponse{
		OriginalRunID:     original.ID,
		ProjectID:         project.ID,
		RunID:             newRun.ID,
		FailedSatelliteID: satID,
		Metrics:           newRun.Metrics,
		Compare:           cmp,
	}, nil
}

func resolveFailedSatellite(run model.Run, req model.WhatIfRequest) (string, error) {
	if req.SatelliteID != "" {
		if !hasSatellite(run.EffectiveScenario, req.SatelliteID) {
			return "", fmt.Errorf("%w: unknown satellite %q", model.ErrInvalidArgument, req.SatelliteID)
		}
		return req.SatelliteID, nil
	}

	clientID := req.ClientID
	if clientID == "" {
		ids := run.EffectiveScenario.ClientIDs()
		if len(ids) == 0 {
			return "", fmt.Errorf("%w: client_id required", model.ErrInvalidArgument)
		}
		clientID = ids[0]
	}

	t := int(req.TS)
	for _, rec := range run.Routes {
		if rec.ClientID == clientID && rec.TS == t {
			satID := satelliteFromPath(run.EffectiveScenario, rec.Path)
			if satID == "" {
				return "", fmt.Errorf("%w: no satellite on path at t_s=%d", model.ErrInvalidArgument, t)
			}
			return satID, nil
		}
	}

	return "", fmt.Errorf("%w: no route for client %s at t_s=%d", model.ErrInvalidArgument, clientID, t)
}

func hasSatellite(sc model.Scenario, id string) bool {
	for _, sat := range sc.Design.Satellites {
		if sat.ID == id {
			return true
		}
	}
	return false
}

func satelliteFromPath(sc model.Scenario, path []string) string {
	sats := map[string]struct{}{}
	for _, sat := range sc.Design.Satellites {
		sats[sat.ID] = struct{}{}
	}
	for _, id := range path {
		if _, ok := sats[id]; ok {
			return id
		}
	}
	return ""
}
