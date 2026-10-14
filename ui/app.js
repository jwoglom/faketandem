const $ = (id) => document.getElementById(id);

const elements = {
  wsStatus: $("ws-status"),
  pumpStatus: $("pump-status"),
  reconnectStatus: $("reconnect-status"),
  connectBtn: $("connect-btn"),
  disconnectBtn: $("disconnect-btn"),
  getStateBtn: $("get-state-btn"),
  autoReconnectToggle: $("auto-reconnect"),
  clearLogBtn: $("clear-log-btn"),
  eventLog: $("event-log"),
  commandForm: $("command-form"),
  characteristicSelect: $("characteristic"),
  hexInput: $("hex-data"),
  hexError: $("hex-error"),
  notifyBtn: $("notify-btn"),
  setCharBtn: $("set-char-btn"),
  refreshSettingsBtn: $("refresh-settings-btn"),
  settingsFilter: $("settings-filter"),
  settingsList: $("settings-list"),
  pairingStateStatus: $("pairing-state-status"),
  pairingStateForm: $("pairing-state-form"),
  refreshPairingBtn: $("refresh-pairing-btn"),
  pairingAuthStatus: $("pairing-auth-status"),
  pairingConnectionStatus: $("pairing-connection-status"),
  pairingForm: $("pairing-form"),
  pairingCodeInput: $("pairing-code"),
  pairingError: $("pairing-error"),
  pairingProgress: $("pairing-progress"),
  setPairingBtn: $("set-pairing-btn"),
  resetPairingBtn: $("reset-pairing-btn"),
  disconnectPumpBtn: $("disconnect-pump-btn"),
  longTermKeyForm: $("long-term-key-form"),
  longTermKeyInput: $("long-term-key"),
  longTermKeyError: $("long-term-key-error"),
  resetLongTermKeyBtn: $("reset-long-term-key-btn"),
  editorTitle: $("editor-title"),
  dirtyBadge: $("dirty-badge"),
  editorEmpty: $("editor-empty"),
  editorBody: $("editor-body"),
  modeSelect: $("mode-select"),
  configMeta: $("config-meta"),
  constantValue: $("constant-value"),
  incrementalValues: $("incremental-values"),
  timeBasedValues: $("time-based-values"),
  addIncrementalBtn: $("add-incremental"),
  addTimeBasedBtn: $("add-time-based"),
  saveConfigBtn: $("save-config-btn"),
  resetConfigBtn: $("reset-config-btn"),
  reloadConfigBtn: $("reload-config-btn"),
  configError: $("config-error"),
  toasts: $("toasts"),
};

const MAX_LOG_ENTRIES = 500;
const TOAST_MS = 3200;
const TOAST_ERROR_MS = 6000;

const PAIRING_STATES = {
  NotDiscoverable: { text: "NotDiscoverable (0x10)", indicator: "bad" },
  DiscoverableOnly: { text: "DiscoverableOnly (0x10)", indicator: "ok" },
  PairStep1: { text: "PairStep1 (0x11)", indicator: "ok" },
  PairStep2: { text: "PairStep2 (0x12)", indicator: "ok" },
};

const state = {
  ws: null,
  autoReconnect: true,
  reconnectDelay: 1000,
  reconnectTimer: null,
  countdownTimer: null,
  manualClose: false,
  configs: {},
  selectedMessageType: "",
  dirty: false,
  appConnected: null,
  pairingSignature: "",
  replayUntil: 0,
  pairingStateRequestId: 0,
  pairingCodeTouched: false,
  hexTouched: false,
};

/* ---------- small helpers ---------- */

const setIndicator = (el, indicatorState, text) => {
  el.dataset.state = indicatorState;
  el.textContent = text;
};

const setFieldError = (input, errorEl, message) => {
  errorEl.textContent = message || "";
  if (message) {
    input.setAttribute("aria-invalid", "true");
  } else {
    input.removeAttribute("aria-invalid");
  }
};

