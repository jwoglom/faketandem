# Tandem Pump Simulator Improvement Plan

## Executive Summary

This document outlines a detailed plan to transform the current basic Tandem pump BLE simulator into a full-fledged simulator that accurately emulates a real Tandem t:slim X2 / Mobi insulin pump. The key strategy is to leverage the existing **pumpX2** Java library's `cliparser` tool for all message parsing and building, rather than reimplementing the complex protocol logic in Go.

## Current State Analysis

### What We Have
- ✅ BLE service with correct Tandem pump UUIDs and characteristics
- ✅ Basic BLE advertisement and connection handling
- ✅ WebSocket API for monitoring and control
- ✅ Placeholder handlers for read/write/notify operations

### What's Missing
- ❌ Actual message parsing and response generation
- ❌ Authentication flow (JPAKE and legacy)
- ❌ Transaction ID management
- ❌ Message packet chunking (18-byte and 40-byte)
- ❌ CRC16 and HMAC-SHA1 validation
- ❌ Pump state simulation (basal rates, boluses, reservoir, battery, etc.)
- ❌ History log generation
- ❌ Alert and alarm simulation
- ❌ Realistic timing and sequences

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────┐
│                    BLE Client (pumpX2)                       │
└──────────────────────┬──────────────────────────────────────┘
                       │ BLE Messages (hex)
                       ↓
┌─────────────────────────────────────────────────────────────┐
│              Go Simulator (faketandem)                       │
│                                                              │
│  ┌────────────────┐         ┌──────────────────┐           │
│  │  BLE Layer     │ ←────→  │  Message Router  │           │
│  │  (bluetooth/)  │         │  (handler/)      │           │
│  └────────────────┘         └─────────┬────────┘           │
│                                       │                     │
│                             ┌─────────▼─────────┐           │
│                             │  pumpX2 Bridge    │           │
│                             │  (cliparser exec) │           │
│                             └─────────┬─────────┘           │
│                                       │                     │
│                             ┌─────────▼─────────┐           │
│                             │  Pump State       │           │
│                             │  Simulator        │           │
│                             │  (state/)         │           │
│                             └───────────────────┘           │
│                                                              │
│  ┌────────────────┐                                         │
│  │  WebSocket API │  ← Monitor & Control                   │
│  │  (api/)        │                                         │
│  └────────────────┘                                         │
└─────────────────────────────────────────────────────────────┘
                       │
                       ↓
              ┌────────────────┐
              │  pumpX2 repo   │
              │  (external)    │
              │                │
              │  cliparser JAR │
              │  via gradle    │
              └────────────────┘
```

## Phase 1: pumpX2 Integration Infrastructure

### 1.1 Command-Line Configuration
**Goal:** Add support for pointing to a local pumpX2 repository

**Tasks:**
- [ ] Add `-pumpx2-path` command-line flag to specify pumpX2 repo location
- [ ] Validate that the path exists and contains expected structure
- [ ] Add `-pumpx2-gradle-cmd` flag to override gradle command (default: `./gradlew`)
- [ ] Add environment variable support for `PUMPX2_PATH`

**Files to Create/Modify:**
- `main.go` - Add flag parsing
- `pkg/config/config.go` - Configuration struct

**Testing:**
```bash
./faketandem -pumpx2-path=/path/to/pumpX2
```

### 1.2 pumpX2 Cliparser Bridge
**Goal:** Create Go package to execute pumpX2 cliparser commands

**Tasks:**
- [ ] Create `pkg/pumpx2/bridge.go`
- [ ] Implement cliparser JAR building (via gradle)
  - Execute `./gradlew :cliparser:shadowJar`
  - Locate output JAR at `cliparser/build/libs/pumpx2-cliparser-all.jar`
- [ ] Implement message parsing function
  - Execute: `java -jar cliparser.jar parse <hex>`
  - Parse JSON output
- [ ] Implement message building function
  - Execute: `java -jar cliparser.jar encode <txId> <messageName> [params]`
  - Parse JSON output containing hex packets and characteristic UUID
- [ ] Implement opcode identification
  - Execute: `java -jar cliparser.jar opcode <hex>`
- [ ] Add environment variable passing for authentication keys
  - `PUMP_AUTHENTICATION_KEY`
  - `PUMP_PAIRING_CODE`
  - `PUMP_TIME_SINCE_RESET`
- [ ] Add caching/singleton for cliparser JAR path
- [ ] Add proper error handling and logging

**Files to Create:**
- `pkg/pumpx2/bridge.go` - Main bridge interface
- `pkg/pumpx2/types.go` - Go structs for JSON parsing
- `pkg/pumpx2/builder.go` - JAR builder

**Data Structures:**
```go
type CliParserBridge struct {
    jarPath      string
    pumpX2Path   string
    javaCmd      string
    gradleCmd    string
    authKey      string
    pairingCode  string
}

