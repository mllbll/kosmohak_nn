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

func (s *APISuite) TestCreateSuccess() {
	var (
		sc = testScenario()

		createProjectRequest = model.CreateProjectRequest{
			Scenario: sc,
		}

		project = model.Project{
			ID:        gofakeit.UUID(),
			Base:      sc,
			Effective: sc,
		}

		createProjectResponse = model.CreateProjectResponse{
			Project: project,
		}
	)

	s.projectService.On("Create", mock.Anything, createProjectRequest).Return(createProjectResponse, nil)

	rec, req := s.newRequest(http.MethodPost, "/api/projects", "", sc)
	s.api.Create(rec, req)

	s.Require().Equal(http.StatusCreated, rec.Code)

	var got model.Project
	s.Require().NoError(json.NewDecoder(rec.Body).Decode(&got))
	s.Require().Equal(project, got)
}

func (s *APISuite) TestCreateInvalidArgument() {
	req := httptest.NewRequest(http.MethodPost, "/api/projects", bytes.NewReader([]byte("{")))
	req = req.WithContext(s.ctx)
	rec := httptest.NewRecorder()

	s.api.Create(rec, req)

	s.Require().Equal(http.StatusBadRequest, rec.Code)
	s.projectService.AssertNotCalled(s.T(), "Create")
}

func (s *APISuite) TestCreateError() {
	var (
		sc         = testScenario()
		serviceErr = gofakeit.Error()

		createProjectRequest = model.CreateProjectRequest{
			Scenario: sc,
		}
	)

	s.projectService.On("Create", mock.Anything, createProjectRequest).Return(model.CreateProjectResponse{}, serviceErr)

	rec, req := s.newRequest(http.MethodPost, "/api/projects", "", sc)
	s.api.Create(rec, req)

	s.Require().Equal(http.StatusInternalServerError, rec.Code)
}

func (s *APISuite) TestCreateFromExportSuccess() {
	var (
		sc  = testScenario()
		doc = model.ExportDocument{
			SchemaVersion:     model.ResultSchemaVersion,
			EffectiveScenario: sc,
			Routes: []model.ExportRoute{
				{TS: 0, ClientID: "C65", Path: []string{}},
			},
		}
		project = model.Project{
			ID:        gofakeit.UUID(),
			Base:      sc,
			Effective: sc,
		}
	)

	s.projectService.On("Create", mock.Anything, model.CreateProjectRequest{Scenario: sc}).Return(
		model.CreateProjectResponse{Project: project}, nil,
	)

	rec, req := s.newRequest(http.MethodPost, "/api/projects", "", doc)
	s.api.Create(rec, req)

	s.Require().Equal(http.StatusCreated, rec.Code)
}

func (s *APISuite) TestCreateFromExportMissingScenario() {
	doc := map[string]any{
		"schema_version": model.ResultSchemaVersion,
		"routes":         []any{},
	}

	rec, req := s.newRequest(http.MethodPost, "/api/projects", "", doc)
	s.api.Create(rec, req)

	s.Require().Equal(http.StatusBadRequest, rec.Code)
	s.projectService.AssertNotCalled(s.T(), "Create")
}
