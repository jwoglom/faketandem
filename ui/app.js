const elements = {
  wsStatus: document.getElementById("ws-status"),
  pumpStatus: document.getElementById("pump-status"),
  reconnectStatus: document.getElementById("reconnect-status"),
  connectBtn: document.getElementById("connect-btn"),
  disconnectBtn: document.getElementById("disconnect-btn"),
  getStateBtn: document.getElementById("get-state-btn"),
  autoReconnectToggle: document.getElementById("auto-reconnect"),
  clearLogBtn: document.getElementById("clear-log-btn"),
  eventLog: document.getElementById("event-log"),
  characteristicSelect: document.getElementById("characteristic"),
  hexInput: document.getElementById("hex-data"),
  hexError: document.getElementById("hex-error"),
  notifyBtn: document.getElementById("notify-btn"),
  setCharBtn: document.getElementById("set-char-btn"),
  refreshSettingsBtn: document.getElementById("refresh-settings-btn"),
  settingsList: document.getElementById("settings-list"),
  refreshDiscoverableBtn: document.getElementById("refresh-discoverable-btn"),
  discoverableStatus: document.getElementById("discoverable-status"),
  enableDiscoverableBtn: document.getElementById("enable-discoverable-btn"),
  disableDiscoverableBtn: document.getElementById("disable-discoverable-btn"),
  refreshAllowPairingBtn: document.getElementById("refresh-allow-pairing-btn"),
  allowPairingStatus: document.getElementById("allow-pairing-status"),
  enableAllowPairingBtn: document.getElementById("enable-allow-pairing-btn"),
  disableAllowPairingBtn: document.getElementById("disable-allow-pairing-btn"),
  refreshPairingBtn: document.getElementById("refresh-pairing-btn"),
  pairingAuthStatus: document.getElementById("pairing-auth-status"),
  pairingConnectionStatus: document.getElementById("pairing-connection-status"),
  pairingCodeInput: document.getElementById("pairing-code"),
  pairingError: document.getElementById("pairing-error"),
  setPairingBtn: document.getElementById("set-pairing-btn"),
  resetPairingBtn: document.getElementById("reset-pairing-btn"),
  disconnectPumpBtn: document.getElementById("disconnect-pump-btn"),
  messageTypeInput: document.getElementById("message-type"),
  modeSelect: document.getElementById("mode-select"),
  configMeta: document.getElementById("config-meta"),
  constantValue: document.getElementById("constant-value"),
  incrementalValues: document.getElementById("incremental-values"),
  timeBasedValues: document.getElementById("time-based-values"),
  addIncrementalBtn: document.getElementById("add-incremental"),
  addTimeBasedBtn: document.getElementById("add-time-based"),
  saveConfigBtn: document.getElementById("save-config-btn"),
  resetConfigBtn: document.getElementById("reset-config-btn"),
  reloadConfigBtn: document.getElementById("reload-config-btn"),
  configStatus: document.getElementById("config-status"),
  configError: document.getElementById("config-error"),
};

const state = {
  ws: null,
  autoReconnect: true,
  reconnectDelay: 1000,
  reconnectTimer: null,
  manualClose: false,
  configs: {},
  selectedMessageType: "",
};

const statusClasses = ["connected", "disconnected", "reconnecting"];

const resetStatusClasses = (el) => {
  statusClasses.forEach((cls) => el.classList.remove(cls));
};

const updateWsStatus = (status, detail = "") => {
  resetStatusClasses(elements.wsStatus);
  elements.wsStatus.classList.add(status);
  const label = status === "connected" ? "Connected" : status === "reconnecting" ? "Reconnecting" : "Disconnected";
  elements.wsStatus.textContent = `WebSocket: ${label}`;
  elements.reconnectStatus.textContent = detail;
};

