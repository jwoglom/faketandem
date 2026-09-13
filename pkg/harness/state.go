package harness

import (
	"net/http"
	"time"

	"github.com/jwoglom/faketandem/pkg/bluetooth"
	"github.com/jwoglom/faketandem/pkg/state"

	log "github.com/sirupsen/logrus"
)

// historySnapshotLimit bounds how many history-log records the snapshot
// carries. A scenario asserting on history reads the tail; the full log is
// available through the protocol's own HistoryLog messages.
const historySnapshotLimit = 50

// Snapshot is the harness's view of the whole pump: everything a driver could
// observe through the protocol, plus the pump-side truth behind it.
type Snapshot struct {
	Identity   identitySnapshot   `json:"identity"`
	Clock      clockBody          `json:"clock"`
	Auth       authSnapshot       `json:"auth"`
	Connection connectionSnapshot `json:"connection"`
	Basal      basalSnapshot      `json:"basal"`
	Bolus      bolusSnapshot      `json:"bolus"`
	LastBolus  *lastBolusSnapshot `json:"last_bolus"`
	ControlIQ  controlIQSnapshot  `json:"control_iq"`
	Insulin    insulinSnapshot    `json:"insulin"`
	Battery    batterySnapshot    `json:"battery"`
	CGM        cgmSnapshot        `json:"cgm"`
	Alerts     []alertSnapshot    `json:"alerts"`
	// AlarmsHistory is every alarm that has been raised and cleared, with the
	// instant it was cleared. Alerts only ever reports what is standing now, so
	// without this an alarm that came and went left nothing a timeline could
	// place an interval from.
	AlarmsHistory []clearedAlarmSnapshot `json:"alarms_history"`
	History       historySnapshot        `json:"history"`
	RequestLog requestLogSnapshot `json:"request_log"`
}

type identitySnapshot struct {
	SerialNumber    string `json:"serial_number"`
	Model           string `json:"model"`
	FirmwareVersion string `json:"firmware_version"`
	APIVersionMajor int    `json:"api_version_major"`
	APIVersionMinor int    `json:"api_version_minor"`
}

type authSnapshot struct {
	Authenticated      bool   `json:"authenticated"`
	PairingCode        string `json:"pairing_code"`
	LongTermKeyPresent bool   `json:"long_term_key_present"`
	PairingState       string `json:"pairing_state"`
}

type connectionSnapshot struct {
	Connected bool `json:"connected"`
	// RadioEnabled is reported only for transports that can switch their radio
	// (the virtual link); it reads true for the real BLE transport.
	RadioEnabled bool `json:"radio_enabled"`
}

type basalSnapshot struct {
	ProfileRate      float64 `json:"profile_rate"`
	CurrentRate      float64 `json:"current_rate"`
	Suspended        bool    `json:"suspended"`
	SuspendReason    string  `json:"suspend_reason,omitempty"`
	TempActive       bool    `json:"temp_active"`
	TempRate         float64 `json:"temp_rate"`
	TempPercent      int     `json:"temp_percent"`
	TempStart        string  `json:"temp_start,omitempty"`
	TempEnd          string  `json:"temp_end,omitempty"`
	TempRateID       int     `json:"temp_rate_id"`
	TempPumpTimeSecs uint32  `json:"temp_start_pump_seconds,omitempty"`
}

type bolusSnapshot struct {
	Active             bool    `json:"active"`
	Stalled            bool    `json:"stalled"`
	BolusID            uint32  `json:"bolus_id"`
	UnitsTotal         float64 `json:"units_total"`
	UnitsDelivered     float64 `json:"units_delivered"`
	SourceID           int     `json:"source_id"`
	TypeBitmask        int     `json:"type_bitmask"`
	StartTime          string  `json:"start_time,omitempty"`
	StartPumpSeconds   uint32  `json:"start_pump_seconds,omitempty"`
	RateUnitsPerSecond float64 `json:"rate_units_per_second"`
}

type lastBolusSnapshot struct {
	BolusID           uint32  `json:"bolus_id"`
	RequestedUnits    float64 `json:"requested_units"`
	DeliveredUnits    float64 `json:"delivered_units"`
	SourceID          int     `json:"source_id"`
	TypeBitmask       int     `json:"type_bitmask"`
	EndReasonID       int     `json:"end_reason_id"`
	EndTime           string  `json:"end_time"`
	EndPumpSeconds    uint32  `json:"end_pump_seconds"`
	SecondsSinceReset uint32  `json:"seconds_since_reset"`
}

