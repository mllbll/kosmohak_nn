export interface Environment {
  altitude_km: number;
  inclination_deg: number;
  earth_angle0_deg: number;
  horizon_s: number;
  step_s: number;
  min_elevation_deg: number;
  isl_range_km: number;
  target_availability: number;
}
export interface Plane {
  id: string;
  raan_deg: number;
  phase_deg: number;
}
export interface Satellite {
  id: string;
  plane_id: string;
  slot_deg: number;
  launch_batch: number;
}
export interface GroundSite {
  id: string;
  name: string;
  role: "client" | "gateway";
  lat_deg: number;
  lon_deg: number;
}
export interface Failure {
  satellite_id: string;
  start_s: number;
  end_s: number;
}
export interface GatewayOutage {
  gateway_id: string;
  start_s: number;
  end_s: number;
}
export interface Scenario {
  schema_version: string;
  meta: { id: string; title: string };
  environment: Environment;
  design: { launch_stage: number; planes: Plane[]; satellites: Satellite[] };
  ground_sites: GroundSite[];
  failures: Failure[];
  gateway_outages: GatewayOutage[];
}
export interface Project {
  id: string;
  base: Scenario;
  effective: Scenario;
}
export type Reason =
  "no_visible_sat" | "no_gateway_contact" | "gateway_outage" | "isl_partition";
export interface Route {
  t_s: number;
  client_id: string;
  path: string[];
  hops?: number;
  reason?: Reason;
}
export interface Metric {
  client_id: string;
  visibility_ratio: number;
  path_ratio: number;
  max_gap_s: number;
  mean_hops: number;
  meets_target: boolean;
  gaps: {
    start_s: number;
    end_s: number;
    duration_s: number;
    reason?: Reason;
  }[];
}
export interface Run {
  id: string;
  project_id: string;
  effective_scenario: Scenario;
  routes: Route[];
  metrics: Metric[];
  summary?: string;
}
export interface SatState {
  id: string;
  x_km: number;
  y_km: number;
  z_km: number;
  active: boolean;
}
export interface Snapshot {
  snapshot: {
    t_s: number;
    satellites: SatState[];
    edges: [string, string, number][];
    elevation_deg: Record<string, Record<string, number>>;
  };
  route: {
    path: string[];
    hops?: number;
    reason?: Reason;
    alternatives?: string[][];
  };
  visible_satellites: string[];
  network_delta: { changed: boolean; explanation: string };
}
export interface Comparison {
  recommendation: {
    run_id?: string;
    better: string;
    reason: string;
    advantages: string[];
    conditions: string[];
    limitations: string[];
    conclusion: string;
  };
  config_diff?: Record<string, unknown>;
}
export interface Variant {
  key: string;
  title: string;
  scenario: Scenario;
  projectId?: string;
  runId?: string;
  savedAt?: string;
}
export type Tab = "project" | "failures" | "network" | "compare";
export type Layers = {
  satellites: boolean;
  sites: boolean;
  links: boolean;
  route: boolean;
  orbits: boolean;
  labels: boolean;
};
