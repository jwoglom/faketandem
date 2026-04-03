# Implementation Plan: Closing Simulator Gaps

Based on the [Gap Analysis](GAP_ANALYSIS.md) comparing faketandem against
pumpX2 (v1.8.8) and controlX2.

## Current State (Post Phase 1 Commit)

Already implemented since the gap analysis was written:
- `CancelBolusRequest` handler (router.go:158)
- `SuspendPumpingRequest` / `ResumePumpingRequest` handlers (pump_control.go)
- `SetTempRateRequest` / `StopTempRateRequest` handlers (pump_control.go)
- `CartridgeHandler` for Enter/Exit cartridge/tubing/cannula modes (cartridge.go)
- `SimpleControlHandler` for DismissNotification, PlaySound, ChangeTimeDate,
  DisconnectPump, UserInteraction, StreamDataPreflight (pump_control.go)
- `IDPSegmentRequest`, `IDPSettingsRequest`, `GetSavedG7PairingCodeRequest`,
  `CurrentActiveIdpValuesRequest` as GenericSettingsHandler entries

**Remaining gaps to address:**

---

## Phase 1: Dynamic Status Responses (Critical)

Several status handlers return static data via GenericSettingsHandler but need
to reflect live pump state for controlX2 compatibility.

### 1A. Dynamic CurrentBolusStatusRequest

**Problem**: controlX2 polls `CurrentBolusStatusRequest` every 1 second during
active bolus. It detects completion when `bolusId` transitions from non-zero
to 0. The current static `{"bolusStatus": 0}` response breaks this flow.

**Files to modify**:
- `pkg/handler/bolus.go` -- Add `CurrentBolusStatusHandler` struct
- `pkg/handler/router.go` -- Replace GenericSettingsHandler on line 106

**Implementation**:
```go
type CurrentBolusStatusHandler struct {
    bridge *pumpx2.Bridge
}

func (h *CurrentBolusStatusHandler) HandleMessage(msg *pumpx2.ParsedMessage, ps *state.PumpState) (*Response, error) {
    cargo := map[string]interface{}{}
    if ps.Bolus.Active {
        cargo["bolusStatus"] = 1
        cargo["bolusId"] = ps.Bolus.BolusID
        cargo["deliveredVolume"] = int(ps.Bolus.UnitsDelivered * 1000)
        cargo["totalVolume"] = int(ps.Bolus.UnitsTotal * 1000)
    } else {
        cargo["bolusStatus"] = 0
        cargo["bolusId"] = 0
    }
    // encode and return
}
```

### 1B. Dynamic CurrentBasalStatusRequest

**Problem**: Should reflect actual basal state (temp basal active/inactive,
rate). Currently returns static `{"basalRate": 850, "tempBasalActive": false}`.

**Files to modify**:
- `pkg/handler/bolus.go` or new `pkg/handler/basal_status.go`
- `pkg/handler/router.go` -- Replace GenericSettingsHandler on line 105

**Implementation**: Read `pumpState.Basal.*` fields, return dynamic response.

### 1C. Dynamic TempRateStatusRequest

**Problem**: Should reflect whether a temp rate is active and its parameters.
Currently static.

**Files to modify**:
- `pkg/handler/router.go` -- Replace GenericSettingsHandler on line 135

### 1D. Dynamic InsulinStatusRequest

**Problem**: Should reflect current reservoir level from `pumpState.Reservoir`.

**Files to modify**:
- `pkg/handler/router.go` -- Replace GenericSettingsHandler on line 102

---

## Phase 2: StateChange Infrastructure Completion

### 2A. Handle StateChangeBasal in Router

**Problem**: `SetTempRateHandler` and `StopTempRateHandler` return
`StateChange{Type: StateChangeBasal}` but `Router.applyStateChange()` has a
`TODO` comment and falls through to default with a warning.

**Files to modify**:
- `pkg/handler/router.go` -- Add `case StateChangeBasal:` in `applyStateChange()`

**Implementation**:
```go
case StateChangeBasal:
    if basalState, ok := change.Data.(*state.BasalState); ok {
        ps.SetBasalState(basalState)
        if r.qeNotifier != nil {
            r.qeNotifier.NotifyBasalRateChange(basalState.CurrentRate, basalState.TempBasalRate)
        }
    }
```

### 2B. Add StateChangeSuspend

**Problem**: Suspend/Resume handlers mutate state directly (line 37/80 of
pump_control.go) instead of using the StateChange pattern. This means no
qualifying event is emitted.

**Files to modify**:
- `pkg/handler/handler.go` -- Add `StateChangeSuspend StateChangeType`
- `pkg/handler/pump_control.go` -- Return StateChange instead of direct mutation
- `pkg/handler/router.go` -- Handle StateChangeSuspend, emit QE notification

