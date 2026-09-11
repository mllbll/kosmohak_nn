package mocks

import (
	"context"

	"github.com/mllbll/kosmohak_nn/internal/model"
	"github.com/stretchr/testify/mock"
)

type RunService struct {
	mock.Mock
}

func NewRunService(t interface {
	mock.TestingT
	Cleanup(func())
}) *RunService {
	m := &RunService{}
	m.Mock.Test(t)
	t.Cleanup(func() { m.AssertExpectations(t) })
	return m
}

func (m *RunService) Create(ctx context.Context, req model.CreateRunRequest) (model.CreateRunResponse, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(model.CreateRunResponse), args.Error(1)
}

func (m *RunService) Get(ctx context.Context, req model.GetRunRequest) (model.GetRunResponse, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(model.GetRunResponse), args.Error(1)
}

func (m *RunService) GetSnapshot(ctx context.Context, req model.GetSnapshotRequest) (model.GetSnapshotResponse, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(model.GetSnapshotResponse), args.Error(1)
}

func (m *RunService) Export(ctx context.Context, req model.ExportRunRequest) (model.ExportDocument, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(model.ExportDocument), args.Error(1)
}

func (m *RunService) Compare(ctx context.Context, req model.CompareRunsRequest) (model.CompareRunsResponse, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(model.CompareRunsResponse), args.Error(1)
}

func (m *RunService) WhatIf(ctx context.Context, req model.WhatIfRequest) (model.WhatIfResponse, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(model.WhatIfResponse), args.Error(1)
}
