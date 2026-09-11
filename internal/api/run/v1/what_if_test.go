package v1

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"

	"github.com/brianvoe/gofakeit/v7"
	"github.com/go-chi/chi/v5"
	"github.com/mllbll/kosmohak_nn/internal/model"
	"github.com/stretchr/testify/mock"
)

func (s *APISuite) TestWhatIfSuccess() {
	var (
		runID = gofakeit.UUID()

		whatIfRequest = model.WhatIfRequest{
			RunID:       runID,
			SatelliteID: "S01",
			TS:          0,
			ClientID:    "C65",
		}

		whatIfResponse = model.WhatIfResponse{
			OriginalRunID:     runID,
			ProjectID:         gofakeit.UUID(),
			RunID:             gofakeit.UUID(),
			FailedSatelliteID: "S01",
			Metrics: []model.ClientMetrics{
				{ClientID: "C65", PathRatio: 1, VisibilityRatio: 1, MeetsTarget: true},
			},
		}
	)

	s.runService.On("WhatIf", mock.Anything, whatIfRequest).Return(whatIfResponse, nil)

	rec, req := s.newRequest(http.MethodPost, "/api/runs/"+runID+"/what-if", runID, model.WhatIfRequest{
		SatelliteID: "S01",
		TS:          0,
		ClientID:    "C65",
	})
	s.api.WhatIf(rec, req)

	s.Require().Equal(http.StatusCreated, rec.Code)

	var got model.WhatIfResponse
	s.Require().NoError(json.NewDecoder(rec.Body).Decode(&got))
	s.Require().Equal(whatIfResponse, got)
}

func (s *APISuite) TestWhatIfNotFoundError() {
	var (
		runID = gofakeit.UUID()

		whatIfRequest = model.WhatIfRequest{
			RunID: runID,
		}
	)

	s.runService.On("WhatIf", mock.Anything, whatIfRequest).Return(model.WhatIfResponse{}, model.ErrRunNotFound)

	rec, req := s.newRequest(http.MethodPost, "/api/runs/"+runID+"/what-if", runID, model.WhatIfRequest{})
	s.api.WhatIf(rec, req)

	s.Require().Equal(http.StatusNotFound, rec.Code)
}

func (s *APISuite) TestWhatIfInvalidArgument() {
	var (
		runID = gofakeit.UUID()
		rctx  = chi.NewRouteContext()
	)
	rctx.URLParams.Add("id", runID)

	req := httptest.NewRequest(http.MethodPost, "/api/runs/"+runID+"/what-if", bytes.NewReader([]byte("{")))
	req = req.WithContext(context.WithValue(s.ctx, chi.RouteCtxKey, rctx))
	rec := httptest.NewRecorder()

	s.api.WhatIf(rec, req)

	s.Require().Equal(http.StatusBadRequest, rec.Code)
	s.runService.AssertNotCalled(s.T(), "WhatIf")
}

func (s *APISuite) TestWhatIfError() {
	var (
		runID      = gofakeit.UUID()
		serviceErr = gofakeit.Error()

		whatIfRequest = model.WhatIfRequest{
			RunID: runID,
		}
	)

	s.runService.On("WhatIf", mock.Anything, whatIfRequest).Return(model.WhatIfResponse{}, serviceErr)

	rec, req := s.newRequest(http.MethodPost, "/api/runs/"+runID+"/what-if", runID, model.WhatIfRequest{})
	s.api.WhatIf(rec, req)

	s.Require().Equal(http.StatusInternalServerError, rec.Code)
}
