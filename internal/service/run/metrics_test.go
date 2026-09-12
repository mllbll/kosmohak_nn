package run

import "github.com/mllbll/kosmohak_nn/internal/model"

func (s *ServiceSuite) TestAggregateUsesTimeGridForGaps() {
	sc := testScenario()
	sc.Environment.HorizonS = 360
	sc.Environment.StepS = 120

	routes := []model.RouteRecord{
		{TS: 0, ClientID: "C65", Path: []string{"C65", "S01", "G_MUR"}, Hops: 2},
		{TS: 240, ClientID: "C65", Path: []string{"C65", "S01", "G_MUR"}, Hops: 2},
	}
	visible := map[int]map[string]bool{
		0:   {"C65": true},
		120: {"C65": true},
		240: {"C65": false},
	}

	got := Aggregate(sc, routes, visible)

	s.Require().Len(got, 1)
	s.Require().Equal(2.0/3.0, got[0].PathRatio)
	s.Require().Equal(120, got[0].MaxGapS)
	s.Require().InDelta(2.0/3.0, got[0].VisibilityRatio, 1e-9)
	s.Require().False(got[0].MeetsTarget)
	s.Require().Equal([]model.GapInterval{{
		StartS:    120,
		EndS:      240,
		DurationS: 120,
	}}, got[0].Gaps)
}

func (s *ServiceSuite) TestAggregateDoesNotWrapStartAndEndGaps() {
	sc := testScenario()
	sc.Environment.HorizonS = 360
	sc.Environment.StepS = 120

	routes := []model.RouteRecord{
		{TS: 120, ClientID: "C65", Path: []string{"C65", "S01", "G_MUR"}, Hops: 2},
	}
	visible := map[int]map[string]bool{
		0:   {"C65": true},
		120: {"C65": true},
		240: {"C65": true},
	}

	got := Aggregate(sc, routes, visible)

	s.Require().Equal(1.0/3.0, got[0].PathRatio)
	s.Require().Equal(120, got[0].MaxGapS)
	s.Require().Len(got[0].Gaps, 2)
	s.Require().Equal(0, got[0].Gaps[0].StartS)
	s.Require().Equal(120, got[0].Gaps[0].EndS)
	s.Require().Equal(240, got[0].Gaps[1].StartS)
	s.Require().Equal(360, got[0].Gaps[1].EndS)
}

func (s *ServiceSuite) TestAggregateMissingStepCountsAsGap() {
	sc := testScenario()
	sc.Environment.HorizonS = 240
	sc.Environment.StepS = 120

	routes := []model.RouteRecord{
		{TS: 0, ClientID: "C65", Path: []string{"C65", "S01", "G_MUR"}, Hops: 2},
	}
	visible := map[int]map[string]bool{
		0: {"C65": true},
	}

	got := Aggregate(sc, routes, visible)

	s.Require().Equal(0.5, got[0].PathRatio)
	s.Require().Equal(120, got[0].MaxGapS)
	s.Require().Equal(0.5, got[0].VisibilityRatio)
}

func (s *ServiceSuite) TestClientVisibleUsesElevation() {
	sc := testScenario()
	snap := model.Snapshot{
		TS: 0,
		Satellites: []model.SatState{
			{ID: "S01", Active: true},
		},
		Edges: []model.Edge{
			{A: "C65", B: "S01", DistanceKM: 100},
		},
		ElevationDeg: map[string]map[string]float64{
			"C65": {"S01": 5},
		},
	}

	s.Require().False(clientVisible(sc, snap, "C65"))

	snap.ElevationDeg["C65"]["S01"] = 15
	s.Require().True(clientVisible(sc, snap, "C65"))
}

func (s *ServiceSuite) TestClientVisibleIgnoresInactiveSatellite() {
	sc := testScenario()
	snap := model.Snapshot{
		Satellites: []model.SatState{
			{ID: "S01", Active: false},
		},
		ElevationDeg: map[string]map[string]float64{
			"C65": {"S01": 40},
		},
	}

	s.Require().False(clientVisible(sc, snap, "C65"))
}

func (s *ServiceSuite) TestClientVisibleFallsBackToUplink() {
	sc := testScenario()
	snap := testSnapshot()

	s.Require().True(clientVisible(sc, snap, "C65"))
	s.Require().False(clientVisible(sc, snap, "UNKNOWN"))
}
