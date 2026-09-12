import type { Scenario, Snapshot } from "./types";
import { satelliteStatus, xyz } from "./domain";

// The same circular-orbit/contact model as python/geometry.py, evaluated for
// animation frames. Saved metrics remain the result of the server time grid.
export function liveSnapshot(s: Scenario, t: number, client: string): Snapshot {
  const e = s.environment,
    rad = Math.PI / 180,
    radius = 6371 + e.altitude_km;
  const motion = Math.sqrt(398600.435507 / radius ** 3);
  const theta = e.earth_angle0_deg * rad + (2 * Math.PI * t) / 86164.09054;
  const planes = new Map(s.design.planes.map((p) => [p.id, p]));
  const satellites = s.design.satellites.map((sat) => {
    const plane = planes.get(sat.plane_id)!;
    const u = (sat.slot_deg + plane.phase_deg) * rad + motion * t;
    const om = plane.raan_deg * rad,
      inc = e.inclination_deg * rad;
    const x =
      radius *
      (Math.cos(om) * Math.cos(u) - Math.sin(om) * Math.sin(u) * Math.cos(inc));
    const y =
      radius *
      (Math.sin(om) * Math.cos(u) + Math.cos(om) * Math.sin(u) * Math.cos(inc));
    return {
      id: sat.id,
      x_km: Math.cos(theta) * x + Math.sin(theta) * y,
      y_km: -Math.sin(theta) * x + Math.cos(theta) * y,
      z_km: radius * Math.sin(u) * Math.sin(inc),
      active: satelliteStatus(s, sat.id, t) === "active",
    };
  });
  const edges: Snapshot["snapshot"]["edges"] = [];
  const active = satellites.filter((p) => p.active);
  for (let i = 0; i < active.length; i++)
    for (let j = i + 1; j < active.length; j++) {
      const a = active[i],
        b = active[j];
      const dx = b.x_km - a.x_km,
        dy = b.y_km - a.y_km,
        dz = b.z_km - a.z_km;
      const d2 = dx * dx + dy * dy + dz * dz;
      const f = Math.max(
        0,
        Math.min(
          1,
          -(a.x_km * dx + a.y_km * dy + a.z_km * dz) / Math.max(d2, 1e-12),
        ),
      );
      if (
        Math.sqrt(d2) < e.isl_range_km &&
        Math.hypot(a.x_km + f * dx, a.y_km + f * dy, a.z_km + f * dz) > 6371
      )
        edges.push([a.id, b.id, Math.sqrt(d2)]);
    }
  const elevation: Record<string, Record<string, number>> = {};
  const offline = new Set(
    s.gateway_outages
      .filter((f) => f.start_s <= t && t < f.end_s)
      .map((f) => f.gateway_id),
  );
  for (const site of s.ground_sites) {
    const gp = xyz(site);
    elevation[site.id] = {};
    for (const sat of active) {
      const dx = sat.x_km - gp[0],
        dy = sat.y_km - gp[1],
        dz = sat.z_km - gp[2];
      const distance = Math.hypot(dx, dy, dz);
      const el =
        Math.asin(
          Math.max(
            -1,
            Math.min(
              1,
              (dx * gp[0] + dy * gp[1] + dz * gp[2]) / (6371 * distance),
            ),
          ),
        ) / rad;
      elevation[site.id][sat.id] = el;
      if (el >= e.min_elevation_deg && !offline.has(site.id))
        edges.push([site.id, sat.id, distance]);
    }
  }
  const adj = new Map<string, string[]>();
  for (const [a, b] of edges) {
    if (!adj.has(a)) adj.set(a, []);
    if (!adj.has(b)) adj.set(b, []);
    adj.get(a)!.push(b);
    adj.get(b)!.push(a);
  }
  for (const neighbors of adj.values()) neighbors.sort();
  const gateways = new Set(
    s.ground_sites.filter((g) => g.role === "gateway").map((g) => g.id),
  );
  const sats = new Set(active.map((p) => p.id));
  const queue = [client],
    previous = new Map<string, string | null>([[client, null]]);
  let end: string | undefined;
  for (let i = 0; i < queue.length; i++) {
    const id = queue[i];
    if (gateways.has(id)) {
      end = id;
      break;
    }
    for (const next of adj.get(id) ?? []) {
      if ((!sats.has(next) && !gateways.has(next)) || previous.has(next))
        continue;
      previous.set(next, id);
      queue.push(next);
    }
  }
  const path: string[] = [];
  if (end)
    for (let id: string | null = end; id !== null; id = previous.get(id)!)
      path.unshift(id);
  const visible = Object.entries(elevation[client] ?? {})
    .filter(([, el]) => el >= e.min_elevation_deg)
    .map(([id]) => id);
  const route: Snapshot["route"] = path.length
    ? { path, hops: path.length - 1 }
    : {
        path,
        reason: [...gateways].every((id) => offline.has(id))
          ? "gateway_outage"
          : !visible.length
            ? "no_visible_sat"
            : ![...gateways].some((id) => (adj.get(id)?.length ?? 0) > 0)
              ? "no_gateway_contact"
              : "isl_partition",
      };
  return {
    snapshot: { t_s: t, satellites, edges, elevation_deg: elevation },
    route,
    visible_satellites: visible,
    network_delta: {
      changed: false,
      explanation: "Непрерывный расчёт геометрии и кратчайшего маршрута",
    },
  };
}
