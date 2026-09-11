package model

import domain "github.com/mllbll/kosmohak_nn/internal/model"

type Project struct {
	ID        string
	Base      domain.Scenario
	Effective domain.Scenario
}
