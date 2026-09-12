package run

import (
	"context"
	"testing"

	"github.com/brianvoe/gofakeit/v7"
	geometryMocks "github.com/mllbll/kosmohak_nn/internal/client/geometry/mocks"
	"github.com/mllbll/kosmohak_nn/internal/model"
	"github.com/mllbll/kosmohak_nn/internal/repository/mocks"
	"github.com/stretchr/testify/suite"
)

type ServiceSuite struct {
	suite.Suite

	ctx context.Context

	projectRepository *mocks.ProjectRepository

	runRepository *mocks.RunRepository

	geometryClient *geometryMocks.GeometryClient

	service *service
}

func (s *ServiceSuite) SetupTest() {
	s.ctx = context.Background()

	s.projectRepository = mocks.NewProjectRepository(s.T())
	s.runRepository = mocks.NewRunRepository(s.T())
	s.geometryClient = geometryMocks.NewGeometryClient(s.T())

	s.service = NewService(
		s.projectRepository,
		s.runRepository,
		s.geometryClient,
	)
}

func (s *ServiceSuite) TearDownTest() {
}

func TestServiceIntegration(t *testing.T) {
	suite.Run(t, new(ServiceSuite))
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

func testSnapshot() model.Snapshot {
	return model.Snapshot{
		TS: 0,
		Satellites: []model.SatState{
			{ID: "S01", Active: true},
		},
		Edges: []model.Edge{
			{A: "C65", B: "S01", DistanceKM: 100},
			{A: "S01", B: "G_MUR", DistanceKM: 100},
		},
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
			{ClientID: "C65", PathRatio: 1, VisibilityRatio: 1, MeanHops: 2, MeetsTarget: true, Gaps: []model.GapInterval{}},
		},
	}
}
