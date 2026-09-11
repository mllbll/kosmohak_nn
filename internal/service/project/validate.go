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
	if p.LaunchStage != nil {
		sc.Design.LaunchStage = *p.LaunchStage
	}
	if p.Failures != nil {
		sc.Failures = *p.Failures
	}
	if p.GatewayOutages != nil {
		sc.GatewayOutages = *p.GatewayOutages
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
	return sc, ValidateScenario(sc)
}
