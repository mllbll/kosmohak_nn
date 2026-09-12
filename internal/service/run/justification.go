package run

import (
	"fmt"
	"strings"

	"github.com/mllbll/kosmohak_nn/internal/model"
)

func justifyRecommendation(
	runs []model.Run,
	variants []model.CompareVariant,
	best int,
	tied bool,
) model.CompareRecommendation {
	winner := variants[best]
	winnerRun := runs[best]
	sc := winnerRun.EffectiveScenario
	target := sc.Environment.TargetAvailability

	rec := model.CompareRecommendation{
		Advantages:  []string{},
		Conditions:  conditionsOf(sc, winner),
		Limitations: limitationsOf(runs, variants, best),
	}

	if tied {
		rec.Better = "tie"
		rec.Reason = "Ничья: варианты дают одинаковую доступность для заданных пунктов"
		rec.Advantages = []string{
			"По расчёту варианты неразличимы: число пунктов с целью, средний path_ratio и max_gap совпадают",
		}
		rec.Conclusion = fmt.Sprintf(
			"Итог: нет единственного победителя. Цель %.0f%% и перерывы совпадают у сравниваемых прогонов — выбирать по этапу развёртывания или проверить устойчивость через what-if",
			target*100,
		)
		return rec
	}

	rec.RunID = winner.RunID
	rec.Reason = fmt.Sprintf(
		"Рекомендуется %s: %d из %d пунктов достигают цели %.0f%%, средняя доступность пути %.0f%%",
		variantLabel(winner),
		winner.ClientsMeetingTarget,
		winner.ClientsTotal,
		target*100,
		winner.MeanPathRatio*100,
	)
	rec.Advantages = advantagesOf(variants, best)
	rec.Conclusion = conclusionOf(winner, sc, rec)
	return rec
}

func variantLabel(v model.CompareVariant) string {
	title := v.Title
	if title == "" {
		title = v.RunID
	}
	return fmt.Sprintf("«%s» (этап %d)", title, v.LaunchStage)
}

func advantagesOf(variants []model.CompareVariant, best int) []string {
	winner := variants[best]
	out := make([]string, 0)
	add := uniqueAppender(&out)

	for i, other := range variants {
		if i == best {
			continue
		}
		otherLabel := variantLabel(other)
		if winner.ClientsMeetingTarget > other.ClientsMeetingTarget {
			add(fmt.Sprintf(
				"Цель выполняют %d из %d пунктов против %d из %d у %s",
				winner.ClientsMeetingTarget, winner.ClientsTotal,
				other.ClientsMeetingTarget, other.ClientsTotal,
				otherLabel,
			))
		}
		if winner.MeanPathRatio > other.MeanPathRatio {
			add(fmt.Sprintf(
				"Средняя доступность пути %.0f%% против %.0f%% у %s (%+.0f п.п.)",
				winner.MeanPathRatio*100, other.MeanPathRatio*100, otherLabel,
				(winner.MeanPathRatio-other.MeanPathRatio)*100,
			))
		}
		if winner.MeanMaxGapS < other.MeanMaxGapS {
			add(fmt.Sprintf(
				"Средний макс. перерыв %.0f с против %.0f с у %s",
				winner.MeanMaxGapS, other.MeanMaxGapS, otherLabel,
			))
		}
		if winner.LaunchStage > other.LaunchStage {
			add(fmt.Sprintf(
				"Выше этап развёртывания (%d против %d) — активны спутники с launch_batch ≤ %d",
				winner.LaunchStage, other.LaunchStage, winner.LaunchStage,
			))
		}
		for _, m := range winner.Metrics {
			om := metricByClient(other.Metrics, m.ClientID)
			if m.MeetsTarget && !om.MeetsTarget {
				add(fmt.Sprintf(
					"Пункт %s достигает цели только в выбранном проекте: path_ratio %.0f%% против %.0f%%, max_gap %d с против %d с",
					m.ClientID, m.PathRatio*100, om.PathRatio*100, m.MaxGapS, om.MaxGapS,
				))
				continue
			}
			if m.PathRatio > om.PathRatio {
				add(fmt.Sprintf(
					"Пункт %s: path_ratio %.0f%% против %.0f%% у %s",
					m.ClientID, m.PathRatio*100, om.PathRatio*100, otherLabel,
				))
			}
			if om.MaxGapS > m.MaxGapS {
				add(fmt.Sprintf(
					"Пункт %s: перерыв %d с против %d с у %s",
					m.ClientID, m.MaxGapS, om.MaxGapS, otherLabel,
				))
			}
		}
	}

	if len(out) == 0 {
		add("Выбранный проект лучше остальных по совокупности цели, path_ratio и перерывов")
	}
	return out
}

