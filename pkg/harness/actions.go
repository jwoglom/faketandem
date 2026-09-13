package harness

import (
	"fmt"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/jwoglom/faketandem/pkg/state"

	log "github.com/sirupsen/logrus"
)

// Suspend reasons accepted by POST /api/state/suspend.
const (
	// SuspendReasonUser is a person stopping delivery on the pump itself.
	SuspendReasonUser = "user"
	// SuspendReasonOcclusion is the pump stopping delivery because it detected
	// an occlusion. It raises an alarm as well as suspending.
	SuspendReasonOcclusion = "occlusion"
	// SuspendReasonAlarm is any other alarm-driven stop.
	SuspendReasonAlarm = "alarm"
)

// handleStateAction dispatches everything under /api/state/.
func (h *Harness) handleStateAction(w http.ResponseWriter, r *http.Request) {
	if h.pumpState == nil {
		writeError(w, http.StatusInternalServerError, "no pump state attached")
		return
	}

	action := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/state"), "/")
	if action == "" {
		h.handleState(w, r)
		return
	}
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method %s not allowed on /api/state/%s (use POST)", r.Method, action)
		return
	}

	switch action {
	case "bolus/start":
		h.actionBolusStart(w, r)
	case "bolus/stall":
		h.actionBolusStall(w, r)
	case "bolus/resume":
		h.actionBolusResume(w, r)
	case "bolus/abort":
		h.actionBolusEnd(w, r, state.BolusEndReasonStopped)
	case "bolus/complete":
		h.actionBolusEnd(w, r, state.BolusEndReasonCompleted)
	case "tempbasal/start":
		h.actionTempBasalStart(w, r)
	case "tempbasal/stop":
		h.actionTempBasalStop(w, r)
	case "suspend":
		h.actionSuspend(w, r)
	case "resume":
		h.actionResume(w, r)
	case "history/append":
		h.actionHistoryAppend(w, r)
	case "qualifyingevent":
		h.actionQualifyingEvent(w, r)
	default:
		writeError(w, http.StatusNotFound, "unknown state action %q", action)
	}
}

// ---------------------------------------------------------------------------
// Bolus
// ---------------------------------------------------------------------------

type bolusStartBody struct {
	// Units is the requested volume. Required.
	Units float64 `json:"units"`
	// Source is how the bolus was commanded: "pump"/"quick" (the pump's own
	// screen, source id 0) or "remote"/"bluetooth" (an app, source id 8). A
	// numeric source_id overrides it. Pump-initiated is the default, since
	// that is the case a driver cannot produce for itself.
	Source string `json:"source,omitempty"`
	// SourceID sets the pumpX2 BolusSource ordinal directly.
	SourceID *int `json:"source_id,omitempty"`
	// TypeBitmask is the pumpX2 BolusType bitmask.
	TypeBitmask int `json:"type_bitmask,omitempty"`
	// BolusID overrides the pump's own allocator, for a test that wants a
	// known id.
	BolusID *uint32 `json:"bolus_id,omitempty"`
	// DurationSeconds makes the bolus take that long: the delivery rate is
	// derived from it. Ignored when Rate is given.
	DurationSeconds float64 `json:"duration_seconds,omitempty"`
	// Rate sets the delivery speed in units/second directly.
	Rate float64 `json:"rate,omitempty"`
}