const showToast = (message, kind = "info") => {
  const toast = document.createElement("div");
  toast.className = "toast";
  toast.dataset.kind = kind;
  toast.textContent = message;
  elements.toasts.append(toast);
  const lifetime = kind === "error" ? TOAST_ERROR_MS : TOAST_MS;
  window.setTimeout(() => {
    toast.classList.add("leaving");
    window.setTimeout(() => toast.remove(), 220);
  }, lifetime);
};

const clearLogPlaceholder = () => {
  const placeholder = elements.eventLog.querySelector(".log-empty");
  if (placeholder) {
    placeholder.remove();
  }
};

const showLogPlaceholder = () => {
  elements.eventLog.innerHTML = "";
  const placeholder = document.createElement("div");
  placeholder.className = "log-empty";
  placeholder.textContent = "Events will appear here as they happen.";
  elements.eventLog.append(placeholder);
};

// logEvent adds a newest-first entry. `detail` carries long hex payloads on
// their own line so they wrap instead of stretching the row.
const logEvent = (message, type = "info", detail = "") => {
  clearLogPlaceholder();
  const entry = document.createElement("div");
  entry.className = "log-entry";
  entry.dataset.type = type;

  const time = document.createElement("time");
  time.className = "log-time";
  time.textContent = new Date().toLocaleTimeString([], { hour12: false });

  const tag = document.createElement("span");
  tag.className = "log-tag";
  tag.textContent = type === "info" ? "" : type;

  const body = document.createElement("span");
  body.className = "log-body";
  body.textContent = message;
  if (detail) {
    const hex = document.createElement("span");
    hex.className = "log-hex";
    hex.textContent = detail;
    body.append(hex);
  }

  entry.append(time, tag, body);
  elements.eventLog.prepend(entry);
  while (elements.eventLog.childElementCount > MAX_LOG_ENTRIES) {
    elements.eventLog.lastElementChild.remove();
  }
};

/* ---------- connection status ---------- */

const updateWsStatus = (status, detail = "") => {
  const labels = {
    connected: ["ok", "Connected"],
    connecting: ["warn", "Connecting"],
    reconnecting: ["warn", "Reconnecting"],
    disconnected: ["bad", "Disconnected"],
  };
  const [indicator, label] = labels[status];
  setIndicator(elements.wsStatus, indicator, `WebSocket: ${label}`);
  elements.reconnectStatus.textContent = detail;
  elements.connectBtn.disabled = status !== "disconnected";
  elements.disconnectBtn.disabled = status === "disconnected";
  elements.getStateBtn.disabled = status !== "connected";
  elements.refreshPairingBtn.disabled = status !== "connected";
};

const setAppConnected = (connected) => {
  const changed = state.appConnected !== null && state.appConnected !== connected;
  state.appConnected = connected;
  if (connected === true) {
    setIndicator(elements.pumpStatus, "ok", "Phone app: Connected");
    setIndicator(elements.pairingConnectionStatus, "ok", "App connection: Connected");
  } else if (connected === false) {
    setIndicator(elements.pumpStatus, "bad", "Phone app: Disconnected");
    setIndicator(elements.pairingConnectionStatus, "bad", "App connection: Disconnected");
  } else {
    setIndicator(elements.pumpStatus, "idle", "Phone app: Unknown");
    setIndicator(elements.pairingConnectionStatus, "idle", "App connection: Unknown");
  }
  if (changed) {
    logEvent(`Phone app ${connected ? "connected" : "disconnected"}.`);
  }
};

/* ---------- pairing ---------- */

const updatePairingStateStatus = (pairingState) => {
  const desc = PAIRING_STATES[pairingState];
  if (!desc) {
    setIndicator(elements.pairingStateStatus, "idle", "Mode: Unknown");
    return;
  }
  setIndicator(elements.pairingStateStatus, desc.indicator, `Mode: ${desc.text}`);
  const radio = elements.pairingStateForm.querySelector(`input[value="${pairingState}"]`);
  if (radio) {
    radio.checked = true;
  }
};

