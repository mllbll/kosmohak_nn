package run

import (
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
