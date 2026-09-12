package project

import (
	"github.com/brianvoe/gofakeit/v7"
	"github.com/mllbll/kosmohak_nn/internal/model"
	"github.com/stretchr/testify/mock"
)

func (s *ServiceSuite) TestCreateSuccess() {
	var (
		sc = testScenario()

		createProjectRequest = model.CreateProjectRequest{
			Scenario: sc,
		}
	)

	s.geometryClient.On("Snapshot", s.ctx, sc, float64(0)).Return(model.Snapshot{}, nil)
	s.projectRepository.On("Create", s.ctx, mock.MatchedBy(func(project model.Project) bool {
		return project.ID != "" &&
			project.Effective.Meta.Title == sc.Meta.Title &&
			project.Base.Design.LaunchStage == sc.Design.LaunchStage
	})).Return(nil)

	res, err := s.service.Create(s.ctx, createProjectRequest)

	s.Require().NoError(err)
	s.Require().NotEmpty(res.Project.ID)
	s.Require().NotEqual(sc.Meta.ID, res.Project.ID)
	s.Require().Equal(sc.Meta.Title, res.Project.Effective.Meta.Title)
}

func (s *ServiceSuite) TestCreateInvalidArgument() {
	var (
		createProjectRequest = model.CreateProjectRequest{
			Scenario: model.Scenario{},
		}
	)

	s.geometryClient.AssertNotCalled(s.T(), "Snapshot")
	s.projectRepository.AssertNotCalled(s.T(), "Create")

	res, err := s.service.Create(s.ctx, createProjectRequest)

	s.Require().Error(err)
	s.Require().ErrorIs(err, model.ErrInvalidArgument)
	s.Require().Empty(res)
}

func (s *ServiceSuite) TestCreateRejectsNamedField() {
	sc := testScenario()
	sc.Environment.ISLRangeKM = 0

	res, err := s.service.Create(s.ctx, model.CreateProjectRequest{Scenario: sc})

	s.Require().Error(err)
	s.Require().ErrorIs(err, model.ErrInvalidArgument)
	s.Require().Equal("invalid argument: environment.isl_range_km must be in (0, 10000]", err.Error())
	s.Require().Empty(res)
}

func (s *ServiceSuite) TestCreateGeometryError() {
	var (
		sc      = testScenario()
		geomErr = gofakeit.Error()

		createProjectRequest = model.CreateProjectRequest{
			Scenario: sc,
		}
	)

	s.geometryClient.On("Snapshot", s.ctx, sc, float64(0)).Return(model.Snapshot{}, geomErr)
	s.projectRepository.AssertNotCalled(s.T(), "Create")

	res, err := s.service.Create(s.ctx, createProjectRequest)

	s.Require().Error(err)
	s.Require().Equal(err, geomErr)
	s.Require().Empty(res)
}

func (s *ServiceSuite) TestCreateRepositoryError() {
	var (
		sc      = testScenario()
		repoErr = gofakeit.Error()

		createProjectRequest = model.CreateProjectRequest{
			Scenario: sc,
		}
	)

	s.geometryClient.On("Snapshot", s.ctx, sc, float64(0)).Return(model.Snapshot{}, nil)
	s.projectRepository.On("Create", s.ctx, mock.Anything).Return(repoErr)

	res, err := s.service.Create(s.ctx, createProjectRequest)

	s.Require().Error(err)
	s.Require().Equal(err, repoErr)
	s.Require().Empty(res)
}
