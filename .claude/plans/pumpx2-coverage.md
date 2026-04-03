# pumpX2/controlX2 Coverage Gap Analysis & Implementation Plan

## Overview

Comparison of faketandem's handler coverage against pumpX2's message catalog (130+ message types) and controlX2's actual usage (~65 unique message types). This plan addresses gaps that would cause controlX2 to fail or hang when connected to the simulator.

---

## Phase 1: Critical controlX2 Compatibility

These are messages controlX2 actively sends that faketandem has **no handler** for.
Without these, basic controlX2 flows will break.

### 1a. CancelBolusRequest handler

controlX2 uses `CancelBolusRequest` (not `BolusTerminationRequest`) to cancel boluses.

- **File:** `pkg/handler/bolus.go`
- **Action:** Add `CancelBolusHandler` struct mirroring `BolusTerminationHandler`
- **Request:** `CancelBolusRequest` -> `CancelBolusResponse`
- **Cargo:** Read `bolusId` from request cargo
- **State change:** Set `Bolus.Active = false`, same as BolusTerminationHandler
- **Register in:** `pkg/handler/router.go`

### 1b. Pump suspend/resume handlers

controlX2 sends these from the Actions screen and as prerequisite for cartridge changes.

- **File:** `pkg/handler/pump_control.go` (new)
- **Handlers:**
  1. `SuspendPumpingRequest` -> `SuspendPumpingResponse` (status: 0)
     - State change: set a new `PumpingSuspended bool` field on PumpState
  2. `ResumePumpingRequest` -> `ResumePumpingResponse` (status: 0)
     - State change: set `PumpingSuspended = false`
- **State change in:** `pkg/state/pump.go` — add `PumpingSuspended bool` field to PumpState, initialize to `false`
- **Register in:** `pkg/handler/router.go`

### 1c. Cartridge change flow handlers

controlX2 has a full cartridge change wizard that sends these in sequence.

- **File:** `pkg/handler/cartridge.go` (new)
- **Handlers** (all return `{status: 0}` success response):
  1. `EnterChangeCartridgeModeRequest` -> `EnterChangeCartridgeModeResponse`
  2. `ExitChangeCartridgeModeRequest` -> `ExitChangeCartridgeModeResponse`
  3. `EnterFillTubingModeRequest` -> `EnterFillTubingModeResponse`
  4. `ExitFillTubingModeRequest` -> `ExitFillTubingModeResponse`
  5. `FillCannulaRequest` -> `FillCannulaResponse`
- **Register in:** `pkg/handler/router.go`

### 1d. Temp rate control handlers

controlX2 TempRateWindow sends these to set/stop temp basal rates.

- **File:** `pkg/handler/pump_control.go` (same new file from 1b)
- **Handlers:**
  1. `SetTempRateRequest` -> `SetTempRateResponse` (status: 0)
     - Read `percentage` and `duration` from cargo
     - State change: update `Basal.TempBasalActive`, `TempBasalRate`, `TempBasalEnd`
  2. `StopTempRateRequest` -> `StopTempRateResponse` (status: 0)
     - State change: set `Basal.TempBasalActive = false`
- **Register in:** `pkg/handler/router.go`

### 1e. Missing status handlers

controlX2 requests these on various screens, currently unhandled.

- **File:** `pkg/settings/defaults.go` — add registerConstant entries
- **File:** `pkg/handler/router.go` — add GenericSettingsHandler registrations
- **New handlers (all via GenericSettingsHandler):**
  1. `IDPSegmentRequest` — return empty/default segment data `{status: 0}`
  2. `IDPSettingsRequest` — return default IDP settings
  3. `GetSavedG7PairingCodeRequest` — return `{pairingCode: ""}`
  4. `CurrentActiveIdpValuesRequest` — return default active values

### 1f. Other control commands controlX2 uses

