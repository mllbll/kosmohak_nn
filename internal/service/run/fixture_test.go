package run

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/mllbll/kosmohak_nn/internal/client/geometry"
	"github.com/mllbll/kosmohak_nn/internal/model"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type FixtureSuite struct {
	suite.Suite

	ctx    context.Context
	client geometry.Client
	root   string
}

func (s *FixtureSuite) SetupSuite() {
	s.ctx = context.Background()
	s.root = repoRoot(s.T())
	s.client = geometry.NewClient("python3", filepath.Join(s.root, "python", "runner.py"))

	if _, err := exec.LookPath("python3"); err != nil {
		if os.Getenv("REQUIRE_GEOMETRY_TESTS") == "1" {
			s.Require().NoError(err)
		}
		s.T().Skip("python3 not found")
	}
	sc := s.load("data/01_full_constellation.json")
	sc.Environment.HorizonS = sc.Environment.StepS
	if _, err := s.client.Snapshot(s.ctx, sc, 0); err != nil {
		if os.Getenv("REQUIRE_GEOMETRY_TESTS") == "1" {
			s.Require().NoError(err)
		}
		s.T().Skipf("geometry.py unavailable: %v", err)
	}
}

func TestFixtureIntegration(t *testing.T) {
	suite.Run(t, new(FixtureSuite))
}

func (s *FixtureSuite) TestFixture01FullConstellation() {
	s.assertFixtureGrid("data/01_full_constellation.json", fixtureOpts{})
}

func (s *FixtureSuite) TestFixture02FirstLaunch() {
	sc := s.load("data/02_first_launch.json")
	snaps := s.assertFixtureGrid("data/02_first_launch.json", fixtureOpts{})
	s.Require().NotEmpty(snaps)
	for _, sat := range snaps[0].Satellites {
		s.Require().Equal(launchBatch(sc, sat.ID) <= 1, sat.Active, sat.ID)
	}
}

func (s *FixtureSuite) TestFixture03SatelliteOutages() {
	s.assertFixtureGrid("data/03_satellite_outages.json", fixtureOpts{alignFailures: true})

	sc := s.load("data/03_satellite_outages.json")
	before, err := s.client.Snapshot(s.ctx, sc, 0)
	s.Require().NoError(err)
	after, err := s.client.Snapshot(s.ctx, sc, 21600)
	s.Require().NoError(err)

	for _, f := range sc.Failures {
		s.Require().True(satelliteActive(before, f.SatelliteID), f.SatelliteID)
		s.Require().False(satelliteActive(after, f.SatelliteID), f.SatelliteID)
	}
}

func (s *FixtureSuite) TestFixture04LinkRange() {
	s.assertFixtureGrid("data/04_link_range.json", fixtureOpts{})

	full := s.load("data/01_full_constellation.json")
	short := s.load("data/04_link_range.json")
	fullSnap, err := s.client.Snapshot(s.ctx, full, 0)
	s.Require().NoError(err)
	shortSnap, err := s.client.Snapshot(s.ctx, short, 0)
	s.Require().NoError(err)

	ids := satelliteIDs(full)
	s.Require().Less(islEdgeCount(shortSnap, ids), islEdgeCount(fullSnap, ids))
}

type fixtureOpts struct {
	alignFailures bool
}

func (s *FixtureSuite) assertFixtureGrid(rel string, opts fixtureOpts) []model.Snapshot {
	sc := s.load(rel)
	sc.Environment.HorizonS = sc.Environment.StepS * 4
	if opts.alignFailures {
		for i := range sc.Failures {
			sc.Failures[i].StartS = 0
			sc.Failures[i].EndS = float64(sc.Environment.HorizonS)
		}
	}

	grid := sc.TimeGrid()
	s.Require().Len(grid, 4)

	snaps := make([]model.Snapshot, 0, len(grid))
	err := s.client.WalkSnapshots(s.ctx, sc, func(snap model.Snapshot) error {
		snaps = append(snaps, snap)
		return nil
	})
	s.Require().NoError(err)
	s.Require().Len(snaps, len(grid))

	var routes []model.RouteRecord
	visible := map[int]map[string]bool{}
	clients := sc.ClientIDs()
	s.Require().NotEmpty(clients)

	for i, snap := range snaps {
		t := int(snap.TS)
		s.Require().Equal(grid[i], t)
		visible[t] = map[string]bool{}
		for _, sat := range snap.Satellites {
			if launchBatch(sc, sat.ID) > sc.Design.LaunchStage {
				s.Require().False(sat.Active, sat.ID)
			}
		}
		for _, clientID := range clients {
			elev, ok := snap.ElevationDeg[clientID]
			s.Require().True(ok, clientID)
			fromElev := false
			for satID, el := range elev {
				if el >= sc.Environment.MinElevationDeg && satelliteActive(snap, satID) {
					fromElev = true
					break
				}
			}
			s.Require().Equal(fromElev, clientVisible(sc, snap, clientID), clientID)

			res := Route(sc, snap, clientID)
			path := res.Path
			if path == nil {
				path = []string{}
			}
			routes = append(routes, model.RouteRecord{
				TS:       t,
				ClientID: clientID,
				Path:     path,
				Reason:   res.Reason,
				Hops:     res.Hops,
			})
			visible[t][clientID] = clientVisible(sc, snap, clientID)
		}
	}

	metrics := Aggregate(sc, routes, visible)
	s.Require().Len(metrics, len(clients))

	byClient := map[string][]model.RouteRecord{}
	for _, rec := range routes {
		byClient[rec.ClientID] = append(byClient[rec.ClientID], rec)
	}
	for _, m := range metrics {
		recs := byClient[m.ClientID]
		s.Require().Len(recs, len(grid))
		ok := 0
		vis := 0
		streak := 0
		maxStreak := 0
		for _, t := range grid {
			if visible[t][m.ClientID] {
				vis++
			}
			var rec model.RouteRecord
			for _, r := range recs {
				if r.TS == t {
					rec = r
					break
				}
			}
			if len(rec.Path) > 0 {
				ok++
				streak = 0
				continue
			}
			streak++
			if streak > maxStreak {
				maxStreak = streak
			}
		}
		s.Require().InDelta(float64(ok)/float64(len(grid)), m.PathRatio, 1e-9, m.ClientID)
		s.Require().Equal(maxStreak*sc.Environment.StepS, m.MaxGapS, m.ClientID)
		s.Require().InDelta(float64(vis)/float64(len(grid)), m.VisibilityRatio, 1e-9, m.ClientID)
	}

	return snaps
}

func (s *FixtureSuite) load(rel string) model.Scenario {
	s.T().Helper()
	raw, err := os.ReadFile(filepath.Join(s.root, rel))
	s.Require().NoError(err)
	var sc model.Scenario
	s.Require().NoError(json.Unmarshal(raw, &sc))
	return sc
}

func launchBatch(sc model.Scenario, id string) int {
	for _, sat := range sc.Design.Satellites {
		if sat.ID == id {
			return sat.LaunchBatch
		}
	}
	return 0
}

func repoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	require.NoError(t, err)
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("go.mod not found")
		}
		dir = parent
	}
}