### 2C. Handle StateChangeReservoir, StateChangeBattery, StateChangeAlert

**Problem**: Constants exist in handler.go but none are handled in
`applyStateChange()`.

**Files to modify**:
- `pkg/handler/router.go` -- Add cases for remaining state change types

---

## Phase 3: Qualifying Events Bitmask System

### 3A. Define Bitmask Constants

**Problem**: pumpX2 uses numeric bitmask IDs (ALERT=1, ALARM=2, REMINDER=4,
etc.) for qualifying events. faketandem sends custom named events.

**Files to create**:
- `pkg/handler/qualifying_event_types.go`

**Implementation**:
```go
const (
    QEAlert                     uint32 = 1
    QEAlarm                     uint32 = 2
    QEReminder                  uint32 = 4
    QEMalfunction               uint32 = 8
    QECGMAlert                  uint32 = 16
    QEHomeScreenChange          uint32 = 32
    QEPumpSuspend               uint32 = 64
    QEPumpResume                uint32 = 128
    QETimeChange                uint32 = 256
    QEBasalChange               uint32 = 512
    QEBolusChange               uint32 = 1024
    QEIOBChange                 uint32 = 2048
    QEExtendedBolusChange       uint32 = 4096
    QEProfileChange             uint32 = 8192
    QEBG                        uint32 = 16384
    QECGMChange                 uint32 = 32768
    QEBattery                   uint32 = 65536
    QEBasalIQ                   uint32 = 131072
    QERemainingInsulin          uint32 = 262144
    QEPumpCommSuspended         uint32 = 524288
    QEActiveProfileSegChange    uint32 = 1048576
    QEBasalIQStatus             uint32 = 2097152
    QEControlIQInfo             uint32 = 4194304
    QEControlIQSleep            uint32 = 8388608
    QEGlobalPumpSettings        uint32 = 16777216
    QESnoozeStatus              uint32 = 33554432
    QEPumpingStatus             uint32 = 67108864
    QEPumpReset                 uint32 = 134217728
    QEHeartbeat                 uint32 = 268435456
    QEBolusPermissionRevoked    uint32 = 2147483648
)
```

### 3B. Refactor QualifyingEventsNotifier

**Files to modify**:
- `pkg/handler/qualifying_events.go` -- Send proper bitmask-based notifications

**Implementation**: Each `Notify*` method should encode a qualifying events
response with `{"bitmap": <combined_bitmask>}` on CharQualifyingEvents.

### 3C. Add Missing Notify Methods

Add methods for all 30 event types to `QualifyingEventsNotifier` and the
`EventNotifier` interface. Wire them into appropriate state mutation points:
- `NotifyIOBChange` -- from simulator when IOB changes
- `NotifyBattery` -- from simulator battery drain
- `NotifyHeartbeat` -- periodic timer (every 60s)
- `NotifyCGMChange` -- when EGV updates
- `NotifyHomeScreenChange` -- on significant state changes

---

## Phase 4: Settings Write Handlers

### 4A. Generic SettingsWriteHandler Pattern

**Files to create**:
- `pkg/handler/settings_write.go`

**Implementation**: A handler that extracts value from cargo, updates the
settings.Manager constant (so subsequent reads reflect the write), updates
pump state, and returns `{"status": 0}`.

```go
type SettingsWriteHandler struct {
    bridge          *pumpx2.Bridge
    settingsManager *settings.Manager
    msgType         string
    responseType    string
    stateUpdater    func(*state.PumpState, map[string]interface{})
    settingsKey     string  // which GenericSettings response to update
}
```

### 4B. Register Settings Writers

| Handler | Settings Key to Update | State Field |
|---|---|---|
| `SetMaxBolusLimitRequest` | `GlobalMaxBolusSettingsRequest` | `PumpState.MaxBolusLimit` |
| `SetMaxBasalLimitRequest` | `BasalLimitSettingsRequest` | `PumpState.MaxBasalLimit` |
| `SetSleepScheduleRequest` | `ControlIQSleepScheduleRequest` | `PumpState.SleepSchedule` |
| `SetQuickBolusSettingsRequest` | (new) | `PumpState.QuickBolusSettings` |
| `SetPumpSoundsRequest` | (new) | N/A (log only) |
| `SetPumpAlertSnoozeRequest` | (new) | N/A (log only) |
| `SetAutoOffAlertRequest` | (new) | N/A (log only) |
| `SetBgReminderRequest` | (new) | N/A (log only) |
| `SetSiteChangeReminderRequest` | (new) | N/A (log only) |
| `SetMissedMealBolusReminderRequest` | (new) | N/A (log only) |
| `SetLowInsulinAlertRequest` | (new) | N/A (log only) |

Lower-priority settings (Sounds, Snooze, Reminders) can use `SimpleControlHandler`
since controlX2 doesn't read-back those values.

