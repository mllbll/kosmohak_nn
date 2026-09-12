import type {
  Scenario,
  Run,
  Route,
  Reason,
  SatState,
  GroundSite,
} from "./types";
export const reasons: Record<Reason, string> = {
  no_visible_sat: "Нет видимого спутника",
  no_gateway_contact: "Нет контакта со шлюзом",
  gateway_outage: "Шлюзы недоступны",
  isl_partition: "Разрыв межспутниковой сети",
};
export const clone = <T>(value: T): T => structuredClone(value);
export const same = (a: unknown, b: unknown) =>
  JSON.stringify(a) === JSON.stringify(b);
export const percent = (n: number) =>
  (n * 100).toLocaleString("ru-RU", {
    minimumFractionDigits: 1,
    maximumFractionDigits: 1,
  }) + "%";
export const number = (n: number) =>
  n.toLocaleString("ru-RU", { maximumFractionDigits: 2 });
export function clock(s: number) {
  return (
    [Math.floor(s / 3600), Math.floor(s / 60) % 60, Math.floor(s) % 60]
      .map((n) => String(n).padStart(2, "0"))
      .join(":") +
    (s % 1
      ? "." +
        s
          .toLocaleString("en-US", {
            useGrouping: false,
            maximumFractionDigits: 20,
          })
          .split(".")[1]
      : "")
  );
}
export function parseClock(s: string) {
  if (!/^\d{1,2}:\d{2}:\d{2}(\.\d+)?$/.test(s)) return NaN;
  const [h, m, t] = s.split(":").map(Number);
  return m < 60 && t < 60 ? h * 3600 + m * 60 + t : NaN;
}
export function duration(s: number) {
  return s >= 3600
    ? number(s / 3600) + " ч"
    : s >= 60
      ? number(s / 60) + " мин"
      : number(s) + " с";
}
export function snapTime(t: number, s: Scenario) {
  return Math.max(
    0,
    Math.min(
      s.environment.horizon_s - s.environment.step_s,
      Math.floor(t / s.environment.step_s) * s.environment.step_s,
    ),
  );
}
export function satelliteStatus(s: Scenario, id: string, t: number) {
  const sat = s.design.satellites.find((x) => x.id === id);
  if (!sat) return "unknown";
  if (sat.launch_batch > s.design.launch_stage) return "pending";
  return s.failures.some(
    (f) => f.satellite_id === id && f.start_s <= t && t < f.end_s,
  )
    ? "failed"
    : "active";
}
export function routeState(r: Route) {
  return r.path.length
    ? "available"
    : r.reason === "no_visible_sat"
      ? "invisible"
      : r.reason === "gateway_outage"
        ? "gateway"
        : "partition";
}
export function intervals(routes: Route[], client: string, step: number) {
  const output: {
    start: number;
    end: number;
    state: string;
    reason?: Reason;
  }[] = [];
  for (const r of routes
    .filter((r) => r.client_id === client)
    .sort((a, b) => a.t_s - b.t_s)) {
    const state = routeState(r),
      last = output.at(-1);
    if (
      last &&
      last.end === r.t_s &&
      last.state === state &&
      last.reason === r.reason
    )
      last.end = r.t_s + step;
    else
      output.push({ start: r.t_s, end: r.t_s + step, state, reason: r.reason });
  }
  return output;
}
export function comparisonProblem(a: Run, b: Run) {
  const ea = a.effective_scenario.environment,
    eb = b.effective_scenario.environment;
  if (ea.target_availability !== eb.target_availability)
    return "Разные целевые доли доступности. Установите одинаковую цель и повторите расчёт.";
  if (ea.horizon_s !== eb.horizon_s || ea.step_s !== eb.step_s)
    return "Разная временная сетка. Установите одинаковые период и шаг во вкладке «Проект» и повторите расчёт.";
  const pick = (s: Scenario) =>
    s.ground_sites
      .filter((x) => x.role === "client")
      .map(({ id, lat_deg, lon_deg }) => ({ id, lat_deg, lon_deg }));
  const ca = pick(a.effective_scenario),
    cb = pick(b.effective_scenario);
  if (
    !same(
      [...ca].sort((x, y) => x.id.localeCompare(y.id)),
      [...cb].sort((x, y) => x.id.localeCompare(y.id)),
    )
  )
    return "Состав или координаты клиентских пунктов различаются. Для корректного сравнения выберите одинаковые пункты.";
  return "";
}
export function configChanges(a: Scenario, b: Scenario) {
  const rows: { label: string; a: string; b: string }[] = [];
  const add = (label: string, x: unknown, y: unknown) => {
    if (!same(x, y))
      rows.push({
        label,
        a: typeof x === "object" ? JSON.stringify(x) : String(x),
        b: typeof y === "object" ? JSON.stringify(y) : String(y),
      });
  };
  add("Очередь запуска", a.design.launch_stage, b.design.launch_stage);
  const names: Record<string, string> = {
    altitude_km: "Высота, км",
    inclination_deg: "Наклонение, °",
    earth_angle0_deg: "Поворот Земли, °",
    horizon_s: "Период, с",
    step_s: "Шаг, с",
    min_elevation_deg: "Мин. возвышение, °",
    isl_range_km: "Дальность ISL, км",
    target_availability: "Целевая доля доступности",
  };
  for (const key of Object.keys(names) as (keyof Scenario["environment"])[])
    add(names[key], a.environment[key], b.environment[key]);
  for (const id of new Set(
    [...a.design.planes, ...b.design.planes].map((x) => x.id),
  )) {
    const x = a.design.planes.find((p) => p.id === id),
      y = b.design.planes.find((p) => p.id === id);
    add("RAAN " + id, x?.raan_deg ?? "—", y?.raan_deg ?? "—");
    add("Фаза " + id, x?.phase_deg ?? "—", y?.phase_deg ?? "—");
  }
  const failures = (s: Scenario) =>
    s.failures
      .map(
        (f) => f.satellite_id + ": " + clock(f.start_s) + "–" + clock(f.end_s),
      )
      .join("; ") || "Нет отказов";
  const gateways = (s: Scenario) =>
    s.gateway_outages
      .map((f) => f.gateway_id + ": " + clock(f.start_s) + "–" + clock(f.end_s))
      .join("; ") || "Нет отказов";
  add("Отказы спутников", failures(a), failures(b));
  add("Отказы шлюзов", gateways(a), gateways(b));
  const sites = (s: Scenario) =>
    s.ground_sites
      .map(
        (g) =>
          g.id +
          " (" +
          (g.role === "gateway" ? "шлюз" : "пункт") +
          "): " +
          String(g.lat_deg) +
          "°, " +
          String(g.lon_deg) +
          "°",
      )
      .join("; ");
  const satellites = (s: Scenario) =>
    s.design.satellites
      .map(
        (x) =>
          x.id +
          ": " +
          x.plane_id +
          ", " +
          String(x.slot_deg) +
          "°, очередь " +
          x.launch_batch,
      )
      .join("; ");
  add("Наземные пункты", sites(a), sites(b));
  add("Состав спутников", satellites(a), satellites(b));
  return rows;
}
export function validateScenario(input: unknown): string[] {
  const errors: string[] = [];
  if (!input || typeof input !== "object")
    return ["Ожидается объект проекта JSON"];
  const s = input as Scenario;
  if (s.schema_version !== "cosmo-A-1.0")
    errors.push("schema_version: ожидается cosmo-A-1.0");
  if (
    !s.meta ||
    typeof s.meta.title !== "string" ||
    typeof s.meta.id !== "string"
  )
    errors.push("meta: нужны строковые id и title");
  if (
    !s.environment ||
    !s.design ||
    !Array.isArray(s.design.planes) ||
    !Array.isArray(s.design.satellites) ||
    !Array.isArray(s.ground_sites)
  )
    return [
      ...errors,
      "Нужны environment, design.planes, design.satellites и ground_sites",
    ];
  if (
    [...s.design.planes, ...s.design.satellites, ...s.ground_sites].some(
      (x) => !x || typeof x !== "object",
    )
  )
    return [...errors, "Плоскости, спутники и пункты должны быть объектами"];
  const e = s.environment;
  const range = (
    label: string,
    n: unknown,
    min: number,
    max: number,
    exclusive = false,
  ) => {
    if (
      typeof n !== "number" ||
      !Number.isFinite(n) ||
      n < min ||
      (exclusive ? n >= max : n > max)
    )
      errors.push(
        label +
          ": допустимо от " +
          min +
          " до " +
          max +
          (exclusive ? " (верхняя граница не включена)" : ""),
      );
  };
  range("Высота, км", e.altitude_km, 200, 1200);
  range("Наклонение, °", e.inclination_deg, Number.MIN_VALUE, 180);
  range("Минимальное возвышение, °", e.min_elevation_deg, 0, 90, true);
  range("Дальность ISL, км", e.isl_range_km, Number.MIN_VALUE, 10000);
  range("Целевая доступность", e.target_availability, 0, 1);
  if (!Number.isFinite(e.earth_angle0_deg))
    errors.push("Начальный поворот Земли: нужно конечное число");
  if (
    !Number.isInteger(e.step_s) ||
    !Number.isInteger(e.horizon_s) ||
    e.step_s <= 0 ||
    e.horizon_s > 172800 ||
    e.horizon_s < e.step_s ||
    e.horizon_s % e.step_s !== 0
  )
    errors.push(
      "Сетка времени: положительные целые секунды, период кратен шагу и не больше 172800 с",
    );
  if (![1, 2, 3].includes(s.design.launch_stage))
    errors.push("Очередь запуска: 1, 2 или 3");
  const ids = (items: { id: string }[], label: string) => {
    const seen = new Set<string>();
    for (const item of items) {
      if (
        !item ||
        typeof item.id !== "string" ||
        !item.id.trim() ||
        seen.has(item.id)
      )
        errors.push(label + ": пустой или повторяющийся ID");
      seen.add(item?.id);
    }
    return seen;
  };
  const planes = ids(s.design.planes, "Плоскости"),
    sats = ids(s.design.satellites, "Спутники");
  ids(s.ground_sites, "Наземные пункты");
  if (!planes.size || !sats.size)
    errors.push("Нужны хотя бы одна плоскость и один спутник");
  for (const p of s.design.planes) {
    if (!p) continue;
    range(p.id + " · RAAN", p.raan_deg, 0, 360, true);
    range(p.id + " · Фаза", p.phase_deg, 0, 360, true);
  }
  for (const x of s.design.satellites) {
    if (!x) continue;
    if (!planes.has(x.plane_id)) errors.push(x.id + ": неизвестная плоскость");
    if (![1, 2, 3].includes(x.launch_batch))
      errors.push(x.id + ": очередь от 1 до 3");
    if (!Number.isFinite(x.slot_deg))
      errors.push(x.id + ": некорректное угловое положение");
  }
  for (const g of s.ground_sites) {
    if (!g) continue;
    range(g.id + " · Широта", g.lat_deg, -90, 90);
    range(g.id + " · Долгота", g.lon_deg, -180, 180);
    if (!["client", "gateway"].includes(g.role))
      errors.push(g.id + ": роль client или gateway");
    if (sats.has(g.id)) errors.push(g.id + ": совпадает с ID спутника");
  }
  if (
    !s.ground_sites.some((g) => g?.role === "client") ||
    !s.ground_sites.some((g) => g?.role === "gateway")
  )
    errors.push("Нужны хотя бы один клиентский пункт и один шлюз");
  for (const [list, key, allowed] of [
    [s.failures ?? [], "satellite_id", sats],
    [
      s.gateway_outages ?? [],
      "gateway_id",
      new Set(
        s.ground_sites.filter((g) => g?.role === "gateway").map((g) => g.id),
      ),
    ],
  ] as const) {
    if (!Array.isArray(list)) {
      errors.push(key + ": ожидается массив интервалов");
      continue;
    }
    for (const f of list) {
      const id = (f as unknown as Record<string, unknown>)?.[key];
      if (!f || !allowed.has(String(id)))
        errors.push(key + ": неизвестный объект " + id);
      if (
        !f ||
        !Number.isFinite(f.start_s) ||
        !Number.isFinite(f.end_s) ||
        f.start_s < 0 ||
        f.end_s > e.horizon_s ||
        f.start_s >= f.end_s
      )
        errors.push(
          String(id) +
            ": интервал должен удовлетворять 0 ≤ начало < конец ≤ период",
        );
    }
  }
  return errors;
}
export function parseScenario(text: string): Scenario {
  let value;
  try {
    value = JSON.parse(text);
  } catch {
    throw new Error("Некорректный JSON. Проверьте запятые, кавычки и скобки.");
  }
  if (value?.schema_version === "cosmo-A-result-1.0")
    value = value.effective_scenario;
  const errors = validateScenario(value);
  if (errors.length) throw new Error(errors.join("\n"));
  return {
    ...value,
    failures: value.failures ?? [],
    gateway_outages: value.gateway_outages ?? [],
  };
}
export function download(name: string, value: unknown) {
  const url = URL.createObjectURL(
    new Blob([JSON.stringify(value, null, 2)], {
      type: "application/json;charset=utf-8",
    }),
  );
  const a = document.createElement("a");
  a.href = url;
  a.download = name;
  a.click();
  setTimeout(() => URL.revokeObjectURL(url), 1000);
}
export function xyz(g: GroundSite): [number, number, number] {
  const lat = (g.lat_deg * Math.PI) / 180,
    lon = (g.lon_deg * Math.PI) / 180;
  return [
    6371 * Math.cos(lat) * Math.cos(lon),
    6371 * Math.cos(lat) * Math.sin(lon),
    6371 * Math.sin(lat),
  ];
}
export function lonLat(s: SatState): [number, number] {
  return [
    (Math.atan2(s.y_km, s.x_km) * 180) / Math.PI,
    (Math.atan2(s.z_km, Math.hypot(s.x_km, s.y_km)) * 180) / Math.PI,
  ];
}