// actionBolusStart starts a bolus on the pump, as if it had been programmed
// there: the pump allocates the id, records the activation and raises the
// bolus-change qualifying event, so a driver that polls CurrentBolusStatus
// sees a bolus it did not command.
func (h *Harness) actionBolusStart(w http.ResponseWriter, r *http.Request) {
	var body bolusStartBody
	if err := decodeBody(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "%v", err)
		return
	}
	if body.Units <= 0 {
		writeError(w, http.StatusBadRequest, "units must be positive")
		return
	}
	if h.pumpState.IsBolusActive() {
		writeError(w, http.StatusConflict, "a bolus is already in progress; abort or complete it first")
		return
	}

	sourceID, err := resolveBolusSource(body)
	if err != nil {
		writeError(w, http.StatusBadRequest, "%v", err)
		return
	}

	bolusID := h.pumpState.GetNextBolusID()
	if body.BolusID != nil {
		bolusID = *body.BolusID
	}

	switch {
	case body.Rate > 0:
		h.pumpState.SetBolusRate(body.Rate)
	case body.DurationSeconds > 0:
		h.pumpState.SetBolusRate(body.Units / body.DurationSeconds)
	}

	h.pumpState.StartBolusWithSource(body.Units, bolusID, sourceID, body.TypeBitmask)
	h.pumpState.RecordBolusActivated(bolusID, body.Units, sourceID)

	if n := h.notifier(); n != nil {
		h.emit("bolus start", func() error { return n.NotifyBolusStart(bolusID, body.Units) })
	}

	log.Infof("harness: pump-initiated bolus started: %.2f units, id=%d, sourceId=%d, rate=%.4f U/s",
		body.Units, bolusID, sourceID, h.pumpState.GetBolusRate())

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"bolus_id": bolusID,
		"state":    h.Snapshot(),
	})
}

// resolveBolusSource maps the request's source naming onto a pumpX2 source id.
func resolveBolusSource(body bolusStartBody) (int, error) {
	if body.SourceID != nil {
		return *body.SourceID, nil
	}
	switch strings.ToLower(strings.TrimSpace(body.Source)) {
	case "", "pump", "quick", "quickbolus", "pump_ui":
		return state.BolusSourceQuickBolus, nil
	case "remote", "bluetooth", "app":
		return state.BolusSourceBluetoothRemote, nil
	default:
		return 0, fmt.Errorf("unknown bolus source %q (expected \"pump\" or \"remote\", or a numeric source_id)", body.Source)
	}
}

// actionBolusStall freezes an in-progress bolus: it stays in progress and
// keeps answering CurrentBolusStatus, but delivers nothing further. That is
// what lets a test hold a bolus open across an arbitrary number of driver
// polls without racing the simulator.
func (h *Harness) actionBolusStall(w http.ResponseWriter, r *http.Request) {
	if err := decodeBody(r, &struct{}{}); err != nil {
		writeError(w, http.StatusBadRequest, "%v", err)
		return
	}
	if !h.pumpState.IsBolusActive() {
		writeError(w, http.StatusConflict, "no bolus is in progress")
		return
	}

	h.pumpState.Lock()
	h.pumpState.Bolus.Stalled = true
	h.pumpState.Bolus.StalledAt = h.pumpState.Now()
	delivered := h.pumpState.Bolus.UnitsDelivered
	h.pumpState.Unlock()

	log.Infof("harness: bolus stalled at %.3f units delivered", delivered)
	writeJSON(w, http.StatusOK, map[string]interface{}{"state": h.Snapshot()})
}

// actionBolusResume lets a stalled bolus continue. The stall's duration is
// added to the start time, so delivered-volume arithmetic stays continuous:
// resuming does not make the pump "catch up" on insulin it never gave.
func (h *Harness) actionBolusResume(w http.ResponseWriter, r *http.Request) {
	if err := decodeBody(r, &struct{}{}); err != nil {
		writeError(w, http.StatusBadRequest, "%v", err)
		return
	}

	h.pumpState.Lock()
	if !h.pumpState.Bolus.Active || !h.pumpState.Bolus.Stalled {
		h.pumpState.Unlock()
		writeError(w, http.StatusConflict, "no stalled bolus to resume")
		return
	}
	stalledFor := time.Duration(0)
	if !h.pumpState.Bolus.StalledAt.IsZero() {
		stalledFor = h.pumpState.Now().Sub(h.pumpState.Bolus.StalledAt)
	}
	h.pumpState.Bolus.StartTime = h.pumpState.Bolus.StartTime.Add(stalledFor)
	h.pumpState.Bolus.Stalled = false
	h.pumpState.Bolus.StalledAt = time.Time{}
	h.pumpState.Unlock()

	log.Infof("harness: bolus resumed after a %v stall", stalledFor)
	h.tick()
	writeJSON(w, http.StatusOK, map[string]interface{}{"state": h.Snapshot()})
}

