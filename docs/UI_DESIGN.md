# Pumpx2 Web UI Design TODO

## Overview
Build a basic HTML + JavaScript interface that connects to the pump emulator WebSocket API and the REST settings API, allowing end users to observe events and configure virtual pump settings.

## TODO Checklist with Completion Instructions

### 1. Create the HTML shell + layout
- [ ] Add a single-page HTML document with a header + two-column layout.
  - **Header**: app title, WebSocket status pill (connected/disconnected), pump connection state, buttons for Connect/Disconnect/Get State.
  - **Left column**: Live event log + WebSocket command panel.
  - **Right column**: Settings list + editor panel.
- **Completion criteria**:
  - `index.html` loads without errors in a browser.
  - Layout shows three regions (header, left column, right column) with readable spacing and simple styling.

### 2. Implement WebSocket connection + event log
- [ ] Connect to `ws://<host>:8080/ws` (or `wss` if the page is served over HTTPS).
- [ ] On open, send `{"command":"getState"}` and update UI status pill.
- [ ] On message:
  - If payload includes `connected`, treat it as PumpState and update pump connection status.
  - Otherwise treat it as BleEvent (`type`, `characteristic`, `data`) and append to log.
- [ ] Provide reconnect logic (exponential backoff) with a “stop reconnecting” toggle.
- **Completion criteria**:
  - WebSocket connection status changes in the UI.
  - Event log updates with type/characteristic/data and timestamps.
  - Reconnect attempts are visible in UI (e.g., “Reconnecting in 4s”).

### 3. Add WebSocket command panel
- [ ] Characteristic dropdown: `CurrentStatus`, `QualifyingEvents`, `HistoryLog`, `Authorization`, `Control`, `ControlStream`.
- [ ] Hex input field with validation (only even-length hex).
- [ ] Buttons:
  - “Notify” sends `{"command":"notify","characteristic":"<Name>","data":"<hex>"}`.
  - “Set Characteristic” sends `{"command":"setCharacteristic","characteristic":"<Name>","data":"<hex>"}`.
- **Completion criteria**:
  - Invalid hex prevents send and displays error.
  - Valid commands are sent over the active WebSocket.

### 4. Fetch settings list + list view
- [ ] On page load (and on demand), call `GET /api/settings`.
- [ ] Render a list of message types with a compact JSON preview for each config.
- [ ] Clicking a message type loads that configuration into the editor panel.
- **Completion criteria**:
  - Settings list is visible with correct message types.
  - Selecting a message type loads data into the editor.

### 5. Build settings editor with mode-aware controls
- [ ] Mode selector: `constant`, `incremental`, `time_based`.
- [ ] Mode-specific editing:
  - `constant`: JSON editor for `value` object.
  - `incremental`: array editor for `values` (add/remove entries).
  - `time_based`: array editor for `values` + `timing_seconds` list (same length).
- [ ] Read-only fields (if displayed): `current_index`, `start_time`.
- **Completion criteria**:
  - Mode changes update the visible editor sections.
  - Client-side validation prevents invalid configs:
    - constant: requires `value`
    - incremental: requires at least 1 entry in `values`
    - time_based: `values` length matches `timing_seconds`, timings ascending

### 6. Save + reset configuration actions
- [ ] “Save” button issues `PUT /api/settings/{messageType}` with a `ResponseConfig` JSON body.
- [ ] “Reset State” button issues `POST /api/settings/{messageType}/reset`.
- [ ] “Reload” button re-fetches the config from the server.
- **Completion criteria**:
  - Success and error messages shown in UI.
  - Save updates are reflected in the list view after reload.

### 7. CSS polish + accessibility
- [ ] Basic styling for cards, headers, buttons, status pills.
- [ ] Use color + text for status (don’t rely on color alone).
- [ ] Ensure controls are keyboard-navigable and labels are associated with inputs.
- **Completion criteria**:
  - Page is readable without external CSS frameworks.
  - Inputs have labels and focus styles.

### 8. Optional: Serve UI from the API server
- [ ] Add a static file handler to serve the UI at `/ui/`.
- [ ] Ensure WebSocket path remains `/ws` and REST endpoints stay under `/api`.
- **Completion criteria**:
  - Visiting `/ui/` loads the UI without manual file hosting.

## Implementation Notes (reference API behavior)
- WebSocket commands supported:
  - `getState`, `notify`, `setCharacteristic`.
- BleEvent payloads include `type`, optional `characteristic`, optional hex `data`.
- REST settings endpoints:
  - `GET /api/settings`
  - `GET /api/settings/{messageType}`
  - `PUT /api/settings/{messageType}`
  - `POST /api/settings/{messageType}/reset`

## Done Definition
The UI is considered complete when a user can:
1. Connect/disconnect WebSocket and see live events.
2. Issue notify/setCharacteristic commands with valid hex input.
3. Browse and edit any settings config, save it, and reset state.
4. Understand current status at a glance from the header and logs.
