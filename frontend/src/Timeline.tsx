import { useMemo, useState, useEffect } from "react";
import { Play, Pause, SkipBack, SkipForward } from "lucide-react";
import type { Run } from "./types";
import {
  clock,
  parseClock,
  intervals,
  reasons,
  percent,
  duration,
} from "./domain";
interface Props {
  run: Run;
  time: number;
  displayedTime: number;
  onTime: (t: number) => void;
  client: string;
  onClient: (id: string) => void;
  playing: boolean;
  onPlaying: (v: boolean) => void;
  loading: boolean;
  playbackStep: number;
  onPlaybackStep: (step: number) => void;
}
export default function Timeline({
  run,
  time,
  displayedTime,
  onTime,
  client,
  onClient,
  playing,
  onPlaying,
  loading,
  playbackStep,
  onPlaybackStep,
}: Props) {
  const e = run.effective_scenario.environment,
    [timeText, setTimeText] = useState(clock(time)),
    [invalid, setInvalid] = useState(false),
    [stepText, setStepText] = useState(String(playbackStep));
  const stepInvalid =
    stepText.trim() === "" ||
    !Number.isFinite(Number(stepText)) ||
    Number(stepText) < 0 ||
    Number(stepText) > e.horizon_s;
  const lastTime = e.horizon_s - 0.001;
  const progress = Math.max(
    0,
    Math.min(100, (displayedTime / e.horizon_s) * 100),
  );
  useEffect(() => setStepText(String(playbackStep)), [playbackStep]);
  const rows = useMemo(
    () =>
      run.metrics.map((m) => ({
        metric: m,
        intervals: intervals(run.routes, m.client_id, e.step_s),
      })),
    [run, e.step_s],
  );
  useEffect(() => {
    setTimeText(clock(time));
    setInvalid(false);
  }, [time]);
  function commit() {
    const parsed = parseClock(timeText);
    if (!Number.isFinite(parsed) || parsed < 0 || parsed >= e.horizon_s) {
      setInvalid(true);
      return;
    }
    onPlaying(false);
    onTime(parsed);
    setInvalid(false);
  }
  return (
    <section className="panel timeline">
      <div className="panel-heading">
        <h2>Доступность по пунктам</h2>
        <div className="legend">
          <span>
            <i className="available" />
            Есть маршрут
          </span>
          <span>
            <i className="partition" />
            Спутник виден, пути нет
          </span>
          <span>
            <i className="invisible" />
            Нет видимого спутника
          </span>
          <span>
            <i className="gateway" />
            Шлюзы недоступны
          </span>
        </div>
      </div>
      <div className="playback">
        <button
          className="primary icon-button"
          aria-label={playing ? "Пауза" : "Воспроизвести"}
          disabled={stepInvalid}
          onClick={() => onPlaying(!playing)}
        >
          {playing ? <Pause size={18} /> : <Play size={18} />}
        </button>
        <button
          className="icon-button"
          aria-label="Предыдущий шаг"
          disabled={time === 0}
          onClick={() => {
            onPlaying(false);
            onTime(Math.max(0, time - (playbackStep || 1)));
          }}
        >
          <SkipBack size={17} />
        </button>
        <button
          className="icon-button"
          aria-label="Следующий шаг"
          disabled={time >= lastTime}
          onClick={() => {
            onPlaying(false);
            onTime(Math.min(lastTime, time + (playbackStep || 1)));
          }}
        >
          <SkipForward size={17} />
        </button>
        <input
          className="time-input mono"
          aria-label="Время расчёта"
          aria-invalid={invalid}
          value={timeText}
          onChange={(e) => setTimeText(e.target.value)}
          onBlur={commit}
          onKeyDown={(e) => {
            if (e.key === "Enter") commit();
          }}
        />
        <label className="simulation-step">
          Шаг симуляции, с
          <input
            aria-label="Шаг симуляции"
            type="number"
            min={0}
            max={e.horizon_s}
            step="any"
            aria-invalid={stepInvalid}
            aria-describedby="simulation-step-help"
            value={stepText}
            onChange={(event) => {
              const text = event.target.value,
                value = Number(text);
              setStepText(text);
              onPlaying(false);
              if (
                text.trim() !== "" &&
                Number.isFinite(value) &&
                value >= 0 &&
                value <= e.horizon_s
              )
                onPlaybackStep(value);
            }}
          />
        </label>
        <span
          id="simulation-step-help"
          className={stepInvalid ? "error-text" : "muted"}
        >
          {stepInvalid
            ? "Введите число от 0 до " + e.horizon_s
            : "0 — плавно (1×); 1 — шаг в секунду"}
        </span>
        <span className="muted">
          {invalid
            ? "Введите время от 00:00:00 до " + clock(lastTime)
            : loading
              ? "Обновление снимка…"
              : "Шаг метрик: " + duration(e.step_s)}
        </span>
        <span className="playback-note">Время от начала расчёта</span>
      </div>
      <div className="timeline-grid axis">
        <span />
        <div className="time-plot">
          <div className="tick-labels">
            {[0, 0.25, 0.5, 0.75, 1].map((f) => (
              <span key={f} style={{ left: f * 100 + "%" }}>
                {duration(f * e.horizon_s)}
              </span>
            ))}
          </div>
          <div className="time-progress-rail" aria-hidden="true">
            <span
              className="time-progress-fill"
              style={{ width: progress + "%" }}
            />
            <span
              className="time-progress-marker"
              style={{ left: progress + "%" }}
            />
            <span
              className="time-progress-label mono"
              style={{
                left: `clamp(0px, calc(${progress}% - 44px), calc(100% - 88px))`,
              }}
            >
              {clock(displayedTime)}
            </span>
          </div>
          <input
            className="time-slider"
            aria-label="Временная шкала"
            aria-valuetext={clock(displayedTime)}
            type="range"
            min={0}
            max={lastTime}
            step="any"
            value={displayedTime}
            onKeyDown={(event) => {
              const direction =
                event.key === "ArrowRight" || event.key === "ArrowUp"
                  ? 1
                  : event.key === "ArrowLeft" || event.key === "ArrowDown"
                    ? -1
                    : 0;
              if (!direction) return;
              event.preventDefault();
              onPlaying(false);
              onTime(
                Math.max(
                  0,
                  Math.min(
                    lastTime,
                    displayedTime + direction * (playbackStep || 1),
                  ),
                ),
              );
            }}
            onChange={(event) => {
              onPlaying(false);
              onTime(Math.min(lastTime, Number(event.target.value)));
            }}
          />
        </div>
        <span className="muted">Доступность</span>
      </div>
      {rows.map(({ metric: m, intervals: segments }) => (
        <div
          className={
            "timeline-grid timeline-row " +
            (client === m.client_id ? "selected" : "")
          }
          key={m.client_id}
        >
          <button className="text-button" onClick={() => onClient(m.client_id)}>
            {m.client_id}
          </button>
          <div className="availability-track">
            {segments.map((x) => (
              <button
                key={x.start}
                className={"availability-segment " + x.state}
                style={{
                  left: (x.start / e.horizon_s) * 100 + "%",
                  width: ((x.end - x.start) / e.horizon_s) * 100 + "%",
                }}
                title={
                  m.client_id +
                  " · " +
                  clock(x.start) +
                  "–" +
                  clock(x.end) +
                  " · " +
                  duration(x.end - x.start) +
                  "\n" +
                  (x.reason ? reasons[x.reason] : "Есть маршрут")
                }
                aria-label={
                  m.client_id +
                  " " +
                  clock(x.start) +
                  " " +
                  (x.reason ? reasons[x.reason] : "Есть маршрут")
                }
                onClick={() => {
                  onPlaying(false);
                  onClient(m.client_id);
                  onTime(x.start);
                }}
              />
            ))}
            <span className="time-cursor" style={{ left: progress + "%" }} />
          </div>
          <strong className={m.meets_target ? "success-text" : "warning-text"}>
            {percent(m.path_ratio)}
          </strong>
        </div>
      ))}
      <p className="footnote">
        Нажмите на интервал, чтобы перейти к нему. Последний отсчёт —{" "}
        {clock(e.horizon_s - e.step_s)}; правый конец периода не входит в сетку.
      </p>
    </section>
  );
}
