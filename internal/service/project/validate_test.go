package project

import (
	"encoding/json"
	"math"

	"github.com/mllbll/kosmohak_nn/internal/model"
)

func (s *ServiceSuite) TestApplyPatchDoesNotMutateOriginal() {
	var (
		sc          = testScenario()
		launchStage = 1
	)

	patched, err := ApplyPatch(sc, model.Patch{LaunchStage: &launchStage})

	s.Require().NoError(err)
	s.Require().Equal(3, sc.Design.LaunchStage)
	s.Require().Equal(launchStage, patched.Design.LaunchStage)
}

func (s *ServiceSuite) TestValidateScenarioRejectsUnknownFailure() {
	sc := testScenario()
	sc.Failures = []model.Failure{{
		SatelliteID: "UNKNOWN",
		StartS:      0,
		EndS:        120,
	}}

	err := ValidateScenario(sc)

	s.Require().Error(err)
	s.Require().ErrorIs(err, model.ErrInvalidArgument)
	s.Require().Contains(err.Error(), "failures[0].satellite_id")
}

func (s *ServiceSuite) TestValidateScenarioRejectsPlaneAngle() {
	sc := testScenario()
	sc.Design.Planes[0].RAANDeg = 360

	err := ValidateScenario(sc)

	s.Require().Error(err)
	s.Require().ErrorIs(err, model.ErrInvalidArgument)
	s.Require().Equal("invalid argument: planes[0].raan_deg must be in [0, 360)", err.Error())
}

func (s *ServiceSuite) TestValidateScenarioRejectsBadOrbit() {
	sc := testScenario()
	sc.Environment.AltitudeKM = 50

	err := ValidateScenario(sc)

	s.Require().Error(err)
	s.Require().ErrorIs(err, model.ErrInvalidArgument)
	s.Require().Equal("invalid argument: environment.altitude_km must be in [200, 1200]", err.Error())
}

func (s *ServiceSuite) TestValidateScenarioAcceptsRangeBoundaries() {
	sc := testScenario()
	sc.Environment.AltitudeKM = 200
	sc.Environment.InclinationDeg = 180
	sc.Environment.MinElevationDeg = 0
	sc.Environment.ISLRangeKM = 10000
	sc.Environment.TargetAvailability = 0
	sc.Design.Planes[0].RAANDeg = 0
	sc.Design.Planes[0].PhaseDeg = 0
	s.Require().NoError(ValidateScenario(sc))

	sc = testScenario()
	sc.Environment.AltitudeKM = 1200
	sc.Environment.TargetAvailability = 1
	sc.Environment.HorizonS = 172800
	sc.Environment.StepS = 120
	s.Require().NoError(ValidateScenario(sc))
}

