package v1

import (
	"encoding/json"
	"net/http"

	"github.com/brianvoe/gofakeit/v7"
	"github.com/mllbll/kosmohak_nn/internal/model"
	"github.com/stretchr/testify/mock"
)

func (s *APISuite) TestCopySuccess() {
	var (
		projectID = gofakeit.UUID()
		sc        = testScenario()

		copyProjectRequest = model.CopyProjectRequest{
			ProjectID: projectID,
		}

		project = model.Project{
			ID:        gofakeit.UUID(),
			Base:      sc,
			Effective: sc,
		}

		copyProjectResponse = model.CopyProjectResponse{
			Project: project,
		}
	)

	s.projectService.On("Copy", mock.Anything, copyProjectRequest).Return(copyProjectResponse, nil)

	rec, req := s.newRequest(http.MethodPost, "/api/projects/"+projectID+"/copy", projectID, nil)
	s.api.Copy(rec, req)

	s.Require().Equal(http.StatusCreated, rec.Code)

	var got model.Project
	s.Require().NoError(json.NewDecoder(rec.Body).Decode(&got))
	s.Require().Equal(project, got)
}

func (s *APISuite) TestCopyNotFoundError() {
	var (
		projectID = gofakeit.UUID()

		copyProjectRequest = model.CopyProjectRequest{
			ProjectID: projectID,
		}
	)

	s.projectService.On("Copy", mock.Anything, copyProjectRequest).Return(model.CopyProjectResponse{}, model.ErrProjectNotFound)

	rec, req := s.newRequest(http.MethodPost, "/api/projects/"+projectID+"/copy", projectID, nil)
	s.api.Copy(rec, req)

	s.Require().Equal(http.StatusNotFound, rec.Code)
}

func (s *APISuite) TestCopyError() {
	var (
		projectID  = gofakeit.UUID()
		serviceErr = gofakeit.Error()

		copyProjectRequest = model.CopyProjectRequest{
			ProjectID: projectID,
		}
	)

	s.projectService.On("Copy", mock.Anything, copyProjectRequest).Return(model.CopyProjectResponse{}, serviceErr)

	rec, req := s.newRequest(http.MethodPost, "/api/projects/"+projectID+"/copy", projectID, nil)
	s.api.Copy(rec, req)

	s.Require().Equal(http.StatusInternalServerError, rec.Code)
}
