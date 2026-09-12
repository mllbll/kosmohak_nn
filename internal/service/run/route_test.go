package run

import (
	"testing"

	"github.com/mllbll/kosmohak_nn/internal/model"
)

func TestRouteFindsMinHopPath(t *testing.T) {
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
	if len(res.Path) != 4 {
		t.Fatalf("expected path of 4 nodes, got %v", res.Path)
	}
	if res.Path[0] != "C65" || res.Path[len(res.Path)-1] != "G_MUR" {
		t.Fatalf("unexpected path %v", res.Path)
	}
	if res.Hops != 3 {
		t.Fatalf("expected 3 hops, got %d", res.Hops)
	}
	if res.Algorithm.Name != "bfs_min_hops" {
		t.Fatalf("algorithm %s", res.Algorithm.Name)
	}
}

func TestRouteCollectsAlternateMinHopPaths(t *testing.T) {
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
	if res.Hops != 2 {
		t.Fatalf("min hops %d path %v", res.Hops, res.Path)
	}
	if len(res.Alternatives) == 0 {
		t.Fatal("expected alternate min-hop path")
	}
	if res.Alternatives[0][1] == res.Path[1] {
		t.Fatalf("alternate should use another satellite, path=%v alt=%v", res.Path, res.Alternatives[0])
	}
}

func TestRouteIncludesBackupLongerPath(t *testing.T) {
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
	if got := []string{"C65", "S1", "G_MUR"}; len(res.Path) != 3 || res.Path[1] != "S1" {
		t.Fatalf("primary %v want %v", res.Path, got)
	}
	foundBackup := false
	for _, alt := range res.Alternatives {
		if len(alt) == 4 && alt[1] == "S1" && alt[2] == "S2" {
			foundBackup = true
		}
	}
	if !foundBackup {
		t.Fatalf("expected +1 hop backup via S2, alts=%v", res.Alternatives)
	}
}

func TestRouteNoVisibleSat(t *testing.T) {
	sc := sampleScenario()
	snap := model.Snapshot{
		TS:         0,
		Satellites: []model.SatState{{ID: "S1", Active: true}},
		Edges:      []model.Edge{{A: "S1", B: "G_MUR", DistanceKM: 100}},
	}

	res := Route(sc, snap, "C65")
	if len(res.Path) != 0 {
		t.Fatalf("expected empty path, got %v", res.Path)
	}
	if res.Reason != model.GapNoVisibleSat {
		t.Fatalf("expected no_visible_sat, got %s", res.Reason)
	}
}

func TestRouteISLPartition(t *testing.T) {
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
	if res.Reason != model.GapISLPartition {
		t.Fatalf("expected isl_partition, got %s path=%v", res.Reason, res.Path)
	}
}

func TestGroundDoesNotRelay(t *testing.T) {
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
	if len(res.Path) != 0 {
		t.Fatalf("client must not relay, got %v", res.Path)
	}
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
