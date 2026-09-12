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
	Path         []string       `json:"path"`
	Reason       GapReason      `json:"reason,omitempty"`
	Hops         int            `json:"hops,omitempty"`
	Alternatives [][]string     `json:"alternatives,omitempty"`
	Algorithm    RouteAlgorithm `json:"algorithm"`
}

type RouteAlgorithm struct {
	Name        string   `json:"name"`
	Objective   string   `json:"objective"`
	Constraints []string `json:"constraints"`
	Rationale   string   `json:"rationale"`
	MinHops     int      `json:"min_hops,omitempty"`
}

type NetworkDelta struct {
	Changed            bool     `json:"changed"`
	PreviousStillValid bool     `json:"previous_still_valid"`
	PreviousPath       []string `json:"previous_path,omitempty"`
	CurrentPath        []string `json:"current_path,omitempty"`
	Explanation        string   `json:"explanation"`
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
	Snapshot     Snapshot     `json:"snapshot"`
	Route        RouteResult  `json:"route"`
	NetworkDelta NetworkDelta `json:"network_delta"`
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
	RunAID string   `json:"run_a"`
	RunBID string   `json:"run_b"`
	RunIDs []string `json:"run_ids"`
}

type CompareRunsResponse struct {
	RunAID         string                `json:"run_a_id,omitempty"`
	RunBID         string                `json:"run_b_id,omitempty"`
	Config         map[string]any        `json:"config_diff,omitempty"`
	Variants       []CompareVariant      `json:"variants"`
	Clients        []ClientDiff          `json:"clients"`
	Recommendation CompareRecommendation `json:"recommendation"`
}

type CompareVariant struct {
	RunID                string          `json:"run_id"`
	ProjectID            string          `json:"project_id"`
	Title                string          `json:"title"`
	LaunchStage          int             `json:"launch_stage"`
	Planes               []Plane         `json:"planes"`
	ClientsMeetingTarget int             `json:"clients_meeting_target"`
	ClientsTotal         int             `json:"clients_total"`
	MeanPathRatio        float64         `json:"mean_path_ratio"`
	MeanMaxGapS          float64         `json:"mean_max_gap_s"`
	Metrics              []ClientMetrics `json:"metrics"`
	Summary              string          `json:"summary,omitempty"`
}

type CompareRecommendation struct {
	RunID       string   `json:"run_id,omitempty"`
	Better      string   `json:"better"`
	Reason      string   `json:"reason"`
	Advantages  []string `json:"advantages"`
	Conditions  []string `json:"conditions"`
	Limitations []string `json:"limitations"`
	Conclusion  string   `json:"conclusion"`
}

type ClientDiff struct {
	ClientID         string            `json:"client_id"`
	Better           string            `json:"better"`
	PathRatioA       float64           `json:"path_ratio_a"`
	PathRatioB       float64           `json:"path_ratio_b"`
	DeltaPathRatio   float64           `json:"delta_path_ratio"`
	VisibilityRatioA float64           `json:"visibility_ratio_a"`
	VisibilityRatioB float64           `json:"visibility_ratio_b"`
	MaxGapSA         int               `json:"max_gap_s_a"`
	MaxGapSB         int               `json:"max_gap_s_b"`
	DeltaMaxGapS     int               `json:"delta_max_gap_s"`
	MeetsTargetA     bool              `json:"meets_target_a"`
	MeetsTargetB     bool              `json:"meets_target_b"`
	ByRun            []ClientRunMetric `json:"by_run"`
}

type ClientRunMetric struct {
	RunID           string  `json:"run_id"`
	PathRatio       float64 `json:"path_ratio"`
	VisibilityRatio float64 `json:"visibility_ratio"`
	MaxGapS         int     `json:"max_gap_s"`
	MeanHops        float64 `json:"mean_hops"`
	MeetsTarget     bool    `json:"meets_target"`
}

type WhatIfRequest struct {
	RunID       string   `json:"-"`
	ClientID    string   `json:"client_id"`
	TS          float64  `json:"t_s"`
	SatelliteID string   `json:"satellite_id"`
	GatewayID   string   `json:"gateway_id"`
	StartS      *float64 `json:"start_s"`
	EndS        *float64 `json:"end_s"`
}

type WhatIfResponse struct {
	OriginalRunID     string              `json:"original_run_id"`
	ProjectID         string              `json:"project_id"`
	RunID             string              `json:"run_id"`
	FailedSatelliteID string              `json:"failed_satellite_id,omitempty"`
	FailedGatewayID   string              `json:"failed_gateway_id,omitempty"`
	Metrics           []ClientMetrics     `json:"metrics"`
	Compare           CompareRunsResponse `json:"compare"`
	Analysis          ResilienceAnalysis  `json:"analysis"`
}

type ResilienceAnalysis struct {
	FailedSatelliteID string             `json:"failed_satellite_id,omitempty"`
	FailedGatewayID   string             `json:"failed_gateway_id,omitempty"`
	Interval          FailureInterval    `json:"interval"`
	AffectedClients   []string           `json:"affected_clients"`
	PreservedClients  []string           `json:"preserved_clients"`
	Clients           []ResilienceClient `json:"clients"`
	GapReasons        GapReasonDiff      `json:"gap_reasons"`
	Vulnerabilities   []string           `json:"vulnerabilities"`
	Mitigations       []string           `json:"mitigations"`
	Summary           string             `json:"summary"`
}

type FailureInterval struct {
	StartS float64 `json:"start_s"`
	EndS   float64 `json:"end_s"`
}

type ResilienceClient struct {
	ClientID          string    `json:"client_id"`
	Affected          bool      `json:"affected"`
	RoutePreserved    bool      `json:"route_preserved"`
	PathBefore        []string  `json:"path_before"`
	PathAfter         []string  `json:"path_after"`
	ReasonBefore      GapReason `json:"reason_before,omitempty"`
	ReasonAfter       GapReason `json:"reason_after,omitempty"`
	PathRatioBefore   float64   `json:"path_ratio_before"`
	PathRatioAfter    float64   `json:"path_ratio_after"`
	DeltaPathRatio    float64   `json:"delta_path_ratio"`
	MaxGapBefore      int       `json:"max_gap_s_before"`
	MaxGapAfter       int       `json:"max_gap_s_after"`
	DeltaMaxGapS      int       `json:"delta_max_gap_s"`
	MeetsTargetBefore bool      `json:"meets_target_before"`
	MeetsTargetAfter  bool      `json:"meets_target_after"`
	LostSteps         int       `json:"lost_steps"`
	WindowSteps       int       `json:"window_steps"`
	WindowPathBefore  int       `json:"window_path_before"`
	WindowPathAfter   int       `json:"window_path_after"`
	DominantGapReason GapReason `json:"dominant_gap_reason,omitempty"`
}

type GapReasonDiff struct {
	Before map[string]int `json:"before"`
	After  map[string]int `json:"after"`
	Delta  map[string]int `json:"delta"`
}
