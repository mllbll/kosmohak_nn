package run

import (
	"fmt"
	"sort"

	"github.com/mllbll/kosmohak_nn/internal/model"
)

func buildResilienceAnalysis(
	original, failed model.Run,
	satID, gwID string,
	startS, endS float64,
) model.ResilienceAnalysis {
	t := int(startS)
	beforeIdx := indexRoutes(original.Routes)
	afterIdx := indexRoutes(failed.Routes)
	beforeMetrics := indexMetrics(original.Metrics)
	afterMetrics := indexMetrics(failed.Metrics)

	clients := original.EffectiveScenario.ClientIDs()
	if len(clients) == 0 {
		clients = metricClientIDs(original.Metrics, failed.Metrics)
	}

	outClients := make([]model.ResilienceClient, 0, len(clients))
	affected := make([]string, 0)
	preserved := make([]string, 0)
	singlePoint := make([]string, 0)

	for _, clientID := range clients {
		before := beforeIdx[routeKey{clientID, t}]
		after := afterIdx[routeKey{clientID, t}]
		bm := beforeMetrics[clientID]
		am := afterMetrics[clientID]
		lost, window, pathBefore, pathAfter := windowStats(original.Routes, failed.Routes, clientID, startS, endS)
		reason := dominantReason(failed.Routes, clientID, startS, endS)

		item := model.ResilienceClient{
			ClientID:          clientID,
			PathBefore:        copyPath(before.Path),
			PathAfter:         copyPath(after.Path),
			ReasonBefore:      before.Reason,
			ReasonAfter:       after.Reason,
			PathRatioBefore:   bm.PathRatio,
			PathRatioAfter:    am.PathRatio,
			DeltaPathRatio:    am.PathRatio - bm.PathRatio,
			MaxGapBefore:      bm.MaxGapS,
			MaxGapAfter:       am.MaxGapS,
			DeltaMaxGapS:      am.MaxGapS - bm.MaxGapS,
			MeetsTargetBefore: bm.MeetsTarget,
			MeetsTargetAfter:  am.MeetsTarget,
			LostSteps:         lost,
			WindowSteps:       window,
			WindowPathBefore:  pathBefore,
			WindowPathAfter:   pathAfter,
			DominantGapReason: reason,
			RoutePreserved:    len(after.Path) > 0,
		}
		item.Affected = lost > 0 || item.DeltaPathRatio < 0 || item.DeltaMaxGapS > 0 || (len(before.Path) > 0 && len(after.Path) == 0)
		if item.Affected {
			affected = append(affected, clientID)
		} else {
			preserved = append(preserved, clientID)
		}
		if satID != "" && pathContains(before.Path, satID) {
			singlePoint = append(singlePoint, clientID)
		}
		outClients = append(outClients, item)
	}

	gapBefore := countGapReasons(original.Routes)
	gapAfter := countGapReasons(failed.Routes)
	analysis := model.ResilienceAnalysis{
		FailedSatelliteID: satID,
		FailedGatewayID:   gwID,
		Interval:          model.FailureInterval{StartS: startS, EndS: endS},
		AffectedClients:   affected,
		PreservedClients:  preserved,
		Clients:           outClients,
		GapReasons: model.GapReasonDiff{
			Before: gapBefore,
			After:  gapAfter,
			Delta:  gapDelta(gapBefore, gapAfter),
		},
	}
	analysis.Vulnerabilities = vulnerabilities(analysis, satID, gwID, singlePoint, t)
	analysis.Mitigations = mitigations(original.EffectiveScenario, analysis)
	analysis.Summary = resilienceSummary(analysis, satID, gwID)
	return analysis
}

type routeKey struct {
	clientID string
	ts       int
}

func indexRoutes(routes []model.RouteRecord) map[routeKey]model.RouteRecord {
	out := map[routeKey]model.RouteRecord{}
	for _, rec := range routes {
		out[routeKey{rec.ClientID, rec.TS}] = rec
	}
	return out
}

func indexMetrics(metrics []model.ClientMetrics) map[string]model.ClientMetrics {
	out := map[string]model.ClientMetrics{}
	for _, m := range metrics {
		out[m.ClientID] = m
	}
	return out
}

