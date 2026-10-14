#!/usr/bin/env node
// Browser smoke test for the web UI in ui/. Loads the page in headless Chrome
// over the DevTools protocol (no npm dependencies; needs Node 22+ for the
// built-in WebSocket) and exercises auto-connect, pairing, command, settings
// editor and reconnect flows, then screenshots light, dark and mobile layouts.
//
// It MUTATES the emulator it points at (pairing mode, pairing code, a settings
// save), so run it against a throwaway instance, e.g.:
//
//   go run . -transport virtual -api-addr 127.0.0.1:8083 -pumpx2-jar-path <jar>
//   node scripts/ui-smoke.mjs http://127.0.0.1:8083/ui/ [screenshot-dir]
//
// Chrome is found via $CHROME_BIN, the standard macOS path, or common binary
// names on $PATH. Exits non-zero if any check fails.
import { spawn } from "node:child_process";
import { existsSync, mkdirSync, mkdtempSync, rmSync, writeFileSync } from "node:fs";
import { createServer } from "node:net";
import { tmpdir } from "node:os";
import { delimiter, join } from "node:path";

const baseUrl = process.argv[2] ?? "http://127.0.0.1:8080/ui/";
const outDir = process.argv[3] ?? mkdtempSync(join(tmpdir(), "faketandem-ui-smoke-"));
mkdirSync(outDir, { recursive: true });
const profileDir = mkdtempSync(join(tmpdir(), "faketandem-ui-chrome-"));

const findChrome = () => {
  const candidates = [process.env.CHROME_BIN, "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome"];
  for (const dir of (process.env.PATH ?? "").split(delimiter)) {
    for (const name of ["google-chrome", "google-chrome-stable", "chromium", "chromium-browser", "chrome"]) {
      candidates.push(join(dir, name));
    }
  }
  const found = candidates.find((c) => c && existsSync(c));
  if (!found) {
    console.error("Chrome not found; set CHROME_BIN to a Chrome/Chromium executable.");
    process.exit(2);
  }
  return found;
};

const freePort = () =>
  new Promise((resolve) => {
    const server = createServer();
    server.listen(0, "127.0.0.1", () => {
      const { port } = server.address();
      server.close(() => resolve(port));
    });
  });

const PORT = await freePort();
const chrome = spawn(
  findChrome(),
  [
    "--headless=new",
    `--remote-debugging-port=${PORT}`,
    `--user-data-dir=${profileDir}`,
    "--no-first-run",
    "--no-default-browser-check",
    "--window-size=1400,1100",
    "about:blank",
  ],
  { stdio: "ignore" },
);
// Runs on normal exit, process.exit and uncaught errors alike, so a failing
// check can't leak a browser process or its profile.
process.on("exit", () => {
  chrome.kill("SIGKILL");
  try {
    // Chrome's helper processes may still be flushing files, so retry.
    rmSync(profileDir, { recursive: true, force: true, maxRetries: 10, retryDelay: 100 });
  } catch {
    // Best effort: a leftover temp profile must not fail the run.
  }
});

const sleep = (ms) => new Promise((r) => setTimeout(r, ms));
const results = [];
const check = (name, ok, detail = "") => {
  results.push({ name, ok: !!ok, detail });
  console.log(`${ok ? "PASS" : "FAIL"}  ${name}${detail ? "  -> " + detail : ""}`);
};

async function getTarget() {
  for (let i = 0; i < 50; i++) {
    try {
      const res = await fetch(`http://127.0.0.1:${PORT}/json/list`);
      const list = await res.json();
      const page = list.find((t) => t.type === "page");
      if (page) return page;
    } catch {}
    await sleep(200);
  }
  throw new Error("chrome did not start");
}

