package model

const SchemaVersion = "cosmo-A-1.0"
const ResultSchemaVersion = "cosmo-A-result-1.0"

// Scenario входной проект группировки (cosmo-A-1.0)
type Scenario struct {
	SchemaVersion  string          `json:"schema_version"`
	Meta           Meta            `json:"meta"`
	Environment    Environment     `json:"environment"`
	Design         Design          `json:"design"`
	GroundSites    []GroundSite    `json:"ground_sites"`
	Failures       []Failure       `json:"failures"`
	GatewayOutages []GatewayOutage `json:"gateway_outages"`
}

type Meta struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

type Environment struct {
	AltitudeKM         float64 `json:"altitude_km"`
	InclinationDeg     float64 `json:"inclination_deg"`
	EarthAngle0Deg     float64 `json:"earth_angle0_deg"`
	HorizonS           int     `json:"horizon_s"`
	StepS              int     `json:"step_s"`
	MinElevationDeg    float64 `json:"min_elevation_deg"`
	ISLRangeKM         float64 `json:"isl_range_km"`
	TargetAvailability float64 `json:"target_availability"`
}

type Design struct {
	LaunchStage int         `json:"launch_stage"`
	Planes      []Plane     `json:"planes"`
	Satellites  []Satellite `json:"satellites"`
}

type Plane struct {
	ID       string  `json:"id"`
	RAANDeg  float64 `json:"raan_deg"`
	PhaseDeg float64 `json:"phase_deg"`
}

type Satellite struct {
	ID          string  `json:"id"`
	PlaneID     string  `json:"plane_id"`
	SlotDeg     float64 `json:"slot_deg"`
	LaunchBatch int     `json:"launch_batch"`
}

type GroundSite struct {
	ID     string  `json:"id"`
	Name   string  `json:"name"`
	Role   string  `json:"role"`
	LatDeg float64 `json:"lat_deg"`
	LonDeg float64 `json:"lon_deg"`
}

type Failure struct {
	SatelliteID string  `json:"satellite_id"`
	StartS      float64 `json:"start_s"`
	EndS        float64 `json:"end_s"`
}

type GatewayOutage struct {
	GatewayID string  `json:"gateway_id"`
	StartS    float64 `json:"start_s"`
	EndS      float64 `json:"end_s"`
}

func (s Scenario) TimeGrid() []int {
	out := make([]int, 0, s.Environment.HorizonS/max(s.Environment.StepS, 1))
	for t := 0; t < s.Environment.HorizonS; t += s.Environment.StepS {
		out = append(out, t)
	}
	return out
}

func (s Scenario) ClientIDs() []string {
	ids := make([]string, 0)
	for _, g := range s.GroundSites {
		if g.Role == "client" {
			ids = append(ids, g.ID)
		}
	}
	return ids
}

func (s Scenario) GatewayIDs() []string {
	ids := make([]string, 0)
	for _, g := range s.GroundSites {
		if g.Role == "gateway" {
			ids = append(ids, g.ID)
		}
	}
	return ids
}

func (s Scenario) IsGatewayOutaged(gatewayID string, t float64) bool {
	for _, o := range s.GatewayOutages {
		if o.GatewayID == gatewayID && o.StartS <= t && t < o.EndS {
			return true
		}
	}
	return false
}

func CloneScenario(sc Scenario) Scenario {
	out := sc
	out.Design.Planes = append([]Plane{}, sc.Design.Planes...)
	out.Design.Satellites = append([]Satellite{}, sc.Design.Satellites...)
	out.GroundSites = append([]GroundSite{}, sc.GroundSites...)
	out.Failures = append([]Failure{}, sc.Failures...)
	out.GatewayOutages = append([]GatewayOutage{}, sc.GatewayOutages...)
	return out
}
