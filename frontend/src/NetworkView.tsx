import { lazy, Suspense, useEffect, useState } from "react";
import {
  Settings2,
  Layers3,
  Download,
  ChevronRight,
  Satellite,
  RadioTower,
  MapPin,
  CheckCircle2,
  XCircle,
  PanelRightClose,
  PanelRightOpen,
} from "lucide-react";
import type { Scenario, Run, Snapshot, Layers } from "./types";
import { api, ApiError } from "./api";
import {
  clock,
  percent,
  duration,
  reasons,
  satelliteStatus,
  number,
} from "./domain";
import { Empty, Notice } from "./ui";
import Timeline from "./Timeline";
import type { FailureSelection } from "./FailuresView";
const Globe = lazy(() => import("./Globe"));
const Map2D = lazy(() => import("./Map2D"));
interface Props {
  scenario: Scenario;
  run?: Run;
  stale: boolean;
  onConfigure: () => void;
  onRun: () => void;
  onFailure: (v: FailureSelection) => void;
  onExport: () => void;
  busy: boolean;
}
export default function NetworkView(p: Props) {
  const s = p.run?.effective_scenario ?? p.scenario,
    clients = s.ground_sites.filter((x) => x.role === "client");
  const [mode, setMode] = useState<"2d" | "3d">("3d"),
    [layers, setLayers] = useState<Layers>({
      satellites: true,
      sites: true,
      links: true,
      route: true,
      orbits: true,
      labels: true,
    }),
    [layersOpen, setLayersOpen] = useState(false),
    [inspector, setInspector] = useState(true);
  const [client, setClient] = useState(clients[0]?.id ?? ""),
    [selected, setSelected] = useState(""),
    [time, setTime] = useState(0),
    [playing, setPlaying] = useState(false),
    [snapshot, setSnapshot] = useState<Snapshot>(),
    [loading, setLoading] = useState(false),
    [error, setError] = useState(""),
    [retry, setRetry] = useState(0);
  useEffect(() => {
    setTime(0);
    setPlaying(false);
    setSnapshot(undefined);
    setClient(
      p.run?.effective_scenario.ground_sites.find((x) => x.role === "client")
        ?.id ??
        p.scenario.ground_sites.find((x) => x.role === "client")?.id ??
        "",
    );
    setSelected("");
  }, [p.run?.id, p.scenario.meta.id]);
  useEffect(() => {
    if (!p.run || !client) {
      setLoading(false);
      setSnapshot(undefined);
      setError("");
      return;
    }
    const controller = new AbortController();
    setLoading(true);
    setSnapshot(undefined);
    setError("");
    const timer = setTimeout(() => {
      api
        .snapshot(p.run!.id, time, client, controller.signal)
        .then((value) => {
          if (!controller.signal.aborted) setSnapshot(value);
        })
        .catch((err) => {
          if (!controller.signal.aborted) {
            setError(
              err instanceof ApiError && err.status === 404
                ? "Сервер больше не хранит этот расчёт. Запустите его повторно; сохранённые метрики остаются доступны."
                : err.message,
            );
            setPlaying(false);
          }
        })
        .finally(() => {
          if (!controller.signal.aborted) setLoading(false);
        });
    }, 90);
    return () => {
      clearTimeout(timer);
      controller.abort();
    };
  }, [p.run?.id, time, client, retry]);
  useEffect(() => {
    if (!playing || loading || !snapshot) return;
    const id = setTimeout(() => {
      if (time >= s.environment.horizon_s - s.environment.step_s) {
        setPlaying(false);
        return;
      }
      setTime((t) => t + s.environment.step_s);
    }, 650);
    return () => clearTimeout(id);
  }, [playing, loading, snapshot, time, s.environment]);
  function select(id: string) {
    setSelected(id);
    setInspector(true);
    if (s.ground_sites.some((x) => x.id === id && x.role === "client"))
      setClient(id);
  }
  const selectedSat = s.design.satellites.find((x) => x.id === selected),
    selectedGround = s.ground_sites.find((x) => x.id === selected),
    metric = p.run?.metrics.find((x) => x.client_id === client),
    route = snapshot?.route;
  const counts = snapshot
    ? {
        active: snapshot.snapshot.satellites.filter((x) => x.active).length,
        failed: snapshot.snapshot.satellites.filter(
          (x) => satelliteStatus(s, x.id, time) === "failed",
        ).length,
      }
    : null;
  const launched = s.design.satellites.filter(
    (x) => x.launch_batch <= s.design.launch_stage,
  ).length;
  return (
    <div className="page network-page">
      <div className="network-summary">
        <span>
          <strong>{s.meta.title}</strong>
          <i />
          Очередь {s.design.launch_stage}
          <i />
          Запущено {launched}
          {counts && (
            <>
              <i />
              <span className="success-text">Работают {counts.active}</span>
              <i />
              <span className={counts.failed ? "error-text" : "muted"}>
                В отказе {counts.failed}
              </span>
            </>
          )}
          <i />
          {duration(s.environment.horizon_s)} · шаг{" "}
          {duration(s.environment.step_s)}
        </span>
        <div className="actions">
          <button className="small" onClick={p.onConfigure}>
            <Settings2 size={15} />
            Настроить
          </button>
          <button
            className="small"
            disabled={!p.run || p.busy}
            onClick={p.onExport}
          >
            <Download size={15} />
            Экспорт результата
          </button>
        </div>
      </div>
      {p.stale && (
        <Notice>
          Настройки изменены. Здесь показан сохранённый результат «
          {s.meta.title}». Для применения изменений запустите новый расчёт.
        </Notice>
      )}
      {error && (
        <Notice tone="error">
          {error}{" "}
          <button onClick={() => setRetry((x) => x + 1)}>
            Повторить загрузку снимка
          </button>
        </Notice>
      )}
      <div className={"network-layout " + (!inspector ? "expanded" : "")}>
        <section className="panel map-panel">
          <div className="map-heading">
            <div>
              <h1>Состояние сети</h1>
              <span className="muted mono">
                {p.run ? "t = " + clock(time) : "Обзор наземных пунктов"}
              </span>
            </div>
            <div className="actions">
              <div className="segmented small">
                <button
                  className={mode === "2d" ? "selected" : ""}
                  onClick={() => setMode("2d")}
                >
                  2D-карта
                </button>
                <button
                  className={mode === "3d" ? "selected" : ""}
                  onClick={() => setMode("3d")}
                >
                  3D-глобус
                </button>
              </div>
              <div className="layers-menu">
                <button
                  className="small"
                  aria-expanded={layersOpen}
                  onClick={() => setLayersOpen(!layersOpen)}
                >
                  <Layers3 size={16} />
                  Слои
                </button>
                {layersOpen && (
                  <div className="layers-popover">
                    {(
                      Object.entries({
                        satellites: "Спутники",
                        sites: "Наземные пункты",
                        links: "Связи",
                        route: "Маршрут",
                        orbits: "Орбиты",
                        labels: "Подписи",
                      }) as [keyof Layers, string][]
                    ).map(([key, title]) => (
                      <label key={key}>
                        <input
                          type="checkbox"
                          checked={layers[key]}
                          onChange={(e) =>
                            setLayers({ ...layers, [key]: e.target.checked })
                          }
                        />
                        {title}
                      </label>
                    ))}
                  </div>
                )}
              </div>
              <button
                className="icon-button"
                aria-label={
                  inspector ? "Скрыть инспектор" : "Показать инспектор"
                }
                onClick={() => setInspector(!inspector)}
              >
                {inspector ? (
                  <PanelRightClose size={18} />
                ) : (
                  <PanelRightOpen size={18} />
                )}
              </button>
            </div>
          </div>
          <div className="map-viewport">
            <Suspense
              fallback={<div className="map-loading">Загрузка карты…</div>}
            >
              {mode === "3d" ? (
                <Globe
                  scenario={s}
                  snapshot={snapshot}
                  layers={layers}
                  selected={selected || client}
                  onSelect={select}
                  onFallback={() => setMode("2d")}
                />
              ) : (
                <Map2D
                  scenario={s}
                  snapshot={snapshot}
                  layers={layers}
                  selected={selected || client}
                  onSelect={select}
                />
              )}
            </Suspense>
            {loading && (
              <span className="map-busy">
                <span className="spinner" />
                Обновление снимка
              </span>
            )}
            <div className="map-legend">
              <span className="sat-color">
                ● <em>Спутник</em>
              </span>
              <span className="success-text">
                ▲ <em>Пункт</em>
              </span>
              <span className="warning-text">
                ■ <em>Шлюз</em>
              </span>
              <span className="error-text">
                × <em>Отказ</em>
              </span>
              <span className="muted">
                ○ <em>Не запущен</em>
              </span>
            </div>
            <span className="map-hint">
              {mode === "3d"
                ? "Вращение — мышью · Масштаб — колесом"
                : "Перемещение — мышью · Масштаб — колесом"}
            </span>
          </div>
        </section>
        {inspector && (
          <aside className="panel inspector">
            <div className="panel-heading">
              <h2>Маршрут до шлюза</h2>
              <RadioTower size={18} />
            </div>
            <label className="field">
              <span>Пункт</span>
              <select
                aria-label="Выбранный наземный пункт"
                value={client}
                onChange={(e) => {
                  setClient(e.target.value);
                  setSelected(e.target.value);
                }}
              >
                {clients.map((g) => (
                  <option key={g.id} value={g.id}>
                    {g.id}
                  </option>
                ))}
              </select>
            </label>
            {!p.run ? (
              <Empty
                title="Сеть ещё не рассчитана"
                action={
                  <button
                    className="primary"
                    disabled={p.busy}
                    onClick={p.onRun}
                  >
                    Запустить расчёт
                  </button>
                }
              >
                Рассчитайте группировку, чтобы увидеть спутники, связи и
                маршруты.
              </Empty>
            ) : loading ? (
              <div className="inspector-loading">
                <span className="spinner" /> Загрузка состояния…
              </div>
            ) : route ? (
              <>
                <div
                  className={
                    "route-status " + (route.path.length ? "success" : "error")
                  }
                >
                  {route.path.length ? (
                    <CheckCircle2 size={19} />
                  ) : (
                    <XCircle size={19} />
                  )}
                  <strong>
                    {route.path.length ? "Путь доступен" : "Маршрут недоступен"}
                  </strong>
                </div>
                {route.path.length ? (
                  <ol className="route-list">
                    {route.path.map((id, i) => (
                      <li key={id}>
                        <button onClick={() => select(id)}>
                          {i === 0 ? (
                            <MapPin size={18} />
                          ) : i === route.path.length - 1 ? (
                            <RadioTower size={18} />
                          ) : (
                            <Satellite size={18} />
                          )}
                          <strong>{id}</strong>
                          <small>
                            {i === 0
                              ? "Пункт"
                              : i === route.path.length - 1
                                ? "Шлюз"
                                : "Спутник"}
                          </small>
                        </button>
                      </li>
                    ))}
                  </ol>
                ) : (
                  <p className="route-reason">
                    {route.reason ? reasons[route.reason] : "Путь отсутствует"}
                  </p>
                )}
                <p className="muted">
                  {route.path.length
                    ? route.path.length - 1 + " переходов"
                    : "Переходы: —"}
                </p>
                <details className="route-details">
                  <summary>
                    Видимые спутники ·{" "}
                    {snapshot?.visible_satellites.length ?? 0}
                  </summary>
                  <div className="chip-list">
                    {snapshot?.visible_satellites.map((id) => (
                      <button
                        key={id}
                        className="small"
                        onClick={() => select(id)}
                      >
                        {id}
                      </button>
                    ))}
                  </div>
                  {snapshot?.visible_satellites.length === 0 && (
                    <p className="muted">
                      Нет активных спутников над горизонтом.
                    </p>
                  )}
                </details>
              </>
            ) : null}
            {metric && (
              <div className="inspector-metrics">
                <h3>За весь период · {duration(s.environment.horizon_s)}</h3>
                <div>
                  <span>Видимость спутников</span>
                  <strong>{percent(metric.visibility_ratio)}</strong>
                </div>
                <div>
                  <span>Доступность до шлюза</span>
                  <strong
                    className={
                      metric.meets_target ? "success-text" : "warning-text"
                    }
                  >
                    {percent(metric.path_ratio)}
                  </strong>
                </div>
                <div>
                  <span>Макс. перерыв</span>
                  <strong>{duration(metric.max_gap_s)}</strong>
                </div>
                <div>
                  <span>
                    Цель ≥ {percent(s.environment.target_availability)}
                  </span>
                  <span
                    className={
                      metric.meets_target ? "success-text" : "warning-text"
                    }
                  >
                    {metric.meets_target ? "Достигнута" : "Не достигнута"}
                  </span>
                </div>
              </div>
            )}
            {(selectedSat || selectedGround) && (
              <div className="selected-object">
                <h3>
                  {selected}{" "}
                  <span className="badge info">
                    {selectedSat
                      ? "Спутник"
                      : selectedGround?.role === "gateway"
                        ? "Шлюз"
                        : "Пункт"}
                  </span>
                </h3>
                {selectedSat ? (
                  <>
                    <p className="muted">
                      Плоскость {selectedSat.plane_id} · Очередь{" "}
                      {selectedSat.launch_batch}
                    </p>
                    <p>
                      {satelliteStatus(s, selected, time) === "active"
                        ? "Активен"
                        : satelliteStatus(s, selected, time) === "failed"
                          ? "В отказе"
                          : "Ещё не запущен"}
                    </p>
                  </>
                ) : (
                  <div>
                    <p className="muted">
                      {selectedGround?.name}
                      <br />
                      {number(selectedGround!.lat_deg)}°,{" "}
                      {number(selectedGround!.lon_deg)}°
                    </p>
                    {selectedGround?.role === "gateway" && (
                      <p
                        className={
                          s.gateway_outages.some(
                            (x) =>
                              x.gateway_id === selected &&
                              x.start_s <= time &&
                              time < x.end_s,
                          )
                            ? "error-text"
                            : "success-text"
                        }
                      >
                        {s.gateway_outages.some(
                          (x) =>
                            x.gateway_id === selected &&
                            x.start_s <= time &&
                            time < x.end_s,
                        )
                          ? "Шлюз недоступен"
                          : "Шлюз работает"}
                      </p>
                    )}
                  </div>
                )}
                {(selectedSat || selectedGround?.role === "gateway") && (
                  <button
                    className="small"
                    disabled={p.busy}
                    onClick={() =>
                      p.onFailure({
                        kind: selectedSat ? "satellite" : "gateway",
                        id: selected,
                        start: time,
                      })
                    }
                  >
                    Задать отказ <ChevronRight size={14} />
                  </button>
                )}
              </div>
            )}
          </aside>
        )}
      </div>
      {p.run ? (
        <Timeline
          run={p.run}
          time={time}
          onTime={setTime}
          client={client}
          onClient={(id) => {
            setClient(id);
            setSelected(id);
          }}
          playing={playing}
          onPlaying={(value) => {
            if (value && time >= s.environment.horizon_s - s.environment.step_s)
              setTime(0);
            setPlaying(value);
          }}
          loading={loading}
        />
      ) : (
        <Notice tone="info">
          Показаны координаты пунктов из проекта. Спутники, связи и диаграмма
          доступности появятся после расчёта.
        </Notice>
      )}
    </div>
  );
}