type ParsedMessage struct {
    Opcode        int
    MessageType   string
    TxID          int
    Cargo         map[string]interface{}
    IsSigned      bool
    IsValid       bool
}

type EncodedMessage struct {
    Characteristic string
    Packets        []string  // Hex strings
    MessageType    string
    TxID           int
}
```

**Testing Strategy:**
- Unit tests with mock messages
- Integration test: parse → encode → parse round-trip
- Test with real pumpX2 sample messages

### 1.3 Packet Assembly/Disassembly
**Goal:** Handle message chunking and reassembly

**Tasks:**
- [ ] Create `pkg/protocol/packet.go`
- [ ] Implement packet disassembly
  - Parse packet header (remaining count, txID)
  - Reassemble packets by (characteristic, txID) key
  - Extract CRC16 trailer
- [ ] Implement packet assembly
  - Split messages into 18-byte or 40-byte chunks based on characteristic
  - Add packet headers
  - Handle CRC16 calculation (delegate to cliparser or implement)
- [ ] Add packet buffer management
  - Track in-progress multi-packet messages
  - Timeout old incomplete messages
- [ ] Implement characteristic-specific chunking rules
  - Authorization: 40 bytes
  - Control: 18 bytes
  - ControlStream: 18 bytes
  - Others: 18 bytes (verify with pumpX2)

**Files to Create:**
- `pkg/protocol/packet.go` - Packet handling
- `pkg/protocol/reassembler.go` - Multi-packet message reassembly

**Data Structures:**
```go
type PacketHeader struct {
    RemainingPackets uint8
    TxID            uint8
}

type PacketBuffer struct {
    CharType      bluetooth.CharacteristicType
    TxID          uint8
    Packets       [][]byte
    ExpectedCount int
    Timestamp     time.Time
}

type Reassembler struct {
    buffers map[string]*PacketBuffer
    mutex   sync.RWMutex
}
```

## Phase 2: Message Handling Framework

### 2.1 Transaction ID Management
**Goal:** Implement proper transaction ID tracking

**Tasks:**
- [ ] Create `pkg/protocol/transaction.go`
- [ ] Implement TX ID counter (0-255, wraps around)
- [ ] Track outstanding requests by TX ID
- [ ] Implement request/response matching
- [ ] Add timeout for unanswered requests

**Files to Create:**
- `pkg/protocol/transaction.go`

**Data Structures:**
```go
type TransactionManager struct {
    nextTxID      uint8
    mutex         sync.Mutex
    pendingReqs   map[uint8]*PendingRequest
}

type PendingRequest struct {
    TxID          uint8
    MessageType   string
    Timestamp     time.Time
    ResponseChan  chan *ParsedMessage
}
```

### 2.2 Message Router
**Goal:** Route incoming messages to appropriate handlers

**Tasks:**
- [ ] Create `pkg/handler/router.go`
- [ ] Implement message type registry
- [ ] Create handler interface
- [ ] Implement routing logic based on opcode/message type
- [ ] Add default/fallback handler
- [ ] Add logging for unknown messages

**Files to Create:**
- `pkg/handler/router.go`
- `pkg/handler/handler.go` - Handler interface

**Data Structures:**
```go
type MessageHandler interface {
    HandleMessage(msg *pumpx2.ParsedMessage, state *state.PumpState) (*pumpx2.EncodedMessage, error)
    MessageType() string
}

