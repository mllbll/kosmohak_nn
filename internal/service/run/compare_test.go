package run

import (
	"strings"

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
	runB.EffectiveScenario.Meta.Title = "Этап 1"
	runB.Metrics = []model.ClientMetrics{
		{ClientID: "C65", PathRatio: 0.5, VisibilityRatio: 0.6, MaxGapS: 120, MeetsTarget: false},
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
	s.Require().Len(res.Variants, 2)
	s.Require().Equal(3, res.Variants[0].LaunchStage)
	s.Require().Equal(1, res.Variants[1].LaunchStage)
	s.Require().Len(res.Clients, 1)
	s.Require().Equal(-0.5, res.Clients[0].DeltaPathRatio)
	s.Require().Equal("a", res.Clients[0].Better)
	s.Require().True(res.Clients[0].MeetsTargetA)
	s.Require().False(res.Clients[0].MeetsTargetB)
	s.Require().Equal("a", res.Recommendation.Better)
	s.Require().Equal(runA.ID, res.Recommendation.RunID)
	s.Require().NotEmpty(res.Recommendation.Reason)
	s.Require().NotEmpty(res.Recommendation.Advantages)
	s.Require().NotEmpty(res.Recommendation.Conditions)
	s.Require().NotEmpty(res.Recommendation.Limitations)
	s.Require().NotEmpty(res.Recommendation.Conclusion)
	s.Require().Contains(strings.Join(res.Recommendation.Advantages, "\n"), "C65")
	s.Require().Contains(res.Recommendation.Conditions[0], "90")
	s.Require().Contains(res.Recommendation.Conclusion, "этап 3")
}

func (s *ServiceSuite) TestCompareRunIDsSuccess() {
	var (
		projectID = gofakeit.UUID()
		runA      = testRun(projectID)
		runB      = testRun(projectID)
		runC      = testRun(projectID)
	)
	runB.EffectiveScenario.Design.LaunchStage = 2
	runB.Metrics = []model.ClientMetrics{
		{ClientID: "C65", PathRatio: 0.8, MeetsTarget: false},
	}
	runC.EffectiveScenario.Design.LaunchStage = 1
	runC.Metrics = []model.ClientMetrics{
		{ClientID: "C65", PathRatio: 0.4, MeetsTarget: false},
	}

	var (
		compareRunsRequest = model.CompareRunsRequest{
			RunIDs: []string{runA.ID, runB.ID, runC.ID},
		}
	)

	s.runRepository.On("Get", s.ctx, runA.ID).Return(runA, nil)
	s.runRepository.On("Get", s.ctx, runB.ID).Return(runB, nil)
	s.runRepository.On("Get", s.ctx, runC.ID).Return(runC, nil)

	res, err := s.service.Compare(s.ctx, compareRunsRequest)

	s.Require().NoError(err)
	s.Require().Len(res.Variants, 3)
	s.Require().Equal(runA.ID, res.Recommendation.RunID)
	s.Require().Equal(runA.ID, res.Clients[0].Better)
	s.Require().Len(res.Clients[0].ByRun, 3)
}

func (s *ServiceSuite) TestCompareTie() {
	var (
		projectID = gofakeit.UUID()
		runA      = testRun(projectID)
		runB      = testRun(projectID)

		compareRunsRequest = model.CompareRunsRequest{
			RunAID: runA.ID,
			RunBID: runB.ID,
		}
	)

	s.runRepository.On("Get", s.ctx, runA.ID).Return(runA, nil)
	s.runRepository.On("Get", s.ctx, runB.ID).Return(runB, nil)

	res, err := s.service.Compare(s.ctx, compareRunsRequest)

	s.Require().NoError(err)
	s.Require().Equal("tie", res.Recommendation.Better)
	s.Require().Empty(res.Recommendation.RunID)
	s.Require().Equal("tie", res.Clients[0].Better)
	s.Require().NotEmpty(res.Recommendation.Conditions)
	s.Require().NotEmpty(res.Recommendation.Limitations)
	s.Require().Contains(res.Recommendation.Conclusion, "нет единственного победителя")
}

func (s *ServiceSuite) TestCompareInvalidArgument() {
	var (
		compareRunsRequest = model.CompareRunsRequest{
			RunAID: gofakeit.UUID(),
		}
	)

	res, err := s.service.Compare(s.ctx, compareRunsRequest)

	s.Require().Error(err)
	s.Require().ErrorIs(err, model.ErrInvalidArgument)
	s.Require().Empty(res)
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
