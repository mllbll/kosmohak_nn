package run

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"

	"github.com/go-chi/chi/v5"
	projectV1 "github.com/mllbll/kosmohak_nn/internal/api/project/v1"
	runV1 "github.com/mllbll/kosmohak_nn/internal/api/run/v1"
	"github.com/mllbll/kosmohak_nn/internal/model"
	projectRepository "github.com/mllbll/kosmohak_nn/internal/repository/project"
	runRepository "github.com/mllbll/kosmohak_nn/internal/repository/run"
	projectService "github.com/mllbll/kosmohak_nn/internal/service/project"
)

func (s *FixtureSuite) TestNilFailuresEncodeForPython() {
	sc := s.load("data/01_full_constellation.json")
	sc.Failures = nil
	sc.GatewayOutages = nil
	sc.Environment.HorizonS = sc.Environment.StepS

	_, err := s.client.Snapshot(s.ctx, sc, 0)
	s.Require().NoError(err)
}

func (s *FixtureSuite) TestOfficialFixturesFullDay() {
	projRepo := projectRepository.NewRepository()
	runRepo := runRepository.NewRepository()
	ps := projectService.NewService(projRepo, s.client)
	rs := NewService(projRepo, runRepo, s.client)

	files := []string{
		"data/01_full_constellation.json",
		"data/02_first_launch.json",
		"data/03_satellite_outages.json",
		"data/04_link_range.json",
	}
	runs := map[string]model.Run{}
	ids := map[string]string{}

	for _, file := range files {
		sc := s.load(file)
		created, err := ps.Create(s.ctx, model.CreateProjectRequest{Scenario: sc})
		s.Require().NoError(err, file)
		s.Require().NotEqual(sc.Meta.ID, created.Project.ID, file)

		createdRun, err := rs.Create(s.ctx, model.CreateRunRequest{ProjectID: created.Project.ID})
		s.Require().NoError(err, file)

		got, err := rs.Get(s.ctx, model.GetRunRequest{RunID: createdRun.RunID})
		s.Require().NoError(err, file)
		s.assertRunInvariants(got.Run)
		runs[file] = got.Run
		ids[file] = got.Run.ID
	}

	mean01 := meanPathRatio(runs["data/01_full_constellation.json"])
	mean02 := meanPathRatio(runs["data/02_first_launch.json"])
	mean03 := meanPathRatio(runs["data/03_satellite_outages.json"])
	mean04 := meanPathRatio(runs["data/04_link_range.json"])
	s.T().Logf("mean path_ratio 01=%.4f 02=%.4f 03=%.4f 04=%.4f", mean01, mean02, mean03, mean04)
	for file, run := range runs {
		for _, m := range run.Metrics {
			s.T().Logf("%s %s path=%.4f vis=%.4f gap=%d hops=%.2f target=%v", file, m.ClientID, m.PathRatio, m.VisibilityRatio, m.MaxGapS, m.MeanHops, m.MeetsTarget)
		}
	}
	s.Require().Greater(mean01, mean02, "полная группировка должна быть доступнее первой очереди")
	s.Require().Greater(mean01, mean03, "отказы спутников должны снижать доступность")
	s.Require().GreaterOrEqual(mean01, mean04, "ISL 3000 км не должен быть хуже 2000 км")
	s.InDelta(0.981, mean01, 0.02)
	s.InDelta(0.186, mean02, 0.02)
	s.InDelta(0.809, mean03, 0.02)
	s.InDelta(0.683, mean04, 0.02)
	for _, m := range runs["data/01_full_constellation.json"].Metrics {
		s.True(m.MeetsTarget, m.ClientID)
	}
	for _, file := range []string{"data/02_first_launch.json", "data/03_satellite_outages.json"} {
		for _, m := range runs[file].Metrics {
			s.False(m.MeetsTarget, file+" "+m.ClientID)
		}
	}
	hasISL := false
	for _, rec := range runs["data/04_link_range.json"].Routes {
		if rec.Reason == model.GapISLPartition {
			hasISL = true
			break
		}
	}
	s.True(hasISL, "04_link_range должен показывать разрыв ISL")
	for _, m := range runs["data/04_link_range.json"].Metrics {
		s.GreaterOrEqual(m.VisibilityRatio, 0.97, m.ClientID)
		s.Greater(m.VisibilityRatio, m.PathRatio, m.ClientID)
	}

	cmp, err := rs.Compare(s.ctx, model.CompareRunsRequest{
		RunAID: ids["data/01_full_constellation.json"],
		RunBID: ids["data/02_first_launch.json"],
	})
	s.Require().NoError(err)
	s.Require().Equal("a", cmp.Recommendation.Better)
	s.Require().NotEmpty(cmp.Recommendation.Advantages)
	s.Require().NotEmpty(cmp.Recommendation.Conditions)
	s.Require().NotEmpty(cmp.Recommendation.Limitations)
	s.Require().NotEmpty(cmp.Recommendation.Conclusion)

	cmpISL, err := rs.Compare(s.ctx, model.CompareRunsRequest{
		RunAID: ids["data/01_full_constellation.json"],
		RunBID: ids["data/04_link_range.json"],
	})
	s.Require().NoError(err)
	s.Require().Contains([]string{"a", "tie"}, cmpISL.Recommendation.Better)

	run03 := runs["data/03_satellite_outages.json"]
	failed := map[string]struct{}{}
	for _, f := range run03.EffectiveScenario.Failures {
		failed[f.SatelliteID] = struct{}{}
	}
	s.Require().NotEmpty(failed)
	for _, rec := range run03.Routes {
		if rec.TS < 21600 {
			continue
		}
		for _, node := range rec.Path {
			_, ok := failed[node]
			s.Require().False(ok, "failed sat %s on path at t=%d", node, rec.TS)
		}
	}

	doc, err := rs.Export(s.ctx, model.ExportRunRequest{RunID: ids["data/01_full_constellation.json"]})
	s.Require().NoError(err)
	s.Require().Equal(model.ResultSchemaVersion, doc.SchemaVersion)
	s.Require().Len(doc.Routes, len(runs["data/01_full_constellation.json"].Routes))
	for _, rec := range doc.Routes {
		s.Require().NotNil(rec.Path)
	}

	snap, err := rs.GetSnapshot(s.ctx, model.GetSnapshotRequest{
		RunID:    ids["data/01_full_constellation.json"],
		TS:       0,
		ClientID: "C65",
	})
	s.Require().NoError(err)
	s.Require().Equal("bfs_min_hops", snap.Route.Algorithm.Name)
	s.Require().NotEmpty(snap.VisibleSatellites)
	s.Require().NotEmpty(snap.NetworkDelta.Explanation)

	satID := ""
	for _, rec := range runs["data/01_full_constellation.json"].Routes {
		if rec.TS == 0 && len(rec.Path) >= 2 {
			satID = rec.Path[1]
			break
		}
	}
	s.Require().NotEmpty(satID)
	wi, err := rs.WhatIf(s.ctx, model.WhatIfRequest{
		RunID:       ids["data/01_full_constellation.json"],
		SatelliteID: satID,
		TS:          0,
	})
	s.Require().NoError(err)
	s.Require().Equal(satID, wi.FailedSatelliteID)
	s.Require().NotEmpty(wi.Analysis.Summary)
	s.Require().NotEmpty(wi.Analysis.Mitigations)
	s.Require().Contains(wi.Compare.Recommendation.Limitations[0], "искусственным отказом")
	s.Require().Equal("a", wi.Compare.Recommendation.Better)
	s.Require().Equal(ids["data/01_full_constellation.json"], wi.Compare.Recommendation.RunID)
	s.Require().NotContains(wi.Compare.Recommendation.Conclusion, "нет единственного победителя")
}