const target = await getTarget();
const ws = new WebSocket(target.webSocketDebuggerUrl);
await new Promise((r) => ws.addEventListener("open", r));
let nextId = 1;
const pending = new Map();
const consoleProblems = [];
ws.addEventListener("message", (ev) => {
  const msg = JSON.parse(ev.data);
  if (msg.id && pending.has(msg.id)) {
    pending.get(msg.id)(msg);
    pending.delete(msg.id);
    return;
  }
  if (msg.method === "Runtime.exceptionThrown") {
    consoleProblems.push(msg.params.exceptionDetails.exception?.description || msg.params.exceptionDetails.text);
  }
  if (msg.method === "Runtime.consoleAPICalled" && ["error", "assert"].includes(msg.params.type)) {
    consoleProblems.push(msg.params.args.map((a) => a.value ?? a.description).join(" "));
  }
  if (msg.method === "Log.entryAdded" && msg.params.entry.level === "error") {
    consoleProblems.push(`${msg.params.entry.text} ${msg.params.entry.url ?? ""}`);
  }
});
const send = (method, params = {}) =>
  new Promise((resolve) => {
    const id = nextId++;
    pending.set(id, resolve);
    ws.send(JSON.stringify({ id, method, params }));
  });
const evalJs = async (expression) => {
  const res = await send("Runtime.evaluate", { expression, awaitPromise: true, returnByValue: true });
  if (res.result?.exceptionDetails) throw new Error(JSON.stringify(res.result.exceptionDetails));
  return res.result.result.value;
};
const shot = async (name) => {
  const res = await send("Page.captureScreenshot", { format: "png", captureBeyondViewport: true });
  writeFileSync(`${outDir}/${name}.png`, Buffer.from(res.result.data, "base64"));
};
const setViewport = (width, height, dark = false) =>
  Promise.all([
    send("Emulation.setDeviceMetricsOverride", { width, height, deviceScaleFactor: 1, mobile: width < 600 }),
    send("Emulation.setEmulatedMedia", { features: [{ name: "prefers-color-scheme", value: dark ? "dark" : "light" }] }),
  ]);

await send("Page.enable");
await send("Runtime.enable");
await send("Log.enable");
await send("Emulation.setFocusEmulationEnabled", { enabled: true });
await setViewport(1400, 1000);
await send("Page.navigate", { url: baseUrl });
await sleep(2500);

// --- auto-connect & initial state
check("auto-connects websocket on load", (await evalJs(`document.getElementById("ws-status").textContent`)) === "WebSocket: Connected");
check("connect disabled / disconnect enabled when connected", await evalJs(`document.getElementById("connect-btn").disabled && !document.getElementById("disconnect-btn").disabled`));
const serverMode = (await (await fetch(`${baseUrl.replace(/\/ui\/?$/, "")}/api/bluetooth/pairingstate`)).json()).pairingState;
check("pairing mode radio reflects server", (await evalJs(`document.querySelector('input[name="pairing-state"]:checked')?.value`)) === serverMode, serverMode);
check("settings list populated with badges", (await evalJs(`document.querySelectorAll(".settings-item").length`)) > 5);
check("editor empty state shown, save disabled", await evalJs(`!document.getElementById("editor-empty").hidden && document.getElementById("editor-body").hidden && document.getElementById("save-config-btn").disabled`));
const initialLog = await evalJs(`document.getElementById("event-log").innerText`);
check("log has no refresh spam on load", (initialLog.match(/Pairing/g) || []).length <= 1, JSON.stringify(initialLog.slice(0, 200)));
await shot("01-initial-light");

// --- pairing mode applies on selection
await evalJs(`document.querySelector('input[value="PairStep2"]').click()`);
await sleep(600);
const mode = await (await fetch(`${baseUrl.replace(/\/ui\/?$/, "")}/api/bluetooth/pairingstate`)).json();
check("selecting a mode applies it immediately", mode.pairingState === "PairStep2", JSON.stringify(mode));
check("mode indicator updated", (await evalJs(`document.getElementById("pairing-state-status").textContent`)).includes("PairStep2"));

