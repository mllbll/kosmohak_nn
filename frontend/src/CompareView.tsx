import { useEffect, useMemo, useState } from "react";
import { Download, ArrowUpRight, CheckCircle2 } from "lucide-react";
import { Panel, Notice, Empty } from "./ui";
import { api, ApiError } from "./api";
import {
  comparisonProblem,
  configChanges,
  percent,
  number,
  duration,
  download,
} from "./domain";
import type { Run, Comparison } from "./types";
interface Props {
  runs: Run[];
  onOpen: (run: Run) => void;
  onProject: () => void;
}
const readable = (text: string) =>
  text
    .replace(/path_ratio/g, "доступность связи")
    .replace(/max_gap/g, "максимальный перерыв")
    .replace(/mean_hops/g, "среднее число переходов")
    .replace(/hops/g, "переходов")
    .replace(/what-if/g, "сценарий отказов");
export default function CompareView({ runs, onOpen, onProject }: Props) {
  const [aId, setAId] = useState(runs[0]?.id ?? ""),
    [bId, setBId] = useState(runs[1]?.id ?? ""),
    [result, setResult] = useState<Comparison>(),
    [error, setError] = useState(""),
    [loading, setLoading] = useState(false);
  const a = runs.find((x) => x.id === aId),
    b = runs.find((x) => x.id === bId),
    problem =
      a && b
        ? a.id === b.id
          ? "Выберите два разных расчёта."
          : comparisonProblem(a, b)
        : "";
  const changes = useMemo(
    () =>
      a && b ? configChanges(a.effective_scenario, b.effective_scenario) : [],
    [a, b],
  );
  useEffect(() => {
    if (!aId && runs[0]) setAId(runs[0].id);
    if (!bId && runs[1]) setBId(runs[1].id);
  }, [runs, aId, bId]);
  useEffect(() => {
    setResult(undefined);
    setError("");
    if (!a || !b || problem) {
      setLoading(false);
      return;
    }
    let active = true;
    setLoading(true);
    api
      .compare(a.id, b.id)
      .then((r) => {
        if (active) setResult(r);
      })
      .catch((e) => {
        if (active)
          setError(
            e instanceof ApiError && e.status === 404
              ? "Сервер перезапущен: локальные показатели сохранены, но для серверного заключения повторите расчёты."
              : e.message,
          );
      })
      .finally(() => {
        if (active) setLoading(false);
      });
    return () => {
      active = false;
    };
  }, [a, b, problem]);
  const label = (r: Run) =>
    r.effective_scenario.meta.title + " · " + r.id.slice(0, 6);
  return (
    <div className="page">
      <div className="page-heading">
        <div>
          <div className="eyebrow">АНАЛИЗ ВАРИАНТОВ / 04</div>
          <h1>Сравнение вариантов</h1>
          <p>Оцените изменения доступности и устойчивости сети</p>
        </div>
        <button
          disabled={!a || !b || !!problem}
          onClick={() =>
            download("comparison.json", {
              run_a: a,
              run_b: b,
              config_changes: changes,
              comparison: result ?? null,
            })
          }
        >
          <Download />
          Экспорт сравнения
        </button>
      </div>
      {runs.length < 2 ? (
        <Panel>
          <Empty
            title="Нужны два расчёта"
            action={
              <button className="primary" onClick={onProject}>
                Настроить следующий вариант
              </button>
            }
          >
            Рассчитайте исходную группировку, затем измените конфигурацию или
            отказы и запустите второй расчёт. Оба результата появятся здесь.
          </Empty>
        </Panel>
      ) : (
        <>
          <div className="compare-selects">
            <label>
              <span className="variant-letter a">A</span>
              <select
                aria-label="Вариант A"
                value={aId}
                onChange={(e) => setAId(e.target.value)}
              >
                {runs.map((r) => (
                  <option key={r.id} value={r.id}>
                    {label(r)}
                  </option>
                ))}
              </select>
            </label>
            <span className="muted">→</span>
            <label>
              <span className="variant-letter b">B</span>
              <select
                aria-label="Вариант B"
                value={bId}
                onChange={(e) => setBId(e.target.value)}
              >
                {runs.map((r) => (
                  <option key={r.id} value={r.id}>
                    {label(r)}
                  </option>
                ))}
              </select>
            </label>
          </div>
          {problem ? (
            <Notice tone="error">{problem}</Notice>
          ) : (
            a &&
            b && (
              <>
                <Notice tone="success">
                  <strong>
                    Единая сетка:{" "}
                    {duration(a.effective_scenario.environment.horizon_s)} · шаг{" "}
                    {duration(a.effective_scenario.environment.step_s)}
                  </strong>
                  <span className="notice-detail">
                    Показатели рассчитаны для одинаковых клиентских пунктов.
                  </span>
                </Notice>
                {changes.some((x) =>
                  [
                    "Высота, км",
                    "Дальность ISL, км",
                    "Мин. возвышение, °",
                    "Наклонение, °",
                    "Поворот Земли, °",
                  ].includes(x.label),
                ) && (
                  <Notice>
                    Условия эксперимента различаются. Изменения орбиты и связи
                    перечислены ниже; вывод относится к этим условиям.
                  </Notice>
                )}
                <div className="compare-grid">
                  <Panel title="Изменения конфигурации">
                    <div className="table-scroll">
                      <table>
                        <thead>
                          <tr>
                            <th>Параметр</th>
                            <th>A · Базовый</th>
                            <th>B · Сравниваемый</th>
                          </tr>
                        </thead>
                        <tbody>
                          {changes.map((x, i) => (
                            <tr key={i}>
                              <td>{x.label}</td>
                              <td className="diff-value">{x.a}</td>
                              <td className="diff-value cyan-text">{x.b}</td>
                            </tr>
                          ))}
                        </tbody>
                      </table>
                    </div>
                    {!changes.length && (
                      <p className="muted">Параметры проектов совпадают.</p>
                    )}
                  </Panel>
                  <Panel
                    title="Доступность до шлюза"
                    action={
                      <div className="legend">
                        <span>
                          <i className="bar-a" />A
                        </span>
                        <span>
                          <i className="bar-b" />B
                        </span>
                      </div>
                    }
                  >
                    <div className="comparison-chart">
                      {a.metrics.map((m) => {
                        const n = b.metrics.find(
                          (x) => x.client_id === m.client_id,
                        )!;
                        return (
                          <div className="comparison-bar-row" key={m.client_id}>
                            <strong>{m.client_id}</strong>
                            <div className="comparison-bars">
                              <span
                                className="target-line"
                                style={{
                                  left:
                                    a.effective_scenario.environment
                                      .target_availability *
                                      100 +
                                    "%",
                                }}
                                title={
                                  "Цель A: " +
                                  percent(
                                    a.effective_scenario.environment
                                      .target_availability,
                                  )
                                }
                              />
                              {a.effective_scenario.environment
                                .target_availability !==
                                b.effective_scenario.environment
                                  .target_availability && (
                                <span
                                  className="target-line target-b"
                                  style={{
                                    left:
                                      b.effective_scenario.environment
                                        .target_availability *
                                        100 +
                                      "%",
                                  }}
                                />
                              )}
                              <div
                                className="bar-a"
                                style={{ width: m.path_ratio * 100 + "%" }}
                              />
                              <div
                                className="bar-b"
                                style={{ width: n.path_ratio * 100 + "%" }}
                              />
                            </div>
                            <span className="bar-values">
                              {percent(m.path_ratio)}
                              <br />
                              <strong className="cyan-text">
                                {percent(n.path_ratio)}
                              </strong>
                            </span>
                          </div>
                        );
                      })}
                      <div className="comparison-bar-row">
                        <span />
                        <div className="chart-axis">
                          <span style={{ left: "0%" }}>0%</span>
                          <span style={{ left: "50%" }}>50%</span>
                          <span style={{ left: "100%" }}>100%</span>
                        </div>
                        <span />
                      </div>
                    </div>
                    <p className="footnote">
                      Пунктир: цель A{" "}
                      {percent(
                        a.effective_scenario.environment.target_availability,
                      )}
                      , B{" "}
                      {percent(
                        b.effective_scenario.environment.target_availability,
                      )}
                      .
                    </p>
                  </Panel>
                </div>
                <Panel
                  title="Показатели по наземным пунктам"
                  action={<span className="muted">В каждой ячейке: A → B</span>}
                >
                  <div className="table-scroll">
                    <table>
                      <thead>
                        <tr>
                          <th>Пункт</th>
                          <th>Видимость</th>
                          <th>Доступность</th>
                          <th>Δ, п.п.</th>
                          <th>Макс. перерыв</th>
                          <th>Среднее число переходов*</th>
                        </tr>
                      </thead>
                      <tbody>
                        {a.metrics.map((m) => {
                          const n = b.metrics.find(
                              (x) => x.client_id === m.client_id,
                            )!,
                            d = (n.path_ratio - m.path_ratio) * 100;
                          return (
                            <tr key={m.client_id}>
                              <td>
                                <strong>{m.client_id}</strong>
                              </td>
                              <td>
                                {percent(m.visibility_ratio)}{" "}
                                <span className="muted">→</span>{" "}
                                {percent(n.visibility_ratio)}
                              </td>
                              <td>
                                {percent(m.path_ratio)}{" "}
                                <span className="muted">→</span>{" "}
                                <strong
                                  className={
                                    n.meets_target
                                      ? "success-text"
                                      : "warning-text"
                                  }
                                >
                                  {percent(n.path_ratio)}
                                </strong>
                              </td>
                              <td
                                className={
                                  d > 0
                                    ? "success-text"
                                    : d < 0
                                      ? "error-text"
                                      : "muted"
                                }
                              >
                                {d > 0 ? "+" : ""}
                                {number(d)}
                              </td>
                              <td>
                                {duration(m.max_gap_s)} →{" "}
                                {duration(n.max_gap_s)}
                              </td>
                              <td>
                                {m.path_ratio ? number(m.mean_hops) : "—"} →{" "}
                                {n.path_ratio ? number(n.mean_hops) : "—"}
                              </td>
                            </tr>
                          );
                        })}
                      </tbody>
                    </table>
                  </div>
                  <p className="footnote">
                    * Только для отсчётов с доступным маршрутом. Переход — одно
                    ребро, включая две наземные линии.
                  </p>
                </Panel>
                <div className="compare-grid">
                  <Panel title="Достижение целевой доступности">
                    <div className="target-cards">
                      {[a, b].map((r, i) => (
                        <div key={r.id}>
                          <h3>
                            {i ? "B" : "A"} ·{" "}
                            {r.metrics.filter((m) => m.meets_target).length} из{" "}
                            {r.metrics.length} пунктов
                          </h3>
                          <p className="muted">
                            Цель ≥{" "}
                            {percent(
                              r.effective_scenario.environment
                                .target_availability,
                            )}
                          </p>
                          {r.metrics.map((m) => (
                            <div className="target-row" key={m.client_id}>
                              <span
                                className={
                                  m.meets_target
                                    ? "success-text"
                                    : "warning-text"
                                }
                              >
                                {m.meets_target ? "✓" : "!"} {m.client_id}
                              </span>
                              <strong>{percent(m.path_ratio)}</strong>
                            </div>
                          ))}
                        </div>
                      ))}
                    </div>
                  </Panel>
                  <Panel title="Вывод по сравнению">
                    {loading ? (
                      <p className="muted">
                        <span className="spinner" /> Получение заключения…
                      </p>
                    ) : error ? (
                      <Notice tone="warning">{error}</Notice>
                    ) : result ? (
                      <>
                        <h3 className="cyan-text">
                          <CheckCircle2 size={18} />{" "}
                          {readable(result.recommendation.conclusion)}
                        </h3>
                        <p>{readable(result.recommendation.reason)}</p>
                        {result.recommendation.advantages?.length > 0 && (
                          <ul>
                            {result.recommendation.advantages.map((x, i) => (
                              <li key={i}>{readable(x)}</li>
                            ))}
                          </ul>
                        )}
                        <details>
                          <summary>Условия и ограничения</summary>
                          <ul>
                            {[
                              ...(result.recommendation.conditions ?? []),
                              ...(result.recommendation.limitations ?? []),
                            ].map((x, i) => (
                              <li key={i}>{readable(x)}</li>
                            ))}
                          </ul>
                        </details>
                      </>
                    ) : null}
                    <button className="primary" onClick={() => onOpen(b)}>
                      Открыть вариант B на карте <ArrowUpRight size={16} />
                    </button>
                  </Panel>
                </div>
              </>
            )
          )}
        </>
      )}
    </div>
  );
}
