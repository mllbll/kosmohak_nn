package model

type GapReason string

const (
	GapNone             GapReason = ""
	GapNoVisibleSat     GapReason = "no_visible_sat"
	GapNoGatewayContact GapReason = "no_gateway_contact"
	GapGatewayOutage    GapReason = "gateway_outage"
	GapISLPartition     GapReason = "isl_partition"
)

type RouteRecord struct {
	TS       int       `json:"t_s"`
	ClientID string    `json:"client_id"`
	Path     []string  `json:"path"`
	Reason   GapReason `json:"reason,omitempty"`
	Hops     int       `json:"hops,omitempty"`
}

type RouteResult struct {
	Path   []string  `json:"path"`
	Reason GapReason `json:"reason,omitempty"`
	Hops   int       `json:"hops,omitempty"`
}

type ClientMetrics struct {
	ClientID        string  `json:"client_id"`
	VisibilityRatio float64 `json:"visibility_ratio"`
	PathRatio       float64 `json:"path_ratio"`
	MaxGapS         int     `json:"max_gap_s"`
	MeanHops        float64 `json:"mean_hops"`
	MeetsTarget     bool    `json:"meets_target"`
}

type Run struct {
	ID                string          `json:"id"`
	ProjectID         string          `json:"project_id"`
	EffectiveScenario Scenario        `json:"effective_scenario"`
	Routes            []RouteRecord   `json:"routes"`
	Metrics           []ClientMetrics `json:"metrics"`
	Summary           string          `json:"summary,omitempty"`
}

type CreateRunRequest struct {
	ProjectID string
}

type CreateRunResponse struct {
	RunID     string          `json:"run_id"`
	ProjectID string          `json:"project_id"`
	Metrics   []ClientMetrics `json:"metrics"`
}

type GetRunRequest struct {
	RunID string
}

type GetRunResponse struct {
	Run Run
}

type GetSnapshotRequest struct {
	RunID    string
	TS       float64
	ClientID string
}

type GetSnapshotResponse struct {
	Snapshot Snapshot    `json:"snapshot"`
	Route    RouteResult `json:"route"`
}

type ExportRunRequest struct {
	RunID string
}

type ExportDocument struct {
	SchemaVersion     string          `json:"schema_version"`
	EffectiveScenario Scenario        `json:"effective_scenario"`
	Routes            []ExportRoute   `json:"routes"`
	Metrics           []ClientMetrics `json:"metrics,omitempty"`
	Summary           string          `json:"summary,omitempty"`
}

type ExportRoute struct {
	TS       int      `json:"t_s"`
	ClientID string   `json:"client_id"`
	Path     []string `json:"path"`
}

type CompareRunsRequest struct {
	RunAID string `json:"run_a"`
	RunBID string `json:"run_b"`
}

type CompareRunsResponse struct {
	RunAID  string         `json:"run_a_id"`
	RunBID  string         `json:"run_b_id"`
	Config  map[string]any `json:"config_diff"`
	Clients []ClientDiff   `json:"clients"`
}

type ClientDiff struct {
	ClientID       string  `json:"client_id"`
	PathRatioA     float64 `json:"path_ratio_a"`
	PathRatioB     float64 `json:"path_ratio_b"`
	DeltaPathRatio float64 `json:"delta_path_ratio"`
	MaxGapSA       int     `json:"max_gap_s_a"`
	MaxGapSB       int     `json:"max_gap_s_b"`
	DeltaMaxGapS   int     `json:"delta_max_gap_s"`
}
