package geometry

import (
	"context"

	"github.com/mllbll/kosmohak_nn/internal/model"
)

type Client interface {
	Snapshot(ctx context.Context, sc model.Scenario, t float64) (model.Snapshot, error)
	WalkSnapshots(ctx context.Context, sc model.Scenario, fn func(model.Snapshot) error) error
}
