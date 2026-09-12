package project

import (
	"fmt"
	"math"

	"github.com/mllbll/kosmohak_nn/internal/model"
)

func ValidateScenario(sc model.Scenario) error {
	if sc.SchemaVersion != model.SchemaVersion {
		return fmt.Errorf("%w: schema_version expected %s", model.ErrInvalidArgument, model.SchemaVersion)
	}

	if err := validateEnvironment(sc.Environment); err != nil {
		return err
	}

	if sc.Design.LaunchStage < 1 || sc.Design.LaunchStage > 3 {
		return fmt.Errorf("%w: launch_stage must be 1, 2 or 3", model.ErrInvalidArgument)
	}

	planes := map[string]struct{}{}
	for i, p := range sc.Design.Planes {
		if p.ID == "" {
			return fmt.Errorf("%w: planes[%d].id is empty", model.ErrInvalidArgument, i)
		}
		if _, ok := planes[p.ID]; ok {
			return fmt.Errorf("%w: duplicate plane id %q", model.ErrInvalidArgument, p.ID)
		}
		planes[p.ID] = struct{}{}
		if !finite(p.RAANDeg) {
			return fmt.Errorf("%w: planes[%d].raan_deg must be finite", model.ErrInvalidArgument, i)
		}
		if p.RAANDeg < 0 || p.RAANDeg >= 360 {
			return fmt.Errorf("%w: planes[%d].raan_deg must be in [0, 360)", model.ErrInvalidArgument, i)
		}
		if !finite(p.PhaseDeg) {
			return fmt.Errorf("%w: planes[%d].phase_deg must be finite", model.ErrInvalidArgument, i)
		}
		if p.PhaseDeg < 0 || p.PhaseDeg >= 360 {
			return fmt.Errorf("%w: planes[%d].phase_deg must be in [0, 360)", model.ErrInvalidArgument, i)
		}
	}
	if len(planes) == 0 {
		return fmt.Errorf("%w: planes required", model.ErrInvalidArgument)
	}

	sats := map[string]struct{}{}
	for i, sat := range sc.Design.Satellites {
		if sat.ID == "" {
			return fmt.Errorf("%w: satellites[%d].id is empty", model.ErrInvalidArgument, i)
		}
		if _, ok := sats[sat.ID]; ok {
			return fmt.Errorf("%w: duplicate satellite id %q", model.ErrInvalidArgument, sat.ID)
		}
		sats[sat.ID] = struct{}{}
		if _, ok := planes[sat.PlaneID]; !ok {
			return fmt.Errorf("%w: satellite %q: unknown plane_id %q", model.ErrInvalidArgument, sat.ID, sat.PlaneID)
		}
		if sat.LaunchBatch < 1 || sat.LaunchBatch > 3 {
			return fmt.Errorf("%w: satellites[%d].launch_batch must be 1, 2 or 3", model.ErrInvalidArgument, i)
		}
		if !finite(sat.SlotDeg) {
			return fmt.Errorf("%w: satellites[%d].slot_deg must be finite", model.ErrInvalidArgument, i)
		}
	}
	if len(sats) == 0 {
		return fmt.Errorf("%w: satellites required", model.ErrInvalidArgument)
	}

	gids := map[string]struct{}{}
	for i, g := range sc.GroundSites {
		if g.ID == "" {
			return fmt.Errorf("%w: ground_sites[%d].id is empty", model.ErrInvalidArgument, i)
		}
		if _, ok := gids[g.ID]; ok {
			return fmt.Errorf("%w: duplicate ground site id %q", model.ErrInvalidArgument, g.ID)
		}
		if _, ok := sats[g.ID]; ok {
			return fmt.Errorf("%w: ground site id %q collides with satellite", model.ErrInvalidArgument, g.ID)
		}
		gids[g.ID] = struct{}{}
		if g.Role != "client" && g.Role != "gateway" {
			return fmt.Errorf("%w: ground_sites[%d].role must be client or gateway", model.ErrInvalidArgument, i)
		}
		if !finite(g.LatDeg) {
			return fmt.Errorf("%w: ground_sites[%d].lat_deg must be finite", model.ErrInvalidArgument, i)
		}
		if g.LatDeg < -90 || g.LatDeg > 90 {
			return fmt.Errorf("%w: ground_sites[%d].lat_deg must be in [-90, 90]", model.ErrInvalidArgument, i)
		}
		if !finite(g.LonDeg) {
			return fmt.Errorf("%w: ground_sites[%d].lon_deg must be finite", model.ErrInvalidArgument, i)
		}
		if g.LonDeg < -180 || g.LonDeg > 180 {
			return fmt.Errorf("%w: ground_sites[%d].lon_deg must be in [-180, 180]", model.ErrInvalidArgument, i)
		}
	}

	if len(sc.ClientIDs()) == 0 || len(sc.GatewayIDs()) == 0 {
		return fmt.Errorf("%w: client and gateway required", model.ErrInvalidArgument)
	}

	return validateOutages(sc)
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
	return sc, ValidateScenario(sc)
}

