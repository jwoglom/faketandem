package state

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
)

// PumpState represents the current state of the simulated pump
type PumpState struct {
	// Identity
	SerialNumber    string
	Model           string
	FirmwareVersion string
	APIVersionMajor int
	APIVersionMinor int

	// Time
	TimeSinceReset uint32 // seconds since pump was turned on
	CurrentTime    time.Time
	StartTime      time.Time // When simulation started

	// Authentication
	AuthKey         []byte
	PairingCode     string
	IsAuthenticated bool

	// LongTermKey is the JPAKE-derived secret from a completed full pairing
	// (rounds 1a/1b/2/3/4). Real Tandem apps cache this on the phone and, on a
	// later BLE reconnect, skip straight to a "quick pair" that only re-runs
	// rounds 3/4 (fresh nonce + HKDF session key) against this cached secret
	// instead of repeating the full password-authenticated exchange. It
	// persists across BLE disconnects (unlike AuthKey, which is per-connection)
	// so the emulator can honor that quick-pair flow.
	LongTermKey []byte

	// Insulin Delivery
	Basal *BasalState
	Bolus *BolusState
	IOB   float64 // Insulin on board
	TDD   float64 // Total daily dose

	// Physical State
	Reservoir *ReservoirState
	Battery   *BatteryState
	Cartridge *CartridgeState

	// CGM
	CGM *CGMState

	// History Log
	HistoryLog *HistoryLogState

	// Pump mode
	PumpingSuspended bool
	// suspendReason records WHY delivery is suspended ("user", "occlusion",
	// "alarm", ...). A driver cannot see it directly, but it decides which
	// history records and qualifying events a suspend produces, and a harness
	// asserts on it as pump-side truth.
	suspendReason string
	ControlIQMode int // 0=Normal, 1=Sleep, 2=Exercise

	// ClosedLoopEnabled reports whether Control-IQ closed-loop control is on.
	// It defaults to FALSE: a driver that reads closedLoopEnabled=true refuses
	// to enact temp basals and manual boluses at all, so a pump that claims
	// closed loop by default makes most of the control surface untestable.
	ClosedLoopEnabled bool

	// Weight (kg) and TotalDailyInsulin (units) back the ControlIQInfo response.
	Weight            int
	TotalDailyInsulin int

	// LastBolus is the record of the most recently finished bolus.
	LastBolus *LastBolusRecord

	// Alerts/Alarms
	ActiveAlerts []Alert

	// AlarmBitmask is the pump's active-alarm set as AlarmStatusResponse
	// carries it: a 64-bit mask where bit N is the alarm whose
	// AlarmStatusResponse.AlarmResponseType raw value is N (bit 2 OCCLUSION_ALARM,
	// bit 3 PUMP_RESET_ALARM, bit 8 EMPTY_CARTRIDGE_ALARM, bit 18
	// RESUME_PUMP_ALARM, and so on). Alarms are pump-raised conditions that stop
	// or threaten insulin delivery, which is why they are modeled as a raw mask
	// a scenario can set directly rather than derived from ActiveAlerts.
	AlarmBitmask uint64

	// nextBolusID backs GetNextBolusID's monotonic allocator.
	nextBolusID uint32
	// nextTempRateID backs NextTempRateID's monotonic allocator.
	nextTempRateID int

	// clock is the source of every "now" this state derives (see Clock). It
	// has its own mutex so Now() can be called with the main state mutex
	// already held, which the simulator does on every tick.
	clock Clock
	// pumpClockOffset skews the pump's own clock away from the clock's time.
	// Every timestamp put on the wire in pump-epoch seconds is shifted by it,
	// while internal durations (bolus progress, temp-rate expiry) are not --
	// which is exactly how a real pump whose clock is set wrong behaves, and
	// what makes a driver's pump-time drift handling testable.
	pumpClockOffset time.Duration
	clockMtx        sync.RWMutex

	// BolusRateUnitsPerSecond is the speed the simulator delivers a bolus at.
	// Real pumps deliver far more slowly than the 0.05 U/s default (a Tandem
	// Mobi is roughly 1/28.7 U/s under 10 U), so a harness that wants driver
	// timing behavior to match hardware sets this explicitly.
	BolusRateUnitsPerSecond float64

	mutex sync.RWMutex
}

