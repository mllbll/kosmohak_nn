package run

import (
	"github.com/brianvoe/gofakeit/v7"
	"github.com/mllbll/kosmohak_nn/internal/model"
	"github.com/stretchr/testify/mock"
)

func (s *ServiceSuite) TestCreateSuccess() {
	var (
		projectID = gofakeit.UUID()
		sc        = testScenario()
		snap      = testSnapshot()

		createRunRequest = model.CreateRunRequest{
			ProjectID: projectID,
		}

		project = model.Project{
			ID:        projectID,
			Base:      sc,
			Effective: sc,
		}
	)

	s.projectRepository.On("Get", s.ctx, projectID).Return(project, nil)
	s.geometryClient.On("WalkSnapshots", s.ctx, mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
		fn := args.Get(2).(func(model.Snapshot) error)
		_ = fn(snap)
	}).Return(nil)
	s.runRepository.On("Create", s.ctx, mock.MatchedBy(func(run model.Run) bool {
		return run.ID != "" && run.ProjectID == projectID && len(run.Routes) == 1 && run.Routes[0].AltCount == 0
	})).Return(nil)

	res, err := s.service.Create(s.ctx, createRunRequest)

	s.Require().NoError(err)
	s.Require().NotEmpty(res.RunID)
	s.Require().Equal(projectID, res.ProjectID)
	s.Require().NotEmpty(res.Metrics)
}

func (s *ServiceSuite) TestCreateInvalidArgument() {
	var (
		projectID = gofakeit.UUID()
		sc        = testScenario()

		createRunRequest = model.CreateRunRequest{
			ProjectID: projectID,
		}

		project = model.Project{
			ID:        projectID,
			Base:      sc,
			Effective: sc,
		}
	)
	project.Effective.GroundSites = []model.GroundSite{
		{ID: "G_MUR", Role: "gateway", LatDeg: 68.97, LonDeg: 33.07},
	}

	s.projectRepository.On("Get", s.ctx, projectID).Return(project, nil)
	s.geometryClient.AssertNotCalled(s.T(), "WalkSnapshots")
	s.runRepository.AssertNotCalled(s.T(), "Create")

	res, err := s.service.Create(s.ctx, createRunRequest)

	s.Require().Error(err)
	s.Require().ErrorIs(err, model.ErrInvalidArgument)
	s.Require().Empty(res)
}

func (s *ServiceSuite) TestCreateNotFoundError() {
	var (
		projectID = gofakeit.UUID()

		createRunRequest = model.CreateRunRequest{
			ProjectID: projectID,
		}
	)

	s.projectRepository.On("Get", s.ctx, projectID).Return(model.Project{}, model.ErrProjectNotFound)

	res, err := s.service.Create(s.ctx, createRunRequest)

	s.Require().Error(err)
	s.Require().Equal(err, model.ErrProjectNotFound)
	s.Require().Empty(res)
}

func (s *ServiceSuite) TestCreateGeometryError() {
	var (
		projectID = gofakeit.UUID()
		sc        = testScenario()
		geomErr   = gofakeit.Error()

		createRunRequest = model.CreateRunRequest{
			ProjectID: projectID,
		}

		project = model.Project{
			ID:        projectID,
			Base:      sc,
			Effective: sc,
		}
	)

	s.projectRepository.On("Get", s.ctx, projectID).Return(project, nil)
	s.geometryClient.On("WalkSnapshots", s.ctx, mock.Anything, mock.Anything).Return(geomErr)

	res, err := s.service.Create(s.ctx, createRunRequest)

	s.Require().Error(err)
	s.Require().Equal(err, geomErr)
	s.Require().Empty(res)
}

func (s *ServiceSuite) TestCreateRepositoryError() {
	var (
		projectID = gofakeit.UUID()
		sc        = testScenario()
		snap      = testSnapshot()
		repoErr   = gofakeit.Error()

		createRunRequest = model.CreateRunRequest{
			ProjectID: projectID,
		}

		project = model.Project{
			ID:        projectID,
			Base:      sc,
			Effective: sc,
		}
	)

	s.projectRepository.On("Get", s.ctx, projectID).Return(project, nil)
	s.geometryClient.On("WalkSnapshots", s.ctx, mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
		fn := args.Get(2).(func(model.Snapshot) error)
		_ = fn(snap)
	}).Return(nil)
	s.runRepository.On("Create", s.ctx, mock.Anything).Return(repoErr)

	res, err := s.service.Create(s.ctx, createRunRequest)

	s.Require().Error(err)
	s.Require().Equal(err, repoErr)
	s.Require().Empty(res)
}

func (s *ServiceSuite) TestCreateRejectsMissingSnapshots() {
	var (
		projectID = gofakeit.UUID()
		sc        = testScenario()

		createRunRequest = model.CreateRunRequest{
			ProjectID: projectID,
		}

		project = model.Project{
			ID:        projectID,
			Base:      sc,
			Effective: sc,
		}
	)

	s.projectRepository.On("Get", s.ctx, projectID).Return(project, nil)
	s.geometryClient.On("WalkSnapshots", s.ctx, mock.Anything, mock.Anything).Return(nil)
	s.runRepository.AssertNotCalled(s.T(), "Create")

	res, err := s.service.Create(s.ctx, createRunRequest)

	s.Require().Error(err)
	s.Require().ErrorIs(err, model.ErrGeometryFailed)
	s.Require().Empty(res)
}

func (s *ServiceSuite) TestCreateRejectsWrongSnapshotTime() {
	var (
		projectID = gofakeit.UUID()
		sc        = testScenario()

		createRunRequest = model.CreateRunRequest{
			ProjectID: projectID,
		}

		project = model.Project{
			ID:        projectID,
			Base:      sc,
			Effective: sc,
		}
	)

	s.projectRepository.On("Get", s.ctx, projectID).Return(project, nil)
	s.geometryClient.On("WalkSnapshots", s.ctx, mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
		fn := args.Get(2).(func(model.Snapshot) error)
		bad := testSnapshot()
		bad.TS = 99
		_ = fn(bad)
	}).Return(nil)
	s.runRepository.AssertNotCalled(s.T(), "Create")

	res, err := s.service.Create(s.ctx, createRunRequest)

	s.Require().Error(err)
	s.Require().ErrorIs(err, model.ErrGeometryFailed)
	s.Require().Contains(err.Error(), "expected")
	s.Require().Empty(res)
}
