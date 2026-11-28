// GitHub Pages: there is no server, so the Go program runs in the page as
// WebAssembly (cmd/typetrainer/main_wasm.go) and answers the API through
// goApi. Its data lives in the page's memory: after every change it is
// exported to the browser's IndexedDB and imported again on the next visit.

type Answer = { status: number; body: string };

declare global {
  interface Window {
    Go: new () => { importObject: WebAssembly.Imports; run(instance: WebAssembly.Instance): Promise<void> };
    goApi?: (method: string, path: string, body: string) => Answer;
  }
}

const DB = "typetrainer";
const STORE = "progress";
const KEY = "file";

function idb(): Promise<IDBDatabase | null> {
  return new Promise((resolve) => {
    try {
      const open = indexedDB.open(DB, 1);
      open.onupgradeneeded = () => open.result.createObjectStore(STORE);
      open.onsuccess = () => resolve(open.result);
      open.onerror = () => resolve(null);
    } catch {
      resolve(null);
    }
  });
}

async function kept(): Promise<string | null> {
  const db = await idb();
  if (!db) return null;
  return new Promise((resolve) => {
    const req = db.transaction(STORE).objectStore(STORE).get(KEY);
    req.onsuccess = () => resolve(typeof req.result === "string" ? req.result : null);
    req.onerror = () => resolve(null);
  });
}

async function keep(file: string) {
  const db = await idb();
  db?.transaction(STORE, "readwrite").objectStore(STORE).put(file, KEY);
}

let ready: Promise<void> | null = null;

function start(): Promise<void> {
  ready ??= (async () => {
    await new Promise<void>((resolve, reject) => {
      const s = document.createElement("script");
      s.src = `${import.meta.env.BASE_URL}wasm_exec.js`;
      s.onload = () => resolve();
      s.onerror = () => reject(new Error("wasm_exec.js did not load"));
      document.head.append(s);
    });
    const go = new window.Go();
    const loaded = new Promise((r) => addEventListener("go-ready", r, { once: true }));
    const wasm = await WebAssembly.instantiateStreaming(fetch(`${import.meta.env.BASE_URL}app.wasm`), go.importObject);
    go.run(wasm.instance);
    await loaded;
    const file = await kept();
    if (file) window.goApi?.("POST", "/api/import", file);
  })();
  return ready;
}

export async function pagesCall(method: string, path: string, body: string): Promise<Answer> {
  await start();
  const answer = window.goApi?.(method, path, body) ?? { status: 0, body: "" };
  // A portion changes nothing kept: only a round's end, a step, a reset or an import do.
  if (method === "POST" && answer.status === 200 && !path.endsWith("/portion")) {
    const file = window.goApi?.("GET", "/api/export", "");
    if (file?.status === 200) keep(file.body);
  }
  return answer;
}