// BasalState represents basal delivery state
type BasalState struct {
	CurrentRate     float64 // units/hr
	TempBasalActive bool
	TempBasalRate   float64
	TempBasalEnd    time.Time

	// TempBasalPercent, TempBasalStart and TempRateID mirror what
	// SetTempRateRequest asked for, so TempRateResponse can report the temp
	// rate back to the driver instead of a static "inactive" constant.
	TempBasalPercent int
	TempBasalStart   time.Time
	TempRateID       int
}

// Bolus source IDs, matching pumpX2's BolusDeliveryHistoryLog.BolusSource
// ordinals. A driver uses this to tell a bolus it commanded itself from one a
// user programmed on the pump's own screen, so an app-commanded bolus must
// report BolusSourceBluetoothRemote and not the quickBolus default of 0.
const (
	// BolusSourceQuickBolus is a bolus started from the pump's quick-bolus buttons.
	BolusSourceQuickBolus = 0
	// BolusSourceBluetoothRemote is a bolus commanded by a paired app over BLE.
	BolusSourceBluetoothRemote = 8
)

// Bolus end-reason IDs, matching pumpX2's LastBolusStatusAbstractResponse.BolusStatus.
const (
	// BolusEndReasonCompleted marks a bolus that delivered its full requested volume.
	BolusEndReasonCompleted = 3
	// BolusEndReasonStopped marks a bolus cut short (canceled by the app or the user).
	BolusEndReasonStopped = 2
)

// BolusState represents active bolus state
type BolusState struct {
	Active         bool
	UnitsDelivered float64
	UnitsTotal     float64
	StartTime      time.Time
	BolusID        uint32

	// SourceID is the pumpX2 BolusSource ordinal for how this bolus was
	// commanded (see BolusSource* constants).
	SourceID int
	// TypeBitmask is the pumpX2 BolusType bitmask carried by InitiateBolusRequest.
	TypeBitmask int
	// FoodVolume and CorrectionVolume are the optional metadata split the
	// client attributed to carbs and to a correction, in units.
	FoodVolume       float64
	CorrectionVolume float64

	// Stalled freezes delivery progress without ending the bolus: the pump
	// still reports the bolus as in progress and still answers
	// CurrentBolusStatus for it, but no further insulin goes in. It models a
	// pump that has stopped making progress (occlusion detection pending, a
	// paused delivery) and gives a harness a way to hold a bolus open for as
	// long as a test needs.
	Stalled bool
	// StalledAt records when Stalled was set, so resuming can shift StartTime
	// forward by the stall duration and keep delivered-volume arithmetic
	// continuous.
	StalledAt time.Time
}

// LastBolusRecord is the pump's record of the most recently finished bolus,
// whether it completed or was cut short. It is what LastBolusStatus(V1/V2/V3)
// reports, and it is how a driver reconciles the bolus it commanded with what
// the pump actually delivered: it compares BolusID and reads DeliveredUnits and
// the end timestamp. Keeping it as live state (rather than the static zeros the
// generic settings table used to serve) is what makes a bolus initiated through
// InitiateBolusRequest observable afterwards.
type LastBolusRecord struct {
	BolusID           uint32
	RequestedUnits    float64
	DeliveredUnits    float64
	SourceID          int
	TypeBitmask       int
	EndReasonID       int
	EndTime           time.Time
	SecondsSinceReset uint32
	// Valid is false until a bolus has actually finished, so the response can
	// report "no last bolus" rather than a fabricated zeroed record.
	Valid bool
}

// ReservoirState represents reservoir state
type ReservoirState struct {
	CurrentUnits float64
	MaxUnits     float64
	LastFill     time.Time
}

// BatteryState represents battery state
type BatteryState struct {
	Percentage int
	Charging   bool
}

// CartridgeState represents cartridge/infusion set state
type CartridgeState struct {
	DaysSinceChange int
	LastPrime       time.Time
}

// CGMState represents CGM sensor state
type CGMState struct {
	SensorType    int    // CGM sensor type ordinal
	SessionActive bool   // Whether a CGM session is active
	CurrentEGV    int    // Current estimated glucose value (mg/dL)
	TransmitterID string // CGM transmitter ID
}