const updatePairingStatus = (pairingCode, authenticated, longTermKey) => {
  if (authenticated === true) {
    setIndicator(elements.pairingAuthStatus, "ok", "Pairing: Authenticated");
  } else if (authenticated === false) {
    setIndicator(elements.pairingAuthStatus, "bad", "Pairing: Not authenticated");
  } else {
    setIndicator(elements.pairingAuthStatus, "idle", "Pairing: Unknown");
  }
  // Never overwrite a field the user is typing in.
  if (typeof pairingCode === "string" && pairingCode.length > 0 && document.activeElement !== elements.pairingCodeInput) {
    elements.pairingCodeInput.value = pairingCode;
  }
  if (document.activeElement !== elements.longTermKeyInput) {
    elements.longTermKeyInput.value = typeof longTermKey === "string" ? longTermKey : "";
  }
  updatePairingValidation();

  // The server repeats this message on every refresh; only log real changes.
  const signature = `${authenticated}|${pairingCode ?? ""}|${longTermKey ? "key" : "nokey"}`;
  if (signature !== state.pairingSignature) {
    const first = state.pairingSignature === "";
    state.pairingSignature = signature;
    if (!first) {
      logEvent(`Pairing: ${authenticated ? "authenticated" : "not authenticated"}; long-term key ${longTermKey ? "cached" : "not cached"}.`);
    }
  }
};

const setPairingProgress = (message, failed) => {
  // getPairingState makes the server replay its latest status; don't log that
  // replay as if it were a new event.
  const isReplay = Date.now() < state.replayUntil && message === elements.pairingProgress.textContent;
  elements.pairingProgress.dataset.state = failed ? "error" : "info";
  elements.pairingProgress.textContent = message;
  if (!isReplay) {
    logEvent(message, failed ? "error" : "info");
  }
};

const validatePairingCode = (value) => {
  const normalized = value.trim();
  if (!normalized) {
    return { valid: false, message: "Pairing code is required." };
  }
  if (!/^\d{6}$/.test(normalized)) {
    return { valid: false, message: "Pairing code must be 6 digits." };
  }
  return { valid: true };
};

const updatePairingValidation = () => {
  const { valid, message } = validatePairingCode(elements.pairingCodeInput.value);
  elements.setPairingBtn.disabled = !valid;
  // Don't nag while the user is still typing the first digits.
  setFieldError(elements.pairingCodeInput, elements.pairingError, !valid && state.pairingCodeTouched ? message : "");
};

const validateHex = (value) => {
  const normalized = value.trim();
  if (!normalized) {
    return { valid: true };
  }
  if (!/^[0-9a-fA-F]+$/.test(normalized)) {
    return { valid: false, immediate: true, message: "Hex payload can only include 0-9 and a-f characters." };
  }
  if (normalized.length % 2 !== 0) {
    return { valid: false, immediate: false, message: "Hex payload must have an even length." };
  }
  return { valid: true };
};

const updateHexValidation = () => {
  const { valid, immediate, message } = validateHex(elements.hexInput.value);
  // Odd length is normal mid-typing, so only flag it once the user moves on.
  setFieldError(elements.hexInput, elements.hexError, !valid && (immediate || state.hexTouched) ? message : "");
  elements.notifyBtn.disabled = !valid;
  elements.setCharBtn.disabled = !valid;
};

/* ---------- websocket ---------- */

const wsUrl = () => {
  const scheme = window.location.protocol === "https:" ? "wss" : "ws";
  return `${scheme}://${window.location.host}/ws`;
};

const clearReconnect = () => {
  if (state.reconnectTimer) {
    window.clearTimeout(state.reconnectTimer);
    state.reconnectTimer = null;
  }
  if (state.countdownTimer) {
    window.clearInterval(state.countdownTimer);
    state.countdownTimer = null;
  }
  elements.reconnectStatus.textContent = "";
};

const scheduleReconnect = () => {
  if (!state.autoReconnect || state.reconnectTimer) {
    return;
  }
  const delay = state.reconnectDelay;
  const deadline = Date.now() + delay;
  const renderCountdown = () => {
    const seconds = Math.max(0, Math.ceil((deadline - Date.now()) / 1000));
    updateWsStatus("reconnecting", `Reconnecting in ${seconds}s`);
  };
  renderCountdown();
  state.countdownTimer = window.setInterval(renderCountdown, 250);
  state.reconnectTimer = window.setTimeout(() => {
    clearReconnect();
    connectWebSocket();
  }, delay);
  state.reconnectDelay = Math.min(state.reconnectDelay * 2, 30000);
};

