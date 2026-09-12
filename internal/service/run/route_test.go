package run

import (
	"github.com/mllbll/kosmohak_nn/internal/model"
)

func (s *ServiceSuite) TestRouteFindsMinHopPath() {
	sc := sampleScenario()
	snap := model.Snapshot{
		TS: 0,
		Satellites: []model.SatState{
			{ID: "S1", Active: true},
			{ID: "S2", Active: true},
		},
		Edges: []model.Edge{
			{A: "C65", B: "S1", DistanceKM: 100},
			{A: "S1", B: "S2", DistanceKM: 200},
			{A: "S2", B: "G_MUR", DistanceKM: 150},
		},
	}

	res := Route(sc, snap, "C65")
	s.Require().Len(res.Path, 4)
	s.Require().Equal("C65", res.Path[0])
	s.Require().Equal("G_MUR", res.Path[len(res.Path)-1])
	s.Require().Equal(3, res.Hops)
	s.Require().Equal("bfs_min_hops", res.Algorithm.Name)
}

func (s *ServiceSuite) TestRouteGatewayOutageIgnoresStaleEdge() {
	sc := sampleScenario()
	sc.GatewayOutages = []model.GatewayOutage{{GatewayID: "G_MUR", StartS: 0, EndS: 120}}
	snap := model.Snapshot{
		TS:         0,
		Satellites: []model.SatState{{ID: "S1", Active: true}},
		Edges: []model.Edge{
			{A: "C65", B: "S1", DistanceKM: 100},
			{A: "S1", B: "G_MUR", DistanceKM: 100},
		},
	}

	res := Route(sc, snap, "C65")
	s.Require().Empty(res.Path)
	s.Require().Equal(model.GapGatewayOutage, res.Reason)
}

func (s *ServiceSuite) TestRouteUsesLiveGatewayWhenOtherOutaged() {
	sc := sampleScenario()
	sc.GroundSites = append(sc.GroundSites, model.GroundSite{ID: "G2", Role: "gateway"})
	sc.GatewayOutages = []model.GatewayOutage{{GatewayID: "G_MUR", StartS: 0, EndS: 120}}
	snap := model.Snapshot{
		TS:         0,
		Satellites: []model.SatState{{ID: "S1", Active: true}},
		Edges: []model.Edge{
			{A: "C65", B: "S1", DistanceKM: 100},
			{A: "S1", B: "G_MUR", DistanceKM: 100},
			{A: "S1", B: "G2", DistanceKM: 100},
		},
	}

	res := Route(sc, snap, "C65")
	s.Require().Equal([]string{"C65", "S1", "G2"}, res.Path)
	s.Require().Empty(res.Reason)
}

func (s *ServiceSuite) TestRouteCollectsAlternateMinHopPaths() {
	sc := sampleScenario()
	snap := model.Snapshot{
		TS: 0,
		Satellites: []model.SatState{
			{ID: "S1", Active: true},
			{ID: "S2", Active: true},
		},
		Edges: []model.Edge{
			{A: "C65", B: "S1", DistanceKM: 100},
			{A: "C65", B: "S2", DistanceKM: 100},
			{A: "S1", B: "G_MUR", DistanceKM: 100},
			{A: "S2", B: "G_MUR", DistanceKM: 100},
		},
	}

	res := Route(sc, snap, "C65")
	s.Require().Equal(2, res.Hops)
	s.Require().NotEmpty(res.Alternatives)
	s.Require().NotEqual(res.Path[1], res.Alternatives[0][1])
}

func (s *ServiceSuite) TestRouteIncludesBackupLongerPath() {
	sc := sampleScenario()
	snap := model.Snapshot{
		TS: 0,
		Satellites: []model.SatState{
			{ID: "S1", Active: true},
			{ID: "S2", Active: true},
		},
		Edges: []model.Edge{
			{A: "C65", B: "S1", DistanceKM: 100},
			{A: "S1", B: "G_MUR", DistanceKM: 100},
			{A: "S1", B: "S2", DistanceKM: 100},
			{A: "S2", B: "G_MUR", DistanceKM: 100},
		},
	}

	res := Route(sc, snap, "C65")
	s.Require().Equal([]string{"C65", "S1", "G_MUR"}, res.Path)
	foundBackup := false
	for _, alt := range res.Alternatives {
		if len(alt) == 4 && alt[1] == "S1" && alt[2] == "S2" {
			foundBackup = true
		}
	}
	s.Require().True(foundBackup, "expected +1 hop backup via S2, alts=%v", res.Alternatives)
}

func (s *ServiceSuite) TestRouteNoVisibleSat() {
	sc := sampleScenario()
	snap := model.Snapshot{
		TS:         0,
		Satellites: []model.SatState{{ID: "S1", Active: true}},
		Edges:      []model.Edge{{A: "S1", B: "G_MUR", DistanceKM: 100}},
	}

	res := Route(sc, snap, "C65")
	s.Require().Empty(res.Path)
	s.Require().Equal(model.GapNoVisibleSat, res.Reason)
}

func (s *ServiceSuite) TestRouteISLPartition() {
	sc := sampleScenario()
	snap := model.Snapshot{
		TS: 0,
		Satellites: []model.SatState{
			{ID: "S1", Active: true},
			{ID: "S2", Active: true},
		},
		Edges: []model.Edge{
			{A: "C65", B: "S1", DistanceKM: 100},
			{A: "S2", B: "G_MUR", DistanceKM: 100},
		},
	}

	res := Route(sc, snap, "C65")
	s.Require().Equal(model.GapISLPartition, res.Reason)
	s.Require().Empty(res.Path)
}

func (s *ServiceSuite) TestGroundDoesNotRelay() {
	sc := sampleScenario()
	sc.GroundSites = append(sc.GroundSites, model.GroundSite{ID: "C70", Role: "client"})
	snap := model.Snapshot{
		TS:         0,
		Satellites: []model.SatState{{ID: "S1", Active: true}},
		Edges: []model.Edge{
			{A: "C65", B: "S1", DistanceKM: 100},
			{A: "S1", B: "C70", DistanceKM: 100},
			{A: "C70", B: "G_MUR", DistanceKM: 100},
		},
	}

	res := Route(sc, snap, "C65")
	s.Require().Empty(res.Path)
	s.Require().Equal(model.GapNoGatewayContact, res.Reason)
}

func (s *ServiceSuite) TestRouteIgnoresInactiveSatellite() {
	sc := sampleScenario()
	snap := model.Snapshot{
		TS:         0,
		Satellites: []model.SatState{{ID: "S1", Active: false}},
		Edges: []model.Edge{
			{A: "C65", B: "S1", DistanceKM: 100},
			{A: "S1", B: "G_MUR", DistanceKM: 100},
		},
	}

	res := Route(sc, snap, "C65")
	s.Require().Empty(res.Path)
	s.Require().Equal(model.GapNoVisibleSat, res.Reason)
}

func sampleScenario() model.Scenario {
	return model.Scenario{
		SchemaVersion: model.SchemaVersion,
		Environment:   model.Environment{HorizonS: 120, StepS: 120, TargetAvailability: 0.9},
		GroundSites: []model.GroundSite{
			{ID: "C65", Role: "client"},
			{ID: "G_MUR", Role: "gateway"},
		},
	}
}
