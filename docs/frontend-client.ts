/**
 * Drop-in HTTP client for the kosmohak_nn API.
 * Copy into a frontend project. Keep fields in sync with docs/frontend.md §16.
 *
 * Usage:
 *   const api = new KosmoApi("http://localhost:8080");
 *   const project = await api.createProject(scenario);
 */
export const SESSION_KEY = "kosmohak.session.v1";
export const RUN_TIMEOUT_MS = 5 * 60 * 1000;

export type GapReason =
  | "no_visible_sat"
  | "no_gateway_contact"
  | "gateway_outage"
  | "isl_partition";

export type ApiErrorBody = { error: string };

export type Meta = { id: string; title: string };

export type Environment = {
  altitude_km: number;
  inclination_deg: number;
  earth_angle0_deg: number;
  horizon_s: number;
  step_s: number;
  min_elevation_deg: number;
  isl_range_km: number;
  target_availability: number;
};

export type Plane = { id: string; raan_deg: number; phase_deg: number };
export type Satellite = {
  id: string;
  plane_id: string;
  slot_deg: number;
  launch_batch: number;
};
export type GroundSite = {
  id: string;
  name: string;
  role: "client" | "gateway";
  lat_deg: number;
  lon_deg: number;
};
export type Failure = { satellite_id: string; start_s: number; end_s: number };
export type GatewayOutage = { gateway_id: string; start_s: number; end_s: number };

export type Scenario = {
  schema_version: "cosmo-A-1.0" | string;
  meta: Meta;
  environment: Environment;
  design: { launch_stage: number; planes: Plane[]; satellites: Satellite[] };
  ground_sites: GroundSite[];
  failures: Failure[];
  gateway_outages: GatewayOutage[];
};

export type Project = { id: string; base: Scenario; effective: Scenario };

export type Patch = {
  launch_stage?: number;
  planes?: { id: string; raan_deg?: number; phase_deg?: number }[];
  failures?: Failure[] | null;
  gateway_outages?: GatewayOutage[] | null;
};

export type GapInterval = {
  start_s: number;
  end_s: number;
  duration_s: number;
  reason?: GapReason;
};

export type ClientMetrics = {
  client_id: string;
  visibility_ratio: number;
  path_ratio: number;
  max_gap_s: number;
  mean_hops: number;
  meets_target: boolean;
  gaps: GapInterval[];
};

export type RouteRecord = {
  t_s: number;
  client_id: string;
  path: string[];
  reason?: GapReason;
  hops?: number;
  alt_count: number;
};

export type Run = {
  id: string;
  project_id: string;
  effective_scenario: Scenario;
  routes: RouteRecord[];
  metrics: ClientMetrics[];
  summary?: string;
};

export type CreateRunResponse = {
  run_id: string;
  project_id: string;
  metrics: ClientMetrics[];
};

export type EdgeTuple = [string, string, number];

export type Snapshot = {
  t_s: number;
  satellites: {
    id: string;
    x_km: number;
    y_km: number;
    z_km: number;
    active: boolean;
  }[];
  edges: EdgeTuple[];
  elevation_deg: Record<string, Record<string, number>>;
};

export type RouteResult = {
  path: string[];
  reason?: GapReason;
  hops?: number;
  alternatives?: string[][];
  algorithm: {
    name: string;
    objective: string;
    constraints: string[];
    rationale: string;
    min_hops?: number;
  };
};

export type NetworkDelta = {
  changed: boolean;
  previous_still_valid: boolean;
  previous_path?: string[];
  current_path?: string[];
  explanation: string;
};

export type GetSnapshotResponse = {
  snapshot: Snapshot;
  route: RouteResult;
  network_delta: NetworkDelta;
  visible_satellites: string[];
};

export type ExportDocument = {
  schema_version: "cosmo-A-result-1.0" | string;
  effective_scenario: Scenario;
  routes: Omit<RouteRecord, "alt_count">[];
  metrics?: ClientMetrics[];
  summary?: string;
};

export type CompareRunsRequest = {
  run_a?: string;
  run_b?: string;
  run_ids?: string[];
};

export type CompareVariant = {
  run_id: string;
  project_id: string;
  title: string;
  launch_stage: number;
  planes: Plane[];
  clients_meeting_target: number;
  clients_total: number;
  mean_path_ratio: number;
  mean_max_gap_s: number;
  mean_hops: number;
  metrics: ClientMetrics[];
  summary?: string;
};

export type ClientRunMetric = {
  run_id: string;
  path_ratio: number;
  visibility_ratio: number;
  max_gap_s: number;
  mean_hops: number;
  meets_target: boolean;
};