const handleMessage = (payload) => {
  if (typeof payload.connected === "boolean") {
    setAppConnected(payload.connected);
    return;
  }
  switch (payload.type) {
    case "pairing_state":
      updatePairingStatus(payload.pairing_code, payload.authenticated, payload.long_term_key);
      return;
    case "pairing_progress":
    case "pairing_error":
      setPairingProgress(payload.message, payload.type === "pairing_error");
      return;
    case "connected":
    case "disconnected":
      setAppConnected(payload.type === "connected");
      return;
    case "write":
    case "read":
    case "notify":
      logEvent(payload.characteristic ?? "", payload.type, payload.data ?? "");
      return;
    default:
      logEvent(`${payload.characteristic ? `${payload.characteristic} ` : ""}${payload.message ?? ""}`.trim() || "(empty event)", payload.type ?? "event", payload.data ?? "");
  }
};

const connectWebSocket = () => {
  if (state.ws && (state.ws.readyState === WebSocket.OPEN || state.ws.readyState === WebSocket.CONNECTING)) {
    return;
  }
  clearReconnect();
  updateWsStatus("connecting", "Connecting…");
  const ws = new WebSocket(wsUrl());
  state.ws = ws;

  ws.addEventListener("open", () => {
    state.reconnectDelay = 1000;
    updateWsStatus("connected");
    logEvent("WebSocket connected.");
    requestState();
  });

  ws.addEventListener("message", (event) => {
    try {
      handleMessage(JSON.parse(event.data));
    } catch (error) {
      logEvent(`Unparseable message: ${event.data}`, "error");
    }
  });

  ws.addEventListener("close", () => {
    if (state.ws !== ws) {
      return; // A newer socket has replaced this one.
    }
    state.appConnected = null;
    setAppConnected(null);
    const manual = state.manualClose;
    state.manualClose = false;
    updateWsStatus("disconnected");
    logEvent("WebSocket disconnected.");
    if (!manual) {
      scheduleReconnect();
    }
  });

  ws.addEventListener("error", () => {
    logEvent("WebSocket error encountered.", "error");
  });
};

const disconnectWebSocket = () => {
  clearReconnect();
  if (state.ws && (state.ws.readyState === WebSocket.OPEN || state.ws.readyState === WebSocket.CONNECTING)) {
    state.manualClose = true;
    state.ws.close();
    return;
  }
  updateWsStatus("disconnected");
};

const sendCommand = (payload, { quiet = false } = {}) => {
  if (!state.ws || state.ws.readyState !== WebSocket.OPEN) {
    showToast("Not connected to the emulator. Connect first.", "error");
    logEvent("WebSocket is not connected.", "error");
    return false;
  }
  state.ws.send(JSON.stringify(payload));
  if (!quiet) {
    showToast(`Sent: ${payload.command}`);
  }
  return true;
};

const requestState = () => {
  state.replayUntil = Date.now() + 1500;
  sendCommand({ command: "getState" }, { quiet: true });
  sendCommand({ command: "getPairingState" }, { quiet: true });
  fetchPairingState();
};

/* ---------- pairing mode (REST) ---------- */

const fetchPairingState = async () => {
  const requestId = ++state.pairingStateRequestId;
  try {
    const response = await fetch("/api/bluetooth/pairingstate");
    if (!response.ok) {
      throw new Error(`Failed to load pairing mode (${response.status}).`);
    }
    const data = await response.json();
    if (requestId === state.pairingStateRequestId) {
      updatePairingStateStatus(data.pairingState);
    }
  } catch (error) {
    logEvent(`Failed to fetch pairing mode: ${error.message}`, "error");
  }
};