type bolusEndBody struct {
	// DeliveredUnits overrides how much the pump says went in. By default an
	// abort keeps whatever had been delivered and a completion delivers the
	// full requested volume.
	DeliveredUnits *float64 `json:"delivered_units,omitempty"`
}

// actionBolusEnd finishes an in-progress bolus, either cut short (abort) or in
// full (complete). Both paths do what the simulator's own completion does:
// stop delivery, write the last-bolus record a driver reconciles against, log
// the history record, and raise the bolus-change qualifying event.
func (h *Harness) actionBolusEnd(w http.ResponseWriter, r *http.Request, endReason int) {
	var body bolusEndBody
	if err := decodeBody(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "%v", err)
		return
	}

	// EndBolus settles the delivered volume (the full request for a completion,
	// what went in so far for an abort, or the explicit volume this request
	// named), takes the difference out of the reservoir, and writes both halves
	// of the ending under one lock -- so this cannot race the simulator
	// finishing the same bolus into a second BolusCompleted record.
	record, ended := h.pumpState.EndBolus(endReason, body.DeliveredUnits)
	if !ended {
		writeError(w, http.StatusConflict, "no bolus is in progress")
		return
	}

	if n := h.notifier(); n != nil {
		if endReason == state.BolusEndReasonCompleted {
			h.emit("bolus complete", func() error {
				return n.NotifyBolusComplete(record.BolusID, record.DeliveredUnits, record.RequestedUnits)
			})
		} else {
			h.emit("bolus canceled", func() error {
				return n.NotifyBolusCanceled(record.BolusID, record.DeliveredUnits, record.RequestedUnits)
			})
		}
	}

	log.Infof("harness: bolus %d ended (reason %d) with %.3f of %.3f units delivered",
		record.BolusID, endReason, record.DeliveredUnits, record.RequestedUnits)
	writeJSON(w, http.StatusOK, map[string]interface{}{"state": h.Snapshot()})
}

// ---------------------------------------------------------------------------
// Temp basal
// ---------------------------------------------------------------------------

type tempBasalBody struct {
	// Percent is the temp rate as a percentage of the profile rate.
	Percent int `json:"percent,omitempty"`
	// Rate sets the absolute temp rate in U/hr instead, and the percentage is
	// derived from the profile rate.
	Rate float64 `json:"rate,omitempty"`
	// DurationMinutes is how long the temp rate runs.
	DurationMinutes int `json:"duration_minutes,omitempty"`
}