export type ClientDiff = {
  client_id: string;
  better: "a" | "b" | "tie" | string;
  path_ratio_a: number;
  path_ratio_b: number;
  delta_path_ratio: number;
  visibility_ratio_a: number;
  visibility_ratio_b: number;
  max_gap_s_a: number;
  max_gap_s_b: number;
  delta_max_gap_s: number;
  meets_target_a: boolean;
  meets_target_b: boolean;
  by_run: ClientRunMetric[];
};

export type CompareRecommendation = {
  run_id?: string;
  better: "a" | "b" | "tie" | string;
  reason: string;
  advantages: string[];
  conditions: string[];
  limitations: string[];
  conclusion: string;
};

export type CompareRunsResponse = {
  run_a_id?: string;
  run_b_id?: string;
  config_diff?: Record<string, unknown>;
  variants: CompareVariant[];
  clients: ClientDiff[];
  recommendation: CompareRecommendation;
};

export type WhatIfRequest = {
  client_id?: string;
  t_s?: number;
  satellite_id?: string;
  gateway_id?: string;
  start_s?: number;
  end_s?: number;
};

export type ResilienceClient = {
  client_id: string;
  affected: boolean;
  route_preserved: boolean;
  path_before: string[];
  path_after: string[];
  reason_before?: GapReason;
  reason_after?: GapReason;
  path_ratio_before: number;
  path_ratio_after: number;
  delta_path_ratio: number;
  max_gap_s_before: number;
  max_gap_s_after: number;
  delta_max_gap_s: number;
  meets_target_before: boolean;
  meets_target_after: boolean;
  lost_steps: number;
  window_steps: number;
  window_path_before: number;
  window_path_after: number;
  dominant_gap_reason?: GapReason;
};

export type WhatIfResponse = {
  original_run_id: string;
  project_id: string;
  run_id: string;
  failed_satellite_id?: string;
  failed_gateway_id?: string;
  metrics: ClientMetrics[];
  compare: CompareRunsResponse;
  analysis: {
    failed_satellite_id?: string;
    failed_gateway_id?: string;
    interval: { start_s: number; end_s: number };
    affected_clients: string[];
    preserved_clients: string[];
    clients: ResilienceClient[];
    gap_reasons: {
      before: Record<string, number>;
      after: Record<string, number>;
      delta: Record<string, number>;
    };
    vulnerabilities: string[];
    mitigations: string[];
    summary: string;
  };
};

export type Session = {
  project_id: string;
  run_id: string;
  client_id: string;
  t_s: number;
  compare_run_ids: string[];
};

export class ApiError extends Error {
  constructor(
    public status: number,
    public body: string,
  ) {
    super(body);
    this.name = "ApiError";
  }
}

export function wrapDeg(value: number): number {
  const x = ((value % 360) + 360) % 360;
  return x === 360 ? 0 : x;
}

export function timeGrid(env: Pick<Environment, "horizon_s" | "step_s">): number[] {
  const out: number[] = [];
  for (let t = 0; t < env.horizon_s; t += env.step_s) out.push(t);
  return out;
}

export function pathCount(route: { path: string[]; alternatives?: string[][] }): number {
  if (!route.path.length) return 0;
  return 1 + (route.alternatives?.length ?? 0);
}

export function pathCountFromRecord(rec: RouteRecord): number {
  if (!rec.path.length) return 0;
  return 1 + rec.alt_count;
}

/** Unique-path ticks for the timeline. Not a gap: path exists, no backups. */
export function uniquePathTicks(routes: RouteRecord[], clientId: string): number[] {
  return routes
    .filter((r) => r.client_id === clientId && r.path.length > 0 && r.alt_count === 0)
    .map((r) => r.t_s);
}

export function clientIds(sc: Scenario): string[] {
  return sc.ground_sites.filter((s) => s.role === "client").map((s) => s.id);
}

export function hopSatellites(path: string[]): string[] {
  if (path.length < 3) return [];
  return path.slice(1, -1);
}

export function isDirty(project: Project): boolean {
  const a = project.base;
  const b = project.effective;
  if (a.design.launch_stage !== b.design.launch_stage) return true;
  if (JSON.stringify(a.design.planes) !== JSON.stringify(b.design.planes)) return true;
  if (JSON.stringify(a.failures) !== JSON.stringify(b.failures)) return true;
  if (JSON.stringify(a.gateway_outages) !== JSON.stringify(b.gateway_outages)) return true;
  return false;
}

