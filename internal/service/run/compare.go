package run

import (
	"context"

	"github.com/mllbll/kosmohak_nn/internal/model"
)

func (s *service) Compare(ctx context.Context, req model.CompareRunsRequest) (model.CompareRunsResponse, error) {
	a, err := s.runRepository.Get(ctx, req.RunAID)
	if err != nil {
		return model.CompareRunsResponse{}, err
	}
	b, err := s.runRepository.Get(ctx, req.RunBID)
	if err != nil {
		return model.CompareRunsResponse{}, err
	}

	return model.CompareRunsResponse{
		RunAID:  a.ID,
		RunBID:  b.ID,
		Config:  configDiff(a.EffectiveScenario, b.EffectiveScenario),
		Clients: clientDiffs(a.Metrics, b.Metrics),
	}, nil
}

func configDiff(a, b model.Scenario) map[string]any {
	diff := map[string]any{}
	if a.Design.LaunchStage != b.Design.LaunchStage {
		diff["launch_stage"] = map[string]int{"a": a.Design.LaunchStage, "b": b.Design.LaunchStage}
	}
	if a.Environment.ISLRangeKM != b.Environment.ISLRangeKM {
		diff["isl_range_km"] = map[string]float64{"a": a.Environment.ISLRangeKM, "b": b.Environment.ISLRangeKM}
	}
	if len(a.Failures) != len(b.Failures) {
		diff["failures_count"] = map[string]int{"a": len(a.Failures), "b": len(b.Failures)}
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

func clientDiffs(a, b []model.ClientMetrics) []model.ClientDiff {
	bm := map[string]model.ClientMetrics{}
	for _, m := range b {
		bm[m.ClientID] = m
	}
	out := make([]model.ClientDiff, 0, len(a))
	for _, ma := range a {
		mb := bm[ma.ClientID]
		out = append(out, model.ClientDiff{
			ClientID:       ma.ClientID,
			PathRatioA:     ma.PathRatio,
			PathRatioB:     mb.PathRatio,
			DeltaPathRatio: mb.PathRatio - ma.PathRatio,
			MaxGapSA:       ma.MaxGapS,
			MaxGapSB:       mb.MaxGapS,
			DeltaMaxGapS:   mb.MaxGapS - ma.MaxGapS,
		})
	}
	return out
}
