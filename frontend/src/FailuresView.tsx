import { useEffect, useState } from "react";
import {
  Plus,
  Pencil,
  Trash2,
  Satellite,
  RadioTower,
  Clock3,
  X,
} from "lucide-react";
import { Panel, Notice, Empty } from "./ui";
import { clone, clock, parseClock, duration } from "./domain";
import type { Scenario } from "./types";
export interface FailureSelection {
  kind: "satellite" | "gateway";
  id: string;
  start?: number;
}
interface Props {
  scenario: Scenario;
  setScenario: (s: Scenario) => void;
  busy: boolean;
  selection?: FailureSelection;
  onNetwork: () => void;
}
type Entry = {
  kind: "satellite" | "gateway";
  id: string;
  start: number;
  end: number;
  index: number;
};
export default function FailuresView({
  scenario: s,
  setScenario,
  busy,
  selection,
  onNetwork,
}: Props) {
  const [form, setForm] = useState<{
      kind: "satellite" | "gateway";
      id: string;
      start: string;
      end: string;
      index: number;
    } | null>(null),
    [error, setError] = useState(""),
    [filter, setFilter] = useState("all");
  const horizon = s.environment.horizon_s;
  const entries: Entry[] = [
    ...s.failures.map((x, index) => ({
      kind: "satellite" as const,
      id: x.satellite_id,
      start: x.start_s,
      end: x.end_s,
      index,
    })),
    ...s.gateway_outages.map((x, index) => ({
      kind: "gateway" as const,
      id: x.gateway_id,
      start: x.start_s,
      end: x.end_s,
      index,
    })),
  ];
  useEffect(() => {
    if (selection)
      setForm({
        kind: selection.kind,
        id: selection.id,
        start: clock(selection.start ?? 0),
        end: clock(horizon),
        index: -1,
      });
  }, [selection, horizon]);
  function open(entry?: Entry) {
    setError("");
    setForm(
      entry
        ? { ...entry, start: clock(entry.start), end: clock(entry.end) }
        : {
            kind: "satellite",
            id: s.design.satellites[0].id,
            start: clock(0),
            end: clock(horizon),
            index: -1,
          },
    );
  }
  function save() {
    if (!form) return;
    const start = parseClock(form.start),
      end = parseClock(form.end);
    if (
      !Number.isFinite(start) ||
      !Number.isFinite(end) ||
      start < 0 ||
      start >= end ||
      end > horizon
    ) {
      setError(
        "Укажите время ЧЧ:ММ:СС: 0 ≤ начало < конец ≤ " + clock(horizon),
      );
      return;
    }
    const next = clone(s);
    if (form.kind === "satellite") {
      const f = { satellite_id: form.id, start_s: start, end_s: end };
      if (form.index < 0) next.failures.push(f);
      else next.failures[form.index] = f;
    } else {
      const f = { gateway_id: form.id, start_s: start, end_s: end };
      if (form.index < 0) next.gateway_outages.push(f);
      else next.gateway_outages[form.index] = f;
    }
    setScenario(next);
    setForm(null);
    setError("");
  }
  function remove(entry: Entry) {
    const next = clone(s);
    if (entry.kind === "satellite") next.failures.splice(entry.index, 1);
    else next.gateway_outages.splice(entry.index, 1);
    setScenario(next);
    setForm(null);
  }
  const shown = entries.filter((x) => filter === "all" || x.kind === filter);
  return (
    <div className="page">
      <div className="page-heading">
        <div>
          <div className="eyebrow">УСТОЙЧИВОСТЬ / 02</div>
          <h1>Сценарий отказов</h1>
          <p>Проверьте, как сеть справится с потерей аппаратов и шлюзов</p>
        </div>
        <button className="primary" onClick={() => open()} disabled={busy}>
          <Plus />
          Добавить отказ
        </button>
      </div>
      <div className="failure-grid">
        <div className="stack">
          <div className="metric-grid">
            {[
              [
                Satellite,
                new Set(s.failures.map((x) => x.satellite_id)).size,
                "Спутников с отказами",
              ],
              [
                RadioTower,
                new Set(s.gateway_outages.map((x) => x.gateway_id)).size,
                "Шлюзов с отказами",
              ],
              [Clock3, entries.length, "Интервалов"],
            ].map(([Icon, n, label], i) => {
              const I = Icon as typeof Satellite;
              return (
                <div className="metric-tile" key={i}>
                  <I size={25} />
                  <div>
                    <strong>{String(n)}</strong>
                    <span>{String(label)}</span>
                  </div>
                </div>
              );
            })}
          </div>
          <Panel
            title="Периоды недоступности"
            action={
              <select
                aria-label="Тип отказа"
                value={filter}
                onChange={(e) => setFilter(e.target.value)}
              >
                <option value="all">Все типы</option>
                <option value="satellite">Спутники</option>
                <option value="gateway">Шлюзы</option>
              </select>
            }
          >
            {!shown.length ? (
              <Empty title="Отказы не заданы">
                Добавьте период недоступности или загрузите готовый сценарий во
                вкладке «Проект».
              </Empty>
            ) : (
              <div className="table-scroll">
                <table>
                  <thead>
                    <tr>
                      <th>Объект</th>
                      <th>Тип</th>
                      <th>Начало</th>
                      <th>Конец</th>
                      <th>Длительность</th>
                      <th>Действия</th>
                    </tr>
                  </thead>
                  <tbody>
                    {shown.map((x) => (
                      <tr
                        key={x.kind + x.index}
                        className={
                          form?.kind === x.kind && form.index === x.index
                            ? "selected-row"
                            : ""
                        }
                      >
                        <td>
                          <strong>{x.id}</strong>
                        </td>
                        <td>{x.kind === "satellite" ? "Спутник" : "Шлюз"}</td>
                        <td className="mono">{clock(x.start)}</td>
                        <td className="mono">{clock(x.end)}</td>
                        <td>{duration(x.end - x.start)}</td>
                        <td>
                          <div className="actions">
                            <button
                              className="icon-button"
                              disabled={busy}
                              title={"Редактировать " + x.id}
                              onClick={() => open(x)}
                            >
                              <Pencil size={15} />
                            </button>
                            <button
                              className="icon-button danger"
                              disabled={busy}
                              title={"Удалить отказ " + x.id}
                              onClick={() => remove(x)}
                            >
                              <Trash2 size={15} />
                            </button>
                          </div>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </Panel>
          {entries.length > 0 && (
            <Panel title="Отказы на временной шкале">
              <div className="outage-chart">
                <div className="chart-axis-row">
                  <span />
                  <div className="chart-axis">
                    {[0, 0.25, 0.5, 0.75, 1].map((x) => (
                      <span key={x} style={{ left: x * 100 + "%" }}>
                        {duration(x * horizon)}
                      </span>
                    ))}
                  </div>
                </div>
                {entries.map((x) => (
                  <div className="outage-row" key={x.kind + x.index}>
                    <strong>{x.id}</strong>
                    <div className="outage-track">
                      <button
                        aria-label={
                          x.id + " " + clock(x.start) + "–" + clock(x.end)
                        }
                        title={clock(x.start) + "–" + clock(x.end)}
                        disabled={busy}
                        className={"outage-bar " + x.kind}
                        style={{
                          left: (x.start / horizon) * 100 + "%",
                          width: ((x.end - x.start) / horizon) * 100 + "%",
                        }}
                        onClick={() => open(x)}
                      />
                    </div>
                  </div>
                ))}
              </div>
              <p className="footnote">
                Начало включено, конец не включён. Время от начала расчёта.
              </p>
            </Panel>
          )}
          <Notice tone="info">
            Пересекающиеся интервалы одного объекта учитываются как единый
            период недоступности. Положение отказавшего спутника сохраняется,
            его связи исключаются.
          </Notice>
        </div>
        <div className="stack">
          {form ? (
            <Panel
              title={
                form.index < 0 ? "Добавить отказ" : "Редактирование отказа"
              }
              action={
                <button
                  className="icon-button"
                  aria-label="Закрыть форму"
                  onClick={() => setForm(null)}
                >
                  <X size={17} />
                </button>
              }
            >
              <fieldset disabled={busy} className="form-reset">
                <div className="stack form-stack">
                  <label className="label">Тип объекта</label>
                  <div className="segmented">
                    {(["satellite", "gateway"] as const).map((kind) => (
                      <button
                        key={kind}
                        disabled={form.index >= 0 && form.kind !== kind}
                        className={form.kind === kind ? "selected" : ""}
                        onClick={() =>
                          setForm({
                            ...form,
                            kind,
                            id:
                              kind === "satellite"
                                ? s.design.satellites[0].id
                                : s.ground_sites.find(
                                    (x) => x.role === "gateway",
                                  )!.id,
                          })
                        }
                      >
                        {kind === "satellite" ? (
                          <Satellite size={17} />
                        ) : (
                          <RadioTower size={17} />
                        )}{" "}
                        {kind === "satellite" ? "Спутник" : "Шлюз"}
                      </button>
                    ))}
                  </div>
                  <label className="field">
                    <span>Объект</span>
                    <select
                      value={form.id}
                      onChange={(e) => setForm({ ...form, id: e.target.value })}
                    >
                      {(form.kind === "satellite"
                        ? s.design.satellites
                        : s.ground_sites.filter((x) => x.role === "gateway")
                      ).map((x) => (
                        <option key={x.id}>{x.id}</option>
                      ))}
                    </select>
                  </label>
                  <label className="field">
                    <span>Начало</span>
                    <input
                      aria-label="Начало отказа"
                      className="mono"
                      value={form.start}
                      placeholder="06:00:00"
                      onChange={(e) =>
                        setForm({ ...form, start: e.target.value })
                      }
                    />
                  </label>
                  <label className="field">
                    <span>Окончание</span>
                    <input
                      aria-label="Окончание отказа"
                      className="mono"
                      value={form.end}
                      placeholder="12:00:00"
                      onChange={(e) =>
                        setForm({ ...form, end: e.target.value })
                      }
                    />
                  </label>
                  <p className="footnote">
                    Допустимый период: 00:00:00–{clock(horizon)}. Можно ввести
                    24:00:00 и больше для многосуточного периода.
                  </p>
                  {error && <Notice tone="error">{error}</Notice>}
                  <div className="actions end">
                    <button onClick={() => setForm(null)}>Отмена</button>
                    <button className="primary" onClick={save}>
                      Применить
                    </button>
                  </div>
                </div>
              </fieldset>
            </Panel>
          ) : (
            <Panel>
              <Empty title="Выберите интервал">
                Нажмите на строку или полосу, чтобы изменить период
                недоступности.
              </Empty>
            </Panel>
          )}
          <Notice>
            <strong>После изменений нужен расчёт</strong>
            <p>
              Новые маршруты и затронутые пункты появятся после запуска расчёта.
            </p>
            <button className="text-button" onClick={onNetwork}>
              Перейти к состоянию сети →
            </button>
          </Notice>
        </div>
      </div>
    </div>
  );
}