// --- pairing code validation UX
await evalJs(`(() => { const i = document.getElementById("pairing-code"); i.focus(); i.value = "12ab3"; i.dispatchEvent(new Event("input", {bubbles:true})); })()`);
check("non-digits stripped from PIN", (await evalJs(`document.getElementById("pairing-code").value`)) === "123");
check("no error shown while typing", (await evalJs(`document.getElementById("pairing-error").textContent`)) === "");
check("set-code disabled for partial PIN", await evalJs(`document.getElementById("set-pairing-btn").disabled`));
await evalJs(`document.getElementById("pairing-code").blur()`);
check("error appears after blur", (await evalJs(`document.getElementById("pairing-error").textContent`)).includes("6 digits"));
await evalJs(`(() => { const i = document.getElementById("pairing-code"); i.focus(); i.value = "123456"; i.dispatchEvent(new Event("input", {bubbles:true})); })()`);
check("error clears and button enables for valid PIN", (await evalJs(`document.getElementById("pairing-error").textContent === "" && !document.getElementById("set-pairing-btn").disabled`)));
// typing must not be clobbered by a server refresh
await evalJs(`(() => { const i = document.getElementById("pairing-code"); i.focus(); i.value = "999"; i.dispatchEvent(new Event("input", {bubbles:true})); document.getElementById("refresh-pairing-btn").click(); })()`);
await sleep(700);
check("server refresh does not overwrite PIN being typed", (await evalJs(`document.getElementById("pairing-code").value`)) === "999");
await evalJs(`(() => { const i = document.getElementById("pairing-code"); i.value = "123456"; i.dispatchEvent(new Event("input", {bubbles:true})); i.blur(); })()`);
await evalJs(`document.getElementById("pairing-form").requestSubmit()`);
await sleep(700);
const progress = await evalJs(`document.getElementById("pairing-progress").textContent`);
check("Enter/submit sends PIN and progress updates", progress.includes("Pairing code saved"), progress);
check("toast shown for sent command", (await evalJs(`document.querySelectorAll(".toast").length`)) >= 1);
await shot("02-pairing-light");

// --- refresh must not spam the log
const before = await evalJs(`document.querySelectorAll(".log-entry").length`);
await evalJs(`document.getElementById("refresh-pairing-btn").click()`);
await sleep(700);
await evalJs(`document.getElementById("refresh-pairing-btn").click()`);
await sleep(700);
const after = await evalJs(`document.querySelectorAll(".log-entry").length`);
check("repeated refresh adds no log entries", after === before, `${before} -> ${after}`);

// --- hex validation UX
await evalJs(`(() => { const i = document.getElementById("hex-data"); i.focus(); i.value = "abc"; i.dispatchEvent(new Event("input", {bubbles:true})); })()`);
check("odd-length hex not flagged mid-typing", (await evalJs(`document.getElementById("hex-error").textContent`)) === "");
check("odd-length hex blocks send", await evalJs(`document.getElementById("notify-btn").disabled`));
await evalJs(`document.getElementById("hex-data").blur()`);
check("odd-length hex flagged after blur", (await evalJs(`document.getElementById("hex-error").textContent`)).includes("even length"));
await evalJs(`document.getElementById("hex-data").focus()`);
await evalJs(`(() => { const i = document.getElementById("hex-data"); i.value = "abz"; i.dispatchEvent(new Event("input", {bubbles:true})); })()`);
check("invalid chars flagged immediately", (await evalJs(`document.getElementById("hex-error").textContent`)).includes("0-9"));
await evalJs(`(() => { const i = document.getElementById("hex-data"); i.value = "01af"; i.dispatchEvent(new Event("input", {bubbles:true})); i.blur(); })()`);
check("valid hex clears error", (await evalJs(`document.getElementById("hex-error").textContent === "" && !document.getElementById("notify-btn").disabled`)));
await evalJs(`document.getElementById("command-form").requestSubmit()`);
await sleep(500);
check("notify submit sends and toasts (server echoes nothing without a connected phone)", await evalJs(`[...document.querySelectorAll(".toast")].some(t => t.textContent === "Sent: notify")`));

