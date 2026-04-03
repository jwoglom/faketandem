# Gap Analysis: faketandem Simulator vs pumpX2 / controlX2

Generated: 2026-04-03

This document identifies gaps between the faketandem simulator and the
client implementations in [pumpX2](https://github.com/jwoglom/pumpX2) (Java BLE
protocol library) and [controlX2](https://github.com/jwoglom/controlX2)
(Android/Wear OS app).

---

## Summary

| Category | pumpX2 total | faketandem handled | Missing |
|---|---|---|---|
| CURRENT_STATUS requests | ~50 | ~40 | ~10 |
| CONTROL requests | ~50 | ~10 | ~40 |
| AUTHORIZATION requests | 7 | 7 | 0 |
| CONTROL_STREAM requests | 8 (Nonexistent*) | 0 | 8 |
| HISTORY_LOG stream | 1 | 1 | 0 |
| History log event types | 132 | basic only | ~130 |
| Qualifying event types | 30 | ~10 | ~20 |

\* CONTROL_STREAM "requests" are labeled `Nonexistent*` in pumpX2 because
  they are pump-initiated streaming responses with no client request.

**Overall: faketandem handles ~67 of ~138 request message types (49%).**
The biggest gap is in CONTROL characteristic messages (settings writes, pump
operations) where only 10 of ~50 are handled.

---

## 1. Missing CURRENT_STATUS Messages (Read-Only Status Queries)

These are status queries that pumpX2 defines but faketandem has no handler for.
They would be straightforward to add as GenericSettingsHandler entries.

| Message | Used by controlX2? | Priority | Notes |
|---|---|---|---|
| `CurrentBatteryV1Request` | Yes (via builder) | Medium | Older API version variant |
| `PumpVersionBRequest` | No | Low | Variant of PumpVersionRequest |
| `CgmStatusV2Request` | Possible | Medium | V2 of CGMStatusRequest |
| `CurrentEgvGuiDataV2Request` | Possible | Medium | V2 of CurrentEGVGuiDataRequest |
| `LastBolusStatusRequest` (V1) | Yes (via builder) | Medium | Older variant |
| `PumpFeaturesV1Request` | No | Low | Older variant |
| `CGMHardwareInfoRequest` | No | Low | Dexcom hardware info |
| `CGMGlucoseAlertSettingsRequest` | No | Low | CGM glucose alert thresholds |
| `CGMOORAlertSettingsRequest` | No | Low | CGM out-of-range alert settings |
| `CGMRateAlertSettingsRequest` | No | Low | CGM rate-of-change alert settings |
| `CgmSupportPackageStatusRequest` | No | Low | CGM support package |
| `BasalIQAlertInfoRequest` | No | Low | Basal IQ alert details |
| `IDPSegmentRequest` | Yes | High | Insulin delivery program segments |
| `IDPSettingsRequest` | Yes | High | Insulin delivery program settings |
| `CurrentActiveIdpValuesRequest` | No | Medium | Active IDP values |
| `RemindersRequest` | No | Low | Reminder list |
| `QuickBolusSettingsRequest` | No | Low | Quick bolus config |
| `SecretMenuRequest` | No | Low | Debug/service menu |
| `GetG6TransmitterHardwareInfoRequest` | No | Low | G6 transmitter info |
| `GetSavedG7PairingCodeRequest` | Yes | Medium | G7 pairing code |
| `UnknownMobiOpcode110Request` | No | Low | Unknown Mobi-specific |

### Recommendation
Add IDP (Insulin Delivery Program) messages first - controlX2 uses `IDPSegmentRequest`
and `IDPSettingsRequest` for reading basal profiles. The rest are lower priority
and can be added as GenericSettingsHandler entries with static responses.

---

## 2. Missing CONTROL Messages (Write/Action Commands) -- LARGEST GAP

These are the most significant gaps. CONTROL messages modify pump state (bolus,
basal, settings, CGM, cartridge operations). faketandem only handles 10 of ~50.

### 2a. Pump Control Operations (Critical for controlX2)

| Message | Used by controlX2? | Priority | Notes |
|---|---|---|---|
| **`CancelBolusRequest`** | **Yes (with 5x retry)** | **Critical** | controlX2 sends this to cancel bolus, NOT BolusTerminationRequest |
| **`SuspendPumpingRequest`** | **Yes** | **Critical** | Suspend insulin delivery |
| **`ResumePumpingRequest`** | **Yes** | **Critical** | Resume insulin delivery |
| **`SetTempRateRequest`** | **Yes** | **High** | Set temporary basal rate |
| **`StopTempRateRequest`** | **Yes** | **High** | Cancel temporary basal rate |
| `SetModesRequest` | Yes | High | Switch ControlIQ modes |
| `AdditionalBolusRequest` | No | Medium | Additional bolus during extended |
| `DisconnectPumpRequest` | No | Low | Graceful disconnect |

### 2b. Cartridge/Tubing Operations

| Message | Used by controlX2? | Priority | Notes |
|---|---|---|---|
| `EnterChangeCartridgeModeRequest` | Yes | Medium | Enter cartridge change mode |
| `ExitChangeCartridgeModeRequest` | Yes | Medium | Exit cartridge change mode |
| `EnterFillTubingModeRequest` | Yes | Medium | Enter fill tubing mode |
| `ExitFillTubingModeRequest` | Yes | Medium | Exit fill tubing mode |
| `FillCannulaRequest` | Yes | Medium | Fill cannula after site change |
| `PrimeTubingSuspendRequest` | No | Low | Prime tubing with suspend |

### 2c. Settings Write Operations

| Message | Used by controlX2? | Priority | Notes |
|---|---|---|---|
| `ChangeControlIQSettingsRequest` | Yes | High | Modify ControlIQ settings |
| `SetSleepScheduleRequest` | Yes | Medium | Set sleep schedule |
| `SetMaxBolusLimitRequest` | Yes | Medium | Set max bolus limit |
| `SetMaxBasalLimitRequest` | Yes | Medium | Set max basal limit |
| `SetQuickBolusSettingsRequest` | Yes | Medium | Quick bolus preferences |
| `SetPumpSoundsRequest` | Yes | Low | Audio settings |
| `SetPumpAlertSnoozeRequest` | Yes | Low | Alert snooze settings |
| `SetAutoOffAlertRequest` | No | Low | Auto-off alert |
| `SetBgReminderRequest` | No | Low | BG reminder |
| `SetSiteChangeReminderRequest` | No | Low | Site change reminder |
| `SetMissedMealBolusReminderRequest` | No | Low | Missed meal bolus reminder |
| `SetLowInsulinAlertRequest` | No | Low | Low insulin alert threshold |

### 2d. IDP (Insulin Delivery Program) Management

| Message | Used by controlX2? | Priority | Notes |
|---|---|---|---|
| `CreateIDPRequest` | Yes | Medium | Create new basal profile |
| `SetIDPSettingsRequest` | Yes | Medium | Modify profile settings |
| `SetIDPSegmentRequest` | Yes | Medium | Set time-based profile segments |
| `SetActiveIDPRequest` | Yes | Medium | Switch active profile |
| `DeleteIDPRequest` | Yes | Low | Delete a profile |
| `RenameIDPRequest` | Yes | Low | Rename a profile |

### 2e. CGM Configuration

| Message | Used by controlX2? | Priority | Notes |
|---|---|---|---|
| `SetDexcomG7PairingCodeRequest` | Yes | Medium | Set G7 pairing code |
| `SetG6TransmitterIdRequest` | Yes | Medium | Set G6 transmitter ID |
| `StartDexcomG6SensorSessionRequest` | Yes | Medium | Start CGM session |
| `StopDexcomCGMSensorSessionRequest` | Yes | Medium | Stop CGM session |
| `CgmHighLowAlertRequest` | No | Low | CGM high/low alert config |
| `CgmRiseFallAlertRequest` | No | Low | CGM rate alert config |
| `CgmOutOfRangeAlertRequest` | No | Low | CGM OOR alert config |

### 2f. System/Misc

| Message | Used by controlX2? | Priority | Notes |
|---|---|---|---|
| `ChangeTimeDateRequest` | Yes | Medium | Change pump date/time |
| `DismissNotificationRequest` | Yes | Medium | Dismiss notification on pump |
| `PlaySoundRequest` | Yes | Low | Play sound on pump |
| `ActivateShelfModeRequest` | No | Low | Shelf mode (shipping) |
| `UserInteractionRequest` | No | Low | Dev/test interaction |
| `StreamDataPreflightRequest` | No | Low | Stream data preflight |
| `FactoryResetRequest` | No | Low | Factory reset (v1) |

### Recommendation
**Top priority**: `CancelBolusRequest`, `SuspendPumpingRequest`, `ResumePumpingRequest`.
These are safety-critical operations that controlX2 actively uses.
`SetTempRateRequest`/`StopTempRateRequest` are next for basal management.

---

## 3. Missing CONTROL_STREAM Messages (Pump-Initiated Streaming)

CONTROL_STREAM messages are **pump-to-client streaming responses** sent during
multi-step physical operations (cartridge change, fill tubing, prime). In pumpX2,
the "request" classes are labeled `Nonexistent*` because clients never send them -
the pump streams these autonomously.

| Stream Response | Trigger | Priority |
|---|---|---|
| `PumpingStateStreamResponse` | Pump state changes | Medium |
| `FillTubingStateStreamResponse` | During fill tubing mode | Medium |
| `FillCannulaStateStreamResponse` | During cannula fill | Medium |
| `ExitFillTubingModeStateStreamResponse` | Exit fill tubing | Medium |
| `DetectingCartridgeStateStreamResponse` | Cartridge insertion | Medium |
| `EnterChangeCartridgeModeStateStreamResponse` | Enter cartridge mode | Medium |
| `LoadCartridgeStateStreamResponse` | During cartridge load | Medium |
| `PrimeNudgeStateStreamResponse` | During prime nudge | Medium |

### Recommendation
These are only needed if simulating cartridge/tubing operations end-to-end. They
use **signed negative opcodes** (-23, -29) on the CONTROL_STREAM characteristic.
Lower priority than CONTROL messages but needed for complete cartridge workflow.

---

## 4. Qualifying Events Gap

faketandem implements basic qualifying events but is missing many of the 30
bitmask types defined in pumpX2's `QualifyingEvent` enum.

### Currently implemented in faketandem:
- Bolus start/complete/cancel
- Alert events
- Basal rate change
- Reservoir low / Battery low
- Pump suspended/resumed

### Missing qualifying event types:

| Event | Bitmask ID | Suggested Handlers (from pumpX2) |
|---|---|---|
| `ALARM` | 2 | AlarmStatusRequest, HighestAamRequest, ActiveAamBitsRequest |
| `REMINDER` | 4 | ReminderStatusRequest |
| `MALFUNCTION` | 8 | MalfunctionStatusRequest, HighestAamRequest, ActiveAamBitsRequest |
| `CGM_ALERT` | 16 | CGMAlertStatusRequest |
| `HOME_SCREEN_CHANGE` | 32 | CurrentBasalStatusRequest, CurrentEGVGuiDataRequest, HomeScreenMirrorRequest, ControlIQInfoRequest |
| `TIME_CHANGE` | 256 | TimeSinceResetRequest |
| `IOB_CHANGE` | 2048 | IOBRequest, InsulinStatusRequest |
| `EXTENDED_BOLUS_CHANGE` | 4096 | ExtendedBolusStatusRequest |
| `PROFILE_CHANGE` | 8192 | ProfileStatusRequest, PumpGlobalsRequest, PumpSettingsRequest |
| `BG` | 16384 | LastBGRequest |
| `CGM_CHANGE` | 32768 | CurrentEGVGuiDataRequest, CGMStatusRequest |
| `BATTERY` | 65536 | CurrentBatteryRequest |
| `BASAL_IQ` | 131072 | BasalIQSettingsRequest, ControlIQInfoRequest |
| `REMAINING_INSULIN` | 262144 | InsulinStatusRequest |
| `PUMP_COMMUNICATIONS_SUSPENDED` | 524288 | (none) |
| `ACTIVE_PROFILE_SEGMENT_CHANGE` | 1048576 | CurrentBasalStatusRequest |
| `BASAL_IQ_STATUS` | 2097152 | BasalIQStatusRequest |
| `CONTROL_IQ_INFO` | 4194304 | ControlIQInfoRequest |
| `CONTROL_IQ_SLEEP` | 8388608 | ControlIQSleepScheduleRequest |
| `GLOBAL_PUMP_SETTINGS` | 16777216 | PumpGlobalsRequest, GlobalMaxBolusSettingsRequest, BasalLimitSettingsRequest, LocalizationRequest, PumpSettingsRequest |
| `SNOOZE_STATUS` | 33554432 | (from pumpX2) |
| `PUMPING_STATUS` | 67108864 | LoadStatusRequest, InsulinStatusRequest |
| `PUMP_RESET` | 134217728 | (reconnect) |
| `HEARTBEAT` | 268435456 | TimeSinceResetRequest |
| `BOLUS_PERMISSION_REVOKED` | 2147483648 | BolusPermissionChangeReasonRequest |

### Recommendation
The key insight from pumpX2 is that each qualifying event type has **suggested
handler messages** - when controlX2 receives a qualifying event notification, it
sends the suggested messages to refresh its state. faketandem needs to:
1. Send qualifying events using the correct bitmask IDs
2. Be prepared to receive the suggested handler messages in response

---

## 5. History Log Event Types Gap

pumpX2 defines **132 history log event types**. faketandem currently stores
history entries with basic type/timestamp/data but doesn't generate the full
range of structured events.

### Key categories not yet generated:

| Category | Count | Examples |
|---|---|---|
| Bolus events | 9 | BolusRequestedMsg1-3, BolexActivated/Completed, CorrectionDeclined |
| Basal events | 5 | BasalDelivery, BasalRateChange, TempRateActivated/Completed, DailyBasal |
| Cartridge/tubing | 8 | CartridgeInserted/Removed/Filled, TubingFilled, CannulaFilled |
| CGM - Dexcom G6 | 3 | DexcomG6CGM, CgmTransmitterId, CgmCalibration |
| CGM - Dexcom G7 | 10 | DexcomG7CGM, CgmStartSensorReqG7, CgmPairingCodeG7, etc. |
| CGM - Libre 2 | 8 | CgmDataFsl2, CgmJoinSessionFsl2, CgmStartSessionFsl2, etc. |
| CGM - Libre 3 | 3 | CgmDataFsl3, CgmJoinSessionFsl3, CgmStopSessionFsl3 |
| CGM - Generic | 20 | CgmDataSample, CgmDataGx, CgmCalibrationGx, etc. |
| Alarms/Alerts | 9 | AlarmActivated/Ack/Cleared, AlertActivated/Ack/Cleared, Malfunction |
| Reminders | 3 | ReminderActivated/Dismissed/Snoozed |
| System | 10 | DateChange, TimeChanged, NewDay, UsbConnected/Disconnected, etc. |
| ControlIQ/AutomatedMgmt | 11 | ControlIQUserModeChange, AAExercise/Tdi/Weight/Enable/Sleep changes |
| IDP management | 5 | IdpBolus, IdpAction, IdpList, IdpTimeDependentSegment |
| Hypo protection | 2 | HypoMinimizerSuspend/Resume |

### Recommendation
History log generation should be event-driven - when the simulator performs an
action (bolus, basal change, alert), it should generate the corresponding
history log entry type. Start with bolus and basal events since those are the
most commonly queried by controlX2's `HistoryLogFetcher`.

---

## 6. controlX2-Specific Behavioral Gaps

Beyond missing message types, controlX2 has specific behavioral expectations:

### 6a. Connection/Initialization Sequence

controlX2 expects this exact sequence after BLE connection:
1. `ApiVersionRequest` -> `ApiVersionResponse` (determines API capability)
2. `TimeSinceResetRequest` -> `TimeSinceResetResponse`
3. Authentication (JPAKE rounds 1-4 OR connection sharing detection)
4. Periodic polling begins (every 5 minutes)

**Gap**: faketandem handles all of these. No gap here.

### 6b. Periodic Polling (Every 5 Minutes)

controlX2 sends these in bulk every 5 minutes:
- `CurrentBatteryRequest` (V1 or V2 based on API version)
- `ControlIQIOBRequest`
- `InsulinStatusRequest`
- `HistoryLogStatusRequest`

**Gap**: faketandem handles all except `CurrentBatteryV1Request`. Minor gap.

### 6c. CancelBolusRequest vs BolusTerminationRequest

**Critical behavioral difference**: controlX2 uses `CancelBolusRequest` (CONTROL
characteristic) with 5x retry (0/100/200/300/400ms delays). faketandem handles
`BolusTerminationRequest` but NOT `CancelBolusRequest`. These are **different
message types with different opcodes**.

**Gap**: faketandem should add a `CancelBolusRequest` handler that behaves like
the existing BolusTerminationHandler.

### 6d. Bolus Status Polling (Every 1 Second During Active Bolus)

When a bolus is in progress, controlX2 polls `CurrentBolusStatusRequest` every
1 second. It detects completion when `bolusId` transitions from non-zero to 0.

**Gap**: faketandem handles `CurrentBolusStatusRequest` via GenericSettingsHandler
with a static response. It needs to return **dynamic bolus state** reflecting
the in-progress bolus (bolusId, status, deliveredAmount).

### 6e. Message Response Caching (30 seconds)

controlX2 caches responses by `(Characteristic, opCode)` for 30 seconds. It
also supports `BUST_CACHE_BULK` to force fresh requests.

**Gap**: No simulator-side gap, but good to know for testing - sending the same
request within 30 seconds may not actually reach the simulator.

### 6f. Connection Sharing with tconnect App

controlX2 detects if the official tconnect app is already connected by checking
`Packetize.txId > 0`. It can share the BLE connection.

**Gap**: faketandem doesn't simulate tconnect app coexistence. Low priority
unless testing connection sharing.

### 6g. History Log Fetching

controlX2's `HistoryLogFetcher`:
- Reads `HistoryLogStatusResponse` for `firstSequenceNum`/`lastSequenceNum`
- Fetches in chunks of 32 entries
- 4-second timeout per chunk
- 500ms poll interval to check if entries arrived
- 1000ms delay between chunk requests

**Gap**: faketandem has basic history log support but needs to ensure:
- `HistoryLogStatusResponse` returns correct sequence range
- `HistoryLogStreamResponse` delivers entries as stream notifications on HISTORY_LOG characteristic
- Entries arrive within the 4-second timeout

### 6h. API Version-Dependent Message Builders

controlX2 uses builder classes that select message variants based on API version:
- `CurrentBatteryRequestBuilder.create(apiVersion)` -> V1 or V2
- `IOBRequestBuilder.create(apiVersion)` -> ControlIQIOB or NonControlIQIOB
- `LastBolusStatusRequestBuilder.create(apiVersion)` -> V2 or V3
- `ControlIQInfoRequestBuilder.create(apiVersion)` -> V1 or V2

**Gap**: faketandem handles the newer variants but not all older variants.
Should add V1 fallbacks for older API version testing.

---

## 7. Priority-Ordered Implementation Roadmap

### Phase 1: Critical controlX2 Compatibility (Highest Priority)

1. **`CancelBolusRequest` handler** - controlX2 uses this for bolus cancellation
2. **`SuspendPumpingRequest` handler** - safety-critical pump suspend
3. **`ResumePumpingRequest` handler** - safety-critical pump resume
4. **Dynamic `CurrentBolusStatusRequest` response** - must reflect active bolus state
5. **Proper qualifying event bitmask IDs** - use pumpX2's bitmask values

### Phase 2: Basal/Temp Rate Management

6. **`SetTempRateRequest` handler** - set temporary basal rate
7. **`StopTempRateRequest` handler** - cancel temporary basal rate
8. **`SetModesRequest` handler** - switch ControlIQ modes
9. **`ChangeControlIQSettingsRequest` handler** - modify ControlIQ settings

### Phase 3: Missing Status Queries

10. **`IDPSegmentRequest` / `IDPSettingsRequest`** - basal profile data
11. **`CurrentBatteryV1Request`** - older API version battery
12. **`GetSavedG7PairingCodeRequest`** - CGM pairing
13. **V2 variants** - CgmStatusV2, CurrentEgvGuiDataV2

### Phase 4: Settings Write Operations

14. Settings write handlers (SetMaxBolusLimit, SetMaxBasalLimit, SetSleepSchedule, etc.)
15. IDP management (Create/Set/Delete/Rename IDP)
16. CGM configuration (SetG6TransmitterId, SetDexcomG7PairingCode, Start/StopSession)

### Phase 5: Cartridge/Tubing Operations

17. Enter/Exit change cartridge mode handlers
18. Enter/Exit fill tubing mode handlers
19. FillCannula handler
20. CONTROL_STREAM streaming responses

### Phase 6: History Log Completeness

21. Generate structured history log entries for all pump actions
22. Implement proper HistoryLogStream delivery on HISTORY_LOG characteristic

### Phase 7: Full Qualifying Events

23. Implement all 30 qualifying event bitmask types
24. Send correct suggested handler messages with each event

---

## 8. Message Type Cross-Reference

### faketandem handlers -> pumpX2 message type mapping

| faketandem Handler | Characteristic | In pumpX2? | In controlX2? |
|---|---|---|---|
| ApiVersionRequest | CURRENT_STATUS | Yes | Yes |
| TimeSinceResetRequest | CURRENT_STATUS | Yes | Yes |
| CentralChallengeRequest | AUTHORIZATION | Yes | Yes |
| PumpChallengeRequest | AUTHORIZATION | Yes | Yes |
| JPAKERound1-4Request | AUTHORIZATION | Yes (as Jpake1a/1b/2/3/4) | Yes |
| CurrentStatusRequest | CURRENT_STATUS | **No exact match** | **N/A** |
| HistoryLogRequest | CURRENT_STATUS | Yes | Yes |
| CreateHistoryLogRequest | CURRENT_STATUS | Yes | No |
| HistoryLogStatusRequest | CURRENT_STATUS | Yes | Yes |
| BolusPermissionRequest | CONTROL | Yes | Yes |
| BolusCalcDataSnapshotRequest | CURRENT_STATUS | Yes | Yes |
| InitiateBolusRequest | CONTROL | Yes | Yes |
| BolusTerminationRequest | **Not in pumpX2** | **No** | **No** |
| RemoteBgEntryRequest | CONTROL | Yes | Yes |
| RemoteCarbEntryRequest | CONTROL | Yes | Yes |
| BolusPermissionReleaseRequest | CONTROL | Yes | Yes |
| SetSensorTypeRequest | CONTROL | Yes | No |
| StreamDataReadinessRequest | CURRENT_STATUS | Yes | No |
| FactoryResetBRequest | CONTROL | Yes | No |
| ProfileBasalRequest | CURRENT_STATUS | **Custom** | No |
| (40+ GenericSettings) | CURRENT_STATUS | Yes | Yes |

**Notable findings:**
- `CurrentStatusRequest` in faketandem has no exact pumpX2 counterpart - it may
  be a composite/custom message
- `BolusTerminationRequest` doesn't exist in pumpX2 - controlX2 uses
  `CancelBolusRequest` instead. This is a **critical naming mismatch**.
- faketandem's JPAKE round naming (JPAKERound1-4) maps to pumpX2's
  Jpake1a/Jpake1b/Jpake2/Jpake3SessionKey/Jpake4KeyConfirmation

---

## 9. Conclusion

The simulator has solid coverage of **read-only status queries** (~80%) and
**authentication** (100%), but significant gaps in:

1. **Write/action commands** (CONTROL characteristic) - only 20% coverage
2. **Bolus cancellation** - uses wrong message type (`BolusTerminationRequest`
   instead of `CancelBolusRequest`)
3. **Pump suspend/resume** - not implemented at all
4. **Qualifying events** - missing bitmask IDs and most event types
5. **Dynamic status responses** - `CurrentBolusStatusRequest` returns static data
   instead of reflecting active bolus state
6. **History log generation** - basic storage exists but structured event types
   not generated

The most impactful improvements would be adding `CancelBolusRequest`,
`SuspendPumpingRequest`, `ResumePumpingRequest`, and making
`CurrentBolusStatusRequest` return dynamic bolus state.
