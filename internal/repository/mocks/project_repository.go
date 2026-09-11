package mocks

import (
	"context"

	"github.com/mllbll/kosmohak_nn/internal/model"
	"github.com/stretchr/testify/mock"
)

type ProjectRepository struct {
	mock.Mock
}

func NewProjectRepository(t interface {
	mock.TestingT
	Cleanup(func())
}) *ProjectRepository {
	m := &ProjectRepository{}
	m.Mock.Test(t)
	t.Cleanup(func() { m.AssertExpectations(t) })
	return m
}

func (m *ProjectRepository) Create(ctx context.Context, project model.Project) error {
	args := m.Called(ctx, project)
	return args.Error(0)
}

func (m *ProjectRepository) Get(ctx context.Context, projectID string) (model.Project, error) {
	args := m.Called(ctx, projectID)
	return args.Get(0).(model.Project), args.Error(1)
}

func (m *ProjectRepository) Update(ctx context.Context, project model.Project) error {
	args := m.Called(ctx, project)
	return args.Error(0)
}