type controlIQSnapshot struct {
	ClosedLoopEnabled bool `json:"closed_loop_enabled"`
	Mode              int  `json:"mode"`
	Weight            int  `json:"weight"`
	TotalDailyInsulin int  `json:"total_daily_insulin"`
}

type insulinSnapshot struct {
	ReservoirUnits float64 `json:"reservoir_units"`
	ReservoirMax   float64 `json:"reservoir_max"`
	IOB            float64 `json:"iob"`
	TDD            float64 `json:"tdd"`
}

type batterySnapshot struct {
	Percentage int  `json:"percentage"`
	Charging   bool `json:"charging"`
}

type cgmSnapshot struct {
	SensorType    int    `json:"sensor_type"`
	SessionActive bool   `json:"session_active"`
	CurrentEGV    int    `json:"current_egv"`
	TransmitterID string `json:"transmitter_id"`
}

type alertSnapshot struct {
	ID       uint32 `json:"id"`
	Type     int    `json:"type"`
	TypeName string `json:"type_name"`
	Priority int    `json:"priority"`
	Message  string `json:"message"`
	// Time is the true instant the alarm was raised, on the pump's Clock;
	// PumpSeconds is the same instant as the wire carries it.
	Time         string `json:"time"`
	PumpSeconds  uint32 `json:"pump_seconds"`
	Acknowledged bool   `json:"acknowledged"`
}

// clearedAlarmSnapshot is an alarm that stood and has since been cleared, with
// both ends of the interval.
type clearedAlarmSnapshot struct {
	ID                 uint32 `json:"id"`
	Type               int    `json:"type"`
	TypeName           string `json:"type_name"`
	Priority           int    `json:"priority"`
	Message            string `json:"message"`
	Time               string `json:"time"`
	PumpSeconds        uint32 `json:"pump_seconds"`
	ClearedTime        string `json:"cleared_time"`
	ClearedPumpSeconds uint32 `json:"cleared_pump_seconds"`
}

type requestLogSnapshot struct {
	Entries int `json:"entries"`
	LastSeq int `json:"last_seq"`
}

func (h *Harness) handleState(w http.ResponseWriter, r *http.Request) {
	if h.pumpState == nil {
		writeError(w, http.StatusInternalServerError, "no pump state attached")
		return
	}

	switch r.Method {
	case http.MethodGet:
		writeJSON(w, http.StatusOK, h.Snapshot())

	case http.MethodPut, http.MethodPatch:
		var body stateUpdate
		if err := decodeBody(r, &body); err != nil {
			writeError(w, http.StatusBadRequest, "%v", err)
			return
		}
		applied := h.applyStateUpdate(body)
		log.Infof("harness: applied state update: %v", applied)
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"applied": applied,
			"state":   h.Snapshot(),
		})

	default:
		writeError(w, http.StatusMethodNotAllowed, "method %s not allowed on /api/state", r.Method)
	}
}