type Router struct {
    handlers map[string]MessageHandler
    bridge   *pumpx2.CliParserBridge
    state    *state.PumpState
}
```

### 2.3 Response Builder
**Goal:** Build appropriate responses using cliparser

**Tasks:**
- [ ] Create `pkg/handler/response.go`
- [ ] Implement helper functions for common responses
- [ ] Add response templates for each message type
- [ ] Integrate with cliparser's encode functionality

**Files to Create:**
- `pkg/handler/response.go`

## Phase 3: Pump State Simulation

### 3.1 Core Pump State
**Goal:** Simulate realistic pump state

**Tasks:**
- [ ] Create `pkg/state/pump.go`
- [ ] Implement pump state structure
  - API version
  - Serial number
  - Firmware version
  - Model (X2, Mobi, etc.)
  - Time since reset (pump uptime)
  - Current time
- [ ] Implement insulin delivery state
  - Current basal rate
  - Active temp basal
  - Active bolus
  - Total daily dose (TDD)
  - Insulin on board (IOB)
- [ ] Implement reservoir state
  - Current insulin units
  - Fill status
- [ ] Implement battery state
  - Battery percentage
  - Charging status
- [ ] Implement cartridge/infusion set state
  - Days since change
  - Prime status
- [ ] Add state persistence (save/load from file)

**Files to Create:**
- `pkg/state/pump.go`
- `pkg/state/insulin.go`
- `pkg/state/persistence.go`

**Data Structures:**
```go
type PumpState struct {
    // Identity
    SerialNumber    string
    Model           string
    FirmwareVersion string
    APIVersion      int

    // Time
    TimeSinceReset  uint32  // seconds
    CurrentTime     time.Time

    // Authentication
    AuthKey         []byte
    PairingCode     string
    IsAuthenticated bool

    // Insulin Delivery
    Basal           *BasalState
    Bolus           *BolusState
    IOB             float64
    TDD             float64

    // Physical State
    Reservoir       *ReservoirState
    Battery         *BatteryState
    Cartridge       *CartridgeState

    // Alerts/Alarms
    ActiveAlerts    []Alert

    // History
    History         *HistoryLog

    mutex           sync.RWMutex
}

type BasalState struct {
    CurrentRate     float64  // units/hr
    TempBasalActive bool
    TempBasalRate   float64
    TempBasalEnd    time.Time
}

type BolusState struct {
    Active         bool
    UnitsDelivered float64
    UnitsTotal     float64
    StartTime      time.Time
    BolusID        uint32
}

type ReservoirState struct {
    CurrentUnits   float64
    MaxUnits       float64
    LastFill       time.Time
}

type BatteryState struct {
    Percentage     int
    Charging       bool
}

type CartridgeState struct {
    DaysSinceChange int
    LastPrime       time.Time
}
```

### 3.2 Pump State Evolution
**Goal:** Make pump state change realistically over time

**Tasks:**
- [ ] Create `pkg/state/simulator.go`
- [ ] Implement background goroutine for state updates
- [ ] Simulate insulin delivery
  - Deduct from reservoir based on basal rate
  - Handle bolus delivery over time
  - Update IOB calculations
- [ ] Simulate battery drain
- [ ] Increment time since reset
- [ ] Trigger alerts based on conditions
  - Low reservoir
  - Low battery
  - Occlusion (random simulation)
  - Cartridge age
- [ ] Add configurable simulation speed (for testing)

**Files to Create:**
- `pkg/state/simulator.go`

### 3.3 Alert and Alarm System
**Goal:** Simulate realistic alerts and alarms

**Tasks:**
- [ ] Create `pkg/state/alerts.go`
- [ ] Implement alert types
  - Low reservoir (< 20 units)
  - Low battery (< 20%)
  - Cartridge expired
  - Occlusion
  - Basal suspended
  - etc.
- [ ] Implement alert priority (info, warning, critical)
- [ ] Add alert acknowledgment tracking
- [ ] Trigger qualifying events on state changes

**Files to Create:**
- `pkg/state/alerts.go`

**Data Structures:**
```go
type Alert struct {
    ID          uint32
    Type        AlertType
    Priority    AlertPriority
    Message     string
    Timestamp   time.Time
    Acknowledged bool
}

type AlertType int
const (
    AlertLowReservoir AlertType = iota
    AlertLowBattery
    AlertCartridgeExpired
    AlertOcclusion
    AlertBasalSuspended
    // ... more alert types
)