// HistoryLogEntry represents a single history log entry
type HistoryLogEntry struct {
	Sequence  uint32
	TypeID    int    // Numeric type ID matching pumpX2 history log types
	Type      string // Human-readable type name
	Timestamp time.Time
	// PumpTime is Timestamp expressed the way the wire format carries it:
	// seconds since the Tandem epoch (2008-01-01). Every history-log record
	// pumpX2 and TandemKit decode reads its pumpTimeSec field this way, so it
	// is captured at write time rather than converted at send time -- a later
	// controllable pump clock then cannot retroactively change the timestamp on
	// records already written.
	PumpTime uint32
	Data     map[string]interface{}
	// SourceNibble is the high nibble of the record's type-ID word. The driver
	// masks it off when reading the type, so it only matters when reproducing a
	// captured record byte-for-byte (Mobi captures carry 1, older ones 0).
	SourceNibble uint8
}

// HistoryLogState represents history log storage
type HistoryLogState struct {
	NextSequence uint32
	Entries      []HistoryLogEntry
	mutex        sync.Mutex
}

// Alert represents an alert or alarm
type Alert struct {
	ID           uint32
	Type         AlertType
	Priority     AlertPriority
	Message      string
	Timestamp    time.Time
	Acknowledged bool
}

// AlertType identifies the type of alert
type AlertType int

const (
	AlertLowReservoir AlertType = iota
	AlertLowBattery
	AlertCartridgeExpired
	AlertOcclusion
	AlertBasalSuspended
)

// AlertPriority indicates alert severity
type AlertPriority int

const (
	PriorityInfo AlertPriority = iota
	PriorityWarning
	PriorityCritical
)

// NewPumpState creates a new pump state with default values
func NewPumpState() *PumpState {
	ps := newDefaultPumpState(time.Now())
	ps.clock = RealClock{}
	ps.applyAPIVersionOverride()
	return ps
}

func newDefaultPumpState(now time.Time) *PumpState {
	return &PumpState{
		SerialNumber:    "11223344",
		Model:           "Tandem Mobi",
		FirmwareVersion: "1.0.0.0",
		// Defaults to the Tandem Mobi API version; see defaultAPIVersionMajor.
		// Configurable at startup via FAKETANDEM_API_VERSION and at runtime via
		// SetAPIVersion.
		APIVersionMajor: defaultAPIVersionMajor,
		APIVersionMinor: defaultAPIVersionMinor,

		TimeSinceReset: 0,
		CurrentTime:    now,
		StartTime:      now,

		PairingCode:     "123456", // Default 6-digit pairing code
		IsAuthenticated: false,

		Basal: &BasalState{
			CurrentRate:     0.85,
			TempBasalActive: false,
		},

		Bolus: &BolusState{
			Active: false,
		},

		LastBolus: &LastBolusRecord{},

		ClosedLoopEnabled: false,
		Weight:            70,

		BolusRateUnitsPerSecond: DefaultBolusRateUnitsPerSecond,
		TotalDailyInsulin:       40,

		IOB: 0.0,
		TDD: 0.0,

		Reservoir: &ReservoirState{
			CurrentUnits: 200.0,
			MaxUnits:     300.0,
			LastFill:     now,
		},

		Battery: &BatteryState{
			Percentage: 85,
			Charging:   false,
		},

		Cartridge: &CartridgeState{
			DaysSinceChange: 0,
			LastPrime:       now,
		},

		CGM: &CGMState{
			SensorType:    1, // Dexcom G6
			SessionActive: true,
			CurrentEGV:    120,
			TransmitterID: "80AB12",
		},

		HistoryLog: &HistoryLogState{
			NextSequence: 1,
			Entries:      make([]HistoryLogEntry, 0),
		},

		ActiveAlerts: make([]Alert, 0),
	}
}

// GetTimeSinceReset returns the current time since reset in seconds
func (ps *PumpState) GetTimeSinceReset() uint32 {
	ps.mutex.RLock()
	defer ps.mutex.RUnlock()

	return ps.TimeSinceReset
}

// UpdateTimeSinceReset updates the time since reset
func (ps *PumpState) UpdateTimeSinceReset() {
	ps.mutex.Lock()
	defer ps.mutex.Unlock()

	now := ps.Now()
	elapsed := now.Sub(ps.StartTime)
	ps.TimeSinceReset = uint32(elapsed.Seconds())
	ps.CurrentTime = now
}

// SetAuthenticated marks the pump as authenticated
func (ps *PumpState) SetAuthenticated(authKey []byte) {
	ps.mutex.Lock()
	defer ps.mutex.Unlock()

	ps.IsAuthenticated = true
	ps.AuthKey = authKey

	log.Info("Pump authenticated")
}

