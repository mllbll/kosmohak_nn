package v1

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"

	"github.com/brianvoe/gofakeit/v7"
	"github.com/mllbll/kosmohak_nn/internal/model"
	"github.com/stretchr/testify/mock"
)

func (s *APISuite) TestCompareSuccess() {
	var (
		compareRunsRequest = model.CompareRunsRequest{
			RunAID: gofakeit.UUID(),
			RunBID: gofakeit.UUID(),
		}

		compareRunsResponse = model.CompareRunsResponse{
			RunAID: compareRunsRequest.RunAID,
			RunBID: compareRunsRequest.RunBID,
			Config: map[string]any{
				"launch_stage": map[string]any{"a": float64(3), "b": float64(1)},
			},
			Clients: []model.ClientDiff{
				{ClientID: "C65", PathRatioA: 1, PathRatioB: 0.5, DeltaPathRatio: -0.5},
			},
		}
	)

	s.runService.On("Compare", mock.Anything, compareRunsRequest).Return(compareRunsResponse, nil)

	rec, req := s.newRequest(http.MethodPost, "/api/compare", "", compareRunsRequest)
	s.api.Compare(rec, req)

	s.Require().Equal(http.StatusOK, rec.Code)

	var got model.CompareRunsResponse
	s.Require().NoError(json.NewDecoder(rec.Body).Decode(&got))
	s.Require().Equal(compareRunsResponse, got)
}

func (s *APISuite) TestCompareNotFoundError() {
	var (
		compareRunsRequest = model.CompareRunsRequest{
			RunAID: gofakeit.UUID(),
			RunBID: gofakeit.UUID(),
		}
	)

	s.runService.On("Compare", mock.Anything, compareRunsRequest).Return(model.CompareRunsResponse{}, model.ErrRunNotFound)

	rec, req := s.newRequest(http.MethodPost, "/api/compare", "", compareRunsRequest)
	s.api.Compare(rec, req)

	s.Require().Equal(http.StatusNotFound, rec.Code)
}

func (s *APISuite) TestCompareInvalidArgument() {
	req := httptest.NewRequest(http.MethodPost, "/api/compare", bytes.NewReader([]byte("{")))
	req = req.WithContext(s.ctx)
	rec := httptest.NewRecorder()

	s.api.Compare(rec, req)

	s.Require().Equal(http.StatusBadRequest, rec.Code)
	s.runService.AssertNotCalled(s.T(), "Compare")
}

func (s *APISuite) TestCompareInvalidArgumentEmpty() {
	var (
		compareRunsRequest = model.CompareRunsRequest{}
	)

	s.runService.On("Compare", mock.Anything, compareRunsRequest).Return(model.CompareRunsResponse{}, model.ErrInvalidArgument)

	rec, req := s.newRequest(http.MethodPost, "/api/compare", "", compareRunsRequest)
	s.api.Compare(rec, req)

	s.Require().Equal(http.StatusBadRequest, rec.Code)
}

func (s *APISuite) TestCompareError() {
	var (
		serviceErr = gofakeit.Error()

		compareRunsRequest = model.CompareRunsRequest{
			RunAID: gofakeit.UUID(),
			RunBID: gofakeit.UUID(),
		}
	)

	s.runService.On("Compare", mock.Anything, compareRunsRequest).Return(model.CompareRunsResponse{}, serviceErr)

	rec, req := s.newRequest(http.MethodPost, "/api/compare", "", compareRunsRequest)
	s.api.Compare(rec, req)

	s.Require().Equal(http.StatusInternalServerError, rec.Code)
}