const updatePumpStatus = (connected) => {
  resetStatusClasses(elements.pumpStatus);
  resetStatusClasses(elements.pairingConnectionStatus);
  if (connected === true) {
    elements.pumpStatus.classList.add("connected");
    elements.pumpStatus.textContent = "Pump: Connected";
    elements.pairingConnectionStatus.classList.add("connected");
    elements.pairingConnectionStatus.textContent = "App Connection: Connected";
    return;
  }
  if (connected === false) {
    elements.pumpStatus.classList.add("disconnected");
    elements.pumpStatus.textContent = "Pump: Disconnected";
    elements.pairingConnectionStatus.classList.add("disconnected");
    elements.pairingConnectionStatus.textContent = "App Connection: Disconnected";
    return;
  }
  elements.pumpStatus.textContent = "Pump: Unknown";
  elements.pairingConnectionStatus.textContent = "App Connection: Unknown";
};

const updateDiscoverableStatus = (discoverable) => {
  resetStatusClasses(elements.discoverableStatus);
  if (discoverable === true) {
    elements.discoverableStatus.classList.add("connected");
    elements.discoverableStatus.textContent = "Discoverable: Enabled";
  } else if (discoverable === false) {
    elements.discoverableStatus.classList.add("disconnected");
    elements.discoverableStatus.textContent = "Discoverable: Disabled";
  } else {
    elements.discoverableStatus.textContent = "Discoverable: Unknown";
  }
};

const updateAllowPairingStatus = (allowPairing) => {
  resetStatusClasses(elements.allowPairingStatus);
  if (allowPairing === true) {
    elements.allowPairingStatus.classList.add("connected");
    elements.allowPairingStatus.textContent = "Allow Pairing: Enabled";
  } else if (allowPairing === false) {
    elements.allowPairingStatus.classList.add("disconnected");
    elements.allowPairingStatus.textContent = "Allow Pairing: Disabled";
  } else {
    elements.allowPairingStatus.textContent = "Allow Pairing: Unknown";
  }
};

const updatePairingStatus = (pairingCode, authenticated) => {
  resetStatusClasses(elements.pairingAuthStatus);
  if (authenticated === true) {
    elements.pairingAuthStatus.classList.add("connected");
    elements.pairingAuthStatus.textContent = "Pairing: Authenticated";
  } else if (authenticated === false) {
    elements.pairingAuthStatus.classList.add("disconnected");
    elements.pairingAuthStatus.textContent = "Pairing: Not Authenticated";
  } else {
    elements.pairingAuthStatus.textContent = "Pairing: Unknown";
  }
  if (typeof pairingCode === "string" && pairingCode.length > 0) {
    elements.pairingCodeInput.value = pairingCode;
  }
  updatePairingValidation();
};

const logEvent = (message, type = "info") => {
  const entry = document.createElement("div");
  entry.className = "log-entry";
  const timestamp = new Date().toLocaleTimeString();
  entry.textContent = `[${timestamp}] ${message}`;
  entry.dataset.type = type;
  elements.eventLog.prepend(entry);
};

const wsUrl = () => {
  const scheme = window.location.protocol === "https:" ? "wss" : "ws";
  return `${scheme}://${window.location.host}/ws`;
};

const scheduleReconnect = () => {
  if (!state.autoReconnect || state.reconnectTimer) {
    return;
  }
  const delay = state.reconnectDelay;
  updateWsStatus("reconnecting", `Reconnecting in ${Math.round(delay / 1000)}s`);
  state.reconnectTimer = window.setTimeout(() => {
    state.reconnectTimer = null;
    connectWebSocket();
  }, delay);
  state.reconnectDelay = Math.min(state.reconnectDelay * 2, 30000);
};

const clearReconnect = () => {
  if (state.reconnectTimer) {
    window.clearTimeout(state.reconnectTimer);
    state.reconnectTimer = null;
  }
  elements.reconnectStatus.textContent = "";
};