export function cloneForEnv(
  sc: Scenario,
  patch: Partial<Environment>,
  titleSuffix: string,
): Scenario {
  const next: Scenario = structuredClone(sc);
  next.environment = { ...next.environment, ...patch };
  next.meta = {
    id: `${next.meta.id}_${titleSuffix}`.replace(/\s+/g, "_"),
    title: `${next.meta.title} (${titleSuffix})`,
  };
  for (const plane of next.design.planes) {
    plane.raan_deg = wrapDeg(plane.raan_deg);
    plane.phase_deg = wrapDeg(plane.phase_deg);
  }
  return next;
}

export function loadSession(): Session | null {
  const raw = localStorage.getItem(SESSION_KEY);
  if (!raw) return null;
  try {
    return JSON.parse(raw) as Session;
  } catch {
    return null;
  }
}

export function saveSession(session: Session): void {
  localStorage.setItem(SESSION_KEY, JSON.stringify(session));
}

export function clearSession(): void {
  localStorage.removeItem(SESSION_KEY);
}

export class KosmoApi {
  constructor(public base = "") {}

  async health(): Promise<{ status: string }> {
    return this.request("GET", "/health");
  }

  async createProject(body: Scenario | ExportDocument): Promise<Project> {
    return this.request("POST", "/api/projects", body);
  }

  async getProject(id: string): Promise<Project> {
    return this.request("GET", `/api/projects/${id}`);
  }

  async patchProject(id: string, patch: Patch): Promise<Project> {
    return this.request("PATCH", `/api/projects/${id}`, patch);
  }

  async resetProject(id: string): Promise<Project> {
    return this.request("POST", `/api/projects/${id}/reset`);
  }

  async copyProject(id: string): Promise<Project> {
    return this.request("POST", `/api/projects/${id}/copy`);
  }

  async createRun(projectId: string): Promise<CreateRunResponse> {
    return this.request("POST", `/api/projects/${projectId}/runs`, undefined, RUN_TIMEOUT_MS);
  }

  async getRun(runId: string): Promise<Run> {
    return this.request("GET", `/api/runs/${runId}`);
  }

  async getMetrics(runId: string): Promise<ClientMetrics[]> {
    return this.request("GET", `/api/runs/${runId}/metrics`);
  }

  async getSnapshot(runId: string, tS: number, clientId: string): Promise<GetSnapshotResponse> {
    const q = new URLSearchParams({ t_s: String(tS), client_id: clientId });
    return this.request("GET", `/api/runs/${runId}/snapshot?${q}`);
  }

  async exportRun(runId: string): Promise<ExportDocument> {
    return this.request("GET", `/api/runs/${runId}/export`);
  }

  async whatIf(runId: string, body: WhatIfRequest): Promise<WhatIfResponse> {
    return this.request("POST", `/api/runs/${runId}/what-if`, body, RUN_TIMEOUT_MS);
  }

  async compare(body: CompareRunsRequest): Promise<CompareRunsResponse> {
    return this.request("POST", "/api/compare", body);
  }

  /** Click a satellite on the current route. Do not render compare.recommendation as a design winner. */
  async failSatellite(
    runId: string,
    satelliteId: string,
    tS: number,
    startS = tS,
    endS?: number,
  ): Promise<WhatIfResponse> {
    return this.whatIf(runId, {
      satellite_id: satelliteId,
      t_s: tS,
      start_s: startS,
      end_s: endS,
    });
  }

  /** New project+run with one environment field changed (PATCH cannot do this). */
  async exploreEnvironment(
    source: Project,
    patch: Partial<Environment>,
    titleSuffix: string,
  ): Promise<{ project: Project; run: CreateRunResponse }> {
    const project = await this.createProject(cloneForEnv(source.effective, patch, titleSuffix));
    const run = await this.createRun(project.id);
    return { project, run };
  }

  private async request<T>(
    method: string,
    path: string,
    body?: unknown,
    timeoutMs = 30_000,
  ): Promise<T> {
    const ctrl = new AbortController();
    const timer = setTimeout(() => ctrl.abort(), timeoutMs);
    try {
      const res = await fetch(`${this.base}${path}`, {
        method,
        headers: body === undefined ? undefined : { "Content-Type": "application/json" },
        body: body === undefined ? undefined : JSON.stringify(body),
        signal: ctrl.signal,
      });
      const text = await res.text();
      if (!res.ok) {
        throw new ApiError(res.status, text || res.statusText);
      }
      return text ? (JSON.parse(text) as T) : (undefined as T);
    } finally {
      clearTimeout(timer);
    }
  }
}