// actionTempBasalStart starts a temp rate on the pump itself, with the same
// bookkeeping SetTempRateRequest produces: a new temp rate id, a
// TempRateActivated history record and a basal-change qualifying event.
func (h *Harness) actionTempBasalStart(w http.ResponseWriter, r *http.Request) {
	var body tempBasalBody
	if err := decodeBody(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "%v", err)
		return
	}

	profileRate := h.pumpState.GetProfileBasalRate()
	percent := body.Percent
	rate := body.Rate
	switch {
	case rate > 0 && profileRate > 0:
		// Rounded, not truncated: 1.2 U/hr against a 0.8 U/hr profile is 150%,
		// and float division alone reports 149.
		percent = int(math.Round(rate / profileRate * 100))
	case percent > 0:
		rate = profileRate * float64(percent) / 100.0
	default:
		writeError(w, http.StatusBadRequest, "either percent or rate is required")
		return
	}
	if body.DurationMinutes <= 0 {
		writeError(w, http.StatusBadRequest, "duration_minutes must be positive")
		return
	}

	start := h.pumpState.Now()
	oldRate := h.pumpState.GetBasalRate()

	// A temp rate started while another is running replaces it, and the one it
	// replaces ends HERE -- at the replacement instant, with its own
	// TempRateCompleted record written before the new rate's activation, so a
	// timeline can close it instead of leaving it open forever.
	h.stopTempRate(start)

	tempRateID := h.pumpState.NextTempRateID()
	basal := &state.BasalState{
		CurrentRate:      profileRate,
		TempBasalActive:  true,
		TempBasalRate:    rate,
		TempBasalEnd:     start.Add(time.Duration(body.DurationMinutes) * time.Minute),
		TempBasalPercent: percent,
		TempBasalStart:   start,
		TempRateID:       tempRateID,
	}
	h.pumpState.SetBasalState(basal)
	h.pumpState.RecordTempRateActivated(basal, profileRate)

	if n := h.notifier(); n != nil {
		h.emit("basal change", func() error { return n.NotifyBasalRateChange(oldRate, rate, true) })
	}

	log.Infof("harness: pump-initiated temp rate: %d%% of %.3f U/hr = %.3f U/hr for %d min (id=%d)",
		percent, profileRate, rate, body.DurationMinutes, tempRateID)
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"temp_rate_id": tempRateID,
		"state":        h.Snapshot(),
	})
}

// actionTempBasalStop ends a running temp rate on the pump, as its own expiry
// would: back to the profile rate, a TempRateCompleted record and a
// basal-change qualifying event.
func (h *Harness) actionTempBasalStop(w http.ResponseWriter, r *http.Request) {
	if err := decodeBody(r, &struct{}{}); err != nil {
		writeError(w, http.StatusBadRequest, "%v", err)
		return
	}

	temp, ok := h.stopTempRate(h.pumpState.Now())
	if !ok {
		writeError(w, http.StatusConflict, "no temp rate is running")
		return
	}

	log.Infof("harness: temp rate %d stopped, back to profile %.3f U/hr",
		temp.TempRateID, h.pumpState.GetProfileBasalRate())
	writeJSON(w, http.StatusOK, map[string]interface{}{"state": h.Snapshot()})
}

// stopTempRate ends a running temp rate at `when` and emits the basal-change
// qualifying event for it. Shared by the stop action and by a temp rate that
// replaces another; a suspend ends its temp rate through SuspendDelivery.
//
// PumpState.EndTempRate writes the one TempRateCompleted record, naming the
// temp rate id it closes and how much of the programmed duration was left.
// ok is false when no temp rate was running.
func (h *Harness) stopTempRate(when time.Time) (state.TempRateSnapshot, bool) {
	temp, profileRate, ok := h.pumpState.EndTempRate(when)
	if !ok {
		return temp, false
	}
	if n := h.notifier(); n != nil {
		h.emit("basal change", func() error { return n.NotifyBasalRateChange(temp.Rate, profileRate, false) })
	}
	return temp, true
}

// ---------------------------------------------------------------------------
// Suspend / resume
// ---------------------------------------------------------------------------

type suspendBody struct {
	// Reason is "user", "occlusion" or "alarm".
	Reason string `json:"reason,omitempty"`
	// Message overrides the alarm text recorded for an alarm-driven stop.
	Message string `json:"message,omitempty"`
}

