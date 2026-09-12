package project

import (
	"encoding/json"

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
}

func (s *ServiceSuite) TestValidateScenarioRejectsPlaneAngle() {
	sc := testScenario()
	sc.Design.Planes[0].RAANDeg = 360

	err := ValidateScenario(sc)

	s.Require().Error(err)
	s.Require().ErrorIs(err, model.ErrInvalidArgument)
}

func (s *ServiceSuite) TestValidateScenarioRejectsBadOrbit() {
	sc := testScenario()
	sc.Environment.AltitudeKM = 50

	err := ValidateScenario(sc)

	s.Require().Error(err)
	s.Require().ErrorIs(err, model.ErrInvalidArgument)
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
