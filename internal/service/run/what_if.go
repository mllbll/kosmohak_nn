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

	satID, gwID, err := resolveFailure(original, req)
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
	if satID != "" {
		effective.Failures = append(append([]model.Failure{}, effective.Failures...), model.Failure{
			SatelliteID: satID,
			StartS:      startS,
			EndS:        endS,
		})
	}
	if gwID != "" {
		effective.GatewayOutages = append(append([]model.GatewayOutage{}, effective.GatewayOutages...), model.GatewayOutage{
			GatewayID: gwID,
			StartS:    startS,
			EndS:      endS,
		})
	}

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
	cmp.Recommendation.Limitations = append([]string{
		"run_b — прогон с искусственным отказом, а не альтернативная конструкция группировки",
	}, cmp.Recommendation.Limitations...)

	return model.WhatIfResponse{
		OriginalRunID:     original.ID,
		ProjectID:         project.ID,
		RunID:             newRun.ID,
		FailedSatelliteID: satID,
		FailedGatewayID:   gwID,
		Metrics:           newRun.Metrics,
		Compare:           cmp,
		Analysis:          buildResilienceAnalysis(original, newRun, satID, gwID, startS, endS),
	}, nil
}

func resolveFailure(run model.Run, req model.WhatIfRequest) (string, string, error) {
	var satID, gwID string

	if req.SatelliteID != "" {
		if !hasSatellite(run.EffectiveScenario, req.SatelliteID) {
			return "", "", fmt.Errorf("%w: unknown satellite %q", model.ErrInvalidArgument, req.SatelliteID)
		}
		satID = req.SatelliteID
	}
	if req.GatewayID != "" {
		if !hasGateway(run.EffectiveScenario, req.GatewayID) {
			return "", "", fmt.Errorf("%w: unknown gateway %q", model.ErrInvalidArgument, req.GatewayID)
		}
		gwID = req.GatewayID
	}
	if satID != "" || gwID != "" {
		return satID, gwID, nil
	}

	satID, err := resolveFailedSatellite(run, req)
	if err != nil {
		return "", "", err
	}
	return satID, "", nil
}

func resolveFailedSatellite(run model.Run, req model.WhatIfRequest) (string, error) {
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

func hasGateway(sc model.Scenario, id string) bool {
	for _, g := range sc.GroundSites {
		if g.Role == "gateway" && g.ID == id {
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
