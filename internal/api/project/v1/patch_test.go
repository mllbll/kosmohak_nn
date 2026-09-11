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

func (s *APISuite) TestPatchSuccess() {
	var (
		projectID   = gofakeit.UUID()
		sc          = testScenario()
		launchStage = 1
		patch       = model.Patch{
			LaunchStage: &launchStage,
		}

		patchProjectRequest = model.PatchProjectRequest{
			ProjectID: projectID,
			Patch:     patch,
		}

		project = model.Project{
			ID:        projectID,
			Base:      sc,
			Effective: sc,
		}

		patchProjectResponse = model.PatchProjectResponse{
			Project: project,
		}
	)

	s.projectService.On("Patch", mock.Anything, patchProjectRequest).Return(patchProjectResponse, nil)

	rec, req := s.newRequest(http.MethodPatch, "/api/projects/"+projectID, projectID, patch)
	s.api.Patch(rec, req)

	s.Require().Equal(http.StatusOK, rec.Code)

	var got model.Project
	s.Require().NoError(json.NewDecoder(rec.Body).Decode(&got))
	s.Require().Equal(project, got)
}

func (s *APISuite) TestPatchNotFoundError() {
	var (
		projectID = gofakeit.UUID()
		patch     = model.Patch{}

		patchProjectRequest = model.PatchProjectRequest{
			ProjectID: projectID,
			Patch:     patch,
		}
	)

	s.projectService.On("Patch", mock.Anything, patchProjectRequest).Return(model.PatchProjectResponse{}, model.ErrProjectNotFound)

	rec, req := s.newRequest(http.MethodPatch, "/api/projects/"+projectID, projectID, patch)
	s.api.Patch(rec, req)

	s.Require().Equal(http.StatusNotFound, rec.Code)
}

func (s *APISuite) TestPatchInvalidArgument() {
	var (
		projectID = gofakeit.UUID()
		rctx      = chi.NewRouteContext()
	)
	rctx.URLParams.Add("id", projectID)

	req := httptest.NewRequest(http.MethodPatch, "/api/projects/"+projectID, bytes.NewReader([]byte("{")))
	req = req.WithContext(context.WithValue(s.ctx, chi.RouteCtxKey, rctx))
	rec := httptest.NewRecorder()

	s.api.Patch(rec, req)

	s.Require().Equal(http.StatusBadRequest, rec.Code)
	s.projectService.AssertNotCalled(s.T(), "Patch")
}
