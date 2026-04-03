package state

import (
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
	ControlIQMode    int // 0=Normal, 1=Sleep, 2=Exercise

	// Alerts/Alarms
	ActiveAlerts []Alert

	mutex sync.RWMutex
}

// BasalState represents basal delivery state
type BasalState struct {
	CurrentRate     float64 // units/hr
	TempBasalActive bool
	TempBasalRate   float64
	TempBasalEnd    time.Time
}

// BolusState represents active bolus state
type BolusState struct {
	Active         bool
	UnitsDelivered float64
	UnitsTotal     float64
	StartTime      time.Time
	BolusID        uint32
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
	Data      map[string]interface{}
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
	now := time.Now()

	return &PumpState{
		SerialNumber:    "11223344",
		Model:           "t:slim X2",
		FirmwareVersion: "7.6.0.0",
		APIVersionMajor: 2,
		APIVersionMinor: 5,

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

	elapsed := time.Since(ps.StartTime)
	ps.TimeSinceReset = uint32(elapsed.Seconds())
	ps.CurrentTime = time.Now()
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

// GetNextBolusID returns the next bolus ID
func (ps *PumpState) GetNextBolusID() uint32 {
	ps.mutex.RLock()
	defer ps.mutex.RUnlock()

	// Simple incrementing ID based on time
	return uint32(time.Now().Unix() % 1000000)
}

// StartBolus starts a bolus delivery
func (ps *PumpState) StartBolus(units float64, bolusID uint32) {
	ps.mutex.Lock()
	defer ps.mutex.Unlock()

	ps.Bolus.Active = true
	ps.Bolus.UnitsTotal = units
	ps.Bolus.UnitsDelivered = 0
	ps.Bolus.StartTime = time.Now()
	ps.Bolus.BolusID = bolusID

	log.Infof("Started bolus: %.2f units, ID=%d", units, bolusID)
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

// AddHistoryLogEntryWithTypeID adds a history log entry with a specific type ID.
func (ps *PumpState) AddHistoryLogEntryWithTypeID(typeID int, entryType string, data map[string]interface{}) {
	ps.HistoryLog.mutex.Lock()
	defer ps.HistoryLog.mutex.Unlock()

	entry := HistoryLogEntry{
		Sequence:  ps.HistoryLog.NextSequence,
		TypeID:    typeID,
		Type:      entryType,
		Timestamp: time.Now(),
		Data:      data,
	}
	ps.HistoryLog.Entries = append(ps.HistoryLog.Entries, entry)
	ps.HistoryLog.NextSequence++
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
	ps.mutex.Lock()
	defer ps.mutex.Unlock()
	ps.PumpingSuspended = suspended
}

// IsPumpingSuspended returns whether pumping is suspended
func (ps *PumpState) IsPumpingSuspended() bool {
	ps.mutex.RLock()
	defer ps.mutex.RUnlock()
	return ps.PumpingSuspended
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