// Snapshot builds the full pump-state view.
func (h *Harness) Snapshot() Snapshot {
	ps := h.pumpState

	ps.RLock()
	identity := identitySnapshot{
		SerialNumber:    ps.SerialNumber,
		Model:           ps.Model,
		FirmwareVersion: ps.FirmwareVersion,
		APIVersionMajor: ps.APIVersionMajor,
		APIVersionMinor: ps.APIVersionMinor,
	}
	basal := *ps.Basal
	bolus := *ps.Bolus
	suspended := ps.PumpingSuspended
	suspendReason := ps.SuspendReason()
	insulin := insulinSnapshot{
		ReservoirUnits: ps.Reservoir.CurrentUnits,
		ReservoirMax:   ps.Reservoir.MaxUnits,
		IOB:            ps.IOB,
		TDD:            ps.TDD,
	}
	battery := batterySnapshot{Percentage: ps.Battery.Percentage, Charging: ps.Battery.Charging}
	cgm := cgmSnapshot{
		SensorType:    ps.CGM.SensorType,
		SessionActive: ps.CGM.SessionActive,
		CurrentEGV:    ps.CGM.CurrentEGV,
		TransmitterID: ps.CGM.TransmitterID,
	}
	bolusRate := ps.BolusRateUnitsPerSecond
	authenticated := ps.IsAuthenticated
	pairingCode := ps.PairingCode
	hasLTK := len(ps.LongTermKey) > 0
	activeAlerts := make([]state.Alert, len(ps.ActiveAlerts))
	copy(activeAlerts, ps.ActiveAlerts)
	ps.RUnlock()

	alerts := make([]alertSnapshot, 0, len(activeAlerts))
	for _, a := range activeAlerts {
		alerts = append(alerts, alertSnapshot{
			ID:           a.ID,
			Type:         int(a.Type),
			TypeName:     a.Type.String(),
			Priority:     int(a.Priority),
			Message:      a.Message,
			Time:         formatTime(a.Timestamp),
			PumpSeconds:  pumpTimeOrZero(ps, a.Timestamp),
			Acknowledged: a.Acknowledged,
		})
	}

	clearedAlarms := make([]clearedAlarmSnapshot, 0)
	for _, c := range ps.GetClearedAlerts() {
		clearedAlarms = append(clearedAlarms, clearedAlarmSnapshot{
			ID:                 c.ID,
			Type:               int(c.Type),
			TypeName:           c.Type.String(),
			Priority:           int(c.Priority),
			Message:            c.Message,
			Time:               formatTime(c.Timestamp),
			PumpSeconds:        pumpTimeOrZero(ps, c.Timestamp),
			ClearedTime:        formatTime(c.ClearedAt),
			ClearedPumpSeconds: pumpTimeOrZero(ps, c.ClearedAt),
		})
	}

	controlIQ := ps.GetControlIQInfo()
	lastBolus := h.lastBolusSnapshot()

	return Snapshot{
		Identity: identity,
		Clock:    h.clockSnapshot(),
		Auth: authSnapshot{
			Authenticated:      authenticated,
			PairingCode:        pairingCode,
			LongTermKeyPresent: hasLTK,
			PairingState:       h.pairingState(),
		},
		Connection: h.connectionSnapshot(),
		Basal: basalSnapshot{
			ProfileRate:      basal.CurrentRate,
			CurrentRate:      currentBasalRate(basal, suspended),
			Suspended:        suspended,
			SuspendReason:    suspendReason,
			TempActive:       basal.TempBasalActive,
			TempRate:         basal.TempBasalRate,
			TempPercent:      basal.TempBasalPercent,
			TempStart:        formatTime(basal.TempBasalStart),
			TempEnd:          formatTime(basal.TempBasalEnd),
			TempRateID:       basal.TempRateID,
			TempPumpTimeSecs: pumpTimeOrZero(ps, basal.TempBasalStart),
		},
		Bolus: bolusSnapshot{
			Active:             bolus.Active,
			Stalled:            bolus.Stalled,
			BolusID:            bolus.BolusID,
			UnitsTotal:         bolus.UnitsTotal,
			UnitsDelivered:     bolus.UnitsDelivered,
			SourceID:           bolus.SourceID,
			TypeBitmask:        bolus.TypeBitmask,
			StartTime:          formatTime(bolus.StartTime),
			StartPumpSeconds:   pumpTimeOrZero(ps, bolus.StartTime),
			RateUnitsPerSecond: bolusRate,
		},
		LastBolus: lastBolus,
		ControlIQ: controlIQSnapshot{
			ClosedLoopEnabled: controlIQ.ClosedLoopEnabled,
			Mode:              controlIQ.CurrentUserModeType,
			Weight:            controlIQ.Weight,
			TotalDailyInsulin: controlIQ.TotalDailyInsulin,
		},
		Insulin:    insulin,
		Battery:    battery,
		CGM:        cgm,
		Alerts:        alerts,
		AlarmsHistory: clearedAlarms,
		History:       h.historySnapshot(),
		RequestLog:    h.requestLogSnapshot(),
	}
}

// currentBasalRate is the rate actually being delivered, which is what
// CurrentBasalStatusResponse reports: zero while suspended, the temp rate
// while one runs, otherwise the profile rate.
func currentBasalRate(basal state.BasalState, suspended bool) float64 {
	switch {
	case suspended:
		return 0
	case basal.TempBasalActive:
		return basal.TempBasalRate
	default:
		return basal.CurrentRate
	}
}

