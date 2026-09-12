import { test, expect } from "@playwright/test";

test("simulation keeps the previous map while loading and uses the selected step", async ({
  page,
}) => {
  await page.goto("/");
  await page.getByLabel("Период расчёта", { exact: true }).fill("1200");
  await page
    .getByRole("button", { name: "Запустить расчёт", exact: true })
    .click();
  await expect(page.locator(".globe-host canvas")).toBeVisible();
  await expect(page.locator(".map-busy")).toHaveCount(0);
  const input = page.getByLabel("Время расчёта", { exact: true });
  await input.fill("00:04:00");
  await input.press("Enter");
  await expect(page.locator(".map-heading")).toContainText("00:04:00");
  await expect(page.locator(".map-busy")).toHaveCount(0);
  const markers = page.locator(".globe-labels .map-label");
  await expect(markers).toHaveCount(52);
  const oldMarkers = await markers.evaluateAll((nodes) =>
    nodes.map((n) => n.getAttribute("aria-label")),
  );
  let release!: () => void;
  const gate = new Promise<void>((resolve) => {
    release = resolve;
  });
  await page.route("**/snapshot?**", async (route) => {
    await gate;
    await route.continue();
  });
  await page.getByLabel("Шаг симуляции", { exact: true }).fill("240");
  await page
    .getByRole("button", { name: "Следующий шаг", exact: true })
    .click();
  await expect(page.locator(".map-busy")).toBeVisible();
  await expect(input).toHaveValue("00:08:00");
  await expect(page.getByLabel("Временная шкала", { exact: true })).toHaveValue(
    "240",
  );
  await expect(page.locator(".map-heading")).toContainText("00:04:00");
  expect(
    await markers.evaluateAll((nodes) =>
      nodes.map((n) => n.getAttribute("aria-label")),
    ),
  ).toEqual(oldMarkers);
  release();
  await expect(page.locator(".map-busy")).toHaveCount(0);
  await expect(page.locator(".map-heading")).toContainText("00:08:00");
  await expect(page.getByLabel("Временная шкала", { exact: true })).toHaveValue(
    "480",
  );
  await expect(markers).toHaveCount(52);
  await page
    .getByRole("button", { name: "Воспроизвести", exact: true })
    .click();
  await expect(input).toHaveValue("00:12:00");
  await page.getByRole("button", { name: "Пауза", exact: true }).click();
});

test("3D markers stay at their projected coordinates during rotation and playback", async ({
  page,
}) => {
  await page.goto("/");
  await page.getByLabel("Период расчёта", { exact: true }).fill("1200");
  await page
    .getByRole("button", { name: "Запустить расчёт", exact: true })
    .click();
  const canvas = page.locator(".globe-host canvas");
  await expect(canvas).toBeVisible();
  await expect(page.locator(".map-busy")).toHaveCount(0);
  await expect(
    page.locator(".globe-labels .map-label:visible").first(),
  ).toBeVisible();
  // Measure the visible symbol against the renderer's target on every frame,
  // including the first visible frame of each new simulation snapshot.
  await page.evaluate(() => {
    const probe = { maxError: 0, samples: 0, frames: 0, running: true };
    (window as unknown as { markerProbe: typeof probe }).markerProbe = probe;
    const sample = () => {
      if (!probe.running) return;
      const host = document
        .querySelector(".globe-host")!
        .getBoundingClientRect();
      for (const label of document.querySelectorAll<HTMLButtonElement>(
        ".globe-labels .map-label",
      )) {
        if (getComputedStyle(label).display === "none") continue;
        const match = label.style.transform.match(
          /translate\(([-\d.]+)px,\s*([-\d.]+)px\)$/,
        );
        if (!match) {
          probe.maxError = Infinity;
          continue;
        }
        const symbol = label
          .querySelector(".node-symbol")!
          .getBoundingClientRect();
        probe.maxError = Math.max(
          probe.maxError,
          Math.abs(
            symbol.left + symbol.width / 2 - host.left - Number(match[1]),
          ),
          Math.abs(
            symbol.top + symbol.height / 2 - host.top - Number(match[2]),
          ),
        );
        probe.samples++;
      }
      probe.frames++;
      requestAnimationFrame(sample);
    };
    requestAnimationFrame(sample);
  });
  const box = (await canvas.boundingBox())!;
  await page.mouse.move(box.x + box.width * 0.65, box.y + box.height * 0.6);
  await page.mouse.down();
  await page.mouse.move(box.x + box.width * 0.35, box.y + box.height * 0.45, {
    steps: 30,
  });
  await page.mouse.up();
  const time = page.getByLabel("Время расчёта", { exact: true });
  const before = await time.inputValue();
  await page
    .getByRole("button", { name: "Воспроизвести", exact: true })
    .click();
  await expect(time).not.toHaveValue(before);
  await page.getByRole("button", { name: "Пауза", exact: true }).click();
  await expect(page.locator(".map-busy")).toHaveCount(0);
  const result = await page.evaluate(() => {
    const probe = (
      window as unknown as {
        markerProbe: {
          maxError: number;
          samples: number;
          frames: number;
          running: boolean;
        };
      }
    ).markerProbe;
    probe.running = false;
    return probe;
  });
  expect(result.samples).toBeGreaterThan(100);
  expect(result.frames).toBeGreaterThan(5);
  expect(result.maxError).toBeLessThan(1);
});
