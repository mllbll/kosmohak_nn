package run

import (
	"github.com/brianvoe/gofakeit/v7"
	"github.com/mllbll/kosmohak_nn/internal/model"
	"github.com/stretchr/testify/mock"
)

func (s *ServiceSuite) TestGetSnapshotSuccess() {
	var (
		projectID = gofakeit.UUID()
		run       = testRun(projectID)
		snap      = testSnapshot()

		getSnapshotRequest = model.GetSnapshotRequest{
			RunID:    run.ID,
			TS:       0,
			ClientID: "C65",
		}
	)

	s.runRepository.On("Get", s.ctx, run.ID).Return(run, nil)
	s.geometryClient.On("Snapshot", s.ctx, mock.Anything, float64(0)).Return(snap, nil)

	res, err := s.service.GetSnapshot(s.ctx, getSnapshotRequest)

	s.Require().NoError(err)
	s.Require().Equal(snap, res.Snapshot)
	s.Require().Equal([]string{"C65", "S01", "G_MUR"}, res.Route.Path)
	s.Require().Equal([]string{"S01"}, res.VisibleSatellites)
	s.Require().Equal("bfs_min_hops", res.Route.Algorithm.Name)
	s.Require().NotEmpty(res.Route.Algorithm.Rationale)
	s.Require().NotEmpty(res.NetworkDelta.Explanation)
}

func (s *ServiceSuite) TestGetSnapshotRebuildsBrokenPath() {
	var (
		projectID = gofakeit.UUID()
		run       = testRun(projectID)
		snap      = model.Snapshot{
			TS: 120,
			Satellites: []model.SatState{
				{ID: "S01", Active: true},
				{ID: "S02", Active: true},
			},
			Edges: []model.Edge{
				{A: "C65", B: "S02", DistanceKM: 100},
				{A: "S02", B: "G_MUR", DistanceKM: 100},
			},
		}

		getSnapshotRequest = model.GetSnapshotRequest{
			RunID:    run.ID,
			TS:       120,
			ClientID: "C65",
		}
	)

	s.runRepository.On("Get", s.ctx, run.ID).Return(run, nil)
	s.geometryClient.On("Snapshot", s.ctx, mock.Anything, float64(120)).Return(snap, nil)

	res, err := s.service.GetSnapshot(s.ctx, getSnapshotRequest)

	s.Require().NoError(err)
	s.Require().Equal([]string{"C65", "S02", "G_MUR"}, res.Route.Path)
	s.Require().Equal([]string{"C65", "S01", "G_MUR"}, res.NetworkDelta.PreviousPath)
	s.Require().True(res.NetworkDelta.Changed)
	s.Require().False(res.NetworkDelta.PreviousStillValid)
	s.Require().Contains(res.NetworkDelta.Explanation, "перестроен")
}

func (s *ServiceSuite) TestGetSnapshotOffGridUsesPreviousStep() {
	var (
		projectID = gofakeit.UUID()
		run       = testRun(projectID)
		snap      = testSnapshot()

		getSnapshotRequest = model.GetSnapshotRequest{
			RunID:    run.ID,
			TS:       60,
			ClientID: "C65",
		}
	)
	snap.TS = 60

	s.runRepository.On("Get", s.ctx, run.ID).Return(run, nil)
	s.geometryClient.On("Snapshot", s.ctx, mock.Anything, float64(60)).Return(snap, nil)

	res, err := s.service.GetSnapshot(s.ctx, getSnapshotRequest)

	s.Require().NoError(err)
	s.Require().Equal([]string{"C65", "S01", "G_MUR"}, res.NetworkDelta.PreviousPath)
	s.Require().True(res.NetworkDelta.PreviousStillValid)
	s.Require().Equal([]string{"S01"}, res.VisibleSatellites)
}

func (s *ServiceSuite) TestGetSnapshotNotFoundError() {
	var (
		runID = gofakeit.UUID()

		getSnapshotRequest = model.GetSnapshotRequest{
			RunID: runID,
		}
	)

	s.runRepository.On("Get", s.ctx, runID).Return(model.Run{}, model.ErrRunNotFound)

	res, err := s.service.GetSnapshot(s.ctx, getSnapshotRequest)

	s.Require().Error(err)
	s.Require().Equal(err, model.ErrRunNotFound)
	s.Require().Empty(res)
}

func (s *ServiceSuite) TestGetSnapshotGeometryError() {
	var (
		projectID = gofakeit.UUID()
		run       = testRun(projectID)
		geomErr   = gofakeit.Error()

		getSnapshotRequest = model.GetSnapshotRequest{
			RunID:    run.ID,
			ClientID: "C65",
		}
	)

	s.runRepository.On("Get", s.ctx, run.ID).Return(run, nil)
	s.geometryClient.On("Snapshot", s.ctx, mock.Anything, float64(0)).Return(model.Snapshot{}, geomErr)

	res, err := s.service.GetSnapshot(s.ctx, getSnapshotRequest)

	s.Require().Error(err)
	s.Require().Equal(err, geomErr)
	s.Require().Empty(res)
}

func (s *ServiceSuite) TestGetSnapshotUnknownClient() {
	var (
		projectID = gofakeit.UUID()
		run       = testRun(projectID)

		getSnapshotRequest = model.GetSnapshotRequest{
			RunID:    run.ID,
			ClientID: "UNKNOWN",
		}
	)

	s.runRepository.On("Get", s.ctx, run.ID).Return(run, nil)
	s.geometryClient.AssertNotCalled(s.T(), "Snapshot")

	res, err := s.service.GetSnapshot(s.ctx, getSnapshotRequest)

	s.Require().Error(err)
	s.Require().ErrorIs(err, model.ErrInvalidArgument)
	s.Require().Empty(res)
}

func (s *ServiceSuite) TestGetSnapshotInvalidTime() {
	var (
		projectID = gofakeit.UUID()
		run       = testRun(projectID)

		getSnapshotRequest = model.GetSnapshotRequest{
			RunID:    run.ID,
			TS:       99999,
			ClientID: "C65",
		}
	)

	s.runRepository.On("Get", s.ctx, run.ID).Return(run, nil)
	s.geometryClient.AssertNotCalled(s.T(), "Snapshot")

	res, err := s.service.GetSnapshot(s.ctx, getSnapshotRequest)

	s.Require().Error(err)
	s.Require().ErrorIs(err, model.ErrInvalidArgument)
	s.Require().Empty(res)
}

func (s *ServiceSuite) TestGetSnapshotNegativeTime() {
	var (
		projectID = gofakeit.UUID()
		run       = testRun(projectID)

		getSnapshotRequest = model.GetSnapshotRequest{
			RunID:    run.ID,
			TS:       -1,
			ClientID: "C65",
		}
	)

	s.runRepository.On("Get", s.ctx, run.ID).Return(run, nil)
	s.geometryClient.AssertNotCalled(s.T(), "Snapshot")

	res, err := s.service.GetSnapshot(s.ctx, getSnapshotRequest)

	s.Require().Error(err)
	s.Require().ErrorIs(err, model.ErrInvalidArgument)
	s.Require().Empty(res)
}