### 4C. SetModesRequest / ChangeControlIQSettingsRequest

**Files to create**:
- `pkg/handler/modes.go`

**State changes**:
- Add `ControlIQMode int` to PumpState (Normal=0, Sleep=1, Exercise=2)
- `SetModesRequest` updates mode and emits QEControlIQInfo
- `ChangeControlIQSettingsRequest` updates ControlIQ parameters

### 4D. IDP Management Handlers

**Files to create**:
- `pkg/handler/idp.go`

**State changes**:
- Add `IDPs []*InsulinDeliveryProgram` and `ActiveIDPIndex int` to PumpState
- `CreateIDPRequest` -- append new IDP, return `{"status": 0, "idpId": <id>}`
- `SetIDPSettingsRequest` / `SetIDPSegmentRequest` -- find by ID, update
- `SetActiveIDPRequest` -- set ActiveIDPIndex, emit QEProfileChange
- `DeleteIDPRequest` / `RenameIDPRequest` -- find by ID, mutate

### 4E. CGM Configuration Handlers

**Files to create**:
- `pkg/handler/cgm.go`

**State changes**:
- Add `G7PairingCode`, `G6TransmitterID` to CGMState
- `SetDexcomG7PairingCodeRequest` / `SetG6TransmitterIdRequest` -- store value
- `StartDexcomG6SensorSessionRequest` -- set SessionActive=true, emit QECGMChange
- `StopDexcomCGMSensorSessionRequest` -- set SessionActive=false

---

## Phase 5: Missing Status Query Variants

These are all straightforward GenericSettingsHandler additions.

**Files to modify**:
- `pkg/handler/router.go` -- Register new handlers
- `pkg/settings/defaults.go` -- Add default response configs

| Message | Default Response |
|---|---|
| `CurrentBatteryV1Request` | `{"batteryLevelPercent": 85, "isCharging": false}` |
| `PumpFeaturesV1Request` | Copy from PumpFeaturesV2 |
| `LastBolusStatusRequest` (V1) | `{"lastBolusStatus": 0}` |
| `CgmStatusV2Request` | Copy from CGMStatusRequest |
| `CurrentEgvGuiDataV2Request` | Copy from CurrentEGVGuiDataRequest |
| `PumpVersionBRequest` | Copy from PumpVersionRequest |
| `CGMHardwareInfoRequest` | `{"transmitterBattery": 100}` |
| `CGMGlucoseAlertSettingsRequest` | `{"highAlert": 250, "lowAlert": 70}` |
| `CGMOORAlertSettingsRequest` | `{"enabled": false}` |
| `CGMRateAlertSettingsRequest` | `{"riseEnabled": false, "fallEnabled": false}` |
| `BasalIQAlertInfoRequest` | `{"alertActive": false}` |
| `RemindersRequest` | `{"reminders": []}` |
| `QuickBolusSettingsRequest` | `{"enabled": false}` |
| `SecretMenuRequest` | `{"enabled": false}` |
| `CgmSupportPackageStatusRequest` | `{"status": 0}` |
| `UnknownMobiOpcode110Request` | `{"status": 0}` |

---

## Phase 6: Cartridge/Tubing State + CONTROL_STREAM

### 6A. Enhance CartridgeHandler with State

**Files to modify**:
- `pkg/handler/cartridge.go` -- Add state mutations
- `pkg/state/pump.go` -- Add `CartridgeMode`, `FillTubingMode` booleans

**Implementation**: Each cartridge handler updates the appropriate state flag
and emits a qualifying event.

### 6B. CONTROL_STREAM Notifications

**Files to create**:
- `pkg/handler/control_stream.go`

**Implementation**: `ControlStreamNotifier` sends pump-initiated streaming
responses on CharControlStream during multi-step operations. These use
**signed negative opcodes** (-23, -29). Triggered by cartridge handlers:

- `EnterChangeCartridgeModeRequest` -> stream `DetectingCartridgeStateStream`
- `EnterFillTubingModeRequest` -> stream `FillTubingStateStream`
- `FillCannulaRequest` -> stream `FillCannulaStateStream`

Lower priority -- only needed for full cartridge workflow simulation.

---

## Phase 7: History Log Event Types

### 7A. Define Event Type Constants

**Files to create**:
- `pkg/state/history_types.go`

Match pumpX2's 132 history log type IDs. Start with the most important:

| Type | ID | Generated When |
|---|---|---|
| `BolusRequestedMsg1` | (from pumpX2) | Bolus initiated |
| `BolusCompletedHistoryLog` | (from pumpX2) | Bolus complete |
| `BasalRateChangeHistoryLog` | (from pumpX2) | Basal rate change |
| `TempRateActivatedHistoryLog` | (from pumpX2) | Temp rate set |
| `TempRateCompletedHistoryLog` | (from pumpX2) | Temp rate expired/stopped |
| `PumpingSuspendedHistoryLog` | (from pumpX2) | Pump suspended |
| `PumpingResumedHistoryLog` | (from pumpX2) | Pump resumed |
| `NewDayHistoryLog` | (from pumpX2) | Midnight rollover |
| `CartridgeInsertedHistoryLog` | (from pumpX2) | Cartridge change |