// ResetAuthentication clears authentication state
func (ps *PumpState) ResetAuthentication() {
	ps.mutex.Lock()
	defer ps.mutex.Unlock()

	ps.IsAuthenticated = false
	ps.AuthKey = nil

	log.Info("Pump authentication reset")
}

// SetPairingCode updates the pairing code
func (ps *PumpState) SetPairingCode(code string) {
	ps.mutex.Lock()
	defer ps.mutex.Unlock()

	ps.PairingCode = code
	// A new pairing code means a new JPAKE password, so any previously cached
	// long-term key is no longer valid for a quick-pair reconnect.
	ps.LongTermKey = nil
}

// GetLongTermKey returns the cached JPAKE long-term key, if any
func (ps *PumpState) GetLongTermKey() []byte {
	ps.mutex.RLock()
	defer ps.mutex.RUnlock()

	return ps.LongTermKey
}

// SetLongTermKey caches the JPAKE long-term key derived from a completed full
// pairing (or seeded via CLI flag), so later quick-pair reconnects can reuse it.
func (ps *PumpState) SetLongTermKey(key []byte) {
	ps.mutex.Lock()
	defer ps.mutex.Unlock()

	ps.LongTermKey = key
}

// GetAuthKey returns the authentication key
func (ps *PumpState) GetAuthKey() []byte {
	ps.mutex.RLock()
	defer ps.mutex.RUnlock()

	return ps.AuthKey
}

// GetPairingCode returns the pairing code
func (ps *PumpState) GetPairingCode() string {
	ps.mutex.RLock()
	defer ps.mutex.RUnlock()

	return ps.PairingCode
}

// GetAPIVersionMajor returns the API major version
func (ps *PumpState) GetAPIVersionMajor() int {
	ps.mutex.RLock()
	defer ps.mutex.RUnlock()

	return ps.APIVersionMajor
}

// GetAPIVersionMinor returns the API minor version
func (ps *PumpState) GetAPIVersionMinor() int {
	ps.mutex.RLock()
	defer ps.mutex.RUnlock()

	return ps.APIVersionMinor
}

// GetSerialNumber returns the serial number
func (ps *PumpState) GetSerialNumber() string {
	ps.mutex.RLock()
	defer ps.mutex.RUnlock()

	return ps.SerialNumber
}

// GetIOB returns the pump's current insulin-on-board estimate, in units.
func (ps *PumpState) GetIOB() float64 {
	ps.mutex.RLock()
	defer ps.mutex.RUnlock()
	return ps.IOB
}

// GetReservoirLevel returns the current reservoir level
func (ps *PumpState) GetReservoirLevel() float64 {
	ps.mutex.RLock()
	defer ps.mutex.RUnlock()

	return ps.Reservoir.CurrentUnits
}

// GetBatteryLevel returns the current battery percentage
func (ps *PumpState) GetBatteryLevel() int {
	ps.mutex.RLock()
	defer ps.mutex.RUnlock()

	return ps.Battery.Percentage
}

// GetBasalRate returns the current basal rate
func (ps *PumpState) GetBasalRate() float64 {
	ps.mutex.RLock()
	defer ps.mutex.RUnlock()

	if ps.Basal.TempBasalActive {
		return ps.Basal.TempBasalRate
	}
	return ps.Basal.CurrentRate
}

// GetNextBolusID allocates the bolus ID a BolusPermissionResponse hands out.
// A real pump returns a monotonically increasing counter that the following
// InitiateBolusRequest echoes back and that every later LastBolusStatus query
// is matched against; deriving it from the wall clock (as this used to) made
// two permission grants inside the same second collide and made the ID
// unrelated to anything the pump had actually recorded.
func (ps *PumpState) GetNextBolusID() uint32 {
	ps.mutex.Lock()
	defer ps.mutex.Unlock()

	ps.nextBolusID++
	return ps.nextBolusID
}

// PeekNextBolusID returns the ID GetNextBolusID last handed out, without
// allocating a new one.
func (ps *PumpState) PeekNextBolusID() uint32 {
	ps.mutex.RLock()
	defer ps.mutex.RUnlock()

	return ps.nextBolusID
}

// StartBolus starts a bolus delivery
func (ps *PumpState) StartBolus(units float64, bolusID uint32) {
	ps.StartBolusWithSource(units, bolusID, BolusSourceBluetoothRemote, 0)
}

