import { describe, it, expect } from "vitest";
import fixture from "../public/scenarios/01_full_constellation.json";
import {
  clone,
  parseScenario,
  validateScenario,
  satelliteStatus,
  intervals,
  parseClock,
  clock,
  snapTime,
  comparisonProblem,
  configChanges,
  xyz,
  lonLat,
} from "./domain";
import type { Run, Scenario } from "./types";
const scenario = fixture as Scenario;
describe("scenario input and validation", () => {
  it("imports a scenario and a result without treating result routes as new calculations", () => {
    expect(parseScenario(JSON.stringify(fixture)).ground_sites[0].id).toBe(
      "G_MUR",
    );
    expect(
      parseScenario(
        JSON.stringify({
          schema_version: "cosmo-A-result-1.0",
          effective_scenario: fixture,
          routes: [],
        }),
      ),
    ).toEqual(fixture);
  });
  it("reports unknown references, invalid angles and out-of-period failures", () => {
    const s = clone(scenario);
    s.design.planes[0].raan_deg = 360;
    s.design.satellites[0].plane_id = "unknown";
    s.failures = [
      {
        satellite_id: "unknown",
        start_s: 50,
        end_s: s.environment.horizon_s + 1,
      },
    ];
    const errors = validateScenario(s);
    expect(errors.some((x) => x.includes("RAAN"))).toBe(true);
    expect(errors.some((x) => x.includes("неизвестная плоскость"))).toBe(true);
    expect(errors.some((x) => x.includes("интервал"))).toBe(true);
  });
  it("rejects malformed documents and invalid grids", () => {
    expect(() => parseScenario("{")).toThrow("Некорректный JSON");
    expect(() =>
      parseScenario('{"schema_version":"cosmo-A-result-1.0"}'),
    ).toThrow();
    const s = clone(scenario);
    s.environment.step_s = 121;
    expect(validateScenario(s).some((x) => x.includes("Сетка"))).toBe(true);
  });
  it("keeps launch state separate from outage and uses half-open intervals", () => {
    const s = clone(scenario);
    s.design.launch_stage = 1;
    s.failures = [
      { satellite_id: "S01", start_s: 120, end_s: 240 },
      { satellite_id: "S17", start_s: 0, end_s: 300 },
    ];
    expect(satelliteStatus(s, "S01", 119)).toBe("active");
    expect(satelliteStatus(s, "S01", 120)).toBe("failed");
    expect(satelliteStatus(s, "S01", 240)).toBe("active");
    expect(satelliteStatus(s, "S17", 120)).toBe("pending");
  });
});
describe("timeline semantics", () => {
  it("supports 24:00 and clamps playback to last sampled interval", () => {
    expect(parseClock("24:00:00")).toBe(86400);
    expect(parseClock("00:60:00")).toBeNaN();
    expect(clock(90000)).toBe("25:00:00");
    expect(snapTime(86400, scenario)).toBe(86280);
    expect(snapTime(239, scenario)).toBe(120);
  });
  it("groups only contiguous intervals of equal reason and separates visibility from route", () => {
    const rows = intervals(
      [
        { t_s: 0, client_id: "C", path: [], reason: "no_visible_sat" },
        { t_s: 120, client_id: "C", path: [], reason: "isl_partition" },
        { t_s: 240, client_id: "C", path: [], reason: "isl_partition" },
        { t_s: 360, client_id: "C", path: ["C", "S", "G"] },
        { t_s: 480, client_id: "C", path: [], reason: "gateway_outage" },
      ],
      "C",
      120,
    );
    expect(rows).toEqual([
      { start: 0, end: 120, state: "invisible", reason: "no_visible_sat" },
      { start: 120, end: 360, state: "partition", reason: "isl_partition" },
      { start: 360, end: 480, state: "available", reason: undefined },
      { start: 480, end: 600, state: "gateway", reason: "gateway_outage" },
    ]);
  });
});
describe("comparison and shared coordinates", () => {
  const run = (s: Scenario): Run => ({
    id: "r",
    project_id: "p",
    effective_scenario: s,
    metrics: [],
    routes: [],
  });
  it("blocks incompatible time grids and moved client sites", () => {
    const s = clone(scenario);
    s.environment.step_s = 60;
    expect(comparisonProblem(run(scenario), run(s))).toContain(
      "Разная временная сетка",
    );
    s.environment.step_s = 120;
    s.ground_sites[1].lat_deg += 1;
    expect(comparisonProblem(run(scenario), run(s))).toContain("координаты");
  });
  it("allows orbit experiments but exposes their differences", () => {
    const s = clone(scenario);
    s.environment.isl_range_km = 2000;
    s.design.planes[1].phase_deg = 15;
    expect(comparisonProblem(run(scenario), run(s))).toBe("");
    expect(configChanges(scenario, s).map((x) => x.label)).toEqual(
      expect.arrayContaining(["Дальность ISL, км", "Фаза P2"]),
    );
  });
  it("round trips ground coordinates through the same ECEF representation used by both maps", () => {
    for (const g of scenario.ground_sites) {
      const [x_km, y_km, z_km] = xyz(g),
        [lon, lat] = lonLat({ id: g.id, x_km, y_km, z_km, active: true });
      expect(lon).toBeCloseTo(g.lon_deg, 8);
      expect(lat).toBeCloseTo(g.lat_deg, 8);
    }
  });
});