const setPairingState = async (pairingState) => {
  const requestId = ++state.pairingStateRequestId;
  elements.pairingStateForm.querySelector(".segmented").classList.add("busy");
  try {
    const response = await fetch("/api/bluetooth/pairingstate", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ pairingState }),
    });
    if (!response.ok) {
      const text = await response.text();
      throw new Error(text.trim() || `Failed to set pairing mode (${response.status}).`);
    }
    const data = await response.json();
    if (requestId === state.pairingStateRequestId) {
      updatePairingStateStatus(data.pairingState);
    }
    showToast(`Pairing mode: ${data.pairingState}`, "success");
    logEvent(`Pairing mode set to ${data.pairingState}.`);
  } catch (error) {
    showToast(error.message, "error");
    logEvent(`Failed to set pairing mode: ${error.message}`, "error");
    fetchPairingState(); // Snap the selection back to what the server really has.
  } finally {
    elements.pairingStateForm.querySelector(".segmented").classList.remove("busy");
  }
};

/* ---------- settings editor ---------- */

const setDirty = (dirty) => {
  state.dirty = dirty;
  elements.dirtyBadge.hidden = !dirty;
};

const setConfigError = (text) => {
  elements.configError.textContent = text;
};

const safeJsonParse = (value, label) => {
  if (!value.trim()) {
    return { error: `${label} is required.` };
  }
  try {
    return { value: JSON.parse(value) };
  } catch (error) {
    return { error: `${label} must be valid JSON.` };
  }
};

const createRemoveButton = (row) => {
  const actions = document.createElement("div");
  actions.className = "row-actions";
  const removeButton = document.createElement("button");
  removeButton.type = "button";
  removeButton.textContent = "Remove";
  removeButton.addEventListener("click", () => {
    row.remove();
    setDirty(true);
  });
  actions.append(removeButton);
  return actions;
};

const createIncrementalRow = (value = "{}") => {
  const row = document.createElement("div");
  row.className = "list-row";
  const textarea = document.createElement("textarea");
  textarea.rows = 4;
  textarea.spellcheck = false;
  textarea.value = value;
  textarea.setAttribute("aria-label", "Value (JSON)");
  row.append(textarea, createRemoveButton(row));
  return row;
};

const createTimeBasedRow = (value = "{}", timing = "0") => {
  const row = document.createElement("div");
  row.className = "list-row time-row";

  const valueLabel = document.createElement("label");
  valueLabel.append("Value (JSON)");
  const textarea = document.createElement("textarea");
  textarea.rows = 3;
  textarea.spellcheck = false;
  textarea.value = value;
  valueLabel.append(textarea);

  const timingLabel = document.createElement("label");
  timingLabel.append("Timing (s)");
  const timingInput = document.createElement("input");
  timingInput.type = "number";
  timingInput.min = "0";
  timingInput.value = timing;
  timingLabel.append(timingInput);

  row.append(valueLabel, timingLabel, createRemoveButton(row));
  return row;
};

const renderSettingsList = () => {
  const scrollTop = elements.settingsList.scrollTop;
  const filter = elements.settingsFilter.value.trim().toLowerCase();
  const entries = Object.entries(state.configs)
    .filter(([messageType]) => messageType.toLowerCase().includes(filter))
    .sort((a, b) => a[0].localeCompare(b[0]));

  elements.settingsList.replaceChildren();
  elements.settingsList.setAttribute("aria-busy", "false");
  if (!entries.length) {
    const empty = document.createElement("p");
    empty.className = "empty";
    empty.textContent = Object.keys(state.configs).length ? "No message types match the filter." : "No settings found.";
    elements.settingsList.append(empty);
    return;
  }
  entries.forEach(([messageType, config]) => {
    const item = document.createElement("button");
    item.type = "button";
    item.className = "settings-item";
    item.setAttribute("role", "listitem");
    if (messageType === state.selectedMessageType) {
      item.setAttribute("aria-current", "true");
    }
    const name = document.createElement("span");
    name.className = "name";
    name.textContent = messageType;
    const mode = document.createElement("span");
    mode.className = "badge";
    mode.textContent = config.mode || "constant";
    item.append(name, mode);
    item.addEventListener("click", () => selectConfig(messageType, config));
    elements.settingsList.append(item);
  });
  elements.settingsList.scrollTop = scrollTop;
};

