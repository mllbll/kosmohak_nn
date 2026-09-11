package mocks

import (
	"context"

	"github.com/mllbll/kosmohak_nn/internal/model"
	"github.com/stretchr/testify/mock"
)

type GeometryClient struct {
	mock.Mock
}

func NewGeometryClient(t interface {
	mock.TestingT
	Cleanup(func())
}) *GeometryClient {
	m := &GeometryClient{}
	m.Mock.Test(t)
	t.Cleanup(func() { m.AssertExpectations(t) })
	return m
}

func (m *GeometryClient) Snapshot(ctx context.Context, sc model.Scenario, t float64) (model.Snapshot, error) {
	args := m.Called(ctx, sc, t)
	return args.Get(0).(model.Snapshot), args.Error(1)
}

func (m *GeometryClient) WalkSnapshots(ctx context.Context, sc model.Scenario, fn func(model.Snapshot) error) error {
	args := m.Called(ctx, sc, fn)
	return args.Error(0)
}
