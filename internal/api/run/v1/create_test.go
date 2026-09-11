package v1

import (
	"encoding/json"
	"net/http"

	"github.com/brianvoe/gofakeit/v7"
	"github.com/mllbll/kosmohak_nn/internal/model"
	"github.com/stretchr/testify/mock"
)

func (s *APISuite) TestCreateSuccess() {
	var (
		projectID = gofakeit.UUID()

		createRunRequest = model.CreateRunRequest{
			ProjectID: projectID,
		}

		createRunResponse = model.CreateRunResponse{
			RunID:     gofakeit.UUID(),
			ProjectID: projectID,
			Metrics: []model.ClientMetrics{
				{ClientID: "C65", PathRatio: 1, VisibilityRatio: 1, MeetsTarget: true},
			},
		}
	)

	s.runService.On("Create", mock.Anything, createRunRequest).Return(createRunResponse, nil)

	rec, req := s.newRequest(http.MethodPost, "/api/projects/"+projectID+"/runs", projectID, nil)
	s.api.Create(rec, req)

	s.Require().Equal(http.StatusCreated, rec.Code)

	var got model.CreateRunResponse
	s.Require().NoError(json.NewDecoder(rec.Body).Decode(&got))
	s.Require().Equal(createRunResponse, got)
}

func (s *APISuite) TestCreateNotFoundError() {
	var (
		projectID = gofakeit.UUID()

		createRunRequest = model.CreateRunRequest{
			ProjectID: projectID,
		}
	)

	s.runService.On("Create", mock.Anything, createRunRequest).Return(model.CreateRunResponse{}, model.ErrProjectNotFound)

	rec, req := s.newRequest(http.MethodPost, "/api/projects/"+projectID+"/runs", projectID, nil)
	s.api.Create(rec, req)

	s.Require().Equal(http.StatusNotFound, rec.Code)
}

func (s *APISuite) TestCreateError() {
	var (
		projectID  = gofakeit.UUID()
		serviceErr = gofakeit.Error()

		createRunRequest = model.CreateRunRequest{
			ProjectID: projectID,
		}
	)

	s.runService.On("Create", mock.Anything, createRunRequest).Return(model.CreateRunResponse{}, serviceErr)

	rec, req := s.newRequest(http.MethodPost, "/api/projects/"+projectID+"/runs", projectID, nil)
	s.api.Create(rec, req)

	s.Require().Equal(http.StatusInternalServerError, rec.Code)
}