func (h *Harness) lastBolusSnapshot() *lastBolusSnapshot {
	record, ok := h.pumpState.GetLastBolus()
	if !ok {
		return nil
	}
	return &lastBolusSnapshot{
		BolusID:           record.BolusID,
		RequestedUnits:    record.RequestedUnits,
		DeliveredUnits:    record.DeliveredUnits,
		SourceID:          record.SourceID,
		TypeBitmask:       record.TypeBitmask,
		EndReasonID:       record.EndReasonID,
		EndTime:           formatTime(record.EndTime),
		EndPumpSeconds:    h.pumpState.PumpTimeFor(record.EndTime),
		SecondsSinceReset: record.SecondsSinceReset,
	}
}

func (h *Harness) historySnapshot() historySnapshot {
	first, last := h.pumpState.GetHistoryLogSequenceRange()
	count := h.pumpState.GetHistoryLogCount()

	from := uint32(0)
	if last > historySnapshotLimit {
		from = last - historySnapshotLimit + 1
	}
	entries := h.pumpState.GetHistoryLogEntries(from, last)

	return historySnapshot{
		Count:         count,
		FirstSequence: first,
		LastSequence:  last,
		Entries:       historyEntrySnapshots(entries),
	}
}

func (h *Harness) requestLogSnapshot() requestLogSnapshot {
	if h.requests == nil {
		return requestLogSnapshot{}
	}
	return requestLogSnapshot{Entries: h.requests.Len(), LastSeq: h.requests.LastSeq()}
}

func (h *Harness) connectionSnapshot() connectionSnapshot {
	if h.transport == nil {
		return connectionSnapshot{RadioEnabled: true}
	}
	radio := true
	if rc, ok := h.transport.(bluetooth.RadioController); ok {
		radio = rc.RadioEnabled()
	}
	return connectionSnapshot{Connected: h.transport.IsConnected(), RadioEnabled: radio}
}

func (h *Harness) pairingState() string {
	if h.transport == nil {
		return ""
	}
	return string(h.transport.GetPairingState())
}

// stateUpdate is the PUT/PATCH body: every field is optional, and only those
// present are applied. Pointers rather than values, so "set the battery to 0"
// is distinguishable from "do not touch the battery".
type stateUpdate struct {
	ReservoirUnits    *float64 `json:"reservoir_units,omitempty"`
	BatteryPercent    *int     `json:"battery_percent,omitempty"`
	BatteryCharging   *bool    `json:"battery_charging,omitempty"`
	BasalRate         *float64 `json:"basal_rate,omitempty"`
	Suspended         *bool    `json:"suspended,omitempty"`
	IOB               *float64 `json:"iob,omitempty"`
	TDD               *float64 `json:"tdd,omitempty"`
	ClosedLoopEnabled *bool    `json:"closed_loop_enabled,omitempty"`
	ControlIQMode     *int     `json:"control_iq_mode,omitempty"`
	Weight            *int     `json:"weight,omitempty"`
	TotalDailyInsulin *int     `json:"total_daily_insulin,omitempty"`
	BolusRate         *float64 `json:"bolus_rate_units_per_second,omitempty"`
	CGMEGV            *int     `json:"cgm_egv,omitempty"`
	CGMSessionActive  *bool    `json:"cgm_session_active,omitempty"`
	PairingCode       *string  `json:"pairing_code,omitempty"`
	TimeSinceReset    *uint32  `json:"time_since_reset,omitempty"`
	APIVersionMajor   *int     `json:"api_version_major,omitempty"`
	APIVersionMinor   *int     `json:"api_version_minor,omitempty"`
	ClearAlerts       *bool    `json:"clear_alerts,omitempty"`
}

