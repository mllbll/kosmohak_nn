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