const connectWebSocket = () => {
  if (state.ws && (state.ws.readyState === WebSocket.OPEN || state.ws.readyState === WebSocket.CONNECTING)) {
    return;
  }
  clearReconnect();
  updateWsStatus("reconnecting", "Connecting...");
  const ws = new WebSocket(wsUrl());
  state.ws = ws;

  ws.addEventListener("open", () => {
    state.reconnectDelay = 1000;
    updateWsStatus("connected");
    logEvent("WebSocket connected.");
    sendCommand({ command: "getState" });
    requestPairingState();
  });

  ws.addEventListener("message", (event) => {
    try {
      const payload = JSON.parse(event.data);
      if (typeof payload.connected === "boolean") {
        updatePumpStatus(payload.connected);
        logEvent(`Pump connection state: ${payload.connected ? "connected" : "disconnected"}.`);
        return;
      }
      if (payload.type === "pairing_state") {
        updatePairingStatus(payload.pairing_code, payload.authenticated);
        logEvent("Pairing state updated.");
        return;
      }
      if (payload.type === "connected" || payload.type === "disconnected") {
        updatePumpStatus(payload.type === "connected");
        logEvent(`Pump connection state: ${payload.type}.`);
        return;
      }
      const type = payload.type ?? "event";
      const characteristic = payload.characteristic ? ` (${payload.characteristic})` : "";
      const data = payload.data ? ` data=${payload.data}` : "";
      const message = payload.message ? ` ${payload.message}` : "";
      logEvent(`${type}${characteristic}${data}${message}`);
    } catch (error) {
      logEvent(`Unparseable message: ${event.data}`, "error");
    }
  });

  ws.addEventListener("close", () => {
    updateWsStatus("disconnected", "");
    logEvent("WebSocket disconnected.");
    if (state.manualClose) {
      state.manualClose = false;
      return;
    }
    scheduleReconnect();
  });

  ws.addEventListener("error", () => {
    logEvent("WebSocket error encountered.", "error");
  });
};

const disconnectWebSocket = () => {
  if (!state.ws) {
    return;
  }
  state.manualClose = true;
  state.ws.close();
};

const sendCommand = (payload) => {
  if (!state.ws || state.ws.readyState !== WebSocket.OPEN) {
    logEvent("WebSocket is not connected.", "error");
    return;
  }
  state.ws.send(JSON.stringify(payload));
};

const requestPairingState = () => {
  sendCommand({ command: "getPairingState" });
};

const validateHex = (value) => {
  if (!value) {
    return { valid: true };
  }
  const normalized = value.trim();
  if (normalized.length % 2 !== 0) {
    return { valid: false, message: "Hex payload must have an even length." };
  }
  if (!/^[0-9a-fA-F]+$/.test(normalized)) {
    return { valid: false, message: "Hex payload can only include 0-9 and a-f characters." };
  }
  return { valid: true };
};

const updateHexValidation = () => {
  const { valid, message } = validateHex(elements.hexInput.value);
  elements.hexError.textContent = valid ? "" : message;
  elements.notifyBtn.disabled = !valid;
  elements.setCharBtn.disabled = !valid;
};

const validatePairingCode = (value) => {
  if (!value) {
    return { valid: false, message: "Pairing code is required." };
  }
  const normalized = value.trim();
  if (!/^\d{6}$/.test(normalized)) {
    return { valid: false, message: "Pairing code must be 6 digits." };
  }
  return { valid: true };
};

const updatePairingValidation = () => {
  const value = elements.pairingCodeInput.value.trim();
  if (!value) {
    elements.pairingError.textContent = "";
    elements.setPairingBtn.disabled = true;
    return;
  }
  const { valid, message } = validatePairingCode(value);
  elements.pairingError.textContent = valid ? "" : message;
  elements.setPairingBtn.disabled = !valid;
};

const clearConfigStatus = () => {
  elements.configStatus.textContent = "";
  elements.configStatus.className = "status-banner";
};

