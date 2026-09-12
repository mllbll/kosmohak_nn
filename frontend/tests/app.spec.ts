import { test, expect } from "@playwright/test";
test("project, outage, live calculations, synchronized maps and comparison", async ({
  page,
}) => {
  const errors: string[] = [];
  page.on("pageerror", (e) => errors.push(e.message));
  await page.goto("/");
  await expect(
    page.getByRole("heading", { name: "Настройка проекта" }),
  ).toBeVisible();
  await expect(page.getByLabel("P2 RAAN")).toHaveValue("60");
  await page.screenshot({ path: "test-results/project.png", fullPage: true });
  await page.getByLabel("P2 RAAN").fill("360");
  await expect(
    page.getByRole("button", { name: "Запустить расчёт", exact: true }),
  ).toBeDisabled();
  await page.getByLabel("P2 RAAN").fill("60");
  await page.getByLabel("Период расчёта", { exact: true }).fill("1200");
  await page
    .getByRole("button", { name: "Запустить расчёт", exact: true })
    .click();
  await expect(
    page.getByRole("heading", { name: "Состояние сети", exact: true }),
  ).toBeVisible({ timeout: 150000 });
  await expect(page.locator(".globe-host canvas")).toBeVisible();
  await expect(page.locator(".map-busy")).toHaveCount(0, { timeout: 30000 });
  await expect(page.locator(".timeline-row")).toHaveCount(3);
  await page.getByRole("button", { name: "Слои", exact: true }).click();
  await expect(page.getByLabel("Орбиты", { exact: true })).toBeChecked();
  await page.getByLabel("Орбиты", { exact: true }).uncheck();
  await page.getByRole("button", { name: "Слои", exact: true }).click();
  await page.getByLabel("Время расчёта", { exact: true }).fill("00:04:00");
  await page.getByLabel("Время расчёта", { exact: true }).press("Enter");
  await expect(page.locator(".map-busy")).toHaveCount(0, { timeout: 30000 });
  await page.screenshot({
    path: "test-results/network-3d.png",
    fullPage: true,
  });
  await page.getByRole("button", { name: "2D-карта", exact: true }).click();
  await expect(page.getByLabel("Двумерная карта сети")).toBeVisible();
  await expect(page.getByLabel("Время расчёта", { exact: true })).toHaveValue(
    "00:04:00",
  );
  await expect.poll(() => page.locator(".svg-node").count()).toBeGreaterThan(4);
  await page.screenshot({
    path: "test-results/network-2d.png",
    fullPage: true,
  });
  const alignment = await page.evaluate(() => {
    const ticks = document
        .querySelector(".tick-labels")!
        .getBoundingClientRect(),
      bar = document
        .querySelector(".availability-track")!
        .getBoundingClientRect();
    return Math.abs(ticks.left - bar.left) + Math.abs(ticks.width - bar.width);
  });
  expect(alignment).toBeLessThan(1);
  await page.route("**/snapshot?**", async (route) => {
    await new Promise((resolve) => setTimeout(resolve, 700));
    await route.continue().catch(() => {});
  });
  await page.getByLabel("Время расчёта", { exact: true }).fill("00:06:00");
  await page.getByLabel("Время расчёта", { exact: true }).press("Enter");
  await expect(page.locator(".map-busy")).toBeVisible();
  await page.getByLabel("Текущий проект").selectOption({ index: 0 });
  await expect(page.getByText("Сеть ещё не рассчитана")).toBeVisible();
  await expect(page.locator(".map-busy")).toHaveCount(0);
  await expect(page.locator(".svg-node")).toHaveCount(4);
  await page.unroute("**/snapshot?**");
  await page.getByLabel("Текущий проект").selectOption({ index: 1 });
  await expect.poll(() => page.locator(".svg-node").count()).toBeGreaterThan(4);
  await page.getByRole("button", { name: "Отказы", exact: true }).click();
  await page
    .getByRole("button", { name: "Добавить отказ", exact: true })
    .click();
  await page.getByLabel("Начало отказа").fill("00:08:00");
  await page.getByLabel("Окончание отказа").fill("00:04:00");
  await page.getByRole("button", { name: "Применить", exact: true }).click();
  await expect(page.getByRole("alert")).toContainText("начало < конец");
  await page.getByLabel("Начало отказа").fill("00:02:00");
  await page.getByLabel("Окончание отказа").fill("00:10:00");
  await page.getByRole("button", { name: "Применить", exact: true }).click();
  await expect(page.getByRole("cell", { name: "00:10:00" })).toBeVisible();
  await page.screenshot({ path: "test-results/failures.png", fullPage: true });
  await page
    .getByRole("button", { name: "Запустить расчёт", exact: true })
    .click();
  await expect(
    page.getByRole("heading", { name: "Состояние сети", exact: true }),
  ).toBeVisible({ timeout: 150000 });
  await page.getByRole("button", { name: "Сравнение", exact: true }).click();
  await expect(
    page.getByRole("heading", { name: "Показатели по наземным пунктам" }),
  ).toBeVisible();
  await expect(page.locator(".comparison-bar-row")).toHaveCount(4);
  await expect(page.getByText("Получение заключения…")).toHaveCount(0);
  await page.screenshot({ path: "test-results/compare.png", fullPage: true });
  expect(errors).toEqual([]);
  await page.reload();
  await page.getByRole("button", { name: "Сравнение", exact: true }).click();
  await expect(
    page.getByRole("heading", { name: "Показатели по наземным пунктам" }),
  ).toBeVisible();
});
test("invalid import retains project; small viewport stays usable", async ({
  page,
}) => {
  await page.goto("/");
  await expect(
    page.getByRole("heading", { name: "Настройка проекта" }),
  ).toBeVisible();
  await page
    .locator("input[type=file]")
    .setInputFiles({
      name: "bad.json",
      mimeType: "application/json",
      buffer: Buffer.from('{"schema_version":"wrong"}'),
    });
  await expect(page.getByRole("alert")).toContainText("schema_version");
  await expect(page.getByLabel("P2 RAAN")).toHaveValue("60");
  await page.setViewportSize({ width: 390, height: 844 });
  await page.screenshot({ path: "test-results/mobile.png", fullPage: true });
  const overflow = await page.evaluate(
    () => document.documentElement.scrollWidth - innerWidth,
  );
  expect(overflow).toBeLessThanOrEqual(1);
  await page
    .getByRole("button", { name: "Состояние сети", exact: true })
    .click();
  await expect(page.getByText("Сеть ещё не рассчитана")).toBeVisible();
});
