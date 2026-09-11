package run

import (
	"github.com/brianvoe/gofakeit/v7"
	"github.com/mllbll/kosmohak_nn/internal/model"
)

func (s *ServiceSuite) TestGetSuccess() {
	var (
		projectID = gofakeit.UUID()
		run       = testRun(projectID)

		getRunRequest = model.GetRunRequest{
			RunID: run.ID,
		}
	)

	s.runRepository.On("Get", s.ctx, run.ID).Return(run, nil)

	res, err := s.service.Get(s.ctx, getRunRequest)

	s.Require().NoError(err)
	s.Require().Equal(run, res.Run)
}

func (s *ServiceSuite) TestGetError() {
	var (
		runID   = gofakeit.UUID()
		repoErr = gofakeit.Error()

		getRunRequest = model.GetRunRequest{
			RunID: runID,
		}
	)

	s.runRepository.On("Get", s.ctx, runID).Return(model.Run{}, repoErr)

	res, err := s.service.Get(s.ctx, getRunRequest)

	s.Require().Error(err)
	s.Require().Equal(err, repoErr)
	s.Require().Empty(res)
}

func (s *ServiceSuite) TestGetNotFoundError() {
	var (
		runID = gofakeit.UUID()

		getRunRequest = model.GetRunRequest{
			RunID: runID,
		}
	)

	s.runRepository.On("Get", s.ctx, runID).Return(model.Run{}, model.ErrRunNotFound)

	res, err := s.service.Get(s.ctx, getRunRequest)

	s.Require().Error(err)
	s.Require().Equal(err, model.ErrRunNotFound)
	s.Require().Empty(res)
}