const updateEditorVisibility = () => {
  const mode = elements.modeSelect.value;
  document.querySelectorAll(".editor-section").forEach((section) => {
    section.hidden = section.dataset.mode !== mode;
  });
};

const setEditorEnabled = (enabled) => {
  elements.editorEmpty.hidden = enabled;
  elements.editorBody.hidden = !enabled;
  elements.saveConfigBtn.disabled = !enabled;
  elements.reloadConfigBtn.disabled = !enabled;
  elements.resetConfigBtn.disabled = !enabled;
};

const loadConfig = (messageType, config) => {
  state.selectedMessageType = messageType;
  elements.editorTitle.textContent = messageType;
  elements.modeSelect.value = config.mode || "constant";
  elements.constantValue.value = config.value ? JSON.stringify(config.value, null, 2) : "";
  elements.incrementalValues.replaceChildren();
  elements.timeBasedValues.replaceChildren();
  (config.values || []).forEach((value) => {
    elements.incrementalValues.append(createIncrementalRow(JSON.stringify(value, null, 2)));
  });
  if (!config.values || config.values.length === 0) {
    elements.incrementalValues.append(createIncrementalRow());
  }
  if (config.values && config.timing_seconds) {
    config.values.forEach((value, index) => {
      const timing = config.timing_seconds[index] ?? 0;
      elements.timeBasedValues.append(createTimeBasedRow(JSON.stringify(value, null, 2), timing));
    });
  } else {
    elements.timeBasedValues.append(createTimeBasedRow());
  }
  elements.configMeta.textContent = `Current index: ${config.current_index ?? 0} · Start time: ${config.start_time || "not started"}`;
  setEditorEnabled(true);
  updateEditorVisibility();
  setConfigError("");
  setDirty(false);
  renderSettingsList();
};

const selectConfig = (messageType, config) => {
  if (messageType === state.selectedMessageType) {
    return;
  }
  if (state.dirty && !window.confirm(`Discard unsaved changes to ${state.selectedMessageType}?`)) {
    return;
  }
  loadConfig(messageType, config);
};

const fetchSettings = async ({ notify = false } = {}) => {
  try {
    const response = await fetch("/api/settings");
    if (!response.ok) {
      throw new Error(`Failed to load settings (${response.status}).`);
    }
    state.configs = await response.json();
    renderSettingsList();
    if (notify) {
      showToast("Settings list refreshed.", "success");
    }
  } catch (error) {
    elements.settingsList.setAttribute("aria-busy", "false");
    if (!Object.keys(state.configs).length) {
      elements.settingsList.innerHTML = "";
      const empty = document.createElement("p");
      empty.className = "empty";
      empty.textContent = error.message;
      elements.settingsList.append(empty);
    }
    showToast(error.message, "error");
  }
};

const buildConfigPayload = () => {
  const mode = elements.modeSelect.value;

  if (mode === "constant") {
    const parsed = safeJsonParse(elements.constantValue.value, "Value");
    if (parsed.error) {
      return { error: parsed.error };
    }
    return { payload: { mode, value: parsed.value } };
  }

  if (mode === "incremental") {
    const rows = Array.from(elements.incrementalValues.querySelectorAll("textarea"));
    const values = [];
    for (const row of rows) {
      const parsed = safeJsonParse(row.value, "Each value");
      if (parsed.error) {
        return { error: parsed.error };
      }
      values.push(parsed.value);
    }
    if (!values.length) {
      return { error: "Incremental mode requires at least one value." };
    }
    return { payload: { mode, values } };
  }

  if (mode === "time_based") {
    const rows = Array.from(elements.timeBasedValues.querySelectorAll(".time-row"));
    const values = [];
    const timingSeconds = [];
    for (const row of rows) {
      const textarea = row.querySelector("textarea");
      const timingInput = row.querySelector("input[type='number']");
      const parsed = safeJsonParse(textarea.value, "Each value");
      if (parsed.error) {
        return { error: parsed.error };
      }
      const timing = Number(timingInput.value);
      if (Number.isNaN(timing)) {
        return { error: "Timing must be a number." };
      }
      values.push(parsed.value);
      timingSeconds.push(timing);
    }
    if (!values.length) {
      return { error: "Time-based mode requires at least one value." };
    }
    for (let i = 1; i < timingSeconds.length; i += 1) {
      if (timingSeconds[i] < timingSeconds[i - 1]) {
        return { error: "Timing values must be in ascending order." };
      }
    }
    return { payload: { mode, values, timing_seconds: timingSeconds } };
  }

  return { error: "Unsupported mode." };
};