func validateEnvironment(e model.Environment) error {
	fields := []struct {
		path string
		v    float64
	}{
		{"environment.altitude_km", e.AltitudeKM},
		{"environment.inclination_deg", e.InclinationDeg},
		{"environment.earth_angle0_deg", e.EarthAngle0Deg},
		{"environment.min_elevation_deg", e.MinElevationDeg},
		{"environment.isl_range_km", e.ISLRangeKM},
		{"environment.target_availability", e.TargetAvailability},
	}
	for _, f := range fields {
		if !finite(f.v) {
			return fmt.Errorf("%w: %s must be finite", model.ErrInvalidArgument, f.path)
		}
	}

	if e.StepS <= 0 {
		return fmt.Errorf("%w: environment.step_s must be > 0", model.ErrInvalidArgument)
	}
	if e.HorizonS <= 0 {
		return fmt.Errorf("%w: environment.horizon_s must be > 0", model.ErrInvalidArgument)
	}
	if e.HorizonS > 172800 {
		return fmt.Errorf("%w: environment.horizon_s must be <= 172800", model.ErrInvalidArgument)
	}
	if e.StepS > e.HorizonS {
		return fmt.Errorf("%w: environment.step_s must be <= environment.horizon_s", model.ErrInvalidArgument)
	}
	if e.HorizonS%e.StepS != 0 {
		return fmt.Errorf("%w: environment.horizon_s must be a multiple of environment.step_s", model.ErrInvalidArgument)
	}

	if e.AltitudeKM < 200 || e.AltitudeKM > 1200 {
		return fmt.Errorf("%w: environment.altitude_km must be in [200, 1200]", model.ErrInvalidArgument)
	}
	if e.InclinationDeg <= 0 || e.InclinationDeg > 180 {
		return fmt.Errorf("%w: environment.inclination_deg must be in (0, 180]", model.ErrInvalidArgument)
	}
	if e.MinElevationDeg < 0 || e.MinElevationDeg >= 90 {
		return fmt.Errorf("%w: environment.min_elevation_deg must be in [0, 90)", model.ErrInvalidArgument)
	}
	if e.ISLRangeKM <= 0 || e.ISLRangeKM > 10000 {
		return fmt.Errorf("%w: environment.isl_range_km must be in (0, 10000]", model.ErrInvalidArgument)
	}
	if e.TargetAvailability < 0 || e.TargetAvailability > 1 {
		return fmt.Errorf("%w: environment.target_availability must be in [0, 1]", model.ErrInvalidArgument)
	}
	return nil
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
		if !finite(f.StartS) || !finite(f.EndS) || !(0 <= f.StartS && f.StartS < f.EndS && f.EndS <= horizon) {
			return fmt.Errorf("%w: failures[%d]: interval must be inside [0, horizon_s]", model.ErrInvalidArgument, i)
		}
	}
	for i, o := range sc.GatewayOutages {
		if _, ok := gws[o.GatewayID]; !ok {
			return fmt.Errorf("%w: gateway_outages[%d].gateway_id: unknown id %q", model.ErrInvalidArgument, i, o.GatewayID)
		}
		if !finite(o.StartS) || !finite(o.EndS) || !(0 <= o.StartS && o.StartS < o.EndS && o.EndS <= horizon) {
			return fmt.Errorf("%w: gateway_outages[%d]: interval must be inside [0, horizon_s]", model.ErrInvalidArgument, i)
		}
	}
	return nil
}

func finite(x float64) bool {
	return !math.IsNaN(x) && !math.IsInf(x, 0)
}