const setConfigStatus = (text, type) => {
  elements.configStatus.textContent = text;
  elements.configStatus.className = `status-banner ${type}`;
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

const createIncrementalRow = (value = "{}") => {
  const row = document.createElement("div");
  row.className = "list-row";
  const textarea = document.createElement("textarea");
  textarea.rows = 4;
  textarea.spellcheck = false;
  textarea.value = value;
  const actions = document.createElement("div");
  actions.className = "row-actions";
  const removeButton = document.createElement("button");
  removeButton.type = "button";
  removeButton.textContent = "Remove";
  removeButton.addEventListener("click", () => row.remove());
  actions.append(removeButton);
  row.append(textarea, actions);
  return row;
};

const createTimeBasedRow = (value = "{}", timing = "0") => {
  const row = document.createElement("div");
  row.className = "list-row time-row";
  const valueWrap = document.createElement("div");
  const valueLabel = document.createElement("label");
  valueLabel.textContent = "Value (JSON)";
  const textarea = document.createElement("textarea");
  textarea.rows = 3;
  textarea.spellcheck = false;
  textarea.value = value;
  valueWrap.append(valueLabel, textarea);

  const timingWrap = document.createElement("div");
  const timingLabel = document.createElement("label");
  timingLabel.textContent = "Timing (s)";
  const timingInput = document.createElement("input");
  timingInput.type = "number";
  timingInput.min = "0";
  timingInput.value = timing;
  timingWrap.append(timingLabel, timingInput);

  const actions = document.createElement("div");
  actions.className = "row-actions";
  const removeButton = document.createElement("button");
  removeButton.type = "button";
  removeButton.textContent = "Remove";
  removeButton.addEventListener("click", () => row.remove());
  actions.append(removeButton);

  row.append(valueWrap, timingWrap, actions);
  return row;
};

const renderSettingsList = () => {
  elements.settingsList.innerHTML = "";
  const entries = Object.entries(state.configs).sort((a, b) => a[0].localeCompare(b[0]));
  if (!entries.length) {
    elements.settingsList.textContent = "No settings found.";
    return;
  }
  entries.forEach(([messageType, config]) => {
    const item = document.createElement("div");
    item.className = "settings-item";
    if (messageType === state.selectedMessageType) {
      item.classList.add("active");
    }
    const title = document.createElement("h3");
    title.textContent = messageType;
    const preview = document.createElement("pre");
    preview.textContent = JSON.stringify(config, null, 2);
    item.append(title, preview);
    item.addEventListener("click", () => loadConfig(messageType, config));
    elements.settingsList.append(item);
  });
};

const fetchDiscoverableState = async () => {
  try {
    const response = await fetch("/api/bluetooth/discoverable");
    if (!response.ok) {
      throw new Error(`Failed to load discoverable state (${response.status}).`);
    }
    const data = await response.json();
    updateDiscoverableStatus(data.discoverable);
    logEvent(`Discoverable state: ${data.discoverable ? "enabled" : "disabled"}.`);
  } catch (error) {
    logEvent(`Failed to fetch discoverable state: ${error.message}`, "error");
  }
};

const setDiscoverableState = async (discoverable) => {
  try {
    const response = await fetch("/api/bluetooth/discoverable", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ discoverable }),
    });
    if (!response.ok) {
      const text = await response.text();
      throw new Error(text || `Failed to set discoverable state (${response.status}).`);
    }
    const data = await response.json();
    updateDiscoverableStatus(data.discoverable);
    logEvent(`Discoverable mode ${data.discoverable ? "enabled" : "disabled"}.`);
  } catch (error) {
    logEvent(`Failed to set discoverable state: ${error.message}`, "error");
  }
};