type AlertPriority int
const (
    PriorityInfo AlertPriority = iota
    PriorityWarning
    PriorityCritical
)
```

### 3.4 History Log
**Goal:** Generate realistic history log entries

**Tasks:**
- [ ] Create `pkg/state/history.go`
- [ ] Implement history log storage (circular buffer)
- [ ] Log all significant events
  - Basal rate changes
  - Boluses
  - Temp basals
  - Alarms
  - Settings changes
  - Cartridge changes
- [ ] Implement history log retrieval
- [ ] Use cliparser for history log encoding

**Files to Create:**
- `pkg/state/history.go`

**Data Structures:**
```go
type HistoryLog struct {
    Entries      []HistoryEntry
    MaxEntries   int
    NextID       uint32
    mutex        sync.RWMutex
}

type HistoryEntry struct {
    ID           uint32
    Timestamp    time.Time
    EventType    string
    Data         map[string]interface{}
}
```

## Phase 4: Authentication Implementation

### 4.1 JPAKE Authentication
**Goal:** Implement modern JPAKE authentication flow

**Tasks:**
- [ ] Create `pkg/auth/jpake.go`
- [ ] Implement JPAKE state machine
  - Wait for CentralChallengeRequest
  - Generate and send CentralChallengeResponse
  - Process JPAKE round 1-4 messages
  - Derive authentication key using HKDF
- [ ] Use cliparser for JPAKE message building
  - `java -jar cliparser.jar jpake <pairingCode>`
- [ ] Store authentication key in pump state
- [ ] Implement connection sharing detection
  - Detect unsolicited notifications (existing auth session)
  - Skip authentication but require stored credentials

**Files to Create:**
- `pkg/auth/jpake.go`
- `pkg/auth/state.go`

**Data Structures:**
```go
type JPAKEAuthenticator struct {
    bridge       *pumpx2.CliParserBridge
    pumpState    *state.PumpState
    pairingCode  string
    currentRound int
}

type AuthState int
const (
    AuthStateUnauthenticated AuthState = iota
    AuthStateChallenge
    AuthStateJPAKERound1
    AuthStateJPAKERound2
    AuthStateJPAKERound3
    AuthStateJPAKERound4
    AuthStateAuthenticated
)
```

### 4.2 Legacy Authentication
**Goal:** Support older pumps with challenge-response auth

**Tasks:**
- [ ] Create `pkg/auth/legacy.go`
- [ ] Implement legacy challenge-response flow
  - CentralChallengeRequest → CentralChallengeResponse
  - PumpChallengeRequest → PumpChallengeResponse
- [ ] Use 16-character pairing code
- [ ] Derive HMAC key from pairing code

**Files to Create:**
- `pkg/auth/legacy.go`

### 4.3 Message Signing and Validation
**Goal:** Implement HMAC-SHA1 signing for secure messages

**Tasks:**
- [ ] Create `pkg/auth/signing.go`
- [ ] Implement message signing
  - Determine which messages require signing
  - Add HMAC-SHA1 trailer
  - Include time-since-reset
- [ ] Implement signature validation (if needed for client messages)
- [ ] Leverage cliparser's built-in validation

**Files to Create:**
- `pkg/auth/signing.go`

## Phase 5: Message Handler Implementation

### 5.1 Core Initialization Handlers
**Goal:** Handle pump initialization sequence

**Message Handlers to Implement:**
- [ ] `ApiVersionRequest` → `ApiVersionResponse`
  - Return supported API version (e.g., 4 or 5)
- [ ] `CentralChallengeRequest` → `CentralChallengeResponse`
  - Initiate authentication
- [ ] `PumpChallengeRequest` → `PumpChallengeResponse`
  - Complete legacy auth
- [ ] JPAKE round messages (opcodes vary)

**Files to Create:**
- `pkg/handler/api_version.go`
- `pkg/handler/auth_challenge.go`
- `pkg/handler/jpake_rounds.go`

**Implementation Pattern:**
```go
type ApiVersionHandler struct {
    bridge *pumpx2.CliParserBridge
}

func (h *ApiVersionHandler) HandleMessage(msg *pumpx2.ParsedMessage, state *state.PumpState) (*pumpx2.EncodedMessage, error) {
    // Extract request parameters from msg.Cargo
    apiVersion := 5  // Or from pump state

    // Build response using cliparser
    response, err := h.bridge.EncodeMessage(
        msg.TxID,
        "ApiVersionResponse",
        map[string]interface{}{
            "apiVersion": apiVersion,
        },
    )
    return response, err
}