func metricClientIDs(a, b []model.ClientMetrics) []string {
	seen := map[string]struct{}{}
	ids := make([]string, 0)
	for _, list := range [][]model.ClientMetrics{a, b} {
		for _, m := range list {
			if _, ok := seen[m.ClientID]; ok {
				continue
			}
			seen[m.ClientID] = struct{}{}
			ids = append(ids, m.ClientID)
		}
	}
	return ids
}

func copyPath(path []string) []string {
	if len(path) == 0 {
		return []string{}
	}
	return append([]string{}, path...)
}

func pathContains(path []string, id string) bool {
	for _, node := range path {
		if node == id {
			return true
		}
	}
	return false
}

func windowStats(before, after []model.RouteRecord, clientID string, startS, endS float64) (lost, window, pathBefore, pathAfter int) {
	b := indexRoutes(before)
	for _, rec := range after {
		if rec.ClientID != clientID {
			continue
		}
		t := float64(rec.TS)
		if t < startS || t >= endS {
			continue
		}
		window++
		prev := b[routeKey{clientID, rec.TS}]
		if len(prev.Path) > 0 {
			pathBefore++
		}
		if len(rec.Path) > 0 {
			pathAfter++
		}
		if len(prev.Path) > 0 && len(rec.Path) == 0 {
			lost++
		}
	}
	if window == 0 {
		for _, rec := range after {
			if rec.ClientID != clientID {
				continue
			}
			prev := b[routeKey{clientID, rec.TS}]
			if len(prev.Path) > 0 && len(rec.Path) == 0 {
				lost++
			}
		}
	}
	return lost, window, pathBefore, pathAfter
}

func dominantReason(routes []model.RouteRecord, clientID string, startS, endS float64) model.GapReason {
	counts := map[model.GapReason]int{}
	for _, rec := range routes {
		if rec.ClientID != clientID || rec.Reason == model.GapNone {
			continue
		}
		t := float64(rec.TS)
		if endS > startS && (t < startS || t >= endS) {
			continue
		}
		counts[rec.Reason]++
	}
	var best model.GapReason
	bestN := 0
	for reason, n := range counts {
		if n > bestN {
			best = reason
			bestN = n
		}
	}
	return best
}

func countGapReasons(routes []model.RouteRecord) map[string]int {
	out := map[string]int{
		string(model.GapNoVisibleSat):     0,
		string(model.GapNoGatewayContact): 0,
		string(model.GapGatewayOutage):    0,
		string(model.GapISLPartition):     0,
	}
	for _, rec := range routes {
		if rec.Reason == model.GapNone {
			continue
		}
		out[string(rec.Reason)]++
	}
	return out
}

func gapDelta(before, after map[string]int) map[string]int {
	out := map[string]int{}
	keys := make([]string, 0, len(after))
	for k := range after {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		out[k] = after[k] - before[k]
	}
	return out
}

func failedLabel(satID, gwID string) string {
	switch {
	case satID != "" && gwID != "":
		return satID + " и шлюз " + gwID
	case gwID != "":
		return "шлюз " + gwID
	default:
		return "спутник " + satID
	}
}

func gapTitle(reason model.GapReason) string {
	switch reason {
	case model.GapNoVisibleSat:
		return "нет видимого спутника"
	case model.GapNoGatewayContact:
		return "нет контакта со шлюзом"
	case model.GapGatewayOutage:
		return "отказ шлюза"
	case model.GapISLPartition:
		return "разрыв межспутниковой сети"
	default:
		return string(reason)
	}
}

func vulnerabilities(
	analysis model.ResilienceAnalysis,
	satID, gwID string,
	singlePoint []string,
	t int,
) []string {
	out := make([]string, 0)
	label := failedLabel(satID, gwID)
	if len(singlePoint) > 1 {
		out = append(out, fmt.Sprintf(
			"%s — общая точка отказа для пунктов %v в t_s=%d: все эти направления шли через один узел",
			satID, singlePoint, t,
		))
	}

	for _, c := range analysis.Clients {
		if !c.Affected {
			if c.RoutePreserved && len(c.PathBefore) > 0 && len(c.PathAfter) > 0 && hops(c.PathAfter) > hops(c.PathBefore) {
				out = append(out, fmt.Sprintf(
					"Пункт %s: маршрут сохранился, но удлинился (%d → %d hops) — запас по связности мал",
					c.ClientID, hops(c.PathBefore), hops(c.PathAfter),
				))
			}
			continue
		}
		reason := c.ReasonAfter
		if reason == model.GapNone {
			reason = c.DominantGapReason
		}
		msg := fmt.Sprintf(
			"Пункт %s: отказ %s рвёт направление на шлюз. Маршрут %v пропадает",
			c.ClientID, label, c.PathBefore,
		)
		if reason != model.GapNone {
			msg += ", причина: " + gapTitle(reason)
		}
		if c.DeltaMaxGapS > 0 {
			msg += fmt.Sprintf(". Перерыв вырос на %d с (было %d, стало %d)", c.DeltaMaxGapS, c.MaxGapBefore, c.MaxGapAfter)
		}
		out = append(out, msg)
	}

	if len(out) == 0 {
		out = append(out, fmt.Sprintf(
			"Отказ %s не разрывает направления заданных пунктов: маршруты сохранились",
			label,
		))
	}
	return out
}

