package project

import (
	"fmt"

	"github.com/mllbll/kosmohak_nn/internal/model"
)

func ValidateScenario(sc model.Scenario) error {
	if sc.SchemaVersion != model.SchemaVersion {
		return fmt.Errorf("%w: schema_version expected %s", model.ErrInvalidArgument, model.SchemaVersion)
	}
	if sc.Environment.StepS <= 0 || sc.Environment.HorizonS <= 0 {
		return fmt.Errorf("%w: horizon_s and step_s must be positive", model.ErrInvalidArgument)
	}
	if sc.Environment.HorizonS%sc.Environment.StepS != 0 {
		return fmt.Errorf("%w: horizon_s must be divisible by step_s", model.ErrInvalidArgument)
	}
	if sc.Design.LaunchStage < 1 || sc.Design.LaunchStage > 3 {
		return fmt.Errorf("%w: launch_stage must be 1, 2 or 3", model.ErrInvalidArgument)
	}
	if len(sc.ClientIDs()) == 0 || len(sc.GatewayIDs()) == 0 {
		return fmt.Errorf("%w: client and gateway required", model.ErrInvalidArgument)
	}
	return nil
}

func ApplyPatch(sc model.Scenario, p model.Patch) (model.Scenario, error) {
	sc = model.CloneScenario(sc)
	if p.LaunchStage != nil {
		sc.Design.LaunchStage = *p.LaunchStage
	}
	if p.Failures != nil {
		sc.Failures = append([]model.Failure{}, *p.Failures...)
	}
	if p.GatewayOutages != nil {
		sc.GatewayOutages = append([]model.GatewayOutage{}, *p.GatewayOutages...)
	}
	if len(p.Planes) > 0 {
		idx := map[string]int{}
		for i, pl := range sc.Design.Planes {
			idx[pl.ID] = i
		}
		for _, pp := range p.Planes {
			i, ok := idx[pp.ID]
			if !ok {
				return model.Scenario{}, fmt.Errorf("%w: unknown plane id %q", model.ErrInvalidArgument, pp.ID)
			}
			if pp.RAANDeg != nil {
				sc.Design.Planes[i].RAANDeg = *pp.RAANDeg
			}
			if pp.PhaseDeg != nil {
				sc.Design.Planes[i].PhaseDeg = *pp.PhaseDeg
			}
		}
	}
	if err := validateOutages(sc); err != nil {
		return model.Scenario{}, err
	}
	return sc, ValidateScenario(sc)
}

func validateOutages(sc model.Scenario) error {
	sats := map[string]struct{}{}
	for _, sat := range sc.Design.Satellites {
		sats[sat.ID] = struct{}{}
	}
	gws := map[string]struct{}{}
	for _, id := range sc.GatewayIDs() {
		gws[id] = struct{}{}
	}
	horizon := float64(sc.Environment.HorizonS)
	for i, f := range sc.Failures {
		if _, ok := sats[f.SatelliteID]; !ok {
			return fmt.Errorf("%w: failures[%d].satellite_id: unknown id %q", model.ErrInvalidArgument, i, f.SatelliteID)
		}
		if !(0 <= f.StartS && f.StartS < f.EndS && f.EndS <= horizon) {
			return fmt.Errorf("%w: failures[%d]: interval must be inside [0, horizon_s]", model.ErrInvalidArgument, i)
		}
	}
	for i, o := range sc.GatewayOutages {
		if _, ok := gws[o.GatewayID]; !ok {
			return fmt.Errorf("%w: gateway_outages[%d].gateway_id: unknown id %q", model.ErrInvalidArgument, i, o.GatewayID)
		}
		if !(0 <= o.StartS && o.StartS < o.EndS && o.EndS <= horizon) {
			return fmt.Errorf("%w: gateway_outages[%d]: interval must be inside [0, horizon_s]", model.ErrInvalidArgument, i)
		}
	}
	return nil
}