// StartBolusWithSource starts a bolus delivery, recording how it was commanded.
func (ps *PumpState) StartBolusWithSource(units float64, bolusID uint32, sourceID, typeBitmask int) {
	ps.mutex.Lock()
	defer ps.mutex.Unlock()

	ps.Bolus.Active = true
	ps.Bolus.UnitsTotal = units
	ps.Bolus.UnitsDelivered = 0
	ps.Bolus.StartTime = ps.Now()
	ps.Bolus.BolusID = bolusID
	ps.Bolus.SourceID = sourceID
	ps.Bolus.TypeBitmask = typeBitmask
	ps.Bolus.Stalled = false
	ps.Bolus.StalledAt = time.Time{}

	// Keep the allocator ahead of any externally supplied ID so a later
	// permission grant can never reissue one already in use.
	if bolusID > ps.nextBolusID {
		ps.nextBolusID = bolusID
	}

	log.Infof("Started bolus: %.2f units, ID=%d, sourceId=%d", units, bolusID, sourceID)
}

// StopBolus stops an active bolus
func (ps *PumpState) StopBolus() {
	ps.mutex.Lock()
	defer ps.mutex.Unlock()

	if ps.Bolus.Active {
		log.Infof("Stopped bolus: delivered %.2f of %.2f units",
			ps.Bolus.UnitsDelivered, ps.Bolus.UnitsTotal)
		ps.Bolus.Active = false
	}
}

// UpdateBolusDelivery updates the bolus delivery progress
func (ps *PumpState) UpdateBolusDelivery(delivered float64) {
	ps.mutex.Lock()
	defer ps.mutex.Unlock()

	if ps.Bolus.Active {
		ps.Bolus.UnitsDelivered = delivered
		if ps.Bolus.UnitsDelivered >= ps.Bolus.UnitsTotal {
			ps.Bolus.Active = false
			log.Infof("Bolus complete: %.2f units delivered", ps.Bolus.UnitsDelivered)
		}
	}
}

// IsBolusActive returns true if a bolus is currently active
func (ps *PumpState) IsBolusActive() bool {
	ps.mutex.RLock()
	defer ps.mutex.RUnlock()

	return ps.Bolus.Active
}

// GetCGMSensorType returns the CGM sensor type
func (ps *PumpState) GetCGMSensorType() int {
	ps.mutex.RLock()
	defer ps.mutex.RUnlock()

	return ps.CGM.SensorType
}

// SetCGMSensorType sets the CGM sensor type
func (ps *PumpState) SetCGMSensorType(sensorType int) {
	ps.mutex.Lock()
	defer ps.mutex.Unlock()

	ps.CGM.SensorType = sensorType
	log.Infof("CGM sensor type set to %d", sensorType)
}

// GetCurrentEGV returns the current estimated glucose value
func (ps *PumpState) GetCurrentEGV() int {
	ps.mutex.RLock()
	defer ps.mutex.RUnlock()

	return ps.CGM.CurrentEGV
}

// GetHistoryLogCount returns the number of history log entries
func (ps *PumpState) GetHistoryLogCount() int {
	ps.HistoryLog.mutex.Lock()
	defer ps.HistoryLog.mutex.Unlock()

	return len(ps.HistoryLog.Entries)
}

// AddHistoryLogEntry adds a new history log entry.
// Uses its own mutex so it can be called while pumpState mutex is held.
func (ps *PumpState) AddHistoryLogEntry(entryType string, data map[string]interface{}) {
	ps.AddHistoryLogEntryWithTypeID(0, entryType, data)
}

// AddHistoryLogEntryWithTypeID adds a history log entry with a specific type ID,
// stamped with the pump's current clock. It is a thin wrapper over
// AppendHistory, which is the API to use when the timestamp or the assigned
// sequence number matters.
func (ps *PumpState) AddHistoryLogEntryWithTypeID(typeID int, entryType string, data map[string]interface{}) {
	ps.AddHistoryLogEntryAt(typeID, entryType, ps.Now(), data)
}

// AddHistoryLogEntryAt adds a history log entry stamped at a chosen instant on
// the pump's clock, returning its sequence number. Backdating is what lets a
// harness stage history that predates the connection -- a pump does not start
// its log when a phone shows up. The record itself is built by AppendHistory
// so the wire encoding and sequence allocation live in one place.
func (ps *PumpState) AddHistoryLogEntryAt(typeID int, entryType string, when time.Time, data map[string]interface{}) uint32 {
	entry := ps.AppendHistory(HistoryEvent{
		TypeID:   typeID,
		Name:     entryType,
		PumpTime: ps.PumpTimeFor(when),
		Fields:   data,
	})
	return entry.Sequence
}