// withBusy disables a button for the duration of a request so a slow
// round-trip can't be double-submitted.
const withBusy = async (button, work) => {
  const wasDisabled = button.disabled;
  button.disabled = true;
  try {
    return await work();
  } finally {
    button.disabled = wasDisabled;
  }
};

const apiError = async (response, fallback) => {
  const text = (await response.text()).trim();
  return new Error(text || `${fallback} (${response.status}).`);
};

const reloadConfig = async ({ notify = true } = {}) => {
  if (!state.selectedMessageType) {
    return;
  }
  try {
    const response = await fetch(`/api/settings/${encodeURIComponent(state.selectedMessageType)}`);
    if (!response.ok) {
      throw await apiError(response, "Failed to reload");
    }
    loadConfig(state.selectedMessageType, await response.json());
    if (notify) {
      showToast("Configuration reloaded.", "success");
    }
  } catch (error) {
    showToast(error.message, "error");
  }
};

const saveConfig = async () => {
  if (!state.selectedMessageType) {
    return;
  }
  const { payload, error } = buildConfigPayload();
  setConfigError(error || "");
  if (error) {
    return;
  }
  await withBusy(elements.saveConfigBtn, async () => {
    try {
      const response = await fetch(`/api/settings/${encodeURIComponent(state.selectedMessageType)}`, {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(payload),
      });
      if (!response.ok) {
        throw await apiError(response, "Failed to save settings");
      }
      await fetchSettings();
      await reloadConfig({ notify: false });
      showToast(`Saved ${state.selectedMessageType}.`, "success");
    } catch (saveError) {
      showToast(saveError.message, "error");
    }
  });
};

const resetConfigState = async () => {
  if (!state.selectedMessageType) {
    return;
  }
  await withBusy(elements.resetConfigBtn, async () => {
    try {
      const response = await fetch(`/api/settings/${encodeURIComponent(state.selectedMessageType)}/reset`, { method: "POST" });
      if (!response.ok) {
        throw await apiError(response, "Failed to reset");
      }
      await reloadConfig({ notify: false });
      showToast(`Reset state for ${state.selectedMessageType}.`, "success");
    } catch (error) {
      showToast(error.message, "error");
    }
  });
};

/* ---------- wiring ---------- */

const submitPairingCode = () => {
  state.pairingCodeTouched = true;
  const { valid, message } = validatePairingCode(elements.pairingCodeInput.value);
  updatePairingValidation();
  if (!valid) {
    setFieldError(elements.pairingCodeInput, elements.pairingError, message);
    return;
  }
  sendCommand({ command: "setPairingCode", pairingCode: elements.pairingCodeInput.value.trim() });
};

const submitLongTermKey = () => {
  const longTermKey = elements.longTermKeyInput.value.trim();
  if (!/^[0-9a-fA-F]+$/.test(longTermKey) || longTermKey.length % 2 !== 0) {
    setFieldError(elements.longTermKeyInput, elements.longTermKeyError, "Long-term key must be an even-length hex string.");
    return;
  }
  setFieldError(elements.longTermKeyInput, elements.longTermKeyError, "");
  sendCommand({ command: "setLongTermKey", longTermKey });
};

const sendCharacteristicCommand = (command) => {
  state.hexTouched = true;
  updateHexValidation();
  if (!validateHex(elements.hexInput.value).valid) {
    return;
  }
  sendCommand({
    command,
    characteristic: elements.characteristicSelect.value,
    data: elements.hexInput.value.trim(),
  });
};

