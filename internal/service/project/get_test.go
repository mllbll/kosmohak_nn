package project

import (
	"github.com/brianvoe/gofakeit/v7"
	"github.com/mllbll/kosmohak_nn/internal/model"
)

func (s *ServiceSuite) TestGetSuccess() {
	var (
		projectID = gofakeit.UUID()
		sc        = testScenario()

		getProjectRequest = model.GetProjectRequest{
			ProjectID: projectID,
		}

		project = model.Project{
			ID:        projectID,
			Base:      sc,
			Effective: sc,
		}
	)

	s.projectRepository.On("Get", s.ctx, projectID).Return(project, nil)

	res, err := s.service.Get(s.ctx, getProjectRequest)

	s.Require().NoError(err)
	s.Require().Equal(project, res.Project)
}

func (s *ServiceSuite) TestGetError() {
	var (
		projectID = gofakeit.UUID()
		repoErr   = gofakeit.Error()

		getProjectRequest = model.GetProjectRequest{
			ProjectID: projectID,
		}
	)

	s.projectRepository.On("Get", s.ctx, projectID).Return(model.Project{}, repoErr)

	res, err := s.service.Get(s.ctx, getProjectRequest)

	s.Require().Error(err)
	s.Require().Equal(err, repoErr)
	s.Require().Empty(res)
}

func (s *ServiceSuite) TestGetNotFoundError() {
	var (
		projectID = gofakeit.UUID()

		getProjectRequest = model.GetProjectRequest{
			ProjectID: projectID,
		}
	)

	s.projectRepository.On("Get", s.ctx, projectID).Return(model.Project{}, model.ErrProjectNotFound)

	res, err := s.service.Get(s.ctx, getProjectRequest)

	s.Require().Error(err)
	s.Require().Equal(err, model.ErrProjectNotFound)
	s.Require().Empty(res)
}