// actionSuspend stops delivery on the pump itself, the way a real pump does
// when a person stops it or an alarm fires.
//
// It is deliberately not just "set a flag": a real stop halts an in-progress
// bolus and any running temp rate too, records each of those endings, and
// raises a qualifying event for each -- which is exactly the pile-up a driver's
// reconciliation has to survive, and what makes #330-style tests meaningful.
func (h *Harness) actionSuspend(w http.ResponseWriter, r *http.Request) {
	var body suspendBody
	if err := decodeBody(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "%v", err)
		return
	}

	reason := strings.ToLower(strings.TrimSpace(body.Reason))
	if reason == "" {
		reason = SuspendReasonUser
	}
	switch reason {
	case SuspendReasonUser, SuspendReasonOcclusion, SuspendReasonAlarm:
	default:
		writeError(w, http.StatusBadRequest, "unknown suspend reason %q (expected user, occlusion or alarm)", reason)
		return
	}

	if h.pumpState.IsPumpingSuspended() {
		writeError(w, http.StatusConflict, "delivery is already suspended")
		return
	}

	// One transition: SuspendDelivery ends an in-progress bolus and any running
	// temp rate -- each with its own record, written before the
	// PumpingSuspended record that caused them -- and is the same call the
	// protocol path makes, so a pump-initiated stop and a driver-commanded one
	// leave identical logs.
	outcome := h.pumpState.SuspendDelivery(reason)
	if !outcome.Changed {
		writeError(w, http.StatusConflict, "delivery is already suspended")
		return
	}

	if reason != SuspendReasonUser {
		h.raiseSuspendAlarm(reason, body.Message)
	}

	if n := h.notifier(); n != nil {
		if outcome.Bolus != nil {
			bolus := outcome.Bolus
			h.emit("bolus canceled", func() error {
				return n.NotifyBolusCanceled(bolus.BolusID, bolus.DeliveredUnits, bolus.RequestedUnits)
			})
			log.Infof("harness: suspend stopped bolus %d after %.3f of %.3f units",
				bolus.BolusID, bolus.DeliveredUnits, bolus.RequestedUnits)
		}
		if outcome.TempRate != nil {
			temp := outcome.TempRate
			h.emit("basal change", func() error {
				return n.NotifyBasalRateChange(temp.Rate, outcome.ProfileRate, false)
			})
		}
		h.emit("pump suspend", func() error { return n.NotifyPumpSuspended(reason) })
	}

	log.Infof("harness: pump-initiated suspend (%s)", reason)
	writeJSON(w, http.StatusOK, map[string]interface{}{"state": h.Snapshot()})
}

// raiseSuspendAlarm records the alarm behind an alarm-driven stop, so
// AlarmStatus and the alert-status responses have something to report and the
// alert qualifying event has a cause.
func (h *Harness) raiseSuspendAlarm(reason, message string) {
	alertType := state.AlertBasalSuspended
	if reason == SuspendReasonOcclusion {
		alertType = state.AlertOcclusion
	}
	if message == "" {
		message = map[string]string{
			SuspendReasonOcclusion: "Occlusion detected",
			SuspendReasonAlarm:     "Delivery stopped",
		}[reason]
	}

	alert := state.Alert{
		ID:        h.nextAlertID(),
		Type:      alertType,
		Priority:  state.PriorityCritical,
		Message:   message,
		Timestamp: h.pumpState.Now(),
	}
	h.pumpState.AddAlert(alert)
	h.pumpState.RecordAlarmActivated(alert, reason)

	if n := h.notifier(); n != nil {
		h.emit("alert", func() error { return n.NotifyAlert(alert) })
	}
}

// nextAlertID allocates a monotonic alert id.
func (h *Harness) nextAlertID() uint32 {
	h.pumpState.RLock()
	defer h.pumpState.RUnlock()

	var max uint32
	for _, a := range h.pumpState.ActiveAlerts {
		if a.ID > max {
			max = a.ID
		}
	}
	return max + 1
}

type resumeBody struct {
	// ClearAlarms drops the active alerts, as acknowledging an alarm on the
	// pump would. On by default: a pump that resumes with its occlusion alarm
	// still standing is not a state a real pump reaches.
	ClearAlarms *bool `json:"clear_alarms,omitempty"`
}

