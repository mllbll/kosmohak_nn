package run

import (
	"context"
	"fmt"

	"github.com/mllbll/kosmohak_nn/internal/model"
)

func (s *service) Compare(ctx context.Context, req model.CompareRunsRequest) (model.CompareRunsResponse, error) {
	ids, err := compareRunIDs(req)
	if err != nil {
		return model.CompareRunsResponse{}, err
	}

	runs := make([]model.Run, 0, len(ids))
	for _, id := range ids {
		run, err := s.runRepository.Get(ctx, id)
		if err != nil {
			return model.CompareRunsResponse{}, err
		}
		runs = append(runs, run)
	}

	return buildCompare(runs), nil
}

func compareRunIDs(req model.CompareRunsRequest) ([]string, error) {
	if len(req.RunIDs) >= 2 {
		return req.RunIDs, nil
	}
	if req.RunAID != "" && req.RunBID != "" {
		return []string{req.RunAID, req.RunBID}, nil
	}
	return nil, fmt.Errorf("%w: two or more run ids required", model.ErrInvalidArgument)
}

func buildCompare(runs []model.Run) model.CompareRunsResponse {
	variants := make([]model.CompareVariant, 0, len(runs))
	for _, run := range runs {
		variants = append(variants, toCompareVariant(run))
	}

	resp := model.CompareRunsResponse{
		Variants:       variants,
		Clients:        clientDiffs(runs),
		Recommendation: recommend(runs, variants),
	}

	if len(runs) == 2 {
		resp.RunAID = runs[0].ID
		resp.RunBID = runs[1].ID
		resp.Config = configDiff(runs[0].EffectiveScenario, runs[1].EffectiveScenario)
		if resp.Recommendation.RunID == runs[0].ID {
			resp.Recommendation.Better = "a"
		} else if resp.Recommendation.RunID == runs[1].ID {
			resp.Recommendation.Better = "b"
		} else {
			resp.Recommendation.Better = "tie"
		}
	}

	return resp
}

func toCompareVariant(run model.Run) model.CompareVariant {
	meeting := 0
	pathSum := 0.0
	gapSum := 0.0
	hopsSum := 0.0
	for _, m := range run.Metrics {
		if m.MeetsTarget {
			meeting++
		}
		pathSum += m.PathRatio
		gapSum += float64(m.MaxGapS)
		hopsSum += m.MeanHops
	}
	n := float64(len(run.Metrics))
	meanPath := 0.0
	meanGap := 0.0
	meanHops := 0.0
	if n > 0 {
		meanPath = pathSum / n
		meanGap = gapSum / n
		meanHops = hopsSum / n
	}

	return model.CompareVariant{
		RunID:                run.ID,
		ProjectID:            run.ProjectID,
		Title:                run.EffectiveScenario.Meta.Title,
		LaunchStage:          run.EffectiveScenario.Design.LaunchStage,
		Planes:               append([]model.Plane{}, run.EffectiveScenario.Design.Planes...),
		ClientsMeetingTarget: meeting,
		ClientsTotal:         len(run.Metrics),
		MeanPathRatio:        meanPath,
		MeanMaxGapS:          meanGap,
		MeanHops:             meanHops,
		Metrics:              cloneClientMetrics(run.Metrics),
		Summary:              run.Summary,
	}
}

func clientDiffs(runs []model.Run) []model.ClientDiff {
	indexes := make([]map[string]model.ClientMetrics, 0, len(runs))
	for _, run := range runs {
		idx := map[string]model.ClientMetrics{}
		for _, m := range run.Metrics {
			idx[m.ClientID] = m
		}
		indexes = append(indexes, idx)
	}

	out := make([]model.ClientDiff, 0)
	seen := map[string]struct{}{}
	for _, run := range runs {
		for _, m := range run.Metrics {
			if _, ok := seen[m.ClientID]; ok {
				continue
			}
			seen[m.ClientID] = struct{}{}

			byRun := make([]model.ClientRunMetric, 0, len(runs))
			for i, r := range runs {
				cm := indexes[i][m.ClientID]
				byRun = append(byRun, model.ClientRunMetric{
					RunID:           r.ID,
					PathRatio:       cm.PathRatio,
					VisibilityRatio: cm.VisibilityRatio,
					MaxGapS:         cm.MaxGapS,
					MeanHops:        cm.MeanHops,
					MeetsTarget:     cm.MeetsTarget,
				})
			}

			diff := model.ClientDiff{
				ClientID: m.ClientID,
				Better:   betterClient(byRun),
				ByRun:    byRun,
			}
			if len(byRun) == 2 {
				diff.PathRatioA = byRun[0].PathRatio
				diff.PathRatioB = byRun[1].PathRatio
				diff.DeltaPathRatio = byRun[1].PathRatio - byRun[0].PathRatio
				diff.VisibilityRatioA = byRun[0].VisibilityRatio
				diff.VisibilityRatioB = byRun[1].VisibilityRatio
				diff.MaxGapSA = byRun[0].MaxGapS
				diff.MaxGapSB = byRun[1].MaxGapS
				diff.DeltaMaxGapS = byRun[1].MaxGapS - byRun[0].MaxGapS
				diff.MeetsTargetA = byRun[0].MeetsTarget
				diff.MeetsTargetB = byRun[1].MeetsTarget
				switch diff.Better {
				case byRun[0].RunID:
					diff.Better = "a"
				case byRun[1].RunID:
					diff.Better = "b"
				}
			}
			out = append(out, diff)
		}
	}
	return out
}

