package run

import (
	"encoding/json"

	"github.com/brianvoe/gofakeit/v7"
	"github.com/mllbll/kosmohak_nn/internal/model"
)

func (s *ServiceSuite) TestExportSuccess() {
	var (
		projectID = gofakeit.UUID()
		run       = testRun(projectID)

		exportRunRequest = model.ExportRunRequest{
			RunID: run.ID,
		}
	)

	s.runRepository.On("Get", s.ctx, run.ID).Return(run, nil)

	res, err := s.service.Export(s.ctx, exportRunRequest)

	s.Require().NoError(err)
	s.Require().Equal(model.ResultSchemaVersion, res.SchemaVersion)
	s.Require().Equal(run.EffectiveScenario, res.EffectiveScenario)
	s.Require().Len(res.Routes, 1)
	s.Require().Equal(run.Metrics, res.Metrics)
}

func (s *ServiceSuite) TestExportNotFoundError() {
	var (
		runID = gofakeit.UUID()

		exportRunRequest = model.ExportRunRequest{
			RunID: runID,
		}
	)

	s.runRepository.On("Get", s.ctx, runID).Return(model.Run{}, model.ErrRunNotFound)

	res, err := s.service.Export(s.ctx, exportRunRequest)

	s.Require().Error(err)
	s.Require().Equal(err, model.ErrRunNotFound)
	s.Require().Empty(res)
}

func (s *ServiceSuite) TestExportMarshalsEmptyPathAndGaps() {
	var (
		projectID = gofakeit.UUID()
		run       = testRun(projectID)
	)
	run.Routes = []model.RouteRecord{
		{TS: 0, ClientID: "C65", Path: nil, Reason: model.GapISLPartition},
	}
	run.Metrics[0].Gaps = nil

	s.runRepository.On("Get", s.ctx, run.ID).Return(run, nil)

	res, err := s.service.Export(s.ctx, model.ExportRunRequest{RunID: run.ID})

	s.Require().NoError(err)
	s.Require().Equal([]string{}, res.Routes[0].Path)
	s.Require().Equal(model.GapISLPartition, res.Routes[0].Reason)
	s.Require().NotNil(res.Metrics[0].Gaps)

	raw, err := json.Marshal(res)
	s.Require().NoError(err)
	s.Require().Contains(string(raw), `"path":[]`)
	s.Require().Contains(string(raw), `"reason":"isl_partition"`)
	s.Require().Contains(string(raw), `"gaps":[]`)
	s.Require().NotContains(string(raw), `"path":null`)
	s.Require().NotContains(string(raw), `"failures":null`)
}