const fetchAllowPairingState = async () => {
  try {
    const response = await fetch("/api/bluetooth/allowpairing");
    if (!response.ok) {
      throw new Error(`Failed to load allow pairing state (${response.status}).`);
    }
    const data = await response.json();
    updateAllowPairingStatus(data.allowPairing);
    logEvent(`Allow pairing state: ${data.allowPairing ? "enabled" : "disabled"}.`);
  } catch (error) {
    logEvent(`Failed to fetch allow pairing state: ${error.message}`, "error");
  }
};

const setAllowPairingState = async (allowPairing) => {
  try {
    const response = await fetch("/api/bluetooth/allowpairing", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ allowPairing }),
    });
    if (!response.ok) {
      const text = await response.text();
      throw new Error(text || `Failed to set allow pairing state (${response.status}).`);
    }
    const data = await response.json();
    updateAllowPairingStatus(data.allowPairing);
    logEvent(`Allow pairing mode ${data.allowPairing ? "enabled" : "disabled"}.`);
  } catch (error) {
    logEvent(`Failed to set allow pairing state: ${error.message}`, "error");
  }
};

const fetchSettings = async () => {
  clearConfigStatus();
  try {
    const response = await fetch("/api/settings");
    if (!response.ok) {
      throw new Error(`Failed to load settings (${response.status}).`);
    }
    state.configs = await response.json();
    renderSettingsList();
    setConfigStatus("Settings list refreshed.", "success");
  } catch (error) {
    setConfigStatus(error.message, "error");
  }
};

const loadConfig = (messageType, config) => {
  state.selectedMessageType = messageType;
  renderSettingsList();
  elements.messageTypeInput.value = messageType;
  elements.modeSelect.value = config.mode || "constant";
  elements.constantValue.value = config.value ? JSON.stringify(config.value, null, 2) : "";
  elements.incrementalValues.innerHTML = "";
  elements.timeBasedValues.innerHTML = "";
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
  elements.configMeta.textContent = `Current index: ${config.current_index ?? 0} | Start time: ${config.start_time || "not started"}`;
  updateEditorVisibility();
  clearConfigStatus();
  setConfigError("");
};

const updateEditorVisibility = () => {
  const mode = elements.modeSelect.value;
  document.querySelectorAll(".editor-section").forEach((section) => {
    const sectionMode = section.dataset.mode;
    section.style.display = sectionMode === mode ? "block" : "none";
  });
};