- **File:** `pkg/handler/pump_control.go`
- **Handlers** (simple success responses):
  1. `DismissNotificationRequest` -> `DismissNotificationResponse` (status: 0)
  2. `PlaySoundRequest` -> `PlaySoundResponse` (status: 0)
  3. `ChangeTimeDateRequest` -> `ChangeTimeDateResponse` (status: 0)
  4. `DisconnectPumpRequest` -> `DisconnectPumpResponse` (status: 0)
  5. `UserInteractionRequest` -> `UserInteractionResponse` (status: 0)

### Files modified in Phase 1:
- `pkg/handler/bolus.go` — add CancelBolusHandler
- `pkg/handler/pump_control.go` — **new file** with suspend/resume, temp rate, misc control handlers
- `pkg/handler/cartridge.go` — **new file** with cartridge change handlers
- `pkg/handler/router.go` — register all new handlers
- `pkg/settings/defaults.go` — add default responses for new status handlers
- `pkg/state/pump.go` — add `PumpingSuspended bool` field

### Verification:
```bash
golangci-lint run --fix --timeout=5m
golangci-lint run --timeout=5m
go test -v ./...
```

---

## Phase 2: Settings Write Handlers

controlX2's settings screens send write commands to modify pump configuration.

### 2a. ControlIQ/Mode settings

- **File:** `pkg/handler/settings_write.go` (new)
- **Handlers** (accept input, log it, return success):
  1. `ChangeControlIQSettingsRequest` -> `ChangeControlIQSettingsResponse`
  2. `SetModesRequest` -> `SetModesResponse`
  3. `SetSleepScheduleRequest` -> `SetSleepScheduleResponse`

### 2b. Safety limit settings

- **Handlers:**
  1. `SetMaxBolusLimitRequest` -> `SetMaxBolusLimitResponse`
  2. `SetMaxBasalLimitRequest` -> `SetMaxBasalLimitResponse`

### 2c. IDP (Insulin Delivery Profile) management

- **Handlers:**
  1. `SetActiveIDPRequest` -> `SetActiveIDPResponse`
  2. `SetIDPSegmentRequest` -> `SetIDPSegmentResponse`
  3. `SetIDPSettingsRequest` -> `SetIDPSettingsResponse`
  4. `CreateIDPRequest` -> `CreateIDPResponse`
  5. `DeleteIDPRequest` -> `DeleteIDPResponse`
  6. `RenameIDPRequest` -> `RenameIDPResponse`

### 2d. Alert/reminder/sound settings

- **Handlers:**
  1. `SetQuickBolusSettingsRequest` -> `SetQuickBolusSettingsResponse`
  2. `SetPumpSoundsRequest` -> `SetPumpSoundsResponse`
  3. `SetPumpAlertSnoozeRequest` -> `SetPumpAlertSnoozeResponse`

### 2e. CGM control

- **Handlers:**
  1. `StartDexcomG6SensorSessionRequest` -> `StartDexcomG6SensorSessionResponse`
  2. `StopDexcomCGMSensorSessionRequest` -> `StopDexcomCGMSensorSessionResponse`
  3. `SetG6TransmitterIdRequest` -> `SetG6TransmitterIdResponse`
  4. `SetDexcomG7PairingCodeRequest` -> `SetDexcomG7PairingCodeResponse`

### Pattern for all Phase 2 handlers:
```go
type SimpleControlHandler struct {
    bridge      *pumpx2.Bridge
    msgType     string
    responseType string
}
func (h *SimpleControlHandler) HandleMessage(msg *pumpx2.ParsedMessage, pumpState *state.PumpState) (*Response, error) {
    log.Infof("Handling %s: txID=%d cargo=%v", h.msgType, msg.TxID, msg.Cargo)
    response, err := h.bridge.EncodeMessage(msg.TxID, h.responseType, map[string]interface{}{"status": 0})
    if err != nil { return nil, err }
    return &Response{ResponseMessage: response, Immediate: true}, nil
}
```

### Files modified in Phase 2:
- `pkg/handler/settings_write.go` — **new file** with SimpleControlHandler + registrations
- `pkg/handler/router.go` — register all new handlers

---

## Phase 3: ControlStream & Error Handling

### 3a. ControlStream state responses

controlX2 subscribes to ControlStream characteristic for cartridge change state machines.
These are **unsolicited push notifications** from the pump, not request/response pairs.

