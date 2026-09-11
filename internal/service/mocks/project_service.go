package mocks

import (
	"context"

	"github.com/mllbll/kosmohak_nn/internal/model"
	"github.com/stretchr/testify/mock"
)

type ProjectService struct {
	mock.Mock
}

func NewProjectService(t interface {
	mock.TestingT
	Cleanup(func())
}) *ProjectService {
	m := &ProjectService{}
	m.Mock.Test(t)
	t.Cleanup(func() { m.AssertExpectations(t) })
	return m
}

func (m *ProjectService) Create(ctx context.Context, req model.CreateProjectRequest) (model.CreateProjectResponse, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(model.CreateProjectResponse), args.Error(1)
}

func (m *ProjectService) Get(ctx context.Context, req model.GetProjectRequest) (model.GetProjectResponse, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(model.GetProjectResponse), args.Error(1)
}

func (m *ProjectService) Patch(ctx context.Context, req model.PatchProjectRequest) (model.PatchProjectResponse, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(model.PatchProjectResponse), args.Error(1)
}

func (m *ProjectService) Reset(ctx context.Context, req model.ResetProjectRequest) (model.ResetProjectResponse, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(model.ResetProjectResponse), args.Error(1)
}

func (m *ProjectService) Copy(ctx context.Context, req model.CopyProjectRequest) (model.CopyProjectResponse, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(model.CopyProjectResponse), args.Error(1)
}
