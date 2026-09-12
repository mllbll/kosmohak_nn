import type { Comparison, Project, Run, Scenario, Snapshot } from "./types";
const base = (import.meta.env.VITE_API_URL || "").replace(/\/$/, "");
export class ApiError extends Error {
  constructor(
    message: string,
    public status: number,
  ) {
    super(message);
  }
}
export async function request<T>(
  path: string,
  options: RequestInit = {},
): Promise<T> {
  const controller = new AbortController();
  const abort = () => controller.abort();
  if (options.signal?.aborted) controller.abort();
  else options.signal?.addEventListener("abort", abort, { once: true });
  const timer = setTimeout(abort, 310000);
  try {
    const response = await fetch(base + path, {
      ...options,
      signal: controller.signal,
      headers: { "Content-Type": "application/json", ...options.headers },
    });
    const text = await response.text();
    let body;
    try {
      body = JSON.parse(text);
    } catch {
      body = null;
    }
    if (!response.ok)
      throw new ApiError(
        body?.error || "Ошибка сервера HTTP " + response.status,
        response.status,
      );
    if (!body)
      throw new ApiError(
        "Сервер вернул ответ в неизвестном формате",
        response.status,
      );
    return body as T;
  } catch (error) {
    if (
      error instanceof ApiError ||
      (error instanceof DOMException && error.name === "AbortError")
    )
      throw error;
    throw new Error(
      "Нет соединения с сервером расчётов. Проверьте, что бэкенд запущен.",
    );
  } finally {
    clearTimeout(timer);
    options.signal?.removeEventListener("abort", abort);
  }
}
export const api = {
  create: (scenario: Scenario) =>
    request<Project>("/api/projects", {
      method: "POST",
      body: JSON.stringify(scenario),
    }),
  run: (id: string) =>
    request<{ run_id: string }>(
      "/api/projects/" + encodeURIComponent(id) + "/runs",
      { method: "POST" },
    ),
  getRun: (id: string) => request<Run>("/api/runs/" + encodeURIComponent(id)),
  snapshot: (id: string, t: number, client: string, signal?: AbortSignal) =>
    request<Snapshot>(
      "/api/runs/" +
        encodeURIComponent(id) +
        "/snapshot?t_s=" +
        t +
        "&client_id=" +
        encodeURIComponent(client),
      { signal },
    ),
  compare: (a: string, b: string) =>
    request<Comparison>("/api/compare", {
      method: "POST",
      body: JSON.stringify({ run_a: a, run_b: b }),
    }),
  export: (id: string) =>
    request<unknown>("/api/runs/" + encodeURIComponent(id) + "/export"),
};
