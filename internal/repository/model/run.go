package model

import domain "github.com/mllbll/kosmohak_nn/internal/model"

type Run struct {
	ID                string
	ProjectID         string
	EffectiveScenario domain.Scenario
	Routes            []domain.RouteRecord
	Metrics           []domain.ClientMetrics
	Summary           string
}
