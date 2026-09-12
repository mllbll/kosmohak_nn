import { feature } from "topojson-client";
import atlas from "world-atlas/countries-110m.json";
import type { Topology, GeometryCollection } from "topojson-specification";
import type { Scenario, Snapshot } from "./types";
import { xyz, satelliteStatus } from "./domain";
const topology = atlas as unknown as Topology<{
  land: GeometryCollection;
  countries: GeometryCollection;
}>;
export const land = feature(topology, topology.objects.land);
export const countries = feature(topology, topology.objects.countries);
export interface MapNode {
  id: string;
  kind: "satellite" | "client" | "gateway";
  status: string;
  position: [number, number, number];
}
export function mapNodes(s: Scenario, snapshot?: Snapshot): MapNode[] {
  const t = snapshot?.snapshot.t_s ?? 0;
  return [
    ...s.ground_sites.map((g) => ({
      id: g.id,
      kind: g.role,
      status:
        g.role === "gateway" &&
        s.gateway_outages.some(
          (x) => x.gateway_id === g.id && x.start_s <= t && t < x.end_s,
        )
          ? "failed"
          : "active",
      position: xyz(g),
    })),
    ...(snapshot?.snapshot.satellites ?? []).map((p) => ({
      id: p.id,
      kind: "satellite" as const,
      status: satelliteStatus(s, p.id, t),
      position: [p.x_km, p.y_km, p.z_km] as [number, number, number],
    })),
  ];
}
export function orbitPoints(
  s: Scenario,
  t: number,
  raan: number,
): [number, number, number][] {
  // Orbital guide only. Satellite positions and link geometry always come from the API.
  const rad = Math.PI / 180,
    omega = raan * rad,
    inc = s.environment.inclination_deg * rad,
    theta =
      s.environment.earth_angle0_deg * rad + (2 * Math.PI * t) / 86164.09054,
    r = 6371 + s.environment.altitude_km;
  return Array.from({ length: 181 }, (_, i) => {
    const u = (i / 180) * 2 * Math.PI,
      x =
        r *
        (Math.cos(omega) * Math.cos(u) -
          Math.sin(omega) * Math.sin(u) * Math.cos(inc)),
      y =
        r *
        (Math.sin(omega) * Math.cos(u) +
          Math.cos(omega) * Math.sin(u) * Math.cos(inc));
    return [
      Math.cos(theta) * x + Math.sin(theta) * y,
      -Math.sin(theta) * x + Math.cos(theta) * y,
      r * Math.sin(u) * Math.sin(inc),
    ];
  });
}
export const nodeColor = (n: MapNode) =>
  n.status === "failed"
    ? "#ff6b7e"
    : n.status === "pending"
      ? "#73839b"
      : n.kind === "gateway"
        ? "#ffce70"
        : n.kind === "client"
          ? "#63e6bd"
          : "#62cfff";
export const positionLonLat = (
  p: [number, number, number],
): [number, number] => [
  (Math.atan2(p[1], p[0]) * 180) / Math.PI,
  (Math.atan2(p[2], Math.hypot(p[0], p[1])) * 180) / Math.PI,
];