func (s *FixtureSuite) TestHTTPSmokeFixture01() {
	projRepo := projectRepository.NewRepository()
	runRepo := runRepository.NewRepository()
	ps := projectService.NewService(projRepo, s.client)
	rs := NewService(projRepo, runRepo, s.client)

	r := chi.NewRouter()
	projectAPI := projectV1.NewAPI(ps)
	runAPI := runV1.NewAPI(rs)
	r.Get("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	r.Post("/api/projects", projectAPI.Create)
	r.Get("/api/projects/{id}", projectAPI.Get)
	r.Patch("/api/projects/{id}", projectAPI.Patch)
	r.Post("/api/projects/{id}/reset", projectAPI.Reset)
	r.Post("/api/projects/{id}/copy", projectAPI.Copy)
	r.Post("/api/projects/{id}/runs", runAPI.Create)
	r.Get("/api/runs/{id}", runAPI.Get)
	r.Get("/api/runs/{id}/metrics", runAPI.GetMetrics)
	r.Get("/api/runs/{id}/snapshot", runAPI.GetSnapshot)
	r.Get("/api/runs/{id}/export", runAPI.Export)
	r.Post("/api/runs/{id}/what-if", runAPI.WhatIf)
	r.Post("/api/compare", runAPI.Compare)

	ts := httptest.NewServer(r)
	defer ts.Close()

	health, err := http.Get(ts.URL + "/health")
	s.Require().NoError(err)
	s.Require().Equal(http.StatusOK, health.StatusCode)
	io.Copy(io.Discard, health.Body)
	health.Body.Close()

	sc := s.load("data/01_full_constellation.json")
	project := s.httpJSON(ts.URL, http.MethodPost, "/api/projects", sc, http.StatusCreated)
	projectID := project["id"].(string)
	s.Require().NotEmpty(projectID)

	runBody := s.httpJSON(ts.URL, http.MethodPost, "/api/projects/"+projectID+"/runs", nil, http.StatusCreated)
	runID := runBody["run_id"].(string)
	s.Require().NotEmpty(runID)

	metrics := s.httpJSON(ts.URL, http.MethodGet, "/api/runs/"+runID+"/metrics", nil, http.StatusOK)
	s.Require().NotEmpty(metrics)

	snap := s.httpJSON(ts.URL, http.MethodGet, "/api/runs/"+runID+"/snapshot?t_s=0&client_id=C65", nil, http.StatusOK)
	route := snap["route"].(map[string]any)
	s.Require().NotEmpty(route["path"])
	s.Require().NotEmpty(snap["visible_satellites"])

	fullRun := s.httpJSON(ts.URL, http.MethodGet, "/api/runs/"+runID, nil, http.StatusOK)
	runRoutes := fullRun["routes"].([]any)
	s.Require().NotEmpty(runRoutes)
	var gridRoute map[string]any
	for _, raw := range runRoutes {
		rec := raw.(map[string]any)
		if rec["t_s"] == float64(0) && rec["client_id"] == "C65" {
			gridRoute = rec
			break
		}
	}
	s.Require().NotNil(gridRoute, "GET /runs must include t_s=0 client C65")
	_, hasAltCount := gridRoute["alt_count"]
	s.Require().True(hasAltCount, "GET /runs must encode alt_count for unique-path ticks")
	alts, _ := route["alternatives"].([]any)
	s.Require().Equal(float64(len(alts)), gridRoute["alt_count"])

	export := s.httpJSON(ts.URL, http.MethodGet, "/api/runs/"+runID+"/export", nil, http.StatusOK)
	s.Require().Equal(model.ResultSchemaVersion, export["schema_version"])
	reimported := s.httpJSON(ts.URL, http.MethodPost, "/api/projects", export, http.StatusCreated)
	s.Require().NotEqual(projectID, reimported["id"])

	s.httpJSON(ts.URL, http.MethodGet, "/api/runs/"+runID+"/snapshot?t_s=abc", nil, http.StatusBadRequest)
	s.httpJSON(ts.URL, http.MethodPost, "/api/projects", map[string]any{"schema_version": "bad"}, http.StatusBadRequest)

	copied := s.httpJSON(ts.URL, http.MethodPost, "/api/projects/"+projectID+"/copy", nil, http.StatusCreated)
	s.Require().NotEqual(projectID, copied["id"])

	stage := 1
	s.httpJSON(ts.URL, http.MethodPatch, "/api/projects/"+projectID, model.Patch{LaunchStage: &stage}, http.StatusOK)
	runB := s.httpJSON(ts.URL, http.MethodPost, "/api/projects/"+projectID+"/runs", nil, http.StatusCreated)
	runBID := runB["run_id"].(string)

	cmp := s.httpJSON(ts.URL, http.MethodPost, "/api/compare", model.CompareRunsRequest{
		RunAID: runID,
		RunBID: runBID,
	}, http.StatusOK)
	rec := cmp["recommendation"].(map[string]any)
	s.Require().Equal("a", rec["better"])
	cfg := cmp["config_diff"].(map[string]any)
	s.Require().NotEmpty(cfg["launch_stage"])

	reset := s.httpJSON(ts.URL, http.MethodPost, "/api/projects/"+projectID+"/reset", nil, http.StatusOK)
	effective := reset["effective"].(map[string]any)
	design := effective["design"].(map[string]any)
	s.Require().Equal(float64(3), design["launch_stage"])

	path := route["path"].([]any)
	s.Require().GreaterOrEqual(len(path), 2)
	wi := s.httpJSON(ts.URL, http.MethodPost, "/api/runs/"+runID+"/what-if", model.WhatIfRequest{
		SatelliteID: path[1].(string),
		TS:          0,
	}, http.StatusCreated)
	s.Require().NotEmpty(wi["analysis"])
	cmpWI := wi["compare"].(map[string]any)
	recWI := cmpWI["recommendation"].(map[string]any)
	s.Require().Equal("a", recWI["better"])
}

func (s *FixtureSuite) assertRunInvariants(run model.Run) {
	s.T().Helper()
	sc := run.EffectiveScenario
	grid := sc.TimeGrid()
	clients := sc.ClientIDs()
	gateways := map[string]struct{}{}
	for _, id := range sc.GatewayIDs() {
		gateways[id] = struct{}{}
	}
	clientSet := map[string]struct{}{}
	for _, id := range clients {
		clientSet[id] = struct{}{}
	}
	sats := map[string]model.Satellite{}
	inactive := map[string]struct{}{}
	for _, sat := range sc.Design.Satellites {
		sats[sat.ID] = sat
		if sat.LaunchBatch > sc.Design.LaunchStage {
			inactive[sat.ID] = struct{}{}
		}
	}

	s.Require().Len(grid, sc.Environment.HorizonS/sc.Environment.StepS)
	s.Require().Len(run.Routes, len(grid)*len(clients))
	s.Require().Len(run.Metrics, len(clients))

	byClient := map[string]map[int]model.RouteRecord{}
	for _, rec := range run.Routes {
		if byClient[rec.ClientID] == nil {
			byClient[rec.ClientID] = map[int]model.RouteRecord{}
		}
		byClient[rec.ClientID][rec.TS] = rec

		_, isClient := clientSet[rec.ClientID]
		s.Require().True(isClient, rec.ClientID)
		if len(rec.Path) == 0 {
			s.Require().Contains([]model.GapReason{
				model.GapNoVisibleSat,
				model.GapNoGatewayContact,
				model.GapGatewayOutage,
				model.GapISLPartition,
			}, rec.Reason)
			s.Require().Zero(rec.Hops)
			s.Require().Zero(rec.AltCount)
			continue
		}
		s.Require().GreaterOrEqual(rec.AltCount, 0)
		s.Require().LessOrEqual(rec.AltCount, 5)
		s.Require().Empty(rec.Reason)
		s.Require().Equal(rec.ClientID, rec.Path[0])
		_, isGW := gateways[rec.Path[len(rec.Path)-1]]
		s.Require().True(isGW, rec.Path)
		s.Require().Equal(len(rec.Path)-1, rec.Hops)
		s.Require().GreaterOrEqual(rec.Hops, 2)
		for i, node := range rec.Path {
			if i == 0 {
				continue
			}
			if i == len(rec.Path)-1 {
				continue
			}
			_, isSat := sats[node]
			s.Require().True(isSat, node)
			_, off := inactive[node]
			s.Require().False(off, node)
			_, otherClient := clientSet[node]
			s.Require().False(otherClient, node)
		}
	}

	idxMetrics := map[string]model.ClientMetrics{}
	for _, m := range run.Metrics {
		idxMetrics[m.ClientID] = m
	}
	for _, clientID := range clients {
		recs := byClient[clientID]
		s.Require().Len(recs, len(grid), clientID)
		ok := 0
		for _, t := range grid {
			rec, has := recs[t]
			s.Require().True(has, "%s t=%d", clientID, t)
			if len(rec.Path) > 0 {
				ok++
			}
		}
		m := idxMetrics[clientID]
		s.Require().InDelta(float64(ok)/float64(len(grid)), m.PathRatio, 1e-9, clientID)
		s.Require().GreaterOrEqual(m.VisibilityRatio+1e-9, m.PathRatio, clientID)
		s.Require().Equal(m.PathRatio >= sc.Environment.TargetAvailability, m.MeetsTarget, clientID)
		maxGap := 0
		for _, gap := range m.Gaps {
			if gap.DurationS > maxGap {
				maxGap = gap.DurationS
			}
		}
		s.Require().Equal(m.MaxGapS, maxGap, clientID)
	}
}

func meanPathRatio(run model.Run) float64 {
	if len(run.Metrics) == 0 {
		return 0
	}
	sum := 0.0
	for _, m := range run.Metrics {
		sum += m.PathRatio
	}
	return sum / float64(len(run.Metrics))
}

func (s *FixtureSuite) httpJSON(base, method, path string, body any, want int) map[string]any {
	s.T().Helper()
	var rdr io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		s.Require().NoError(err)
		rdr = bytes.NewReader(raw)
	}
	req, err := http.NewRequest(method, base+path, rdr)
	s.Require().NoError(err)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := http.DefaultClient.Do(req)
	s.Require().NoError(err)
	defer resp.Body.Close()
	s.Require().Equal(want, resp.StatusCode, path)
	if want >= 400 {
		return nil
	}
	var out any
	s.Require().NoError(json.NewDecoder(resp.Body).Decode(&out))
	switch v := out.(type) {
	case map[string]any:
		return v
	default:
		return map[string]any{"value": v}
	}
}
