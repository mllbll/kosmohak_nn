import { useRef, useState } from "react";
import {
  Upload,
  Save,
  Download,
  RotateCcw,
  Search,
  Check,
  ArrowUpRight,
} from "lucide-react";
import { Panel, Field, Notice } from "./ui";
import { clone, duration, satelliteStatus, validateScenario } from "./domain";
import type { Scenario, Variant } from "./types";
interface Props {
  scenario: Scenario;
  setScenario: (s: Scenario) => void;
  variants: Variant[];
  activeKey: string;
  dirty: boolean;
  busy: boolean;
  onImport: (file: File) => void;
  onSave: () => void;
  onReset: () => void;
  onExport: () => void;
  onSelect: (v: Variant) => void;
  onPreset: (name: string) => void;
}
export default function ProjectView(p: Props) {
  const input = useRef<HTMLInputElement>(null),
    [objects, setObjects] = useState<"sites" | "satellites">("sites"),
    [query, setQuery] = useState("");
  const s = p.scenario,
    e = s.environment,
    errors = validateScenario(s);
  const edit = (fn: (s: Scenario) => void) => {
    const next = clone(s);
    fn(next);
    p.setScenario(next);
  };
  const presets = [
    ["01_full_constellation", "Полная группировка", "48 аппаратов"],
    ["02_first_launch", "Первая очередь", "16 аппаратов"],
    ["03_satellite_outages", "Отказы аппаратов", "10 отказов"],
    ["04_link_range", "Дальность 2000 км", "Ограничение ISL"],
  ];
  return (
    <div className="page">
      <div className="page-heading">
        <div>
          <div className="eyebrow">КОНФИГУРАЦИЯ / 01</div>
          <h1>Настройка проекта</h1>
          <p>От первой очереди до полной группировки</p>
        </div>
        <div className="actions">
          <input
            ref={input}
            type="file"
            accept=".json,application/json"
            hidden
            onChange={(e) => {
              const f = e.target.files?.[0];
              if (f) p.onImport(f);
              e.target.value = "";
            }}
          />
          <button disabled={p.busy} onClick={() => input.current?.click()}>
            <Upload />
            Загрузить JSON
          </button>
          <button disabled={p.busy || !!errors.length} onClick={p.onExport}>
            <Download />
            Экспорт проекта
          </button>
          <button
            className="primary"
            disabled={p.busy || !!errors.length}
            onClick={p.onSave}
          >
            <Save />
            Сохранить вариант
          </button>
        </div>
      </div>
      <div className="preset-grid">
        {presets.map(([key, title, detail]) => (
          <button
            key={key}
            className="preset"
            disabled={p.busy}
            onClick={() => p.onPreset(key)}
          >
            <span>
              <strong>{title}</strong>
              <small>{detail}</small>
            </span>
            <ArrowUpRight size={16} />
          </button>
        ))}
      </div>
      <fieldset disabled={p.busy} className="form-reset">
        <div className="project-grid">
          <div className="stack">
            <Panel title="Развёртывание">
              <label className="field title-field">
                <span>Название варианта</span>
                <input
                  value={s.meta.title}
                  maxLength={120}
                  onChange={(e) =>
                    edit((x) => {
                      x.meta.title = e.target.value;
                    })
                  }
                />
              </label>
              <div className="deployment">
                <div>
                  <p className="label">Очередь запуска</p>
                  <div className="segmented">
                    {[1, 2, 3].map((n) => (
                      <button
                        key={n}
                        aria-pressed={s.design.launch_stage === n}
                        className={
                          s.design.launch_stage === n ? "selected" : ""
                        }
                        onClick={() =>
                          edit((x) => {
                            x.design.launch_stage = n;
                          })
                        }
                      >
                        {n}{" "}
                        <span>
                          ·{" "}
                          {
                            s.design.satellites.filter(
                              (x) => x.launch_batch <= n,
                            ).length
                          }
                        </span>
                      </button>
                    ))}
                  </div>
                </div>
                <div className="stat-inline">
                  <strong>
                    {
                      s.design.satellites.filter(
                        (x) => x.launch_batch <= s.design.launch_stage,
                      ).length
                    }
                  </strong>
                  <span>
                    аппаратов запущено
                    <br />
                    {s.design.planes.length} орбитальных плоскостей
                  </span>
                </div>
              </div>
            </Panel>
            <Panel
              title="Орбитальные плоскости"
              action={<span className="muted">Углы в градусах</span>}
            >
              <div className="table-scroll">
                <table>
                  <thead>
                    <tr>
                      <th>Плоскость</th>
                      <th>RAAN, °</th>
                      <th>Фаза, °</th>
                      <th>Аппаратов</th>
                    </tr>
                  </thead>
                  <tbody>
                    {s.design.planes.map((plane, i) => (
                      <tr key={plane.id}>
                        <td>
                          <span
                            className="plane-dot"
                            style={{
                              background: ["#35d5f3", "#978bff", "#6ee6bb"][
                                i % 3
                              ],
                            }}
                          />
                          {plane.id}
                        </td>
                        {(["raan_deg", "phase_deg"] as const).map((key) => (
                          <td key={key}>
                            <input
                              aria-label={
                                plane.id +
                                " " +
                                (key === "raan_deg" ? "RAAN" : "Фаза")
                              }
                              type="number"
                              min={0}
                              max={359.999999}
                              step="any"
                              value={Number.isNaN(plane[key]) ? "" : plane[key]}
                              onChange={(e) =>
                                edit((x) => {
                                  x.design.planes[i][key] =
                                    e.target.value === ""
                                      ? NaN
                                      : Number(e.target.value);
                                })
                              }
                            />
                          </td>
                        ))}
                        <td>
                          {
                            s.design.satellites.filter(
                              (x) => x.plane_id === plane.id,
                            ).length
                          }
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
              <p className="footnote">
                RAAN меняет ориентацию плоскости, фаза — положение аппаратов
                вдоль неё. Допустимо 0° ≤ угол &lt; 360°.
              </p>
            </Panel>
            <Panel
              title={
                <div className="subtabs">
                  <button
                    className={objects === "sites" ? "active" : ""}
                    onClick={() => setObjects("sites")}
                  >
                    Наземные пункты <small>{s.ground_sites.length}</small>
                  </button>
                  <button
                    className={objects === "satellites" ? "active" : ""}
                    onClick={() => setObjects("satellites")}
                  >
                    Спутники <small>{s.design.satellites.length}</small>
                  </button>
                </div>
              }
              action={
                <label className="search">
                  <Search size={15} />
                  <input
                    aria-label="Найти объект"
                    placeholder="Найти объект"
                    value={query}
                    onChange={(e) => setQuery(e.target.value)}
                  />
                </label>
              }
            >
              <div className="table-scroll object-table">
                <table>
                  {objects === "sites" ? (
                    <>
                      <thead>
                        <tr>
                          <th>ID / Название</th>
                          <th>Роль</th>
                          <th>Широта, °</th>
                          <th>Долгота, °</th>
                        </tr>
                      </thead>
                      <tbody>
                        {s.ground_sites
                          .map((g, i) => ({ g, i }))
                          .filter(({ g }) =>
                            (g.id + " " + g.name)
                              .toLowerCase()
                              .includes(query.toLowerCase()),
                          )
                          .map(({ g, i }) => (
                            <tr key={g.id}>
                              <td>
                                <strong>{g.id}</strong>
                                <small className="cell-sub">{g.name}</small>
                              </td>
                              <td>
                                <span
                                  className={
                                    "badge " +
                                    (g.role === "gateway" ? "warning" : "info")
                                  }
                                >
                                  {g.role === "gateway" ? "Шлюз" : "Пункт"}
                                </span>
                              </td>
                              {(["lat_deg", "lon_deg"] as const).map((key) => (
                                <td key={key}>
                                  <input
                                    aria-label={
                                      g.id +
                                      " " +
                                      (key === "lat_deg" ? "Широта" : "Долгота")
                                    }
                                    type="number"
                                    step="any"
                                    min={key === "lat_deg" ? -90 : -180}
                                    max={key === "lat_deg" ? 90 : 180}
                                    value={Number.isNaN(g[key]) ? "" : g[key]}
                                    onChange={(e) =>
                                      edit((x) => {
                                        x.ground_sites[i][key] =
                                          e.target.value === ""
                                            ? NaN
                                            : Number(e.target.value);
                                      })
                                    }
                                  />
                                </td>
                              ))}
                            </tr>
                          ))}
                      </tbody>
                    </>
                  ) : (
                    <>
                      <thead>
                        <tr>
                          <th>ID</th>
                          <th>Плоскость</th>
                          <th>Положение, °</th>
                          <th>Очередь</th>
                          <th>На старте</th>
                        </tr>
                      </thead>
                      <tbody>
                        {s.design.satellites
                          .filter((x) =>
                            (x.id + " " + x.plane_id)
                              .toLowerCase()
                              .includes(query.toLowerCase()),
                          )
                          .map((x) => (
                            <tr key={x.id}>
                              <td>{x.id}</td>
                              <td>{x.plane_id}</td>
                              <td>{x.slot_deg}</td>
                              <td>{x.launch_batch}</td>
                              <td>
                                <span
                                  className={
                                    "badge " +
                                    (satelliteStatus(s, x.id, 0) === "active"
                                      ? "success"
                                      : satelliteStatus(s, x.id, 0) === "failed"
                                        ? "error"
                                        : "neutral")
                                  }
                                >
                                  {satelliteStatus(s, x.id, 0) === "active"
                                    ? "Работает"
                                    : satelliteStatus(s, x.id, 0) === "failed"
                                      ? "Отказ"
                                      : "Не запущен"}
                                </span>
                              </td>
                            </tr>
                          ))}
                      </tbody>
                    </>
                  )}
                </table>
              </div>
            </Panel>
          </div>
          <div className="stack">
            <Panel title="Условия расчёта">
              <div className="fields">
                {(
                  [
                    ["altitude_km", "Высота орбиты", "км", 200, 1200],
                    ["inclination_deg", "Наклонение", "°", 0.001, 180],
                    [
                      "earth_angle0_deg",
                      "Поворот Земли на старте",
                      "°",
                      undefined,
                      undefined,
                    ],
                    ["isl_range_km", "Дальность ISL", "км", 0.001, 10000],
                    ["min_elevation_deg", "Мин. возвышение", "°", 0, 89.999],
                    ["target_availability", "Целевая доступность", "%", 0, 100],
                    ["horizon_s", "Период расчёта", "с", 1, 172800],
                    ["step_s", "Шаг расчёта", "с", 1, 172800],
                  ] as const
                ).map(([key, label, unit, min, max]) => (
                  <Field
                    key={key}
                    label={label}
                    value={
                      key === "target_availability" ? e[key] * 100 : e[key]
                    }
                    min={min}
                    max={max}
                    unit={unit}
                    step={key === "step_s" || key === "horizon_s" ? 1 : "any"}
                    onChange={(n) =>
                      edit((x) => {
                        x.environment[key] =
                          key === "target_availability" ? n / 100 : n;
                      })
                    }
                  />
                ))}
              </div>
              <div className="card-footer">
                {duration(e.horizon_s)} ·{" "}
                {Number.isFinite(e.horizon_s / e.step_s)
                  ? Math.ceil(e.horizon_s / e.step_s)
                  : "—"}{" "}
                шагов
              </div>
              <p className="footnote">
                Изменение условий сохраняется в новом варианте проекта.
              </p>
            </Panel>
            {errors.length > 0 ? (
              <Notice tone="error">
                <strong>Проверьте параметры</strong>
                <ul>
                  {errors.map((x, i) => (
                    <li key={i}>{x}</li>
                  ))}
                </ul>
              </Notice>
            ) : p.dirty ? (
              <Notice>
                <strong>Есть изменения</strong>
                <p>
                  Сохраните вариант или запустите новый расчёт. Предыдущие
                  результаты останутся доступны для сравнения.
                </p>
              </Notice>
            ) : (
              <Notice tone="success">
                <strong>Конфигурация готова</strong>
                <p>Можно запускать расчёт сети.</p>
              </Notice>
            )}
            <Panel
              title="Сохранённые варианты"
              action={<span className="count">{p.variants.length}</span>}
            >
              <div className="saved-list">
                {p.variants.map((v) => (
                  <button
                    key={v.key}
                    className={p.activeKey === v.key ? "active" : ""}
                    onClick={() => p.onSelect(v)}
                  >
                    <span>
                      <strong>{v.title}</strong>
                      <small>
                        Очередь {v.scenario.design.launch_stage} ·{" "}
                        {v.runId ? "Есть расчёт" : "Без расчёта"}
                      </small>
                    </span>
                    {v.key === p.activeKey && <Check size={17} />}
                  </button>
                ))}
              </div>
              <p className="footnote">
                Варианты сохраняются в этом браузере. Скачайте JSON для переноса
                на другое устройство.
              </p>
            </Panel>
            <button className="quiet" onClick={p.onReset} disabled={!p.dirty}>
              <RotateCcw size={16} />
              Сбросить несохранённые изменения
            </button>
          </div>
        </div>
      </fieldset>
    </div>
  );
}