func betterClient(metrics []model.ClientRunMetric) string {
	if len(metrics) == 0 {
		return "tie"
	}

	best := 0
	for i := 1; i < len(metrics); i++ {
		if cmpClientMetric(metrics[i], metrics[best]) > 0 {
			best = i
		}
	}
	for i, m := range metrics {
		if i != best && cmpClientMetric(m, metrics[best]) == 0 {
			return "tie"
		}
	}
	return metrics[best].RunID
}

func cmpClientMetric(a, b model.ClientRunMetric) int {
	switch {
	case a.MeetsTarget != b.MeetsTarget:
		if a.MeetsTarget {
			return 1
		}
		return -1
	case a.PathRatio > b.PathRatio:
		return 1
	case a.PathRatio < b.PathRatio:
		return -1
	case a.MaxGapS < b.MaxGapS:
		return 1
	case a.MaxGapS > b.MaxGapS:
		return -1
	default:
		return 0
	}
}

func recommend(runs []model.Run, variants []model.CompareVariant) model.CompareRecommendation {
	if len(variants) == 0 || len(runs) != len(variants) {
		return model.CompareRecommendation{
			Better:      "tie",
			Reason:      "Нет вариантов для сравнения",
			Advantages:  []string{},
			Conditions:  []string{},
			Limitations: []string{},
			Conclusion:  "Недостаточно прогонов, чтобы обосновать выбор конфигурации",
		}
	}

	best := 0
	for i := 1; i < len(variants); i++ {
		if cmpVariant(variants[i], variants[best]) > 0 {
			best = i
		}
	}
	tied := false
	for i, v := range variants {
		if i != best && cmpVariant(v, variants[best]) == 0 {
			tied = true
			break
		}
	}
	return justifyRecommendation(runs, variants, best, tied)
}

func cmpVariant(a, b model.CompareVariant) int {
	switch {
	case a.ClientsMeetingTarget > b.ClientsMeetingTarget:
		return 1
	case a.ClientsMeetingTarget < b.ClientsMeetingTarget:
		return -1
	case a.MeanPathRatio > b.MeanPathRatio:
		return 1
	case a.MeanPathRatio < b.MeanPathRatio:
		return -1
	case a.MeanMaxGapS < b.MeanMaxGapS:
		return 1
	case a.MeanMaxGapS > b.MeanMaxGapS:
		return -1
	default:
		return 0
	}
}

func configDiff(a, b model.Scenario) map[string]any {
	diff := map[string]any{}
	if a.Design.LaunchStage != b.Design.LaunchStage {
		diff["launch_stage"] = map[string]int{"a": a.Design.LaunchStage, "b": b.Design.LaunchStage}
	}

	envKeys := []struct {
		name string
		av   float64
		bv   float64
	}{
		{"altitude_km", a.Environment.AltitudeKM, b.Environment.AltitudeKM},
		{"inclination_deg", a.Environment.InclinationDeg, b.Environment.InclinationDeg},
		{"earth_angle0_deg", a.Environment.EarthAngle0Deg, b.Environment.EarthAngle0Deg},
		{"min_elevation_deg", a.Environment.MinElevationDeg, b.Environment.MinElevationDeg},
		{"isl_range_km", a.Environment.ISLRangeKM, b.Environment.ISLRangeKM},
		{"target_availability", a.Environment.TargetAvailability, b.Environment.TargetAvailability},
	}
	for _, key := range envKeys {
		if key.av != key.bv {
			diff[key.name] = map[string]float64{"a": key.av, "b": key.bv}
		}
	}
	if a.Environment.HorizonS != b.Environment.HorizonS {
		diff["horizon_s"] = map[string]int{"a": a.Environment.HorizonS, "b": b.Environment.HorizonS}
	}
	if a.Environment.StepS != b.Environment.StepS {
		diff["step_s"] = map[string]int{"a": a.Environment.StepS, "b": b.Environment.StepS}
	}

	if !sameFailures(a.Failures, b.Failures) {
		diff["failures"] = map[string]any{"a": a.Failures, "b": b.Failures}
	}
	if !sameOutages(a.GatewayOutages, b.GatewayOutages) {
		diff["gateway_outages"] = map[string]any{"a": a.GatewayOutages, "b": b.GatewayOutages}
	}

	planes := []map[string]any{}
	bPlanes := map[string]model.Plane{}
	for _, p := range b.Design.Planes {
		bPlanes[p.ID] = p
	}
	for _, pa := range a.Design.Planes {
		pb, ok := bPlanes[pa.ID]
		if !ok {
			continue
		}
		if pa.RAANDeg != pb.RAANDeg || pa.PhaseDeg != pb.PhaseDeg {
			planes = append(planes, map[string]any{
				"id": pa.ID,
				"a":  map[string]float64{"raan_deg": pa.RAANDeg, "phase_deg": pa.PhaseDeg},
				"b":  map[string]float64{"raan_deg": pb.RAANDeg, "phase_deg": pb.PhaseDeg},
			})
		}
	}
	if len(planes) > 0 {
		diff["planes"] = planes
	}
	return diff
}

func sameFailures(a, b []model.Failure) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func sameOutages(a, b []model.GatewayOutage) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
