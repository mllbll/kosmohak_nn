package project

import (
	"github.com/brianvoe/gofakeit/v7"
	"github.com/mllbll/kosmohak_nn/internal/model"
	"github.com/stretchr/testify/mock"
)

func (s *ServiceSuite) TestResetSuccess() {
	var (
		projectID     = gofakeit.UUID()
		base          = testScenario()
		effective     = testScenario()
		launchStage   = 1
	)
	effective.Design.LaunchStage = launchStage

	var (
		resetProjectRequest = model.ResetProjectRequest{
			ProjectID: projectID,
		}

		project = model.Project{
			ID:        projectID,
			Base:      base,
			Effective: effective,
		}
	)

	s.projectRepository.On("Get", s.ctx, projectID).Return(project, nil)
	s.projectRepository.On("Update", s.ctx, mock.MatchedBy(func(p model.Project) bool {
		return p.ID == projectID && p.Effective.Design.LaunchStage == base.Design.LaunchStage
	})).Return(nil)

	res, err := s.service.Reset(s.ctx, resetProjectRequest)

	s.Require().NoError(err)
	s.Require().Equal(base.Design.LaunchStage, res.Project.Effective.Design.LaunchStage)
}

func (s *ServiceSuite) TestResetNotFoundError() {
	var (
		projectID = gofakeit.UUID()

		resetProjectRequest = model.ResetProjectRequest{
			ProjectID: projectID,
		}
	)

	s.projectRepository.On("Get", s.ctx, projectID).Return(model.Project{}, model.ErrProjectNotFound)

	res, err := s.service.Reset(s.ctx, resetProjectRequest)

	s.Require().Error(err)
	s.Require().Equal(err, model.ErrProjectNotFound)
	s.Require().Empty(res)
}

func (s *ServiceSuite) TestResetRepositoryError() {
	var (
		projectID = gofakeit.UUID()
		sc        = testScenario()
		repoErr   = gofakeit.Error()

		resetProjectRequest = model.ResetProjectRequest{
			ProjectID: projectID,
		}

		project = model.Project{
			ID:        projectID,
			Base:      sc,
			Effective: sc,
		}
	)

	s.projectRepository.On("Get", s.ctx, projectID).Return(project, nil)
	s.projectRepository.On("Update", s.ctx, mock.Anything).Return(repoErr)

	res, err := s.service.Reset(s.ctx, resetProjectRequest)

	s.Require().Error(err)
	s.Require().Equal(err, repoErr)
	s.Require().Empty(res)
}
