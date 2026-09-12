package run

import (
	"github.com/brianvoe/gofakeit/v7"
	"github.com/mllbll/kosmohak_nn/internal/model"
)

func (s *ServiceSuite) TestBuildResilienceAnalysisAffected() {
	var (
		original = testRun(gofakeit.UUID())
		failed   = original
	)
	failed.Routes = []model.RouteRecord{
		{TS: 0, ClientID: "C65", Path: []string{}, Reason: model.GapISLPartition},
	}
	failed.Metrics = []model.ClientMetrics{
		{ClientID: "C65", PathRatio: 0, VisibilityRatio: 0, MaxGapS: 120, MeetsTarget: false},
	}

	analysis := buildResilienceAnalysis(original, failed, "S01", "", 0, 120)

	s.Require().Equal([]string{"C65"}, analysis.AffectedClients)
	s.Require().Empty(analysis.PreservedClients)
	s.Require().True(analysis.Clients[0].Affected)
	s.Require().False(analysis.Clients[0].RoutePreserved)
	s.Require().Equal([]string{"C65", "S01", "G_MUR"}, analysis.Clients[0].PathBefore)
	s.Require().Empty(analysis.Clients[0].PathAfter)
	s.Require().Equal(model.GapISLPartition, analysis.Clients[0].DominantGapReason)
	s.Require().Equal(1, analysis.GapReasons.Delta[string(model.GapISLPartition)])
	s.Require().NotEmpty(analysis.Vulnerabilities)
	s.Require().NotEmpty(analysis.Mitigations)
	s.Require().Contains(analysis.Summary, "C65")
}

func (s *ServiceSuite) TestBuildResilienceAnalysisPreserved() {
	var (
		original = testRun(gofakeit.UUID())
		failed   = original
	)
	failed.Routes = []model.RouteRecord{
		{TS: 0, ClientID: "C65", Path: []string{"C65", "S02", "G_MUR"}, Hops: 2},
	}

	analysis := buildResilienceAnalysis(original, failed, "S01", "", 0, 120)

	s.Require().Empty(analysis.AffectedClients)
	s.Require().Equal([]string{"C65"}, analysis.PreservedClients)
	s.Require().True(analysis.Clients[0].RoutePreserved)
	s.Require().Contains(analysis.Summary, "сохранились")
}

func (s *ServiceSuite) TestBuildResilienceAnalysisGateway() {
	var (
		original = testRun(gofakeit.UUID())
		failed   = original
	)
	failed.Routes = []model.RouteRecord{
		{TS: 0, ClientID: "C65", Path: []string{}, Reason: model.GapGatewayOutage},
	}
	failed.Metrics = []model.ClientMetrics{
		{ClientID: "C65", PathRatio: 0, MaxGapS: 120, MeetsTarget: false},
	}

	analysis := buildResilienceAnalysis(original, failed, "", "G_MUR", 0, 120)

	s.Require().Equal("G_MUR", analysis.FailedGatewayID)
	s.Require().Equal([]string{"C65"}, analysis.AffectedClients)
	s.Require().Contains(analysis.Mitigations[0], "шлюз")
}