func conditionsOf(sc model.Scenario, winner model.CompareVariant) []string {
	out := make([]string, 0)
	add := uniqueAppender(&out)
	target := sc.Environment.TargetAvailability

	add(fmt.Sprintf(
		"Цель: path_ratio ≥ %.0f%% (target_availability=%.2f) для заданных наземных пунктов",
		target*100, target,
	))
	add(fmt.Sprintf(
		"Пункты-клиенты: %s; шлюзы: %s",
		joinIDs(sc.ClientIDs()), joinIDs(sc.GatewayIDs()),
	))
	add(fmt.Sprintf(
		"Конфигурация расчёта: этап %d, ISL ≤ %.0f км, мин. угол места %.0f°, горизонт %d с, шаг %d с",
		winner.LaunchStage,
		sc.Environment.ISLRangeKM,
		sc.Environment.MinElevationDeg,
		sc.Environment.HorizonS,
		sc.Environment.StepS,
	))

	if len(sc.Failures) == 0 && len(sc.GatewayOutages) == 0 {
		add("Цель посчитана при штатной работе: без отказов спутников и без outage шлюзов")
	} else {
		add(fmt.Sprintf(
			"Цель посчитана при отказах спутников %s и outage шлюзов %s",
			joinIDs(failureSatIDs(sc)), joinIDs(outageGatewayIDs(sc)),
		))
	}

	meeting := make([]string, 0)
	parts := make([]string, 0)
	for _, m := range winner.Metrics {
		parts = append(parts, fmt.Sprintf("%s=%.0f%%", m.ClientID, m.PathRatio*100))
		if m.MeetsTarget {
			meeting = append(meeting, m.ClientID)
		}
	}
	if len(meeting) == winner.ClientsTotal && winner.ClientsTotal > 0 {
		add(fmt.Sprintf("При этих условиях все пункты достигают цели (%s)", strings.Join(parts, ", ")))
	} else if len(meeting) > 0 {
		add(fmt.Sprintf("При этих условиях цель выполняют пункты %s (%s)", joinIDs(meeting), strings.Join(parts, ", ")))
	} else {
		add(fmt.Sprintf("При этих условиях ни один пункт не достигает цели (%s)", strings.Join(parts, ", ")))
	}
	return out
}

