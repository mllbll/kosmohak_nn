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
		sc        = testScenario()

		getProjectRequest = model.GetProjectRequest{
			ProjectID: projectID,
		}

		project = model.Project{
			ID:        projectID,
			Base:      sc,
			Effective: sc,
		}

		getProjectResponse = model.GetProjectResponse{
			Project: project,
		}
	)

	s.projectService.On("Get", mock.Anything, getProjectRequest).Return(getProjectResponse, nil)

	rec, req := s.newRequest(http.MethodGet, "/api/projects/"+projectID, projectID, nil)
	s.api.Get(rec, req)

	s.Require().Equal(http.StatusOK, rec.Code)

	var got model.Project
	s.Require().NoError(json.NewDecoder(rec.Body).Decode(&got))
	s.Require().Equal(project, got)
}

func (s *APISuite) TestGetNotFoundError() {
	var (
		projectID = gofakeit.UUID()

		getProjectRequest = model.GetProjectRequest{
			ProjectID: projectID,
		}
	)

	s.projectService.On("Get", mock.Anything, getProjectRequest).Return(model.GetProjectResponse{}, model.ErrProjectNotFound)

	rec, req := s.newRequest(http.MethodGet, "/api/projects/"+projectID, projectID, nil)
	s.api.Get(rec, req)

	s.Require().Equal(http.StatusNotFound, rec.Code)
}

func (s *APISuite) TestGetError() {
	var (
		projectID  = gofakeit.UUID()
		serviceErr = gofakeit.Error()

		getProjectRequest = model.GetProjectRequest{
			ProjectID: projectID,
		}
	)

	s.projectService.On("Get", mock.Anything, getProjectRequest).Return(model.GetProjectResponse{}, serviceErr)

	rec, req := s.newRequest(http.MethodGet, "/api/projects/"+projectID, projectID, nil)
	s.api.Get(rec, req)

	s.Require().Equal(http.StatusInternalServerError, rec.Code)
}
