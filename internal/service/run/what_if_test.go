package run

import (
	"context"

	"github.com/brianvoe/gofakeit/v7"
	"github.com/mllbll/kosmohak_nn/internal/model"
	"github.com/stretchr/testify/mock"
)

func (s *ServiceSuite) TestWhatIfSuccess() {
	var (
		projectID  = gofakeit.UUID()
		original   = testRun(projectID)
		snap       = testSnapshot()
		createdRun model.Run

		whatIfRequest = model.WhatIfRequest{
			RunID:       original.ID,
			SatelliteID: "S01",
			TS:          0,
		}
	)

	s.runRepository.On("Get", s.ctx, mock.Anything).Return(
		func(_ context.Context, id string) (model.Run, error) {
			if id == original.ID {
				return original, nil
			}
			return createdRun, nil
		},
	)
	s.geometryClient.On("Snapshot", s.ctx, mock.Anything, float64(0)).Return(model.Snapshot{}, nil)
	s.projectRepository.On("Create", s.ctx, mock.MatchedBy(func(project model.Project) bool {
		return project.ID != "" &&
			project.ID != projectID &&
			len(project.Effective.Failures) == 1 &&
			project.Effective.Failures[0].SatelliteID == "S01"
	})).Return(nil)
	s.geometryClient.On("WalkSnapshots", s.ctx, mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
		fn := args.Get(2).(func(model.Snapshot) error)
		_ = fn(snap)
	}).Return(nil)
	s.runRepository.On("Create", s.ctx, mock.Anything).Run(func(args mock.Arguments) {
		createdRun = args.Get(1).(model.Run)
	}).Return(nil)

	res, err := s.service.WhatIf(s.ctx, whatIfRequest)

	s.Require().NoError(err)
	s.Require().Equal(original.ID, res.OriginalRunID)
	s.Require().NotEmpty(res.ProjectID)
	s.Require().NotEmpty(res.RunID)
	s.Require().Equal("S01", res.FailedSatelliteID)
	s.Require().Equal(original.ID, res.Compare.RunAID)
	s.Require().Equal(res.RunID, res.Compare.RunBID)
}

func (s *ServiceSuite) TestWhatIfFromPathSuccess() {
	var (
		projectID  = gofakeit.UUID()
		original   = testRun(projectID)
		snap       = testSnapshot()
		createdRun model.Run

		whatIfRequest = model.WhatIfRequest{
			RunID:    original.ID,
			ClientID: "C65",
			TS:       0,
		}
	)

	s.runRepository.On("Get", s.ctx, mock.Anything).Return(
		func(_ context.Context, id string) (model.Run, error) {
			if id == original.ID {
				return original, nil
			}
			return createdRun, nil
		},
	)
	s.geometryClient.On("Snapshot", s.ctx, mock.Anything, float64(0)).Return(model.Snapshot{}, nil)
	s.projectRepository.On("Create", s.ctx, mock.Anything).Return(nil)
	s.geometryClient.On("WalkSnapshots", s.ctx, mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
		fn := args.Get(2).(func(model.Snapshot) error)
		_ = fn(snap)
	}).Return(nil)
	s.runRepository.On("Create", s.ctx, mock.Anything).Run(func(args mock.Arguments) {
		createdRun = args.Get(1).(model.Run)
	}).Return(nil)

	res, err := s.service.WhatIf(s.ctx, whatIfRequest)

	s.Require().NoError(err)
	s.Require().Equal("S01", res.FailedSatelliteID)
}

func (s *ServiceSuite) TestWhatIfNotFoundError() {
	var (
		runID = gofakeit.UUID()

		whatIfRequest = model.WhatIfRequest{
			RunID:       runID,
			SatelliteID: "S01",
		}
	)

	s.runRepository.On("Get", s.ctx, runID).Return(model.Run{}, model.ErrRunNotFound)

	res, err := s.service.WhatIf(s.ctx, whatIfRequest)

	s.Require().Error(err)
	s.Require().Equal(err, model.ErrRunNotFound)
	s.Require().Empty(res)
}