func limitationsOf(runs []model.Run, variants []model.CompareVariant, best int) []string {
	winner := variants[best]
	sc := runs[best].EffectiveScenario
	out := make([]string, 0)
	add := uniqueAppender(&out)

	below := make([]string, 0)
	for _, m := range winner.Metrics {
		if !m.MeetsTarget {
			below = append(below, fmt.Sprintf("%s (path_ratio %.0f%%, max_gap %d с)", m.ClientID, m.PathRatio*100, m.MaxGapS))
		}
		if m.MeetsTarget && m.MaxGapS > 0 {
			add(fmt.Sprintf(
				"Даже у выбранного проекта на пункте %s есть перерывы до %d с",
				m.ClientID, m.MaxGapS,
			))
		}
	}
	if len(below) > 0 {
		add(fmt.Sprintf(
			"Выбранный проект не закрывает цель %.0f%% для %s — как рабочий конфиг без доработки не подходит",
			sc.Environment.TargetAvailability*100, strings.Join(below, "; "),
		))
	}

	if winner.LaunchStage < 3 {
		add(fmt.Sprintf(
			"Неполная группировка (этап %d из 3): спутники более поздних партий неактивны, запас по отказам ограничен",
			winner.LaunchStage,
		))
	}

	gws := sc.GatewayIDs()
	if len(gws) == 1 && len(sc.GatewayOutages) == 0 {
		add(fmt.Sprintf(
			"Один шлюз %s — единая точка выхода; отказ шлюза этим сравнением не проверялся (нужен what-if с gateway_id)",
			gws[0],
		))
	}
	if len(sc.Failures) == 0 {
		add("Сравнение не заменяет анализ отказа спутника: устойчивость к потере узла на маршруте отдельно не подтверждена (POST /api/runs/{id}/what-if)")
	}

	add(fmt.Sprintf(
		"Расчёт только на горизонте %d с с шагом %d с; за пределами окна доступность не проверялась",
		sc.Environment.HorizonS, sc.Environment.StepS,
	))
	add("Наземные пункты не ретранслируют трафик: путь только client → satellites → gateway")

	for i, other := range variants {
		if i == best {
			continue
		}
		for _, m := range winner.Metrics {
			om := metricByClient(other.Metrics, m.ClientID)
			if om.PathRatio > m.PathRatio {
				add(fmt.Sprintf(
					"Для пункта %s вариант %s лучше по path_ratio (%.0f%% против %.0f%%) — выбранный проект не доминирует на всех направлениях",
					m.ClientID, variantLabel(other), om.PathRatio*100, m.PathRatio*100,
				))
			}
		}
	}

	if len(out) == 0 {
		add("Существенных ограничений по выполненному расчёту не выявлено")
	}
	return out
}

func conclusionOf(winner model.CompareVariant, sc model.Scenario, rec model.CompareRecommendation) string {
	target := sc.Environment.TargetAvailability * 100
	if winner.ClientsMeetingTarget == winner.ClientsTotal && winner.ClientsTotal > 0 {
		return fmt.Sprintf(
			"Итог: брать %s. Цель %.0f%% достигается для всех заданных пунктов при этапе %d и ISL %.0f км. %s",
			variantLabel(winner),
			target,
			winner.LaunchStage,
			sc.Environment.ISLRangeKM,
			firstLimitation(rec.Limitations),
		)
	}
	return fmt.Sprintf(
		"Итог: %s лучше остальных по расчёту, но цель %.0f%% закрыта только у %d из %d пунктов. %s",
		variantLabel(winner),
		target,
		winner.ClientsMeetingTarget,
		winner.ClientsTotal,
		firstLimitation(rec.Limitations),
	)
}

func firstLimitation(items []string) string {
	if len(items) == 0 {
		return ""
	}
	return "Ограничение: " + items[0]
}

func metricByClient(metrics []model.ClientMetrics, id string) model.ClientMetrics {
	for _, m := range metrics {
		if m.ClientID == id {
			return m
		}
	}
	return model.ClientMetrics{ClientID: id}
}

func failureSatIDs(sc model.Scenario) []string {
	ids := make([]string, 0, len(sc.Failures))
	for _, f := range sc.Failures {
		ids = append(ids, f.SatelliteID)
	}
	return ids
}

func outageGatewayIDs(sc model.Scenario) []string {
	ids := make([]string, 0, len(sc.GatewayOutages))
	for _, o := range sc.GatewayOutages {
		ids = append(ids, o.GatewayID)
	}
	return ids
}

func joinIDs(ids []string) string {
	if len(ids) == 0 {
		return "—"
	}
	return strings.Join(ids, ", ")
}

func uniqueAppender(dst *[]string) func(string) {
	seen := map[string]struct{}{}
	return func(msg string) {
		if msg == "" {
			return
		}
		if _, ok := seen[msg]; ok {
			return
		}
		seen[msg] = struct{}{}
		*dst = append(*dst, msg)
	}
}