// GetHistoryLogEntries returns history log entries in a sequence range
func (ps *PumpState) GetHistoryLogEntries(startSeq, endSeq uint32) []HistoryLogEntry {
	ps.HistoryLog.mutex.Lock()
	defer ps.HistoryLog.mutex.Unlock()

	var entries []HistoryLogEntry
	for _, entry := range ps.HistoryLog.Entries {
		if entry.Sequence >= startSeq && entry.Sequence <= endSeq {
			entries = append(entries, entry)
		}
	}
	return entries
}

// SetPumpingSuspended sets the pumping suspended state
func (ps *PumpState) SetPumpingSuspended(suspended bool) {
	ps.SetPumpingSuspendedWithReason(suspended, "")
}

// SetPumpingSuspendedWithReason sets the pumping suspended state and records
// why. Resuming clears the reason.
func (ps *PumpState) SetPumpingSuspendedWithReason(suspended bool, reason string) {
	ps.mutex.Lock()
	defer ps.mutex.Unlock()
	ps.PumpingSuspended = suspended
	if suspended {
		ps.suspendReason = reason
	} else {
		ps.suspendReason = ""
	}
}

// SuspendReason returns why delivery is suspended, or "" when it is not (or
// when the suspend came from a path that did not record a reason). Callers
// holding the state lock read the field through this only when they do not --
// it takes no lock of its own precisely so the snapshot can call it inside
// RLock.
func (ps *PumpState) SuspendReason() string {
	return ps.suspendReason
}

// IsPumpingSuspended returns whether pumping is suspended
func (ps *PumpState) IsPumpingSuspended() bool {
	ps.mutex.RLock()
	defer ps.mutex.RUnlock()
	return ps.PumpingSuspended
}

// Lock acquires the write lock on the pump state, for the handful of callers
// (the harness's direct field writes) that need to poke fields with no setter.
func (ps *PumpState) Lock() {
	ps.mutex.Lock()
}

// Unlock releases the write lock on the pump state.
func (ps *PumpState) Unlock() {
	ps.mutex.Unlock()
}

// RLock acquires a read lock on the pump state
func (ps *PumpState) RLock() {
	ps.mutex.RLock()
}

// RUnlock releases the read lock on the pump state
func (ps *PumpState) RUnlock() {
	ps.mutex.RUnlock()
}

// SetBasalState updates the basal state
func (ps *PumpState) SetBasalState(basal *BasalState) {
	ps.mutex.Lock()
	defer ps.mutex.Unlock()
	ps.Basal = basal
}

// SetReservoirLevel updates the reservoir level
func (ps *PumpState) SetReservoirLevel(units float64) {
	ps.mutex.Lock()
	defer ps.mutex.Unlock()
	ps.Reservoir.CurrentUnits = units
}

// SetBatteryLevel updates the battery percentage
func (ps *PumpState) SetBatteryLevel(pct int) {
	ps.mutex.Lock()
	defer ps.mutex.Unlock()
	ps.Battery.Percentage = pct
}

// AddAlert adds an alert to the active alerts list
func (ps *PumpState) AddAlert(alert Alert) {
	ps.mutex.Lock()
	defer ps.mutex.Unlock()
	ps.ActiveAlerts = append(ps.ActiveAlerts, alert)
}

// SetAlarmBitmask replaces the pump's active-alarm bitmask wholesale.
func (ps *PumpState) SetAlarmBitmask(mask uint64) {
	ps.mutex.Lock()
	defer ps.mutex.Unlock()
	ps.AlarmBitmask = mask
	log.Infof("Alarm bitmask set to 0x%016x", mask)
}

// SetAlarm raises (active) or clears a single alarm bit. bit is the
// AlarmResponseType raw value, 0-63; out-of-range bits are ignored.
func (ps *PumpState) SetAlarm(bit uint, active bool) {
	if bit > 63 {
		log.Warnf("Ignoring out-of-range alarm bit %d", bit)
		return
	}

	ps.mutex.Lock()
	defer ps.mutex.Unlock()
	if active {
		ps.AlarmBitmask |= uint64(1) << bit
	} else {
		ps.AlarmBitmask &^= uint64(1) << bit
	}
}

