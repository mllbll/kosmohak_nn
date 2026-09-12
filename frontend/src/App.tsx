import { useEffect, useState, useRef, lazy, Suspense } from "react";
import {
  Orbit,
  SlidersHorizontal,
  ShieldAlert,
  Globe2,
  GitCompareArrows,
  Play,
  X,
  CheckCircle2,
} from "lucide-react";
import type { Scenario, Variant, Run, Tab } from "./types";
import { api, ApiError } from "./api";
import {
  clone,
  same,
  parseScenario,
  validateScenario,
  download,
} from "./domain";
import { loadVariants, saveVariants, loadRuns, saveRun } from "./storage";
import { Notice } from "./ui";
import ProjectView from "./ProjectView";
import FailuresView, { type FailureSelection } from "./FailuresView";
const NetworkView = lazy(() => import("./NetworkView"));
const CompareView = lazy(() => import("./CompareView"));
const tabs: { id: Tab; title: string; icon: typeof Orbit }[] = [
  { id: "project", title: "Проект", icon: SlidersHorizontal },
  { id: "failures", title: "Отказы", icon: ShieldAlert },
  { id: "network", title: "Состояние сети", icon: Globe2 },
  { id: "compare", title: "Сравнение", icon: GitCompareArrows },
];
export default function App() {
  const [draft, setDraft] = useState<Scenario>(),
    [variants, setVariants] = useState<Variant[]>([]),
    [activeKey, setActiveKey] = useState(""),
    [runs, setRuns] = useState<Run[]>([]),
    [runId, setRunId] = useState(""),
    [tab, setTab] = useState<Tab>("project");
  const [busy, setBusy] = useState(""),
    [error, setError] = useState(""),
    [toast, setToast] = useState(""),
    [connection, setConnection] = useState<"checking" | "online" | "offline">(
      "checking",
    ),
    [selection, setSelection] = useState<FailureSelection>(),
    [confirm, setConfirm] = useState<{
      message: string;
      action: () => void;
    } | null>(null),
    [initialized, setInitialized] = useState(false);
  const taskLock = useRef(false);
  const active = variants.find((x) => x.key === activeKey),
    dirty = !!draft && (!active || !same(draft, active.scenario)),
    run = runs.find((x) => x.id === runId),
    stale = !!run && !!draft && !same(run.effective_scenario, draft);
  useEffect(() => {
    let live = true;
    async function start() {
      try {
        const saved = loadVariants();
        const initial = saved[0] ?? {
          key: Array.from(crypto.getRandomValues(new Uint8Array(16)), (n) =>
            n.toString(16).padStart(2, "0"),
          ).join(""),
          title: "Полная группировка",
          scenario: parseScenario(
            await (await fetch("/scenarios/01_full_constellation.json")).text(),
          ),
        };
        if (live) {
          setVariants(saved.length ? saved : [initial]);
          setActiveKey(initial.key);
          setDraft(clone(initial.scenario));
        }
        try {
          const restored = await loadRuns();
          if (live) {
            setRuns(restored);
            if (initial.runId && restored.some((r) => r.id === initial.runId))
              setRunId(initial.runId);
          }
        } catch {
          if (live)
            setToast(
              "Хранилище расчётов недоступно. Скачивайте результаты через экспорт.",
            );
        }
      } catch (e) {
        if (live) setError((e as Error).message);
      } finally {
        if (live) setInitialized(true);
      }
    }
    start();
    return () => {
      live = false;
    };
  }, []);
  useEffect(() => {
    if (!initialized) return;
    try {
      saveVariants(variants);
    } catch {
      setToast(
        "Не удалось сохранить варианты в браузере. Скачайте проект JSON.",
      );
    }
  }, [variants, initialized]);
  useEffect(() => {
    if (!toast) return;
    const t = setTimeout(() => setToast(""), 6000);
    return () => clearTimeout(t);
  }, [toast]);
  useEffect(() => {
    let live = true;
    const check = () =>
      fetch((import.meta.env.VITE_API_URL || "") + "/health", {
        signal: AbortSignal.timeout(5000),
      })
        .then((r) => {
          if (live) setConnection(r.ok ? "online" : "offline");
        })
        .catch(() => {
          if (live) setConnection("offline");
        });
    check();
    const t = setInterval(check, 20000);
    return () => {
      live = false;
      clearInterval(t);
    };
  }, []);
  useEffect(() => {
    const before = (e: BeforeUnloadEvent) => {
      if (dirty) {
        e.preventDefault();
        e.returnValue = "";
      }
    };
    window.addEventListener("beforeunload", before);
    return () => window.removeEventListener("beforeunload", before);
  }, [dirty]);
  async function action(label: string, fn: () => Promise<void>) {
    if (taskLock.current) return;
    taskLock.current = true;
    setBusy(label);
    setError("");
    try {
      await fn();
    } catch (e) {
      setError(
        e instanceof DOMException && e.name === "AbortError"
          ? "Расчёт прерван или превышено время ожидания. Повторите запуск."
          : (e as Error).message,
      );
    } finally {
      taskLock.current = false;
      setBusy("");
    }
  }
  function choose(v: Variant) {
    setDraft(clone(v.scenario));
    setActiveKey(v.key);
    setRunId(v.runId ?? "");
    setSelection(undefined);
    setError("");
  }
  function guarded(fn: () => void) {
    if (taskLock.current) return;
    if (dirty)
      setConfirm({
        message:
          "Несохранённые изменения текущего варианта будут сброшены. Продолжить?",
        action: fn,
      });
    else fn();
  }
  function remember(
    s: Scenario,
    projectId?: string,
    newRunId?: string,
  ): Variant {
    const v = {
      key: Array.from(crypto.getRandomValues(new Uint8Array(16)), (n) =>
        n.toString(16).padStart(2, "0"),
      ).join(""),
      title: s.meta.title,
      scenario: clone(s),
      projectId,
      runId: newRunId,
      savedAt: new Date().toISOString(),
    };
    setVariants((old) => [...old, v]);
    setActiveKey(v.key);
    setDraft(clone(s));
    return v;
  }
  function save() {
    if (!draft) return;
    action("Сохранение варианта", async () => {
      const errors = validateScenario(draft);
      if (errors.length) throw new Error(errors.join("\n"));
      const s = clone(draft);
      const project = await api.create(s);
      remember(project.effective, project.id);
      setRunId("");
      setToast("Вариант сохранён");
    });
  }
  function calculate() {
    if (!draft) return;
    action("Расчёт группировки", async () => {
      const issues = validateScenario(draft);
      if (issues.length) throw new Error(issues.join("\n"));
      const s = clone(draft);
      let projectId = !dirty ? active?.projectId : undefined;
      if (!projectId) projectId = (await api.create(s)).id;
      let created;
      try {
        created = await api.run(projectId);
      } catch (e) {
        if (e instanceof ApiError && e.status === 404) {
          projectId = (await api.create(s)).id;
          created = await api.run(projectId);
        } else throw e;
      }
      const result = await api.getRun(created.run_id);
      setRuns((old) => [...old.filter((r) => r.id !== result.id), result]);
      setRunId(result.id);
      if (!dirty && active) {
        setVariants((old) =>
          old.map((v) =>
            v.key === active.key ? { ...v, projectId, runId: result.id } : v,
          ),
        );
      } else remember(result.effective_scenario, projectId, result.id);
      try {
        await saveRun(result);
      } catch {
        setToast(
          "Расчёт готов. Для сохранения скачайте результат: хранилище браузера недоступно.",
        );
      }
      setConnection("online");
      setTab("network");
    });
  }
  function preset(name: string) {
    guarded(() => {
      action("Загрузка сценария", async () => {
        const response = await fetch("/scenarios/" + name + ".json");
        if (!response.ok) throw new Error("Сценарий не найден");
        const s = parseScenario(await response.text());
        const v = remember(s);
        choose(v);
        setTab("project");
      });
    });
  }
  function importFile(file: File) {
    guarded(() => {
      action("Проверка файла", async () => {
        if (file.size > 10 * 1024 * 1024)
          throw new Error(
            "Файл больше 10 МБ. Загрузите исходный проект без большой истории маршрутов.",
          );
        const s = parseScenario(await file.text());
        const project = await api.create(s);
        const v = remember(project.effective, project.id);
        choose(v);
        setToast("Проект загружен и проверен сервером");
      });
    });
  }
  function openRun(r: Run) {
    const v =
      variants.find((v) => v.runId === r.id) ||
      remember(r.effective_scenario, r.project_id, r.id);
    choose(v);
    setRunId(r.id);
    setTab("network");
  }
  function addFailure(v: FailureSelection) {
    if (run && stale) {
      setConfirm({
        message:
          "Отказ будет добавлен к конфигурации отображаемого расчёта. Несохранённые настройки текущего проекта будут заменены.",
        action: () => {
          setDraft(clone(run.effective_scenario));
          setSelection(v);
          setTab("failures");
        },
      });
    } else {
      setSelection(v);
      setTab("failures");
    }
  }
  if (!draft)
    return (
      <div className="startup">
        <Orbit size={48} />
        <h1>Орбитальная сеть</h1>
        {error ? (
          <Notice tone="error">
            {error}
            <button onClick={() => location.reload()}>Повторить</button>
          </Notice>
        ) : (
          <p>Загрузка проекта…</p>
        )}
      </div>
    );
  return (
    <div className="app-shell">
      <header className="app-header">
        <div className="header-top">
          <a
            href="#project"
            className="brand"
            onClick={(e) => {
              e.preventDefault();
              setTab("project");
            }}
          >
            <span className="brand-mark">
              <Orbit size={28} />
            </span>
            <span>
              Орбитальная сеть<small>ПРОЕКТИРОВАНИЕ ГРУППИРОВКИ</small>
            </span>
          </a>
          <div className="header-divider" />
          <select
            className="project-picker"
            aria-label="Текущий проект"
            value={activeKey}
            disabled={!!busy}
            onChange={(e) => {
              const v = variants.find((v) => v.key === e.target.value);
              if (v) guarded(() => choose(v));
            }}
          >
            {variants.map((v) => (
              <option key={v.key} value={v.key}>
                {v.title}
              </option>
            ))}
          </select>
          <div className="header-status">
            <span className={"connection " + connection}>
              <i />
              {connection === "online"
                ? "Сервер подключён"
                : connection === "checking"
                  ? "Проверка сервера"
                  : "Сервер недоступен"}
            </span>
            <span
              className={
                dirty ? "warning-text" : run ? "success-text" : "muted"
              }
            >
              {dirty
                ? "Есть изменения"
                : run
                  ? "Расчёт завершён"
                  : "Готов к расчёту"}
            </span>
          </div>
          <button
            className="primary run-button"
            disabled={!!busy || validateScenario(draft).length > 0}
            onClick={calculate}
          >
            {busy ? (
              <span className="spinner" />
            ) : (
              <Play size={17} fill="currentColor" />
            )}
            {busy ? "Выполняется…" : "Запустить расчёт"}
          </button>
        </div>
        <nav className="main-tabs" aria-label="Основные разделы">
          {tabs.map((t) => (
            <button
              key={t.id}
              className={tab === t.id ? "active" : ""}
              aria-current={tab === t.id ? "page" : undefined}
              onClick={() => setTab(t.id)}
            >
              <t.icon size={17} />
              {t.title}
              {t.id === "failures" &&
                draft.failures.length + draft.gateway_outages.length > 0 && (
                  <span className="tab-count">
                    {draft.failures.length + draft.gateway_outages.length}
                  </span>
                )}
            </button>
          ))}
          <span className="nav-right">
            КОСМОХАКАТОН <strong>2026</strong>
          </span>
        </nav>
      </header>
      <main>
        {error && (
          <div className="global-notice">
            <Notice tone="error">
              <strong>Не удалось выполнить действие</strong>
              <p className="preserve-lines">{error}</p>
              <button className="text-button" onClick={() => setError("")}>
                Закрыть
              </button>
            </Notice>
          </div>
        )}
        {busy && (
          <div className="busy-strip" role="status">
            <span className="spinner" />
            <strong>{busy}</strong>
            <span>
              Расчёт может занять несколько минут. Дождитесь ответа сервера.
            </span>
          </div>
        )}
        <Suspense
          fallback={
            <div className="startup">
              <span className="spinner" />
              Загрузка раздела…
            </div>
          }
        >
          {tab === "project" && (
            <ProjectView
              scenario={draft}
              setScenario={setDraft}
              variants={variants}
              activeKey={activeKey}
              dirty={dirty}
              busy={!!busy}
              onImport={importFile}
              onSave={save}
              onReset={() => {
                if (active) setDraft(clone(active.scenario));
              }}
              onExport={() => download(draft.meta.id + ".json", draft)}
              onSelect={(v) => guarded(() => choose(v))}
              onPreset={preset}
            />
          )}
          {tab === "failures" && (
            <FailuresView
              key={activeKey}
              scenario={draft}
              setScenario={setDraft}
              busy={!!busy}
              selection={selection}
              onNetwork={() => setTab("network")}
            />
          )}
          {tab === "network" && (
            <NetworkView
              scenario={draft}
              run={run}
              stale={stale}
              busy={!!busy}
              onConfigure={() => setTab("project")}
              onRun={calculate}
              onFailure={addFailure}
              onExport={() => {
                if (run)
                  action("Экспорт результата", async () => {
                    try {
                      download(
                        "result-" + run.id.slice(0, 8) + ".json",
                        await api.export(run.id),
                      );
                    } catch (e) {
                      if (e instanceof ApiError && e.status === 404) {
                        download("result-" + run.id.slice(0, 8) + ".json", {
                          schema_version: "cosmo-A-result-1.0",
                          effective_scenario: run.effective_scenario,
                          routes: run.routes,
                          metrics: run.metrics,
                          summary: run.summary,
                        });
                        setToast("Выгружена сохранённая копия результата");
                      } else throw e;
                    }
                  });
              }}
            />
          )}
          {tab === "compare" && (
            <CompareView
              runs={runs}
              onOpen={(r) => guarded(() => openRun(r))}
              onProject={() => setTab("project")}
            />
          )}
        </Suspense>
      </main>
      <footer className="app-footer">
        <span>
          <i className="status-dot" />
          Геометрическая доступность связи
        </span>
        <span>Круговые орбиты · Сферическая Земля</span>
      </footer>
      {toast && (
        <div className="toast" role="status">
          <CheckCircle2 size={18} />
          {toast}
          <button
            className="icon-button"
            aria-label="Закрыть уведомление"
            onClick={() => setToast("")}
          >
            <X size={16} />
          </button>
        </div>
      )}
      {confirm && (
        <div className="modal-backdrop">
          <div
            className="modal panel"
            role="dialog"
            aria-modal="true"
            aria-labelledby="confirm-title"
          >
            <h2 id="confirm-title">Переключение варианта</h2>
            <p>{confirm.message}</p>
            <div className="actions end">
              <button autoFocus onClick={() => setConfirm(null)}>
                Отмена
              </button>
              <button
                className="primary"
                onClick={() => {
                  const fn = confirm.action;
                  setConfirm(null);
                  fn();
                }}
              >
                Продолжить
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
