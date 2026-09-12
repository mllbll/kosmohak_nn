package v1

import (
	"encoding/json"
	"net/http"

	"github.com/brianvoe/gofakeit/v7"
	"github.com/mllbll/kosmohak_nn/internal/model"
	"github.com/stretchr/testify/mock"
)

func (s *APISuite) TestGetSnapshotSuccess() {
	var (
		projectID = gofakeit.UUID()
		run       = testRun(projectID)
		clientID  = "C65"

		getSnapshotRequest = model.GetSnapshotRequest{
			RunID:    run.ID,
			TS:       0,
			ClientID: clientID,
		}

		getSnapshotResponse = model.GetSnapshotResponse{
			Snapshot: model.Snapshot{
				TS: 0,
				Satellites: []model.SatState{
					{ID: "S01", Active: true},
				},
			},
			Route: model.RouteResult{
				Path: []string{"C65", "S01", "G_MUR"},
				Hops: 2,
			},
		}
	)

	s.runService.On("GetSnapshot", mock.Anything, getSnapshotRequest).Return(getSnapshotResponse, nil)

	rec, req := s.newRequest(http.MethodGet, "/api/runs/"+run.ID+"/snapshot?t_s=0&client_id="+clientID, run.ID, nil)
	s.api.GetSnapshot(rec, req)

	s.Require().Equal(http.StatusOK, rec.Code)

	var got model.GetSnapshotResponse
	s.Require().NoError(json.NewDecoder(rec.Body).Decode(&got))
	s.Require().Equal(getSnapshotResponse, got)
}

func (s *APISuite) TestGetSnapshotNotFoundError() {
	var (
		runID = gofakeit.UUID()

		getSnapshotRequest = model.GetSnapshotRequest{
			RunID: runID,
		}
	)

	s.runService.On("GetSnapshot", mock.Anything, getSnapshotRequest).Return(model.GetSnapshotResponse{}, model.ErrRunNotFound)

	rec, req := s.newRequest(http.MethodGet, "/api/runs/"+runID+"/snapshot", runID, nil)
	s.api.GetSnapshot(rec, req)

	s.Require().Equal(http.StatusNotFound, rec.Code)
}

func (s *APISuite) TestGetSnapshotError() {
	var (
		runID      = gofakeit.UUID()
		serviceErr = gofakeit.Error()

		getSnapshotRequest = model.GetSnapshotRequest{
			RunID: runID,
		}
	)

	s.runService.On("GetSnapshot", mock.Anything, getSnapshotRequest).Return(model.GetSnapshotResponse{}, serviceErr)

	rec, req := s.newRequest(http.MethodGet, "/api/runs/"+runID+"/snapshot", runID, nil)
	s.api.GetSnapshot(rec, req)

	s.Require().Equal(http.StatusInternalServerError, rec.Code)
}

func (s *APISuite) TestGetSnapshotInvalidTS() {
	var (
		runID = gofakeit.UUID()
	)

	rec, req := s.newRequest(http.MethodGet, "/api/runs/"+runID+"/snapshot?t_s=abc", runID, nil)
	s.api.GetSnapshot(rec, req)

	s.Require().Equal(http.StatusBadRequest, rec.Code)
	s.runService.AssertNotCalled(s.T(), "GetSnapshot")
}