### 7B. Wire History Generation into State Changes

**Files to modify**:
- `pkg/handler/router.go` -- In `applyStateChange()`, add history entries
- `pkg/state/simulator.go` -- Generate periodic entries (NewDay, DailyBasal)

Every handler that changes state should produce the corresponding history
entry. controlX2's `HistoryLogFetcher` fetches in chunks of 32 with 4-second
timeout, so entries must be retrievable within that window.

---

## Phase 8: Full Qualifying Events Wiring

### 8A. Extend EventNotifier Interface

**Files to modify**:
- `pkg/state/events.go` -- Add methods for all 30 QE types
- Update `NoOpEventNotifier` with stubs

### 8B. Wire Into State Mutations

| QE Type | Trigger Point |
|---|---|
| `QEIOBChange` | Simulator `updateBasalDelivery()` when IOB changes >0.1 |
| `QEBattery` | Simulator `updateBattery()` at thresholds (20%, 10%) |
| `QEHeartbeat` | Periodic timer every 60 seconds |
| `QERemainingInsulin` | Simulator when reservoir drops below threshold |
| `QECGMChange` | When CGM EGV value updates |
| `QEHomeScreenChange` | On bolus/basal/alert state changes |
| `QEControlIQInfo` | On mode changes |
| `QEPumpingStatus` | On suspend/resume |

---

## Dependency Graph

```
Phase 1 (dynamic status) ─────────────────────────────┐
Phase 2 (state change infra) ──┐                       │
                                ├─> Phase 4 (writes)    ├─> Phase 7 (history)
Phase 3 (QE bitmasks) ─────────┤                       │
                                ├─> Phase 6 (cartridge) ├─> Phase 8 (full QE)
Phase 5 (status variants) ─────┘
```

Phases 1, 2, 3, 5 can be worked in parallel. Phase 4 depends on 2.
Phases 6-8 depend on all earlier phases.

---

## New Files Summary

| File | Phase | Purpose |
|---|---|---|
| `pkg/handler/basal_status.go` | 1B, 1C | Dynamic basal/temp rate status |
| `pkg/handler/qualifying_event_types.go` | 3A | Bitmask constants |
| `pkg/handler/settings_write.go` | 4A | Generic settings write pattern |
| `pkg/handler/modes.go` | 4C | SetModes, ChangeControlIQSettings |
| `pkg/handler/idp.go` | 4D | IDP CRUD handlers |
| `pkg/handler/cgm.go` | 4E | CGM config handlers |
| `pkg/handler/control_stream.go` | 6B | CONTROL_STREAM notifier |
| `pkg/state/history_types.go` | 7A | History log type constants |

## Modified Files Summary

| File | Phases | Key Changes |
|---|---|---|
| `pkg/handler/handler.go` | 2B | Add StateChangeSuspend constant |
| `pkg/handler/router.go` | 1, 2, 4, 5 | Replace static handlers, complete applyStateChange |
| `pkg/handler/bolus.go` | 1A | Add CurrentBolusStatusHandler |
| `pkg/handler/pump_control.go` | 2B | Use StateChange pattern for suspend/resume |
| `pkg/handler/qualifying_events.go` | 3B, 3C, 8 | Bitmask-based notifications |
| `pkg/state/pump.go` | 4 | Add ControlIQMode, IDPs, CGM fields |
| `pkg/state/events.go` | 8A | Extend EventNotifier interface |
| `pkg/state/simulator.go` | 7B, 8B | History entries, QE wiring |
| `pkg/settings/defaults.go` | 5 | V1/V2 variant defaults |

---

## Testing Strategy

**Unit tests** (per phase):
- Phase 1: `CurrentBolusStatusHandler` returns dynamic state matching `pumpState.Bolus`
- Phase 2: `applyStateChange(StateChangeBasal)` updates pump state and emits QE
- Phase 3: Qualifying event notifications use correct bitmask values
- Phase 4: Settings writes update both pump state and settings manager read responses
- Phase 7: History entries generated with correct type IDs

**Integration tests**:
- Full bolus flow: Initiate -> poll CurrentBolusStatus -> Cancel -> verify status=0
- Temp basal flow: SetTempRate -> poll TempRateStatus -> StopTempRate -> verify
- Suspend/Resume: Suspend -> verify pumping stopped -> Resume -> verify resumed

**Run after each phase**:
```bash
golangci-lint run --fix --timeout=5m
go test -v ./...
```