// GetAlarmBitmask returns the pump's active-alarm bitmask.
func (ps *PumpState) GetAlarmBitmask() uint64 {
	ps.mutex.RLock()
	defer ps.mutex.RUnlock()
	return ps.AlarmBitmask
}

// SetControlIQMode sets the ControlIQ mode
func (ps *PumpState) SetControlIQMode(mode int) {
	ps.mutex.Lock()
	defer ps.mutex.Unlock()
	ps.ControlIQMode = mode
	log.Infof("ControlIQ mode set to %d", mode)
}

// GetControlIQMode returns the ControlIQ mode
func (ps *PumpState) GetControlIQMode() int {
	ps.mutex.RLock()
	defer ps.mutex.RUnlock()
	return ps.ControlIQMode
}

// RecordLastBolus stores the record of a bolus that has just finished, so
// later LastBolusStatus(V1/V2/V3) queries can report it. Callers must NOT hold
// the pump state mutex.
func (ps *PumpState) RecordLastBolus(record LastBolusRecord) {
	ps.mutex.Lock()
	defer ps.mutex.Unlock()

	record.Valid = true
	if record.EndTime.IsZero() {
		record.EndTime = ps.Now()
	}
	record.SecondsSinceReset = ps.TimeSinceReset
	ps.LastBolus = &record

	log.Infof("Recorded last bolus: ID=%d delivered=%.2f of %.2f units, endReason=%d",
		record.BolusID, record.DeliveredUnits, record.RequestedUnits, record.EndReasonID)
}

// GetLastBolus returns a copy of the last finished bolus record. The second
// result is false when no bolus has finished yet.
func (ps *PumpState) GetLastBolus() (LastBolusRecord, bool) {
	ps.mutex.RLock()
	defer ps.mutex.RUnlock()

	if ps.LastBolus == nil || !ps.LastBolus.Valid {
		return LastBolusRecord{}, false
	}
	return *ps.LastBolus, true
}

// TempRateSnapshot is a point-in-time view of the temp basal, for building a
// TempRateResponse without holding the pump state lock across an encode.
type TempRateSnapshot struct {
	Active     bool
	Percent    int
	Rate       float64
	StartTime  time.Time
	EndTime    time.Time
	TempRateID int
}

// GetTempRate returns the current temp basal state.
func (ps *PumpState) GetTempRate() TempRateSnapshot {
	ps.mutex.RLock()
	defer ps.mutex.RUnlock()

	return TempRateSnapshot{
		Active:     ps.Basal.TempBasalActive,
		Percent:    ps.Basal.TempBasalPercent,
		Rate:       ps.Basal.TempBasalRate,
		StartTime:  ps.Basal.TempBasalStart,
		EndTime:    ps.Basal.TempBasalEnd,
		TempRateID: ps.Basal.TempRateID,
	}
}

// GetProfileBasalRate returns the scheduled (profile) basal rate, ignoring any
// temp rate currently in effect. Unlike GetBasalRate this is what a percentage
// temp rate must be applied to: multiplying the *current* rate compounds
// percentages when one temp rate replaces another.
func (ps *PumpState) GetProfileBasalRate() float64 {
	ps.mutex.RLock()
	defer ps.mutex.RUnlock()

	return ps.Basal.CurrentRate
}

// SetAPIVersion overrides the API version reported by ApiVersionResponse.
func (ps *PumpState) SetAPIVersion(major, minor int) {
	ps.mutex.Lock()
	defer ps.mutex.Unlock()

	ps.APIVersionMajor = major
	ps.APIVersionMinor = minor
	log.Infof("Pump API version set to %d.%d", major, minor)
}

// IsClosedLoopEnabled reports whether Control-IQ closed-loop control is on.
func (ps *PumpState) IsClosedLoopEnabled() bool {
	ps.mutex.RLock()
	defer ps.mutex.RUnlock()

	return ps.ClosedLoopEnabled
}

// SetClosedLoopEnabled turns Control-IQ closed-loop control on or off.
func (ps *PumpState) SetClosedLoopEnabled(enabled bool) {
	ps.mutex.Lock()
	defer ps.mutex.Unlock()

	ps.ClosedLoopEnabled = enabled
	log.Infof("Control-IQ closed loop set to %v", enabled)
}

