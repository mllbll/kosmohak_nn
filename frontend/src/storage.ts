import type { Run, Variant } from "./types";
import { validateScenario } from "./domain";
const key = "orbital-network-variants-v1";
export function loadVariants(): Variant[] {
  try {
    const data = JSON.parse(localStorage.getItem(key) || "[]");
    return Array.isArray(data)
      ? data.filter(
          (x) =>
            x &&
            typeof x.key === "string" &&
            typeof x.title === "string" &&
            validateScenario(x.scenario).length === 0,
        )
      : [];
  } catch {
    return [];
  }
}
export function saveVariants(variants: Variant[]) {
  localStorage.setItem(key, JSON.stringify(variants));
}
function db(): Promise<IDBDatabase> {
  return new Promise((resolve, reject) => {
    const r = indexedDB.open("orbital-network", 1);
    r.onupgradeneeded = () =>
      r.result.createObjectStore("runs", { keyPath: "id" });
    r.onsuccess = () => resolve(r.result);
    r.onerror = () => reject(r.error);
  });
}
export async function loadRuns(): Promise<Run[]> {
  const d = await db();
  return new Promise((resolve, reject) => {
    const tx = d.transaction("runs"),
      r = tx.objectStore("runs").getAll();
    r.onsuccess = () => resolve(r.result as Run[]);
    r.onerror = () => reject(r.error);
    tx.oncomplete = () => d.close();
  });
}
export async function saveRun(run: Run) {
  const d = await db();
  return new Promise<void>((resolve, reject) => {
    const tx = d.transaction("runs", "readwrite");
    tx.objectStore("runs").put(run);
    tx.oncomplete = () => {
      d.close();
      resolve();
    };
    tx.onerror = () => {
      d.close();
      reject(tx.error);
    };
  });
}
