import { test, expect } from "@playwright/test";
import { readFileSync } from "node:fs";
import { liveSnapshot } from "../src/liveSnapshot";
import type { Scenario, Snapshot } from "../src/types";

test("continuous model agrees with Python geometry and server routing at fractional outage boundaries", async ({
  request,
}) => {
  const s = JSON.parse(
    readFileSync("public/scenarios/01_full_constellation.json", "utf8"),
  ) as Scenario;
  s.environment.horizon_s = 600;
  s.design.launch_stage = 2;
  s.failures = [{ satellite_id: "S01", start_s: 120.5, end_s: 240.5 }];
  s.gateway_outages = [{ gateway_id: "G_MUR", start_s: 300, end_s: 400 }];
  const project = await request.post("/api/projects", { data: s });
  expect(project.status()).toBe(201);
  const run = await request.post(
    "/api/projects/" + (await project.json()).id + "/runs",
  );
  expect(run.status()).toBe(201);
  const id = (await run.json()).run_id;
  for (const t of [0, 1, 120.5, 240.5, 350, 599.9]) {
    const response = await request.get(
      `/api/runs/${id}/snapshot?t_s=${t}&client_id=C70`,
    );
    expect(response.ok()).toBe(true);
    const server = (await response.json()) as Snapshot;
    const local = liveSnapshot(s, t, "C70");
    for (const sat of local.snapshot.satellites) {
      const expected = server.snapshot.satellites.find((p) => p.id === sat.id)!;
      expect(sat.active).toBe(expected.active);
      for (const key of ["x_km", "y_km", "z_km"] as const)
        expect(sat[key]).toBeCloseTo(expected[key], 7);
    }
    const key = (a: string, b: string) => JSON.stringify([a, b].sort());
    const edgeMap = new Map(
      server.snapshot.edges.map(([a, b, d]) => [key(a, b), d]),
    );
    expect(local.snapshot.edges).toHaveLength(edgeMap.size);
    for (const [a, b, d] of local.snapshot.edges)
      expect(d).toBeCloseTo(edgeMap.get(key(a, b))!, 7);
    expect(local.visible_satellites.sort()).toEqual(
      server.visible_satellites.sort(),
    );
    expect(local.route.reason).toBe(server.route.reason);
    expect(local.route.hops ?? 0).toBe(server.route.hops ?? 0);
    for (let i = 1; i < local.route.path.length; i++)
      expect(
        edgeMap.has(key(local.route.path[i - 1], local.route.path[i])),
      ).toBe(true);
  }
});

test("typed seconds and zero mode animate continuously without snapshot requests", async ({
  page,
}) => {
  await page.goto("/");
  await page.getByLabel("Период расчёта", { exact: true }).fill("1200");
  await page
    .getByRole("button", { name: "Запустить расчёт", exact: true })
    .click();
  await expect(page.locator(".globe-host canvas")).toBeVisible();
  await expect(page.locator(".map-busy")).toHaveCount(0);
  const step = page.getByLabel("Шаг симуляции", { exact: true });
  const time = page.getByLabel("Время расчёта", { exact: true });
  await step.fill("-1");
  await expect(step).toHaveAttribute("aria-invalid", "true");
  await expect(
    page.getByRole("button", { name: "Воспроизвести", exact: true }),
  ).toBeDisabled();
  await step.fill("1");
  await page
    .getByRole("button", { name: "Следующий шаг", exact: true })
    .click();
  await expect(page.locator(".map-heading")).toContainText("00:00:01");
  await expect(page.getByLabel("Временная шкала", { exact: true })).toHaveValue(
    "1",
  );
  await expect(page.locator(".time-progress-label")).toHaveText("00:00:01");
  await time.fill("00:00:01.5");
  await time.press("Enter");
  await expect(page.locator(".map-heading")).toContainText("00:00:01.5");
  await step.fill("0");
  await expect(page.locator(".map-busy")).toHaveCount(0);
  let requests = 0;
  page.on("request", (request) => {
    if (request.url().includes("/snapshot?")) requests++;
  });
  await page
    .getByRole("button", { name: "Воспроизвести", exact: true })
    .click();
  const observation = await page.evaluate(async () => {
    const times = new Set<string>();
    const positions = new Set<string>();
    let mismatches = 0;
    let minMarkers = Infinity;
    for (let i = 0; i < 45; i++) {
      await new Promise(requestAnimationFrame);
      times.add(document.querySelector(".map-heading .mono")!.textContent!);
      const slider = document.querySelector<HTMLInputElement>(".time-slider")!;
      const marker = document.querySelector<HTMLElement>(
        ".time-progress-marker",
      )!;
      const fill = document.querySelector<HTMLElement>(".time-progress-fill")!;
      positions.add(marker.style.left);
      const expected = (Number(slider.value) / 1200) * 100;
      if (
        Math.abs(parseFloat(marker.style.left) - expected) > 0.00001 ||
        Math.abs(parseFloat(fill.style.width) - expected) > 0.00001 ||
        slider.getAttribute("aria-valuetext") !==
          document
            .querySelector(".map-heading .mono")!
            .textContent!.replace("t = ", "")
      )
        mismatches++;
      minMarkers = Math.min(
        minMarkers,
        document.querySelectorAll(".globe-labels .map-label").length,
      );
    }
    return {
      times: times.size,
      minMarkers,
      positions: positions.size,
      mismatches,
    };
  });
  expect(observation.times).toBeGreaterThan(5);
  expect(observation.minMarkers).toBe(52);
  expect(observation.positions).toBeGreaterThan(5);
  expect(observation.mismatches).toBe(0);
  await page.getByRole("button", { name: "Пауза", exact: true }).click();
  const paused = await time.inputValue();
  await page.getByRole("button", { name: "2D-карта", exact: true }).click();
  await expect(page.getByLabel("Двумерная карта сети")).toBeVisible();
  await expect(time).toHaveValue(paused);
  await page
    .getByRole("button", { name: "Воспроизвести", exact: true })
    .click();
  await expect(time).not.toHaveValue(paused);
  await page.getByRole("button", { name: "Пауза", exact: true }).click();
  expect(requests).toBe(0);
});