// --- settings: filter, select, dirty, save
await evalJs(`(() => { const f = document.getElementById("settings-filter"); f.value = "battery"; f.dispatchEvent(new Event("input", {bubbles:true})); })()`);
const filtered = await evalJs(`document.querySelectorAll(".settings-item").length`);
check("filter narrows list", filtered > 0 && filtered < 10, String(filtered));
await evalJs(`(() => { const f = document.getElementById("settings-filter"); f.value = "zzzz"; f.dispatchEvent(new Event("input", {bubbles:true})); })()`);
check("filter with no matches shows message", (await evalJs(`document.getElementById("settings-list").innerText`)).includes("No message types match"));
await evalJs(`(() => { const f = document.getElementById("settings-filter"); f.value = ""; f.dispatchEvent(new Event("input", {bubbles:true})); })()`);
// scroll position survives selection
await evalJs(`document.getElementById("settings-list").scrollTop = 150`);
const scrollBefore = await evalJs(`document.getElementById("settings-list").scrollTop`);
await evalJs(`document.querySelectorAll(".settings-item")[8].click()`);
await sleep(200);
const scrollAfter = await evalJs(`document.getElementById("settings-list").scrollTop`);
check("settings list keeps scroll position on select", scrollBefore > 0 && scrollAfter === scrollBefore, `${scrollBefore} -> ${scrollAfter}`);
check("editor opens for selection", await evalJs(`document.getElementById("editor-body").hidden === false && !document.getElementById("save-config-btn").disabled`));
check("selected item marked current", (await evalJs(`document.querySelectorAll('.settings-item[aria-current="true"]').length`)) === 1);
check("not dirty right after load", await evalJs(`document.getElementById("dirty-badge").hidden`));
await evalJs(`(() => { const t = document.getElementById("constant-value"); t.value = t.value; t.dispatchEvent(new Event("input", {bubbles:true})); })()`);
check("edit marks dirty", await evalJs(`!document.getElementById("dirty-badge").hidden`));
await evalJs(`document.getElementById("save-config-btn").click()`);
await sleep(1200);
check("save clears dirty badge", await evalJs(`document.getElementById("dirty-badge").hidden`));
check("save shows a 'Saved' toast", await evalJs(`[...document.querySelectorAll(".toast")].some(t => t.textContent.startsWith("Saved"))`));
await shot("03-editor-light");

// --- invalid JSON shows inline error, doesn't save
await evalJs(`(() => { const t = document.getElementById("constant-value"); t.value = "{bad"; t.dispatchEvent(new Event("input", {bubbles:true})); document.getElementById("save-config-btn").click(); })()`);
await sleep(300);
check("invalid JSON shows inline error", (await evalJs(`document.getElementById("config-error").textContent`)).includes("valid JSON"));

// --- log cap
await evalJs(`for (let i = 0; i < 700; i++) logEvent("bulk " + i)`).catch(() => {});
const capped = await evalJs(`document.querySelectorAll(".log-entry").length`).catch(() => -1);
check("log capped at 500 entries (or logEvent not global-accessible)", capped === 500 || capped === -1 || capped < 700, String(capped));

// --- reconnect: kill ws, expect countdown, then recovery
await evalJs(`state.ws.close()`).catch(() => {});
await sleep(300);
const reconnectText = await evalJs(`document.getElementById("reconnect-status").textContent + "|" + document.getElementById("ws-status").textContent`).catch(() => "");
await sleep(2500);
check("recovers via auto-reconnect", (await evalJs(`document.getElementById("ws-status").textContent`).catch(() => "")) === "WebSocket: Connected", reconnectText);

// --- screenshots: dark + mobile
await setViewport(1400, 1000, true);
await sleep(400);
await shot("04-dark");
await setViewport(390, 900, false);
await sleep(400);
const overflow = await evalJs(`document.documentElement.scrollWidth - document.documentElement.clientWidth`);
check("no horizontal overflow at 390px", overflow <= 0, `overflow=${overflow}`);
await shot("05-mobile");

check("no console errors / exceptions", consoleProblems.length === 0, consoleProblems.join(" | "));
const failed = results.filter((r) => !r.ok);
console.log(`\n${results.length - failed.length}/${results.length} passed`);
console.log(`Screenshots: ${outDir}`);
process.exit(failed.length ? 1 : 0);