- **File:** `pkg/handler/control_stream.go` (new)
- **Responses to generate** (triggered by cartridge mode handlers):
  1. `PumpingStateStreamResponse` — sent when entering/exiting pumping state
  2. `DetectingCartridgeStateStreamResponse`
  3. `EnterChangeCartridgeModeStateStreamResponse`
  4. `LoadCartridgeStateStreamResponse`
  5. `FillTubingStateStreamResponse`
  6. `FillCannulaStateStreamResponse`
  7. `ExitFillTubingModeStateStreamResponse`
  8. `PrimeNudgeStateStreamResponse`
- **Integration:** Cartridge handlers from Phase 1c trigger these as notifications

### 3b. ErrorResponse generation

- **File:** `pkg/handler/error_response.go` (new)
- Add ability for handlers to return error responses when state is invalid:
  - Bolus while suspended -> error
  - CancelBolus with no active bolus -> error
  - Temp rate while suspended -> error
- Use pumpX2's `ErrorResponse` message type

### 3c. StreamDataPreflightRequest handler

Note: faketandem has `StreamDataReadinessRequest` but NOT `StreamDataPreflightRequest`.
These are different messages (different opcodes).

- **File:** `pkg/handler/control.go`
- Add `StreamDataPreflightRequest` -> `StreamDataPreflightResponse` handler

### Files modified in Phase 3:
- `pkg/handler/control_stream.go` — **new file**
- `pkg/handler/error_response.go` — **new file**
- `pkg/handler/control.go` — add StreamDataPreflight handler
- `pkg/handler/cartridge.go` — integrate ControlStream notifications
- `pkg/handler/router.go` — register new handlers

---

## Phase 4: Qualifying Events & Remaining Status Handlers

### 4a. Expand qualifying events

pumpX2 defines 28 qualifying event types as a bitmask. faketandem currently simulates ~10.

- **File:** `pkg/handler/qualifying_events.go`
- **Add missing event types and their trigger conditions:**
  - `REMINDER` (4), `MALFUNCTION` (8), `CGM_ALERT` (16)
  - `TIME_CHANGE` (256), `IOB_CHANGE` (2048), `EXTENDED_BOLUS_CHANGE` (4096)
  - `PROFILE_CHANGE` (8192), `BG` (16384), `CGM_CHANGE` (32768)
  - `BATTERY` (65536), `BASAL_IQ` (131072), `REMAINING_INSULIN` (262144)
  - `PUMP_COMMUNICATIONS_SUSPENDED` (524288)
  - `ACTIVE_PROFILE_SEGMENT_CHANGE` (1048576), `BASAL_IQ_STATUS` (2097152)
  - `CONTROL_IQ_INFO` (4194304), `CONTROL_IQ_SLEEP` (8388608)
  - `GLOBAL_PUMP_SETTINGS` (16777216), `SNOOZE_STATUS` (33554432)
  - `PUMPING_STATUS` (67108864), `PUMP_RESET` (134217728)
  - `HEARTBEAT` (268435456), `BOLUS_PERMISSION_REVOKED` (2147483648)

### 4b. Remaining currentStatus handlers

Messages that exist in pumpX2 but faketandem doesn't handle yet.
Lower priority since controlX2 may not actively use all of them.

- **File:** `pkg/settings/defaults.go` + `pkg/handler/router.go`
- **Add via GenericSettingsHandler:**
  - `CGMGlucoseAlertSettingsRequest`
  - `CGMOORAlertSettingsRequest`
  - `CGMRateAlertSettingsRequest`
  - `CGMHardwareInfoRequest`
  - `CgmStatusV2Request`
  - `CgmSupportPackageStatusRequest`
  - `CurrentEgvGuiDataV2Request`
  - `CurrentBatteryV1Request`
  - `BasalIQAlertInfoRequest`
  - `PumpFeaturesV1Request`
  - `PumpVersionBRequest`
  - `LastBolusStatusRequest` (V1)
  - `RemindersRequest`

### 4c. Expand history log entry types

