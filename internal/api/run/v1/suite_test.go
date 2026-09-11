package v1

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/brianvoe/gofakeit/v7"
	"github.com/go-chi/chi/v5"
	"github.com/mllbll/kosmohak_nn/internal/model"
	"github.com/mllbll/kosmohak_nn/internal/service/mocks"
	"github.com/stretchr/testify/suite"
)

type APISuite struct {
	suite.Suite

	ctx context.Context

	runService *mocks.RunService

	api *api
}

func (s *APISuite) SetupTest() {
	s.ctx = context.Background()

	s.runService = mocks.NewRunService(s.T())

	s.api = NewAPI(
		s.runService,
	)
}

func (s *APISuite) TearDownTest() {
}

func TestAPIIntegration(t *testing.T) {
	suite.Run(t, new(APISuite))
}

func testScenario() model.Scenario {
	return model.Scenario{
		SchemaVersion: model.SchemaVersion,
		Meta: model.Meta{
			ID:    gofakeit.UUID(),
			Title: gofakeit.Word(),
		},
		Environment: model.Environment{
			AltitudeKM:         550,
			InclinationDeg:     87,
			HorizonS:           120,
			StepS:              120,
			MinElevationDeg:    10,
			ISLRangeKM:         3000,
			TargetAvailability: 0.9,
		},
		Design: model.Design{
			LaunchStage: 3,
			Planes: []model.Plane{
				{ID: "P1", RAANDeg: 0, PhaseDeg: 0},
			},
			Satellites: []model.Satellite{
				{ID: "S01", PlaneID: "P1", SlotDeg: 0, LaunchBatch: 1},
			},
		},
		GroundSites: []model.GroundSite{
			{ID: "C65", Role: "client", LatDeg: 65, LonDeg: 60},
			{ID: "G_MUR", Role: "gateway", LatDeg: 68.97, LonDeg: 33.07},
		},
		Failures:       []model.Failure{},
		GatewayOutages: []model.GatewayOutage{},
	}
}

func testRun(projectID string) model.Run {
	sc := testScenario()
	return model.Run{
		ID:                gofakeit.UUID(),
		ProjectID:         projectID,
		EffectiveScenario: sc,
		Routes: []model.RouteRecord{
			{TS: 0, ClientID: "C65", Path: []string{"C65", "S01", "G_MUR"}, Hops: 2},
		},
		Metrics: []model.ClientMetrics{
			{ClientID: "C65", PathRatio: 1, VisibilityRatio: 1, MeetsTarget: true},
		},
	}
}

func (s *APISuite) newRequest(method, path, id string, body any) (*httptest.ResponseRecorder, *http.Request) {
	var rdr io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		s.Require().NoError(err)
		rdr = bytes.NewReader(b)
	}

	req := httptest.NewRequest(method, path, rdr)
	if id != "" {
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("id", id)
		req = req.WithContext(context.WithValue(s.ctx, chi.RouteCtxKey, rctx))
	} else {
		req = req.WithContext(s.ctx)
	}

	return httptest.NewRecorder(), req
}