func (s *ServiceSuite) TestValidateScenarioNamesProblemField() {
	cases := []struct {
		name string
		mut  func(*model.Scenario)
		msg  string
	}{
		{
			name: "non_finite_altitude",
			mut:  func(sc *model.Scenario) { sc.Environment.AltitudeKM = math.NaN() },
			msg:  "environment.altitude_km must be finite",
		},
		{
			name: "non_finite_earth_angle",
			mut:  func(sc *model.Scenario) { sc.Environment.EarthAngle0Deg = math.Inf(1) },
			msg:  "environment.earth_angle0_deg must be finite",
		},
		{
			name: "step_not_positive",
			mut:  func(sc *model.Scenario) { sc.Environment.StepS = 0 },
			msg:  "environment.step_s must be > 0",
		},
		{
			name: "horizon_not_positive",
			mut:  func(sc *model.Scenario) { sc.Environment.HorizonS = 0 },
			msg:  "environment.horizon_s must be > 0",
		},
		{
			name: "horizon_too_large",
			mut: func(sc *model.Scenario) {
				sc.Environment.HorizonS = 172801
				sc.Environment.StepS = 1
			},
			msg: "environment.horizon_s must be <= 172800",
		},
		{
			name: "step_greater_than_horizon",
			mut: func(sc *model.Scenario) {
				sc.Environment.HorizonS = 120
				sc.Environment.StepS = 240
			},
			msg: "environment.step_s must be <= environment.horizon_s",
		},
		{
			name: "horizon_not_multiple_of_step",
			mut: func(sc *model.Scenario) {
				sc.Environment.HorizonS = 180
				sc.Environment.StepS = 120
			},
			msg: "environment.horizon_s must be a multiple of environment.step_s",
		},
		{
			name: "inclination",
			mut:  func(sc *model.Scenario) { sc.Environment.InclinationDeg = 0 },
			msg:  "environment.inclination_deg must be in (0, 180]",
		},
		{
			name: "min_elevation",
			mut:  func(sc *model.Scenario) { sc.Environment.MinElevationDeg = 90 },
			msg:  "environment.min_elevation_deg must be in [0, 90)",
		},
		{
			name: "isl_range",
			mut:  func(sc *model.Scenario) { sc.Environment.ISLRangeKM = 0 },
			msg:  "environment.isl_range_km must be in (0, 10000]",
		},
		{
			name: "target_availability",
			mut:  func(sc *model.Scenario) { sc.Environment.TargetAvailability = 1.1 },
			msg:  "environment.target_availability must be in [0, 1]",
		},
		{
			name: "phase_deg",
			mut:  func(sc *model.Scenario) { sc.Design.Planes[0].PhaseDeg = 360 },
			msg:  "planes[0].phase_deg must be in [0, 360)",
		},
		{
			name: "launch_batch",
			mut:  func(sc *model.Scenario) { sc.Design.Satellites[0].LaunchBatch = 0 },
			msg:  "satellites[0].launch_batch must be 1, 2 or 3",
		},
		{
			name: "slot_deg",
			mut:  func(sc *model.Scenario) { sc.Design.Satellites[0].SlotDeg = math.NaN() },
			msg:  "satellites[0].slot_deg must be finite",
		},
		{
			name: "lat_deg",
			mut:  func(sc *model.Scenario) { sc.GroundSites[0].LatDeg = 91 },
			msg:  "ground_sites[0].lat_deg must be in [-90, 90]",
		},
		{
			name: "lon_deg",
			mut:  func(sc *model.Scenario) { sc.GroundSites[1].LonDeg = 181 },
			msg:  "ground_sites[1].lon_deg must be in [-180, 180]",
		},
	}

	for _, tc := range cases {
		s.Run(tc.name, func() {
			sc := testScenario()
			tc.mut(&sc)
			err := ValidateScenario(sc)
			s.Require().Error(err)
			s.Require().ErrorIs(err, model.ErrInvalidArgument)
			s.Require().Equal("invalid argument: "+tc.msg, err.Error())
		})
	}
}

func (s *ServiceSuite) TestApplyPatchRejectsUnknownOutageGateway() {
	sc := testScenario()
	outages := []model.GatewayOutage{{
		GatewayID: "UNKNOWN",
		StartS:    0,
		EndS:      120,
	}}

	_, err := ApplyPatch(sc, model.Patch{GatewayOutages: &outages})

	s.Require().Error(err)
	s.Require().ErrorIs(err, model.ErrInvalidArgument)
	s.Require().Contains(err.Error(), "gateway_outages[0].gateway_id")
}

func (s *ServiceSuite) TestApplyPatchRejectsInvalidPlaneAngle() {
	sc := testScenario()
	raan := 360.0

	_, err := ApplyPatch(sc, model.Patch{Planes: []model.PlanePatch{
		{ID: "P1", RAANDeg: &raan},
	}})

	s.Require().Error(err)
	s.Require().ErrorIs(err, model.ErrInvalidArgument)
	s.Require().Equal("invalid argument: planes[0].raan_deg must be in [0, 360)", err.Error())
}

func (s *ServiceSuite) TestValidateScenarioRejectsOutagePastHorizon() {
	sc := testScenario()
	sc.Failures = []model.Failure{{
		SatelliteID: "S01",
		StartS:      0,
		EndS:        240,
	}}

	err := ValidateScenario(sc)

	s.Require().Error(err)
	s.Require().ErrorIs(err, model.ErrInvalidArgument)
	s.Require().Contains(err.Error(), "failures[0]")
}

func (s *ServiceSuite) TestCloneScenarioEncodesEmptySlices() {
	sc := testScenario()
	sc.Failures = nil
	sc.GatewayOutages = nil

	raw, err := json.Marshal(model.CloneScenario(sc))

	s.Require().NoError(err)
	s.Require().Contains(string(raw), `"failures":[]`)
	s.Require().Contains(string(raw), `"gateway_outages":[]`)
	s.Require().NotContains(string(raw), `"failures":null`)
	s.Require().NotContains(string(raw), `"gateway_outages":null`)
}