const init = () => {
  updateWsStatus("disconnected");
  setAppConnected(null);
  showLogPlaceholder();
  setEditorEnabled(false);
  updateHexValidation();
  updatePairingValidation();
  updateEditorVisibility();
  fetchSettings();
  fetchPairingState();

  elements.connectBtn.addEventListener("click", connectWebSocket);
  elements.disconnectBtn.addEventListener("click", disconnectWebSocket);
  elements.getStateBtn.addEventListener("click", requestState);
  elements.refreshPairingBtn.addEventListener("click", requestState);
  elements.clearLogBtn.addEventListener("click", showLogPlaceholder);
  elements.autoReconnectToggle.addEventListener("change", (event) => {
    state.autoReconnect = event.target.checked;
    if (!state.autoReconnect) {
      clearReconnect();
      if (!state.ws || state.ws.readyState === WebSocket.CLOSED) {
        updateWsStatus("disconnected");
      }
    } else if (!state.ws || state.ws.readyState === WebSocket.CLOSED) {
      scheduleReconnect();
    }
  });

  elements.hexInput.addEventListener("input", updateHexValidation);
  elements.hexInput.addEventListener("blur", () => {
    state.hexTouched = true;
    updateHexValidation();
  });
  elements.commandForm.addEventListener("submit", (event) => {
    event.preventDefault();
    sendCharacteristicCommand("notify");
  });
  elements.setCharBtn.addEventListener("click", () => sendCharacteristicCommand("setCharacteristic"));

  elements.pairingStateForm.addEventListener("change", (event) => {
    if (event.target.name === "pairing-state") {
      setPairingState(event.target.value);
    }
  });
  elements.pairingCodeInput.addEventListener("input", () => {
    const digits = elements.pairingCodeInput.value.replace(/\D/g, "").slice(0, 6);
    if (digits !== elements.pairingCodeInput.value) {
      elements.pairingCodeInput.value = digits;
    }
    updatePairingValidation();
  });
  elements.pairingCodeInput.addEventListener("blur", () => {
    state.pairingCodeTouched = true;
    updatePairingValidation();
  });
  elements.pairingForm.addEventListener("submit", (event) => {
    event.preventDefault();
    submitPairingCode();
  });
  elements.resetPairingBtn.addEventListener("click", () => sendCommand({ command: "resetPairing" }));
  elements.disconnectPumpBtn.addEventListener("click", () => sendCommand({ command: "disconnectPump" }));
  elements.longTermKeyForm.addEventListener("submit", (event) => {
    event.preventDefault();
    submitLongTermKey();
  });
  elements.resetLongTermKeyBtn.addEventListener("click", () => {
    setFieldError(elements.longTermKeyInput, elements.longTermKeyError, "");
    sendCommand({ command: "resetLongTermKey" });
  });

  elements.refreshSettingsBtn.addEventListener("click", () => fetchSettings({ notify: true }));
  elements.settingsFilter.addEventListener("input", renderSettingsList);
  elements.modeSelect.addEventListener("change", updateEditorVisibility);
  elements.editorBody.addEventListener("input", () => setDirty(true));
  elements.addIncrementalBtn.addEventListener("click", () => {
    elements.incrementalValues.append(createIncrementalRow());
    setDirty(true);
  });
  elements.addTimeBasedBtn.addEventListener("click", () => {
    elements.timeBasedValues.append(createTimeBasedRow());
    setDirty(true);
  });
  elements.saveConfigBtn.addEventListener("click", saveConfig);
  elements.reloadConfigBtn.addEventListener("click", () => {
    if (state.dirty && !window.confirm("Discard unsaved changes and reload from the emulator?")) {
      return;
    }
    reloadConfig();
  });
  elements.resetConfigBtn.addEventListener("click", resetConfigState);

  window.addEventListener("beforeunload", (event) => {
    if (state.dirty) {
      event.preventDefault();
    }
  });

  // The old UI required clicking Connect before anything worked.
  connectWebSocket();
};

window.addEventListener("DOMContentLoaded", init);
