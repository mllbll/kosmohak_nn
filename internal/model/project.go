package model

// Project сохранённый вариант группировки
type Project struct {
	ID        string   `json:"id"`
	Base      Scenario `json:"base"`
	Effective Scenario `json:"effective"`
}

type CreateProjectRequest struct {
	Scenario Scenario
}

type CreateProjectResponse struct {
	Project Project
}

type GetProjectRequest struct {
	ProjectID string
}

type GetProjectResponse struct {
	Project Project
}

type PatchProjectRequest struct {
	ProjectID string
	Patch     Patch
}

type PatchProjectResponse struct {
	Project Project
}

type ResetProjectRequest struct {
	ProjectID string
}

type ResetProjectResponse struct {
	Project Project
}

type CopyProjectRequest struct {
	ProjectID string
}

type CopyProjectResponse struct {
	Project Project
}

// Patch частичное обновление из интерфейса
type Patch struct {
	LaunchStage    *int             `json:"launch_stage"`
	Planes         []PlanePatch     `json:"planes"`
	Failures       *[]Failure       `json:"failures"`
	GatewayOutages *[]GatewayOutage `json:"gateway_outages"`
}

type PlanePatch struct {
	ID       string   `json:"id"`
	RAANDeg  *float64 `json:"raan_deg"`
	PhaseDeg *float64 `json:"phase_deg"`
}