// ControlIQSnapshot is a point-in-time view of the Control-IQ related state
// that ControlIQInfoV1/V2 responses report.
type ControlIQSnapshot struct {
	ClosedLoopEnabled   bool
	Weight              int
	TotalDailyInsulin   int
	CurrentUserModeType int
}

// GetControlIQInfo returns the Control-IQ state backing ControlIQInfo responses.
func (ps *PumpState) GetControlIQInfo() ControlIQSnapshot {
	ps.mutex.RLock()
	defer ps.mutex.RUnlock()

	return ControlIQSnapshot{
		ClosedLoopEnabled:   ps.ClosedLoopEnabled,
		Weight:              ps.Weight,
		TotalDailyInsulin:   ps.TotalDailyInsulin,
		CurrentUserModeType: ps.ControlIQMode,
	}
}

// NextTempRateID allocates the ID reported by SetTempRateResponse and echoed by
// the matching TempRateActivated history-log record. A real pump increments
// this per temp rate; it was previously hard-coded to 1, so a driver could not
// tell one temp rate from the next.
func (ps *PumpState) NextTempRateID() int {
	ps.mutex.Lock()
	defer ps.mutex.Unlock()

	ps.nextTempRateID++
	return ps.nextTempRateID
}

// GetHistoryLogSequenceRange returns the first and last sequence numbers
// currently held in the history log. Both are 0 when the log is empty.
func (ps *PumpState) GetHistoryLogSequenceRange() (first, last uint32) {
	ps.HistoryLog.mutex.Lock()
	defer ps.HistoryLog.mutex.Unlock()

	if len(ps.HistoryLog.Entries) == 0 {
		return 0, 0
	}
	return ps.HistoryLog.Entries[0].Sequence, ps.HistoryLog.Entries[len(ps.HistoryLog.Entries)-1].Sequence
}

// Default API version reported by ApiVersionResponse. 3.5 is the Tandem Mobi
// initial-release API. It is chosen to match the captured Mobi
// PumpVersionResponse fixture this emulator serves (modelNum 1004000): a
// driver combines pump model and API version to decide which message variants
// a pump supports, and the Mobi-only paths (LastBolusStatusV3, SetTempRate,
// StopTempRate) all declare minApi 3.5. Reporting the t:slim X2's 2.5 while
// claiming a Mobi model number produced an incoherent pump -- the driver took
// the legacy V2 last-bolus path while still sending Mobi-only control
// messages.
const (
	defaultAPIVersionMajor = 3
	defaultAPIVersionMinor = 5
)

// apiVersionEnvVar overrides the default API version at startup, as
// "<major>.<minor>" (e.g. "2.5" to emulate a t:slim X2 on software v7.6).
// Deliberately an environment variable rather than a CLI flag so the emulator's
// entry point does not have to change.
const apiVersionEnvVar = "FAKETANDEM_API_VERSION"

// applyAPIVersionOverride applies the FAKETANDEM_API_VERSION override, if set.
// A malformed value is logged and ignored rather than being fatal: an
// unparseable version should not stop the emulator from starting.
func (ps *PumpState) applyAPIVersionOverride() {
	raw := strings.TrimSpace(os.Getenv(apiVersionEnvVar))
	if raw == "" {
		return
	}

	major, minor, err := parseAPIVersion(raw)
	if err != nil {
		log.Warnf("Ignoring invalid %s=%q: %v", apiVersionEnvVar, raw, err)
		return
	}

	ps.APIVersionMajor = major
	ps.APIVersionMinor = minor
	log.Infof("Pump API version overridden by %s: %d.%d", apiVersionEnvVar, major, minor)
}

// parseAPIVersion parses a "<major>.<minor>" API version string.
func parseAPIVersion(raw string) (major, minor int, err error) {
	parts := strings.SplitN(raw, ".", 2)
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("expected \"<major>.<minor>\"")
	}
	major, err = strconv.Atoi(strings.TrimSpace(parts[0]))
	if err != nil {
		return 0, 0, fmt.Errorf("invalid major version: %w", err)
	}
	minor, err = strconv.Atoi(strings.TrimSpace(parts[1]))
	if err != nil {
		return 0, 0, fmt.Errorf("invalid minor version: %w", err)
	}
	if major < 0 || minor < 0 {
		return 0, 0, fmt.Errorf("version components must not be negative")
	}
	return major, minor, nil
}
