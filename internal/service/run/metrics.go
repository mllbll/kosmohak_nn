package run

import (
	"github.com/mllbll/kosmohak_nn/internal/model"
)

func Aggregate(sc model.Scenario, routes []model.RouteRecord, visible map[int]map[string]bool) []model.ClientMetrics {
	grid := sc.TimeGrid()
	n := float64(len(grid))
	if n == 0 {
		return nil
	}

	byClient := map[string][]model.RouteRecord{}
	for _, r := range routes {
		byClient[r.ClientID] = append(byClient[r.ClientID], r)
	}

	out := make([]model.ClientMetrics, 0, len(sc.ClientIDs()))
	for _, id := range sc.ClientIDs() {
		recs := byClient[id]
		vis := 0
		ok := 0
		hopsSum := 0
		streak := 0
		maxStreak := 0
		for _, rec := range recs {
			if visible[rec.TS][id] {
				vis++
			}
			if len(rec.Path) > 0 {
				ok++
				hopsSum += rec.Hops
				streak = 0
			} else {
				streak++
				if streak > maxStreak {
					maxStreak = streak
				}
			}
		}
		mean := 0.0
		if ok > 0 {
			mean = float64(hopsSum) / float64(ok)
		}
		pathRatio := float64(ok) / n
		out = append(out, model.ClientMetrics{
			ClientID:        id,
			VisibilityRatio: float64(vis) / n,
			PathRatio:       pathRatio,
			MaxGapS:         maxStreak * sc.Environment.StepS,
			MeanHops:        mean,
			MeetsTarget:     pathRatio >= sc.Environment.TargetAvailability,
		})
	}
	return out
}
