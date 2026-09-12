package run

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/mllbll/kosmohak_nn/internal/model"
	projectsvc "github.com/mllbll/kosmohak_nn/internal/service/project"
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

	if err := projectsvc.ValidateScenario(effective); err != nil {
		return model.WhatIfResponse{}, err
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
	cmp = annotateWhatIfCompare(cmp, original)

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
	ids := run.EffectiveScenario.ClientIDs()
	clientID := req.ClientID
	if clientID == "" {
		if len(ids) == 0 {
			return "", fmt.Errorf("%w: client_id required", model.ErrInvalidArgument)
		}
		clientID = ids[0]
	} else if !hasClient(ids, clientID) {
		return "", fmt.Errorf("%w: unknown client %q", model.ErrInvalidArgument, clientID)
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

func annotateWhatIfCompare(cmp model.CompareRunsResponse, original model.Run) model.CompareRunsResponse {
	const caveat = "run_b — прогон с искусственным отказом, а не альтернативная конструкция группировки"
	if len(cmp.Recommendation.Limitations) == 0 || cmp.Recommendation.Limitations[0] != caveat {
		cmp.Recommendation.Limitations = append([]string{caveat}, cmp.Recommendation.Limitations...)
	}

	origMean, failMean := 0.0, 0.0
	origMeet, failMeet := 0, 0
	if len(cmp.Variants) >= 2 {
		origMean = cmp.Variants[0].MeanPathRatio
		failMean = cmp.Variants[1].MeanPathRatio
		origMeet = cmp.Variants[0].ClientsMeetingTarget
		failMeet = cmp.Variants[1].ClientsMeetingTarget
	}

	cmp.Recommendation.Better = "a"
	cmp.Recommendation.RunID = original.ID
	if origMean > failMean+1e-12 || origMeet > failMeet {
		cmp.Recommendation.Reason = fmt.Sprintf(
			"Рабочая конфигурация — исходный прогон: после отказа средняя доступность пути %.0f%% → %.0f%%",
			origMean*100, failMean*100,
		)
		cmp.Recommendation.Conclusion = "Итог: не выбирать run_b как лучший конфиг. Это what-if с искусственным отказом, исходный вариант устойчивее. Ограничение: " + caveat
		return cmp
	}

	cmp.Recommendation.Reason = "Отказ не ухудшил доступность: запасной маршрут сработал. run_b всё равно не конкурирующая конструкция"
	cmp.Recommendation.Conclusion = "Итог: исходная конфигурация остаётся рабочей; what-if подтвердил устойчивость к этому отказу, а не предложил новый дизайн. Ограничение: " + caveat
	return cmp
}
