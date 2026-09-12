import { test, expect } from "@playwright/test";
import { readFileSync } from "node:fs";
import type { Scenario, Run, Snapshot } from "../src/types";

test("API rejects incompatible comparisons and accepts renamed custom node IDs", async ({ request }) => {
  const scenario = JSON.parse(readFileSync("public/scenarios/01_full_constellation.json", "utf8")) as Scenario;
  scenario.environment.horizon_s = 240;
  for (const sat of scenario.design.satellites) sat.id = "custom-" + sat.id;
  for (const site of scenario.ground_sites) site.id = "custom-" + site.id;
  const calculate = async (s: Scenario) => {
    const created = await request.post("/api/projects", { data: s });
    expect(created.status()).toBe(201);
    const result = await request.post("/api/projects/" + (await created.json()).id + "/runs");
    expect(result.status()).toBe(201);
    return (await result.json()).run_id as string;
  };
  const first = await calculate(scenario);
  const exported = await (await request.get("/api/runs/" + first + "/export")).json();
  expect(exported.routes).toHaveLength(6);
  expect(exported.routes.every((r: {client_id: string}) => r.client_id.startsWith("custom-"))).toBe(true);
  for (const change of ["step", "target", "client"]) {
    const changed = structuredClone(scenario);
    if (change === "step") changed.environment.step_s = 60;
    if (change === "target") changed.environment.target_availability = 0.5;
    if (change === "client") changed.ground_sites.find(x => x.role === "client")!.lat_deg += 1;
    const second = await calculate(changed);
    expect((await request.post("/api/compare", {data: {run_a: first, run_b: second}})).status()).toBe(400);
  }
  expect((await request.post("/api/compare", {data: {run_a: first, run_b: first}})).status()).toBe(400);
});

test("four complete daily scenarios obey the frontend API contract", async ({
  request,
}) => {
  test.setTimeout(300000);
  const runs: Run[] = [];
  for (const name of [
    "01_full_constellation",
    "02_first_launch",
    "03_satellite_outages",
    "04_link_range",
  ]) {
    const scenario = JSON.parse(
      readFileSync("public/scenarios/" + name + ".json", "utf8"),
    ) as Scenario;
    const projectResponse = await request.post("/api/projects", {
      data: scenario,
    });
    expect(projectResponse.status()).toBe(201);
    const project = await projectResponse.json();
    const calculation = await request.post(
      "/api/projects/" + project.id + "/runs",
      { timeout: 180000 },
    );
    expect(calculation.status()).toBe(201);
    const created = await calculation.json();
    const runResponse = await request.get("/api/runs/" + created.run_id);
    const run = (await runResponse.json()) as Run;
    runs.push(run);
    expect(run.routes).toHaveLength(720 * 3);
    expect(run.metrics).toHaveLength(3);
    for (const metric of run.metrics) {
      expect(metric.visibility_ratio).toBeGreaterThanOrEqual(
        metric.path_ratio - 1e-10,
      );
      expect(metric.max_gap_s % scenario.environment.step_s).toBe(0);
      const rows = run.routes.filter((r) => r.client_id === metric.client_id);
      expect(
        rows.filter((r) => r.path.length).length / rows.length,
      ).toBeCloseTo(metric.path_ratio, 8);
    }
    const snapshotResponse = await request.get(
      "/api/runs/" + run.id + "/snapshot?t_s=21600&client_id=C70",
    );
    const snap = (await snapshotResponse.json()) as Snapshot;
    const active = new Set(
      snap.snapshot.satellites.filter((x) => x.active).map((x) => x.id),
    );
    const grounds = new Set(scenario.ground_sites.map((x) => x.id));
    for (const [a, b] of snap.snapshot.edges) {
      expect(active.has(a) || grounds.has(a)).toBe(true);
      expect(active.has(b) || grounds.has(b)).toBe(true);
    }
    if (name === "03_satellite_outages") expect(active.size).toBe(38);
    if (name === "02_first_launch") expect(active.size).toBe(16);
    const exported = await (
      await request.get("/api/runs/" + run.id + "/export")
    ).json();
    expect(exported.schema_version).toBe("cosmo-A-result-1.0");
    expect(exported.routes).toHaveLength(2160);
    expect(
      (await request.post("/api/projects", { data: exported })).status(),
    ).toBe(201);
  }
  const compared = await request.post("/api/compare", {
    data: { run_a: runs[0].id, run_b: runs[1].id },
  });
  expect(compared.ok()).toBe(true);
  const comparison = await compared.json();
  expect(comparison.clients).toHaveLength(3);
  expect(comparison.config_diff.launch_stage).toEqual({ a: 3, b: 1 });
  console.log(
    "Daily scenario path ratios:",
    runs.map((r) => ({
      title: r.effective_scenario.meta.title,
      metrics: r.metrics.map((m) => ({
        id: m.client_id,
        path: m.path_ratio,
        gap: m.max_gap_s,
      })),
    })),
  );
});

test("gateway outage displays its reason and a missing hop count", async ({
  page,
}) => {
  await page.goto("/");
  const scenario = JSON.parse(
    readFileSync("public/scenarios/01_full_constellation.json", "utf8"),
  ) as Scenario;
  scenario.environment.horizon_s = 240;
  scenario.gateway_outages = [{ gateway_id: "G_MUR", start_s: 0, end_s: 240 }];
  await page
    .locator("input[type=file]")
    .setInputFiles({
      name: "gateway.json",
      mimeType: "application/json",
      buffer: Buffer.from(JSON.stringify(scenario)),
    });
  await expect(
    page.getByText("Проект загружен и проверен сервером"),
  ).toBeVisible();
  await page
    .getByRole("button", { name: "Запустить расчёт", exact: true })
    .click();
  await expect(
    page.getByText("Маршрут недоступен", { exact: true }),
  ).toBeVisible({ timeout: 150000 });
  await expect(page.locator(".route-reason")).toHaveText("Шлюзы недоступны");
  await expect(page.getByText("Переходы: —", { exact: true })).toBeVisible();
  await expect(page.locator(".availability-segment.gateway")).toHaveCount(3);
  const exported = page.waitForEvent("download");
  await page
    .getByRole("button", { name: "Экспорт результата", exact: true })
    .click();
  expect((await exported).suggestedFilename()).toMatch(/^result-/);
});
