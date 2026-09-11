package v1

import (
	"encoding/json"
	"net/http"

	"github.com/brianvoe/gofakeit/v7"
	"github.com/mllbll/kosmohak_nn/internal/model"
	"github.com/stretchr/testify/mock"
)

func (s *APISuite) TestGetSuccess() {
	var (
		projectID = gofakeit.UUID()
		run       = testRun(projectID)

		getRunRequest = model.GetRunRequest{
			RunID: run.ID,
		}

		getRunResponse = model.GetRunResponse{
			Run: run,
		}
	)

	s.runService.On("Get", mock.Anything, getRunRequest).Return(getRunResponse, nil)

	rec, req := s.newRequest(http.MethodGet, "/api/runs/"+run.ID, run.ID, nil)
	s.api.Get(rec, req)

	s.Require().Equal(http.StatusOK, rec.Code)

	var got model.Run
	s.Require().NoError(json.NewDecoder(rec.Body).Decode(&got))
	s.Require().Equal(run, got)
}

func (s *APISuite) TestGetNotFoundError() {
	var (
		runID = gofakeit.UUID()

		getRunRequest = model.GetRunRequest{
			RunID: runID,
		}
	)

	s.runService.On("Get", mock.Anything, getRunRequest).Return(model.GetRunResponse{}, model.ErrRunNotFound)

	rec, req := s.newRequest(http.MethodGet, "/api/runs/"+runID, runID, nil)
	s.api.Get(rec, req)

	s.Require().Equal(http.StatusNotFound, rec.Code)
}

func (s *APISuite) TestGetError() {
	var (
		runID      = gofakeit.UUID()
		serviceErr = gofakeit.Error()

		getRunRequest = model.GetRunRequest{
			RunID: runID,
		}
	)

	s.runService.On("Get", mock.Anything, getRunRequest).Return(model.GetRunResponse{}, serviceErr)

	rec, req := s.newRequest(http.MethodGet, "/api/runs/"+runID, runID, nil)
	s.api.Get(rec, req)

	s.Require().Equal(http.StatusInternalServerError, rec.Code)
}
