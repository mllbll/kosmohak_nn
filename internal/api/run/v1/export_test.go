package v1

import (
	"encoding/json"
	"net/http"

	"github.com/brianvoe/gofakeit/v7"
	"github.com/mllbll/kosmohak_nn/internal/model"
	"github.com/stretchr/testify/mock"
)

func (s *APISuite) TestExportSuccess() {
	var (
		projectID = gofakeit.UUID()
		run       = testRun(projectID)

		exportRunRequest = model.ExportRunRequest{
			RunID: run.ID,
		}

		exportDocument = model.ExportDocument{
			SchemaVersion:     model.ResultSchemaVersion,
			EffectiveScenario: run.EffectiveScenario,
			Routes: []model.ExportRoute{
				{TS: 0, ClientID: "C65", Path: []string{"C65", "S01", "G_MUR"}},
			},
			Metrics: run.Metrics,
		}
	)

	s.runService.On("Export", mock.Anything, exportRunRequest).Return(exportDocument, nil)

	rec, req := s.newRequest(http.MethodGet, "/api/runs/"+run.ID+"/export", run.ID, nil)
	s.api.Export(rec, req)

	s.Require().Equal(http.StatusOK, rec.Code)
	s.Require().Equal(`attachment; filename="cosmo-A-result.json"`, rec.Header().Get("Content-Disposition"))

	var got model.ExportDocument
	s.Require().NoError(json.NewDecoder(rec.Body).Decode(&got))
	s.Require().Equal(exportDocument, got)
}

func (s *APISuite) TestExportNotFoundError() {
	var (
		runID = gofakeit.UUID()

		exportRunRequest = model.ExportRunRequest{
			RunID: runID,
		}
	)

	s.runService.On("Export", mock.Anything, exportRunRequest).Return(model.ExportDocument{}, model.ErrRunNotFound)

	rec, req := s.newRequest(http.MethodGet, "/api/runs/"+runID+"/export", runID, nil)
	s.api.Export(rec, req)

	s.Require().Equal(http.StatusNotFound, rec.Code)
}

func (s *APISuite) TestExportError() {
	var (
		runID      = gofakeit.UUID()
		serviceErr = gofakeit.Error()

		exportRunRequest = model.ExportRunRequest{
			RunID: runID,
		}
	)

	s.runService.On("Export", mock.Anything, exportRunRequest).Return(model.ExportDocument{}, serviceErr)

	rec, req := s.newRequest(http.MethodGet, "/api/runs/"+runID+"/export", runID, nil)
	s.api.Export(rec, req)

	s.Require().Equal(http.StatusInternalServerError, rec.Code)
}
