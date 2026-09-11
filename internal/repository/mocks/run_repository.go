package mocks

import (
	"context"

	"github.com/mllbll/kosmohak_nn/internal/model"
	"github.com/stretchr/testify/mock"
)

type RunRepository struct {
	mock.Mock
}

func NewRunRepository(t interface {
	mock.TestingT
	Cleanup(func())
}) *RunRepository {
	m := &RunRepository{}
	m.Mock.Test(t)
	t.Cleanup(func() { m.AssertExpectations(t) })
	return m
}

func (m *RunRepository) Create(ctx context.Context, run model.Run) error {
	args := m.Called(ctx, run)
	return args.Error(0)
}

func (m *RunRepository) Get(ctx context.Context, runID string) (model.Run, error) {
	args := m.Called(ctx, runID)
	if fn, ok := args.Get(0).(func(context.Context, string) (model.Run, error)); ok {
		return fn(ctx, runID)
	}
	return args.Get(0).(model.Run), args.Error(1)
}