func hops(path []string) int {
	if len(path) == 0 {
		return 0
	}
	return len(path) - 1
}

func mitigations(sc model.Scenario, analysis model.ResilienceAnalysis) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0)

	add := func(msg string) {
		if _, ok := seen[msg]; ok {
			return
		}
		seen[msg] = struct{}{}
		out = append(out, msg)
	}

	if len(analysis.AffectedClients) == 0 {
		add("Группировка устойчива к этому отказу. Дополнительно проверьте отказ шлюза и снижение isl_range_km")
		return out
	}

	if sc.Design.LaunchStage < 3 {
		add("Поднять этап развёртывания до 3 — больше активных спутников и запасных маршрутов")
	}

	reasons := map[model.GapReason]struct{}{}
	lostTarget := make([]string, 0)
	for _, c := range analysis.Clients {
		if c.DominantGapReason != model.GapNone {
			reasons[c.DominantGapReason] = struct{}{}
		}
		if c.ReasonAfter != model.GapNone {
			reasons[c.ReasonAfter] = struct{}{}
		}
		if c.MeetsTargetBefore && !c.MeetsTargetAfter {
			lostTarget = append(lostTarget, c.ClientID)
		}
	}

	if _, ok := reasons[model.GapISLPartition]; ok {
		add(fmt.Sprintf(
			"Увеличить isl_range_km (сейчас %.0f км) или сдвинуть RAAN/phase плоскостей, чтобы восстановить межспутниковые связи",
			sc.Environment.ISLRangeKM,
		))
	}
	if _, ok := reasons[model.GapNoVisibleSat]; ok {
		add("Сдвинуть RAAN/phase плоскости, чтобы улучшить видимость спутников с затронутых пунктов")
	}
	if _, ok := reasons[model.GapNoGatewayContact]; ok {
		add("Добавить запасной шлюз или сдвинуть фазу орбиты, чтобы восстановить downlink")
	}
	if _, ok := reasons[model.GapGatewayOutage]; ok {
		add("Дублировать шлюз: один наземный пункт не должен быть единственной точкой выхода")
	}
	if analysis.FailedSatelliteID != "" {
		add("Разнести маршруты по разным спутникам и плоскостям, чтобы один отказ не гасил несколько пунктов")
	}
	if len(lostTarget) > 0 {
		add(fmt.Sprintf(
			"Вернуть пункты %v выше целевой доступности %.0f%% — текущий отказ опускает их ниже цели",
			lostTarget, sc.Environment.TargetAvailability*100,
		))
	}
	if len(out) == 0 {
		add("Сравнить вариант с другим launch_stage и сдвигом плоскостей через PATCH + /api/compare")
	}
	return out
}

func resilienceSummary(analysis model.ResilienceAnalysis, satID, gwID string) string {
	label := failedLabel(satID, gwID)
	if len(analysis.AffectedClients) == 0 {
		return fmt.Sprintf(
			"Отказ %s: маршруты сохранились для всех %d пунктов. Группировка устойчива к этому отказу",
			label, len(analysis.PreservedClients),
		)
	}
	return fmt.Sprintf(
		"Отказ %s затрагивает %d из %d направлений (%v), маршруты сохранились у %v. %s",
		label,
		len(analysis.AffectedClients),
		len(analysis.AffectedClients)+len(analysis.PreservedClients),
		analysis.AffectedClients,
		analysis.PreservedClients,
		analysis.Mitigations[0],
	)
}
