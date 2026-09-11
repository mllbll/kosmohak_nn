package v1

import (
	"encoding/json"
	"net/http"

	"github.com/brianvoe/gofakeit/v7"
	"github.com/mllbll/kosmohak_nn/internal/model"
	"github.com/stretchr/testify/mock"
)

func (s *APISuite) TestResetSuccess() {
	var (
		projectID = gofakeit.UUID()
		sc        = testScenario()

		resetProjectRequest = model.ResetProjectRequest{
			ProjectID: projectID,
		}

		project = model.Project{
			ID:        projectID,
			Base:      sc,
			Effective: sc,
		}

		resetProjectResponse = model.ResetProjectResponse{
			Project: project,
		}
	)

	s.projectService.On("Reset", mock.Anything, resetProjectRequest).Return(resetProjectResponse, nil)

	rec, req := s.newRequest(http.MethodPost, "/api/projects/"+projectID+"/reset", projectID, nil)
	s.api.Reset(rec, req)

	s.Require().Equal(http.StatusOK, rec.Code)

	var got model.Project
	s.Require().NoError(json.NewDecoder(rec.Body).Decode(&got))
	s.Require().Equal(project, got)
}

func (s *APISuite) TestResetNotFoundError() {
	var (
		projectID = gofakeit.UUID()

		resetProjectRequest = model.ResetProjectRequest{
			ProjectID: projectID,
		}
	)

	s.projectService.On("Reset", mock.Anything, resetProjectRequest).Return(model.ResetProjectResponse{}, model.ErrProjectNotFound)

	rec, req := s.newRequest(http.MethodPost, "/api/projects/"+projectID+"/reset", projectID, nil)
	s.api.Reset(rec, req)

	s.Require().Equal(http.StatusNotFound, rec.Code)
}

func (s *APISuite) TestResetError() {
	var (
		projectID  = gofakeit.UUID()
		serviceErr = gofakeit.Error()

		resetProjectRequest = model.ResetProjectRequest{
			ProjectID: projectID,
		}
	)

	s.projectService.On("Reset", mock.Anything, resetProjectRequest).Return(model.ResetProjectResponse{}, serviceErr)

	rec, req := s.newRequest(http.MethodPost, "/api/projects/"+projectID+"/reset", projectID, nil)
	s.api.Reset(rec, req)

	s.Require().Equal(http.StatusInternalServerError, rec.Code)
}
