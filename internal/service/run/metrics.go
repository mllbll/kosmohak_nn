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

	byClient := map[string]map[int]model.RouteRecord{}
	for _, rec := range routes {
		if byClient[rec.ClientID] == nil {
			byClient[rec.ClientID] = map[int]model.RouteRecord{}
		}
		byClient[rec.ClientID][rec.TS] = rec
	}

	out := make([]model.ClientMetrics, 0, len(sc.ClientIDs()))
	for _, id := range sc.ClientIDs() {
		recs := byClient[id]
		vis := 0
		ok := 0
		hopsSum := 0
		streak := 0
		maxStreak := 0
		gaps := make([]model.GapInterval, 0)
		gapStart := -1
		reasonCnt := map[model.GapReason]int{}

		flushGap := func(endIdx int) {
			if gapStart < 0 || endIdx <= gapStart {
				return
			}
			startT := grid[gapStart]
			endT := grid[endIdx-1] + sc.Environment.StepS
			steps := endIdx - gapStart
			gaps = append(gaps, model.GapInterval{
				StartS:    startT,
				EndS:      endT,
				DurationS: steps * sc.Environment.StepS,
				Reason:    dominantReasonCount(reasonCnt),
			})
			gapStart = -1
			reasonCnt = map[model.GapReason]int{}
		}

		for i, t := range grid {
			if visible[t][id] {
				vis++
			}
			rec, has := recs[t]
			if has && len(rec.Path) > 0 {
				flushGap(i)
				ok++
				hopsSum += rec.Hops
				streak = 0
				continue
			}
			if gapStart < 0 {
				gapStart = i
			}
			if rec.Reason != model.GapNone {
				reasonCnt[rec.Reason]++
			}
			streak++
			if streak > maxStreak {
				maxStreak = streak
			}
		}
		flushGap(len(grid))

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
			Gaps:            gaps,
		})
	}
	return out
}

func cloneClientMetrics(in []model.ClientMetrics) []model.ClientMetrics {
	out := make([]model.ClientMetrics, len(in))
	for i, m := range in {
		m.Gaps = append([]model.GapInterval{}, m.Gaps...)
		out[i] = m
	}
	return out
}

func dominantReasonCount(counts map[model.GapReason]int) model.GapReason {
	var best model.GapReason
	bestN := 0
	tied := false
	for reason, n := range counts {
		if n > bestN {
			best = reason
			bestN = n
			tied = false
		} else if n == bestN {
			tied = true
		}
	}
	if tied {
		return model.GapNone
	}
	return best
}
