package project

import (
	"github.com/brianvoe/gofakeit/v7"
	"github.com/mllbll/kosmohak_nn/internal/model"
	"github.com/stretchr/testify/mock"
)

func (s *ServiceSuite) TestCopySuccess() {
	var (
		projectID = gofakeit.UUID()
		sc        = testScenario()

		copyProjectRequest = model.CopyProjectRequest{
			ProjectID: projectID,
		}

		project = model.Project{
			ID:        projectID,
			Base:      sc,
			Effective: sc,
		}
	)

	s.projectRepository.On("Get", s.ctx, projectID).Return(project, nil)
	s.projectRepository.On("Create", s.ctx, mock.MatchedBy(func(p model.Project) bool {
		return p.ID != "" && p.ID != projectID && p.Effective.Meta.Title == sc.Meta.Title
	})).Return(nil)

	res, err := s.service.Copy(s.ctx, copyProjectRequest)

	s.Require().NoError(err)
	s.Require().NotEqual(projectID, res.Project.ID)
	s.Require().Equal(sc.Meta.Title, res.Project.Effective.Meta.Title)
}

func (s *ServiceSuite) TestCopyNotFoundError() {
	var (
		projectID = gofakeit.UUID()

		copyProjectRequest = model.CopyProjectRequest{
			ProjectID: projectID,
		}
	)

	s.projectRepository.On("Get", s.ctx, projectID).Return(model.Project{}, model.ErrProjectNotFound)

	res, err := s.service.Copy(s.ctx, copyProjectRequest)

	s.Require().Error(err)
	s.Require().Equal(err, model.ErrProjectNotFound)
	s.Require().Empty(res)
}

func (s *ServiceSuite) TestCopyRepositoryError() {
	var (
		projectID = gofakeit.UUID()
		sc        = testScenario()
		repoErr   = gofakeit.Error()

		copyProjectRequest = model.CopyProjectRequest{
			ProjectID: projectID,
		}

		project = model.Project{
			ID:        projectID,
			Base:      sc,
			Effective: sc,
		}
	)

	s.projectRepository.On("Get", s.ctx, projectID).Return(project, nil)
	s.projectRepository.On("Create", s.ctx, mock.Anything).Return(repoErr)

	res, err := s.service.Copy(s.ctx, copyProjectRequest)

	s.Require().Error(err)
	s.Require().Equal(err, repoErr)
	s.Require().Empty(res)
}
