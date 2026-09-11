package run

import (
	"github.com/brianvoe/gofakeit/v7"
	"github.com/mllbll/kosmohak_nn/internal/model"
)

func (s *ServiceSuite) TestCompareSuccess() {
	var (
		projectID = gofakeit.UUID()
		runA      = testRun(projectID)
		runB      = testRun(projectID)
	)
	runB.EffectiveScenario.Design.LaunchStage = 1
	runB.Metrics = []model.ClientMetrics{
		{ClientID: "C65", PathRatio: 0.5, MaxGapS: 120},
	}

	var (
		compareRunsRequest = model.CompareRunsRequest{
			RunAID: runA.ID,
			RunBID: runB.ID,
		}
	)

	s.runRepository.On("Get", s.ctx, runA.ID).Return(runA, nil)
	s.runRepository.On("Get", s.ctx, runB.ID).Return(runB, nil)

	res, err := s.service.Compare(s.ctx, compareRunsRequest)

	s.Require().NoError(err)
	s.Require().Equal(runA.ID, res.RunAID)
	s.Require().Equal(runB.ID, res.RunBID)
	s.Require().NotEmpty(res.Config)
	s.Require().Len(res.Clients, 1)
	s.Require().Equal(-0.5, res.Clients[0].DeltaPathRatio)
}

func (s *ServiceSuite) TestCompareNotFoundError() {
	var (
		runAID = gofakeit.UUID()
		runBID = gofakeit.UUID()

		compareRunsRequest = model.CompareRunsRequest{
			RunAID: runAID,
			RunBID: runBID,
		}
	)

	s.runRepository.On("Get", s.ctx, runAID).Return(model.Run{}, model.ErrRunNotFound)

	res, err := s.service.Compare(s.ctx, compareRunsRequest)

	s.Require().Error(err)
	s.Require().Equal(err, model.ErrRunNotFound)
	s.Require().Empty(res)
}

func (s *ServiceSuite) TestCompareRunBNotFoundError() {
	var (
		projectID = gofakeit.UUID()
		runA      = testRun(projectID)
		runBID    = gofakeit.UUID()

		compareRunsRequest = model.CompareRunsRequest{
			RunAID: runA.ID,
			RunBID: runBID,
		}
	)

	s.runRepository.On("Get", s.ctx, runA.ID).Return(runA, nil)
	s.runRepository.On("Get", s.ctx, runBID).Return(model.Run{}, model.ErrRunNotFound)

	res, err := s.service.Compare(s.ctx, compareRunsRequest)

	s.Require().Error(err)
	s.Require().Equal(err, model.ErrRunNotFound)
	s.Require().Empty(res)
}