- **File:** `pkg/state/history.go` or `simulator.go`
- Generate richer history entries matching pumpX2's 130+ types
- Priority types: BasalRateChange, TempRateActivated/Completed, CartridgeFilled, CannulaFilled, PumpingSuspended/Resumed, AlertActivated/Cleared

### Files modified in Phase 4:
- `pkg/handler/qualifying_events.go` — expand event types
- `pkg/settings/defaults.go` — add default responses
- `pkg/handler/router.go` — register new handlers
- `pkg/state/simulator.go` — trigger new event types

---

## Current Handler Inventory (for reference)

### Dedicated handlers in faketandem (21):
1. ApiVersionRequest
2. TimeSinceResetRequest
3. CentralChallengeRequest
4. PumpChallengeRequest
5. JPAKERound1-4Request (4 handlers)
6. CurrentStatusRequest
7. HistoryLogRequest
8. CreateHistoryLogRequest
9. HistoryLogStatusRequest
10. BolusPermissionRequest
11. BolusCalcDataSnapshotRequest
12. InitiateBolusRequest
13. BolusTerminationRequest
14. RemoteBgEntryRequest
15. RemoteCarbEntryRequest
16. BolusPermissionReleaseRequest
17. SetSensorTypeRequest
18. StreamDataReadinessRequest
19. FactoryResetBRequest
20. ProfileBasalRequest

### GenericSettingsHandler entries (44):
BasalIQSettingsRequest, ControlIQSettingsRequest, PumpGlobalsRequest,
TherapySettingsGlobalsRequest, ControlIQGlobalsRequest, CurrentBatteryV2Request,
ControlIQIOBRequest, InsulinStatusRequest, CurrentBasalStatusRequest,
CurrentBolusStatusRequest, CurrentEGVGuiDataRequest, HomeScreenMirrorRequest,
CGMStatusRequest, AlertStatusRequest, AlarmStatusRequest, LoadStatusRequest,
ProfileStatusRequest, LastBolusStatusV2Request, HighestAamRequest,
ActiveAamBitsRequest, MalfunctionStatusRequest, ReminderStatusRequest,
CGMAlertStatusRequest, ControlIQInfoV1Request, ControlIQInfoV2Request,
ControlIQSleepScheduleRequest, BasalIQStatusRequest, NonControlIQIOBRequest,
ExtendedBolusStatusRequest, ExtendedBolusStatusV2Request, LastBolusStatusV3Request,
TempRateRequest, TempRateStatusRequest, LastBGRequest,
BolusPermissionChangeReasonRequest, GlobalMaxBolusSettingsRequest,
BasalLimitSettingsRequest, LocalizationRequest, PumpSettingsRequest,
SendTipsControlGenericTestRequest, PumpFeaturesV2Request, PumpVersionRequest,
BleSoftwareInfoRequest, CommonSoftwareInfoRequest

### controlX2 message types NOT handled by faketandem (Phase 1+2 gaps):
CancelBolusRequest, SuspendPumpingRequest, ResumePumpingRequest,
EnterChangeCartridgeModeRequest, ExitChangeCartridgeModeRequest,
EnterFillTubingModeRequest, ExitFillTubingModeRequest, FillCannulaRequest,
SetTempRateRequest, StopTempRateRequest, DismissNotificationRequest,
PlaySoundRequest, ChangeTimeDateRequest, DisconnectPumpRequest,
ChangeControlIQSettingsRequest, SetModesRequest, SetSleepScheduleRequest,
SetMaxBolusLimitRequest, SetMaxBasalLimitRequest, SetActiveIDPRequest,
SetIDPSegmentRequest, SetIDPSettingsRequest, CreateIDPRequest, DeleteIDPRequest,
RenameIDPRequest, SetQuickBolusSettingsRequest, SetPumpSoundsRequest,
SetPumpAlertSnoozeRequest, StartDexcomG6SensorSessionRequest,
StopDexcomCGMSensorSessionRequest, SetG6TransmitterIdRequest,
SetDexcomG7PairingCodeRequest, IDPSegmentRequest, IDPSettingsRequest,
GetSavedG7PairingCodeRequest, StreamDataPreflightRequest