// applyStateUpdate writes the named fields and returns the names it applied.
//
// Deliberately dumb: it sets exactly what it is told and raises no qualifying
// events. Setting a field is for staging a starting position (a pump that was
// already suspended before the driver connected); the action endpoints exist
// for changes that have to look to the driver like the pump doing something.
func (h *Harness) applyStateUpdate(u stateUpdate) []string {
	ps := h.pumpState
	var applied []string

	if u.ReservoirUnits != nil {
		ps.SetReservoirLevel(*u.ReservoirUnits)
		applied = append(applied, "reservoir_units")
	}
	if u.BatteryPercent != nil {
		ps.SetBatteryLevel(*u.BatteryPercent)
		applied = append(applied, "battery_percent")
	}
	if u.BasalRate != nil {
		basal := ps.GetTempRate()
		ps.SetBasalState(&state.BasalState{
			CurrentRate:      *u.BasalRate,
			TempBasalActive:  basal.Active,
			TempBasalRate:    basal.Rate,
			TempBasalEnd:     basal.EndTime,
			TempBasalPercent: basal.Percent,
			TempBasalStart:   basal.StartTime,
			TempRateID:       basal.TempRateID,
		})
		applied = append(applied, "basal_rate")
	}
	if u.Suspended != nil {
		// Staging only: the flag moves, but nothing is ended and no
		// PumpingSuspended/PumpingResumed record is written. POST
		// /api/state/suspend and /api/state/resume are the transitions.
		ps.SetPumpingSuspended(*u.Suspended)
		applied = append(applied, "suspended")
	}
	if u.ClosedLoopEnabled != nil {
		ps.SetClosedLoopEnabled(*u.ClosedLoopEnabled)
		applied = append(applied, "closed_loop_enabled")
	}
	if u.ControlIQMode != nil {
		ps.SetControlIQMode(*u.ControlIQMode)
		applied = append(applied, "control_iq_mode")
	}
	if u.BolusRate != nil {
		ps.SetBolusRate(*u.BolusRate)
		applied = append(applied, "bolus_rate_units_per_second")
	}
	if u.PairingCode != nil {
		ps.SetPairingCode(*u.PairingCode)
		applied = append(applied, "pairing_code")
	}
	if u.APIVersionMajor != nil || u.APIVersionMinor != nil {
		major, minor := ps.GetAPIVersionMajor(), ps.GetAPIVersionMinor()
		if u.APIVersionMajor != nil {
			major = *u.APIVersionMajor
		}
		if u.APIVersionMinor != nil {
			minor = *u.APIVersionMinor
		}
		ps.SetAPIVersion(major, minor)
		applied = append(applied, "api_version")
	}

	if u.ClearAlerts != nil && *u.ClearAlerts {
		// Staged, not enacted: PUT /api/state writes no history, so the alarms
		// move into the cleared-alarm log without an AlarmCleared record.
		// POST /api/state/resume is the path that records the clearing.
		ps.ClearAlerts(ps.Now())
		applied = append(applied, "clear_alerts")
	}

	applied = append(applied, h.applyDirectFields(u)...)
	return applied
}

// applyDirectFields writes the fields that have no setter of their own and are
// poked straight into the struct under the state lock.
func (h *Harness) applyDirectFields(u stateUpdate) []string {
	ps := h.pumpState
	var applied []string

	ps.Lock()
	defer ps.Unlock()

	if u.BatteryCharging != nil {
		ps.Battery.Charging = *u.BatteryCharging
		applied = append(applied, "battery_charging")
	}
	if u.IOB != nil {
		ps.IOB = *u.IOB
		applied = append(applied, "iob")
	}
	if u.TDD != nil {
		ps.TDD = *u.TDD
		applied = append(applied, "tdd")
	}
	if u.Weight != nil {
		ps.Weight = *u.Weight
		applied = append(applied, "weight")
	}
	if u.TotalDailyInsulin != nil {
		ps.TotalDailyInsulin = *u.TotalDailyInsulin
		applied = append(applied, "total_daily_insulin")
	}
	if u.CGMEGV != nil {
		ps.CGM.CurrentEGV = *u.CGMEGV
		applied = append(applied, "cgm_egv")
	}
	if u.CGMSessionActive != nil {
		ps.CGM.SessionActive = *u.CGMSessionActive
		applied = append(applied, "cgm_session_active")
	}
	if u.TimeSinceReset != nil {
		ps.TimeSinceReset = *u.TimeSinceReset
		ps.StartTime = ps.Now().Add(-time.Duration(*u.TimeSinceReset) * time.Second)
		applied = append(applied, "time_since_reset")
	}
	return applied
}

// formatTime renders a time for the snapshot, or "" when it is unset.
func formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339Nano)
}

// pumpTimeOrZero converts an instant to pump-epoch seconds, or 0 when unset.
func pumpTimeOrZero(ps *state.PumpState, t time.Time) uint32 {
	if t.IsZero() {
		return 0
	}
	return ps.PumpTimeFor(t)
}