func (h *ApiVersionHandler) MessageType() string {
    return "ApiVersionRequest"
}
```

### 5.2 Status Handlers
**Goal:** Handle current status requests

**Message Handlers to Implement:**
- [ ] `CurrentStatusRequest` → `CurrentStatusResponse`
  - Return comprehensive pump status
  - Include basal rate, bolus status, reservoir, battery, alerts
  - Update status characteristic

**Files to Create:**
- `pkg/handler/current_status.go`

**Implementation Notes:**
- Status response is complex with many fields
- Let cliparser handle the encoding
- Pull data from `PumpState`

### 5.3 Time and Reset Handlers
**Goal:** Handle timing synchronization

**Message Handlers to Implement:**
- [ ] `TimeSinceResetRequest` → `TimeSinceResetResponse`
  - Return pump uptime in seconds
- [ ] `PumpTimeRequest` → `PumpTimeResponse`
  - Return current pump time

**Files to Create:**
- `pkg/handler/time.go`

### 5.4 Bolus Handlers
**Goal:** Implement remote bolus functionality

**Message Handlers to Implement:**
- [ ] `BolusPermissionRequest` → `BolusPermissionResponse`
  - Grant or deny bolus permission
  - Check pump state (not suspended, etc.)
- [ ] `BolusCalcDataSnapshotRequest` → `BolusCalcDataSnapshotResponse`
  - Return current bolus calculation data (IOB, basal rate, etc.)
- [ ] `RemoteBgEntryRequest` → `RemoteBgEntryResponse`
  - Accept blood glucose reading
- [ ] `BolusCarbohydrateEntryRequest` → Response
  - Accept carbohydrate entry
- [ ] `InitiateBolusRequest` → `InitiateBolusResponse`
  - Start bolus delivery
  - Update pump state
  - Start background bolus simulation
- [ ] `BolusTerminationRequest` → `BolusTerminationResponse`
  - Cancel active bolus
- [ ] `BolusPermissionReleaseRequest` → `BolusPermissionReleaseResponse`
  - Release bolus permission

**Files to Create:**
- `pkg/handler/bolus_permission.go`
- `pkg/handler/bolus_calc.go`
- `pkg/handler/bolus_entry.go`
- `pkg/handler/bolus_initiate.go`
- `pkg/handler/bolus_terminate.go`

**Implementation Notes:**
- Bolus handlers are critical - implement carefully
- Validate all inputs
- Update pump state atomically
- Generate qualifying events for bolus start/stop
- Simulate bolus delivery over time in background

### 5.5 History Log Handlers
**Goal:** Provide history log access

**Message Handlers to Implement:**
- [ ] `HistoryLogRequest` → `HistoryLogResponse`
  - Return history log entries
  - Support pagination
- [ ] Use cliparser's `historylog` command for encoding

**Files to Create:**
- `pkg/handler/history.go`

### 5.6 Settings and Profile Handlers
**Goal:** Handle pump settings and profiles

**Message Handlers to Implement:**
- [ ] `BasalIQSettingsRequest` → Response
- [ ] `ControlIQSettingsRequest` → Response
- [ ] `ProfileBasalRequest` → Response
- [ ] `ProfileCarbRatiosRequest` → Response
- [ ] `ProfileISFRequest` → Response
- [ ] `ProfileTargetGlucoseRequest` → Response
- [ ] Other settings requests (discover from pumpX2)

**Files to Create:**
- `pkg/handler/settings.go`
- `pkg/handler/profiles.go`
- `pkg/state/settings.go` - Settings state storage

### 5.7 Alarm and Alert Handlers
**Goal:** Handle qualifying events and alerts

**Tasks:**
- [ ] Implement qualifying event notifications
  - Send on CharQualifyingEvents characteristic
  - Trigger on state changes (bolus start, alert, etc.)
- [ ] Implement alert acknowledgment handlers

**Files to Create:**
- `pkg/handler/alerts.go`
- `pkg/handler/qualifying_events.go`

## Phase 6: Characteristic-Specific Logic

### 6.1 Authorization Characteristic
**Goal:** Handle authentication messages on Authorization char

**Tasks:**
- [ ] Route all auth messages through this characteristic
- [ ] Use 40-byte packet chunks
- [ ] Implement proper sequencing

**Files to Modify:**
- `pkg/handler/router.go` - Add characteristic-based routing

### 6.2 Control Characteristic
**Goal:** Handle control commands

**Tasks:**
- [ ] Route control messages (bolus, settings changes)
- [ ] Use 18-byte packet chunks
- [ ] Implement request queuing if needed

### 6.3 ControlStream Characteristic
**Goal:** Handle streaming control data

**Tasks:**
- [ ] Identify what uses ControlStream vs Control
- [ ] Implement appropriate handlers

### 6.4 CurrentStatus Characteristic
**Goal:** Provide current pump status

**Tasks:**
- [ ] Implement periodic status updates via notifications
- [ ] Send status on read requests
- [ ] Update frequency: every 60 seconds or on significant change

### 6.5 QualifyingEvents Characteristic
**Goal:** Send event notifications

**Tasks:**
- [ ] Send notifications for all qualifying events
  - Bolus start/stop
  - Basal rate changes
  - Alerts
  - Settings changes
- [ ] Implement event queuing
- [ ] Use cliparser to encode qualifying event messages

### 6.6 HistoryLog Characteristic
**Goal:** Provide history log data

**Tasks:**
- [ ] Respond to history log requests
- [ ] Support streaming large history logs

## Phase 7: Advanced Features

### 7.1 Realistic Timing Simulation
**Goal:** Make simulator behavior realistic

**Tasks:**
- [ ] Add realistic response delays (100-500ms)
- [ ] Simulate processing time for complex requests
- [ ] Add occasional "busy" responses
- [ ] Implement proper message sequencing

**Files to Create:**
- `pkg/simulation/timing.go`

### 7.2 Error and Edge Case Simulation
**Goal:** Simulate error conditions for testing

**Tasks:**
- [ ] Add configurable error injection
  - Communication errors
  - Invalid state errors
  - Occlusion errors
- [ ] Implement pump suspension simulation
- [ ] Add low battery/reservoir simulation modes

**Files to Create:**
- `pkg/simulation/errors.go`

### 7.3 Multiple Pump Profiles
**Goal:** Support different pump configurations

**Tasks:**
- [ ] Add configuration files for different pump models
  - X2 vs Mobi
  - Different firmware versions
  - Different feature sets
- [ ] Load profile on startup
- [ ] Support switching profiles via API

**Files to Create:**
- `pkg/config/profiles.go`
- `profiles/x2.yaml`
- `profiles/mobi.yaml`

### 7.4 Advanced WebSocket API
**Goal:** Enhanced monitoring and control

**Tasks:**
- [ ] Add endpoints for:
  - View/modify pump state
  - Inject messages
  - Trigger alerts
  - Fast-forward time
  - View message log
- [ ] Add web UI (optional future enhancement)

**Files to Modify:**
- `pkg/api/server.go` - Add new commands

### 7.5 Logging and Debugging
**Goal:** Comprehensive logging for debugging

**Tasks:**
- [ ] Log all incoming/outgoing messages with timestamps
- [ ] Add log levels for different verbosity
- [ ] Save message logs to file
- [ ] Add protocol analyzer mode

**Files to Create:**
- `pkg/logging/message_log.go`

## Phase 8: Testing and Validation

### 8.1 Unit Tests
**Goal:** Test individual components

**Tasks:**
- [ ] Test pumpX2 bridge
- [ ] Test packet assembly/disassembly
- [ ] Test transaction management
- [ ] Test message routing
- [ ] Test state management
- [ ] Test each message handler

**Files to Create:**
- `*_test.go` files alongside each package

### 8.2 Integration Tests
**Goal:** Test end-to-end flows

**Tasks:**
- [ ] Test authentication flows (JPAKE and legacy)
- [ ] Test bolus delivery flow
- [ ] Test status updates
- [ ] Test history log retrieval
- [ ] Test alert generation

**Files to Create:**
- `test/integration/*_test.go`

### 8.3 Real Device Testing
**Goal:** Validate against pumpX2 client

**Tasks:**
- [ ] Test with pumpX2 sample app
- [ ] Test authentication
- [ ] Test bolus delivery
- [ ] Test status reading
- [ ] Test history retrieval
- [ ] Compare behavior with real pump (if available)

### 8.4 Performance Testing
**Goal:** Ensure simulator can handle realistic load

**Tasks:**
- [ ] Test sustained connections
- [ ] Test rapid message sequences
- [ ] Test multiple reconnections
- [ ] Profile memory usage
- [ ] Profile CPU usage

## Phase 9: Documentation and Deployment

### 9.1 Code Documentation
**Goal:** Document all code

**Tasks:**
- [ ] Add godoc comments to all exported functions
- [ ] Add package-level documentation
- [ ] Document configuration options
- [ ] Add inline comments for complex logic

### 9.2 User Documentation
**Goal:** Help users run the simulator

**Tasks:**
- [ ] Update README.md
  - Requirements (Java, Go, pumpX2)
  - Installation instructions
  - Usage examples
  - Configuration guide
- [ ] Create CONTRIBUTING.md
- [ ] Create ARCHITECTURE.md (high-level overview)
- [ ] Add examples directory with sample configs

**Files to Create/Update:**
- `README.md`
- `CONTRIBUTING.md`
- `ARCHITECTURE.md`
- `docs/CONFIGURATION.md`
- `docs/MESSAGE_FLOWS.md`
- `docs/TROUBLESHOOTING.md`

### 9.3 Build and Deployment
**Goal:** Easy build and deployment

**Tasks:**
- [ ] Add Makefile
  - Build targets
  - Test targets
  - pumpX2 setup automation
- [ ] Add CI/CD configuration (GitHub Actions)
  - Build on PR
  - Run tests
- [ ] Create Docker container (optional)
  - Include Java and Go
  - Include pumpX2 repo
  - Expose BLE and WebSocket

**Files to Create:**
- `Makefile`
- `.github/workflows/build.yml`
- `.github/workflows/test.yml`
- `Dockerfile` (optional)

## Implementation Phases Summary

### Phase 1-2: Foundation (Weeks 1-2)
- pumpX2 integration
- Packet handling
- Transaction management
- Message routing framework

### Phase 3: State Simulation (Week 3)
- Pump state structure
- State evolution
- Alerts and history

### Phase 4: Authentication (Week 4)
- JPAKE authentication
- Legacy authentication
- Message signing

### Phase 5-6: Message Handlers (Weeks 5-7)
- Implement all message handlers
- Characteristic-specific logic
- Status updates and events

### Phase 7: Polish (Week 8)
- Advanced features
- Error simulation
- Enhanced API

### Phase 8-9: Testing and Documentation (Weeks 9-10)
- Comprehensive testing
- Documentation
- Deployment automation

## Success Criteria

The simulator will be considered complete when:

1. ✅ It can authenticate with pumpX2 using both JPAKE and legacy methods
2. ✅ It responds correctly to all common message types
3. ✅ It can complete a full bolus delivery flow
4. ✅ It generates realistic status updates
5. ✅ It maintains consistent pump state
6. ✅ It passes integration tests with pumpX2 sample app
7. ✅ It can run continuously for extended periods
8. ✅ All code is documented and tested
9. ✅ Users can easily set up and run the simulator

## Key Design Principles

1. **Leverage pumpX2:** Never reimplement message parsing/building logic - always use cliparser
2. **Modular design:** Each component should be independent and testable
3. **Realistic simulation:** Pump state should evolve realistically over time
4. **Comprehensive logging:** All messages and state changes should be logged
5. **Error handling:** Gracefully handle all error conditions
6. **Testability:** Design for easy unit and integration testing
7. **Documentation:** Code should be self-documenting with good comments
8. **Configuration:** Support different pump models and configurations
9. **Extensibility:** Easy to add new message handlers and features

## Dependencies

### External Dependencies
- **pumpX2 repository** - User must provide path to local checkout
- **Java 11+** - Required to run cliparser
- **Gradle** - Included in pumpX2 repo via wrapper
- **Go 1.19+** - For building the simulator

### Go Packages (add to go.mod)
- `github.com/paypal/gatt` - Already included for BLE
- `github.com/gorilla/websocket` - Already included for WebSocket
- `github.com/sirupsen/logrus` - Already included for logging
- `gopkg.in/yaml.v3` - For configuration files
- `github.com/stretchr/testify` - For testing

## Configuration File Format

Example `config.yaml`:

```yaml
simulator:
  model: "t:slim X2"
  serial_number: "11223344"
  firmware_version: "7.6.0.0"
  api_version: 5

  pumpx2:
    path: "/path/to/pumpX2"
    gradle_cmd: "./gradlew"
    java_cmd: "java"

  authentication:
    pairing_code: "123456"  # 6 digits for JPAKE
    auto_authenticate: true

  pump_state:
    reservoir_units: 200
    battery_percent: 80
    basal_rate: 0.85

  simulation:
    speed_multiplier: 1.0  # 1.0 = real-time, 10.0 = 10x speed
    enable_random_alerts: true
    bolus_delivery_time: 300  # seconds for 30 units

  api:
    websocket_port: 8080
    enable_web_ui: false

  logging:
    level: "info"  # trace, debug, info, warn, error
    log_messages_to_file: true
    message_log_path: "messages.log"
```

## Directory Structure

```
faketandem/
├── main.go
├── go.mod
├── go.sum
├── README.md
├── SIMULATOR_IMPROVEMENT_PLAN.md (this file)
├── ARCHITECTURE.md
├── Makefile
├── config.yaml
├── profiles/
│   ├── x2.yaml
│   └── mobi.yaml
├── pkg/
│   ├── api/
│   │   └── server.go
│   ├── auth/
│   │   ├── jpake.go
│   │   ├── legacy.go
│   │   ├── signing.go
│   │   └── state.go
│   ├── bluetooth/
│   │   └── bluetooth.go
│   ├── config/
│   │   ├── config.go
│   │   └── profiles.go
│   ├── handler/
│   │   ├── router.go
│   │   ├── handler.go
│   │   ├── api_version.go
│   │   ├── auth_challenge.go
│   │   ├── current_status.go
│   │   ├── time.go
│   │   ├── bolus_*.go
│   │   ├── history.go
│   │   ├── settings.go
│   │   ├── profiles.go
│   │   └── alerts.go
│   ├── logging/
│   │   └── message_log.go
│   ├── protocol/
│   │   ├── packet.go
│   │   ├── reassembler.go
│   │   └── transaction.go
│   ├── pumpx2/
│   │   ├── bridge.go
│   │   ├── builder.go
│   │   └── types.go
│   ├── simulation/
│   │   ├── timing.go
│   │   └── errors.go
│   └── state/
│       ├── pump.go
│       ├── insulin.go
│       ├── settings.go
│       ├── alerts.go
│       ├── history.go
│       ├── simulator.go
│       └── persistence.go
├── test/
│   └── integration/
│       ├── auth_test.go
│       ├── bolus_test.go
│       └── status_test.go
└── vendor/
```

## Open Questions and Future Research

1. **Packet sizes:** Confirm exact packet sizes for each characteristic (18 vs 40 bytes)
2. **Message catalog:** Get complete list of message types from pumpX2
3. **Qualifying events:** Understand exact triggering conditions
4. **ControlStream vs Control:** Clarify when to use each
5. **History log format:** Understand pagination and encoding details
6. **Connection sharing:** Test exact behavior with multiple clients
7. **Firmware differences:** Document behavior differences across firmware versions
8. **Edge cases:** Identify edge cases in real pump behavior

## Notes for Implementation

1. Start with Phase 1-2 to build the foundation
2. Use pumpX2's cliparser extensively - don't reinvent the wheel
3. Test each phase thoroughly before moving to the next
4. Keep the implementation modular and testable
5. Document as you go
6. Use the WebSocket API for debugging and testing
7. Consider creating a companion web UI for visualization (future)
8. The simulator should be useful for:
   - Testing pumpX2 changes
   - Developing new pumpX2 features
   - Training on the protocol
   - Integration testing for apps using pumpX2

## References

- pumpX2 repository: https://github.com/jwoglom/pumpX2
- pumpX2 Bluetooth initialization docs: /docs/bluetooth-initialization.md
- pumpX2 mobile bolus docs: /mobile_bolus.md
- Tandem pump service UUID: 0000fdfb-0000-1000-8000-00805f9b34fb