// actionResume restarts delivery on the pump itself.
func (h *Harness) actionResume(w http.ResponseWriter, r *http.Request) {
	var body resumeBody
	if err := decodeBody(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "%v", err)
		return
	}
	if !h.pumpState.IsPumpingSuspended() {
		writeError(w, http.StatusConflict, "delivery is not suspended")
		return
	}

	clearAlarms := body.ClearAlarms == nil || *body.ClearAlarms
	if clearAlarms {
		h.clearAlerts()
	}

	if !h.pumpState.ResumeDelivery() {
		writeError(w, http.StatusConflict, "delivery is not suspended")
		return
	}

	if n := h.notifier(); n != nil {
		h.emit("pump resume", func() error { return n.NotifyPumpResumed() })
		// Delivery going from zero back to the profile rate is a basal change
		// too, and a driver that only watches basal-change events must still
		// see the resume.
		rate := h.pumpState.GetBasalRate()
		h.emit("basal change", func() error { return n.NotifyBasalRateChange(0, rate, false) })
	}

	log.Info("harness: pump-initiated resume")
	writeJSON(w, http.StatusOK, map[string]interface{}{"state": h.Snapshot()})
}

// clearAlerts acknowledges every standing alarm, writing one AlarmCleared
// record per alarm (a pump has no "3 alarms went away" record) and keeping each
// one's clear time in the pump's cleared-alarm log, which the snapshot reports
// as alarms_history.
func (h *Harness) clearAlerts() {
	when := h.pumpState.Now()
	for _, cleared := range h.pumpState.ClearAlerts(when) {
		h.pumpState.RecordAlarmCleared(cleared.Alert, cleared.ClearedAt)
	}
}

// ---------------------------------------------------------------------------
// History and raw qualifying events
// ---------------------------------------------------------------------------

type historyAppendBody struct {
	// Type is the record's human-readable type name (e.g. "BolusCompleted").
	Type string `json:"type"`
	// TypeID is the pumpX2 numeric history type id. When omitted it is looked
	// up from Type.
	TypeID *int `json:"type_id,omitempty"`
	// SecondsAgo backdates the record relative to the pump's clock, which is
	// how a test stages history that predates the connection.
	SecondsAgo float64 `json:"seconds_ago,omitempty"`
	// Data is the record's payload fields.
	Data map[string]interface{} `json:"data,omitempty"`
}

// actionHistoryAppend writes one history-log record, optionally backdated.
func (h *Harness) actionHistoryAppend(w http.ResponseWriter, r *http.Request) {
	var body historyAppendBody
	if err := decodeBody(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "%v", err)
		return
	}
	if strings.TrimSpace(body.Type) == "" && body.TypeID == nil {
		writeError(w, http.StatusBadRequest, "type or type_id is required")
		return
	}

	typeID := 0
	if body.TypeID != nil {
		typeID = *body.TypeID
	} else if id, ok := state.HistoryTypeIDByName(body.Type); ok {
		typeID = id
	}

	when := h.pumpState.Now().Add(-secondsToDuration(body.SecondsAgo))
	seq := h.pumpState.AddHistoryLogEntryAt(typeID, body.Type, when, body.Data)

	log.Infof("harness: appended history record %s (typeId=%d) at %s as sequence %d",
		body.Type, typeID, when.Format(time.RFC3339), seq)
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"sequence": seq,
		"type_id":  typeID,
		"history":  h.historySnapshot(),
	})
}

type qualifyingEventBody struct {
	// Bitmask is the raw little-endian uint32 the pump notifies with.
	Bitmask uint32 `json:"bitmask"`
}

// actionQualifyingEvent raises an arbitrary qualifying-event bitmask, for
// combinations the emulator has no internal trigger for.
func (h *Harness) actionQualifyingEvent(w http.ResponseWriter, r *http.Request) {
	var body qualifyingEventBody
	if err := decodeBody(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "%v", err)
		return
	}

	n := h.notifier()
	if n == nil {
		writeError(w, http.StatusInternalServerError, "no router attached; cannot send qualifying events")
		return
	}
	if err := n.NotifyBitmask(body.Bitmask); err != nil {
		writeError(w, http.StatusConflict, "%v", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"bitmask": body.Bitmask})
}