const buildConfigPayload = () => {
  const mode = elements.modeSelect.value;
  setConfigError("");

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

const saveConfig = async () => {
  clearConfigStatus();
  if (!state.selectedMessageType) {
    setConfigStatus("Select a message type before saving.", "error");
    return;
  }
  const { payload, error } = buildConfigPayload();
  if (error) {
    setConfigError(error);
    return;
  }
  try {
    const response = await fetch(`/api/settings/${state.selectedMessageType}`, {
      method: "PUT",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(payload),
    });
    if (!response.ok) {
      const text = await response.text();
      throw new Error(text || `Failed to save settings (${response.status}).`);
    }
    setConfigStatus("Configuration saved successfully.", "success");
    await fetchSettings();
    await reloadConfig();
  } catch (saveError) {
    setConfigStatus(saveError.message, "error");
  }
};

const reloadConfig = async () => {
  clearConfigStatus();
  if (!state.selectedMessageType) {
    setConfigStatus("Select a message type to reload.", "error");
    return;
  }
  try {
    const response = await fetch(`/api/settings/${state.selectedMessageType}`);
    if (!response.ok) {
      throw new Error(`Failed to reload (${response.status}).`);
    }
    const config = await response.json();
    loadConfig(state.selectedMessageType, config);
    setConfigStatus("Configuration reloaded.", "success");
  } catch (error) {
    setConfigStatus(error.message, "error");
  }
};

const resetConfigState = async () => {
  clearConfigStatus();
  if (!state.selectedMessageType) {
    setConfigStatus("Select a message type to reset.", "error");
    return;
  }
  try {
    const response = await fetch(`/api/settings/${state.selectedMessageType}/reset`, { method: "POST" });
    if (!response.ok) {
      throw new Error(`Failed to reset (${response.status}).`);
    }
    setConfigStatus("State reset successfully.", "success");
    await reloadConfig();
  } catch (error) {
    setConfigStatus(error.message, "error");
  }
};

const init = () => {
  updateWsStatus("disconnected");
  updatePumpStatus(null);
  updateHexValidation();
  updatePairingValidation();
  updateEditorVisibility();
  fetchSettings();
  fetchDiscoverableState();
  fetchAllowPairingState();

  elements.connectBtn.addEventListener("click", connectWebSocket);
  elements.disconnectBtn.addEventListener("click", disconnectWebSocket);
  elements.getStateBtn.addEventListener("click", () => sendCommand({ command: "getState" }));
  elements.clearLogBtn.addEventListener("click", () => {
    elements.eventLog.innerHTML = "";
  });
  elements.autoReconnectToggle.addEventListener("change", (event) => {
    state.autoReconnect = event.target.checked;
    if (!state.autoReconnect) {
      clearReconnect();
    } else if (!state.ws || state.ws.readyState === WebSocket.CLOSED) {
      scheduleReconnect();
    }
  });
  elements.hexInput.addEventListener("input", updateHexValidation);
  elements.pairingCodeInput.addEventListener("input", updatePairingValidation);
  elements.notifyBtn.addEventListener("click", () => {
    const data = elements.hexInput.value.trim();
    sendCommand({
      command: "notify",
      characteristic: elements.characteristicSelect.value,
      data,
    });
  });
  elements.setCharBtn.addEventListener("click", () => {
    const data = elements.hexInput.value.trim();
    sendCommand({
      command: "setCharacteristic",
      characteristic: elements.characteristicSelect.value,
      data,
    });
  });
  elements.refreshDiscoverableBtn.addEventListener("click", fetchDiscoverableState);
  elements.enableDiscoverableBtn.addEventListener("click", () => {
    setDiscoverableState(true);
  });
  elements.disableDiscoverableBtn.addEventListener("click", () => {
    setDiscoverableState(false);
  });
  elements.refreshAllowPairingBtn.addEventListener("click", fetchAllowPairingState);
  elements.enableAllowPairingBtn.addEventListener("click", () => {
    setAllowPairingState(true);
  });
  elements.disableAllowPairingBtn.addEventListener("click", () => {
    setAllowPairingState(false);
  });
  elements.refreshPairingBtn.addEventListener("click", requestPairingState);
  elements.setPairingBtn.addEventListener("click", () => {
    const { valid, message } = validatePairingCode(elements.pairingCodeInput.value);
    if (!valid) {
      elements.pairingError.textContent = message;
      return;
    }
    elements.pairingError.textContent = "";
    sendCommand({ command: "setPairingCode", pairingCode: elements.pairingCodeInput.value.trim() });
  });
  elements.resetPairingBtn.addEventListener("click", () => {
    sendCommand({ command: "resetPairing" });
  });
  elements.disconnectPumpBtn.addEventListener("click", () => {
    sendCommand({ command: "disconnectPump" });
  });
  elements.refreshSettingsBtn.addEventListener("click", fetchSettings);
  elements.modeSelect.addEventListener("change", updateEditorVisibility);
  elements.addIncrementalBtn.addEventListener("click", () => {
    elements.incrementalValues.append(createIncrementalRow());
  });
  elements.addTimeBasedBtn.addEventListener("click", () => {
    elements.timeBasedValues.append(createTimeBasedRow());
  });
  elements.saveConfigBtn.addEventListener("click", saveConfig);
  elements.reloadConfigBtn.addEventListener("click", reloadConfig);
  elements.resetConfigBtn.addEventListener("click", resetConfigState);
};

window.addEventListener("DOMContentLoaded", init);