func (s *ServiceSuite) TestWhatIfInvalidArgument() {
	var (
		projectID = gofakeit.UUID()
		original  = testRun(projectID)

		whatIfRequest = model.WhatIfRequest{
			RunID:       original.ID,
			SatelliteID: "UNKNOWN",
		}
	)

	s.runRepository.On("Get", s.ctx, original.ID).Return(original, nil)
	s.geometryClient.AssertNotCalled(s.T(), "Snapshot")
	s.projectRepository.AssertNotCalled(s.T(), "Create")

	res, err := s.service.WhatIf(s.ctx, whatIfRequest)

	s.Require().Error(err)
	s.Require().ErrorIs(err, model.ErrInvalidArgument)
	s.Require().Empty(res)
}

func (s *ServiceSuite) TestWhatIfInvalidInterval() {
	var (
		projectID = gofakeit.UUID()
		original  = testRun(projectID)
		endS      = 0.0

		whatIfRequest = model.WhatIfRequest{
			RunID:       original.ID,
			SatelliteID: "S01",
			TS:          0,
			EndS:        &endS,
		}
	)

	s.runRepository.On("Get", s.ctx, original.ID).Return(original, nil)
	s.geometryClient.AssertNotCalled(s.T(), "Snapshot")
	s.projectRepository.AssertNotCalled(s.T(), "Create")

	res, err := s.service.WhatIf(s.ctx, whatIfRequest)

	s.Require().Error(err)
	s.Require().ErrorIs(err, model.ErrInvalidArgument)
	s.Require().Empty(res)
}

func (s *ServiceSuite) TestWhatIfGeometryError() {
	var (
		projectID = gofakeit.UUID()
		original  = testRun(projectID)
		geomErr   = gofakeit.Error()

		whatIfRequest = model.WhatIfRequest{
			RunID:       original.ID,
			SatelliteID: "S01",
		}
	)

	s.runRepository.On("Get", s.ctx, original.ID).Return(original, nil)
	s.geometryClient.On("Snapshot", s.ctx, mock.Anything, float64(0)).Return(model.Snapshot{}, geomErr)
	s.projectRepository.AssertNotCalled(s.T(), "Create")

	res, err := s.service.WhatIf(s.ctx, whatIfRequest)

	s.Require().Error(err)
	s.Require().Equal(err, geomErr)
	s.Require().Empty(res)
}

func (s *ServiceSuite) TestWhatIfProjectRepositoryError() {
	var (
		projectID = gofakeit.UUID()
		original  = testRun(projectID)
		repoErr   = gofakeit.Error()

		whatIfRequest = model.WhatIfRequest{
			RunID:       original.ID,
			SatelliteID: "S01",
		}
	)

	s.runRepository.On("Get", s.ctx, original.ID).Return(original, nil)
	s.geometryClient.On("Snapshot", s.ctx, mock.Anything, float64(0)).Return(model.Snapshot{}, nil)
	s.projectRepository.On("Create", s.ctx, mock.Anything).Return(repoErr)

	res, err := s.service.WhatIf(s.ctx, whatIfRequest)

	s.Require().Error(err)
	s.Require().Equal(err, repoErr)
	s.Require().Empty(res)
}

func (s *ServiceSuite) TestSatelliteFromPath() {
	sc := testScenario()

	got := satelliteFromPath(sc, []string{"C65", "S01", "G_MUR"})
	s.Require().Equal("S01", got)

	got = satelliteFromPath(sc, []string{"C65", "G_MUR"})
	s.Require().Empty(got)
}

func (s *ServiceSuite) TestResolveFailedSatelliteFromRequest() {
	var (
		run = testRun(gofakeit.UUID())
	)
	run.Routes = []model.RouteRecord{
		{TS: 120, ClientID: "C65", Path: []string{"C65", "S01", "G_MUR"}},
	}

	id, err := resolveFailedSatellite(run, model.WhatIfRequest{TS: 120, ClientID: "C65"})

	s.Require().NoError(err)
	s.Require().Equal("S01", id)
}
