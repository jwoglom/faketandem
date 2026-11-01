package harness

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/jwoglom/faketandem/pkg/handler"
	"github.com/jwoglom/faketandem/pkg/state"
)

// cgmView is the CGM as GET /api/cgm (and /api/state's "cgm") reports it.
type cgmView struct {
	SensorType     int    `json:"sensor_type"`
	SensorTypeName string `json:"sensor_type_name"`
	// SessionActive is whether a session is running (start pending through
	// stop pending), kept from the snapshot's earlier shape.
	SessionActive bool   `json:"session_active"`
	CurrentEGV    int    `json:"current_egv"`
	TransmitterID string `json:"transmitter_id"`

	Phase            string  `json:"phase"`
	PhaseSince       string  `json:"phase_since"`
	EndReason        string  `json:"end_reason,omitempty"`
	SessionState     int     `json:"session_state"`
	SessionStart     string  `json:"session_start"`
	WarmupEnds       string  `json:"warmup_ends"`
	LifetimeEnds     string  `json:"lifetime_ends"`
	GraceEnds        string  `json:"grace_ends"`
	SessionRemaining float64 `json:"session_remaining_seconds"`
	GracePeriod      bool    `json:"grace_period"`

	G6SensorCode       int    `json:"g6_sensor_code"`
	G7PairingCode      *int   `json:"g7_pairing_code"`
	TransmitterBattery int    `json:"transmitter_battery"`
	EGVStatus          int    `json:"egv_status"`
	EGVTime            string `json:"egv_time"`
	TrendRate          int    `json:"trend_rate"`

	AlertBitmask uint64   `json:"alert_bitmask"`
	Alerts       []string `json:"alerts"`

	Auto    bool           `json:"auto"`
	Timing  cgmTimingView  `json:"timing"`
	Body    cgmBodyView    `json:"body"`
	Refuse  map[string]int `json:"refuse"`
	Targets []string       `json:"advance_targets"`
}

type cgmTimingView struct {
	ConnectSeconds  float64 `json:"connect_seconds"`
	WarmupSeconds   float64 `json:"warmup_seconds"`
	LifetimeSeconds float64 `json:"lifetime_seconds"`
	GraceSeconds    float64 `json:"grace_seconds"`
	StopSeconds     float64 `json:"stop_seconds"`
}

type cgmBodyView struct {
	G7Code        string `json:"g7_code"`
	TransmitterID string `json:"transmitter_id"`
}

func newCGMView(s state.CGMSnapshot) cgmView {
	v := cgmView{
		SensorType: s.SensorType, SensorTypeName: state.CGMSensorTypeName(s.SensorType),
		SessionActive: s.Phase.Running(), CurrentEGV: s.EGV, TransmitterID: s.TransmitterID,
		Phase: string(s.Phase), PhaseSince: formatTime(s.PhaseSince), EndReason: s.EndReason,
		SessionState: s.SessionState, SessionStart: formatTime(s.SessionStart), WarmupEnds: formatTime(s.WarmupEnds),
		LifetimeEnds: formatTime(s.LifetimeEnds), GraceEnds: formatTime(s.GraceEnds),
		SessionRemaining: s.SessionRemaining.Seconds(), GracePeriod: s.GracePeriod,
		G6SensorCode: s.G6SensorCode, TransmitterBattery: s.TransmitterBattery, EGVStatus: s.EGVStatus,
		EGVTime: formatTime(s.EGVTime), TrendRate: s.TrendRate, AlertBitmask: s.AlertBitmask, Alerts: s.AlertNames(),
		Auto: s.Auto, Refuse: s.Refuse, Targets: state.CGMTargets,
		Timing: cgmTimingView{
			ConnectSeconds: s.Timing.Connect.Seconds(), WarmupSeconds: s.Timing.Warmup.Seconds(),
			LifetimeSeconds: s.Timing.Lifetime.Seconds(), GraceSeconds: s.Timing.Grace.Seconds(),
			StopSeconds: s.Timing.Stop.Seconds(),
		},
		Body: cgmBodyView{G7Code: s.BodyG7Code, TransmitterID: s.BodyTransmitterID},
	}
	if s.G7PairingCode >= 0 {
		code := s.G7PairingCode
		v.G7PairingCode = &code
	}
	return v
}

// cgmUpdate is PUT /api/cgm's body. Like PUT /api/state it sets what it is
// told and moves no phase; POST /api/cgm/advance moves phases.
type cgmUpdate struct {
	SensorType         *json.RawMessage `json:"sensor_type,omitempty"`
	TransmitterID      *string          `json:"transmitter_id,omitempty"`
	G6SensorCode       *int             `json:"g6_sensor_code,omitempty"`
	G7PairingCode      *json.RawMessage `json:"g7_pairing_code,omitempty"`
	EGV                *int             `json:"egv,omitempty"`
	TrendRate          *int             `json:"trend_rate,omitempty"`
	TransmitterBattery *json.RawMessage `json:"transmitter_battery,omitempty"`
	Auto               *bool            `json:"auto,omitempty"`
	Timing             *struct {
		ConnectSeconds  *float64 `json:"connect_seconds,omitempty"`
		WarmupSeconds   *float64 `json:"warmup_seconds,omitempty"`
		LifetimeSeconds *float64 `json:"lifetime_seconds,omitempty"`
		GraceSeconds    *float64 `json:"grace_seconds,omitempty"`
		StopSeconds     *float64 `json:"stop_seconds,omitempty"`
	} `json:"timing,omitempty"`
	ResetTiming bool `json:"reset_timing,omitempty"`
	Body        *struct {
		G7Code        *string `json:"g7_code,omitempty"`
		TransmitterID *string `json:"transmitter_id,omitempty"`
	} `json:"body,omitempty"`
	Refuse      map[string]int `json:"refuse,omitempty"`
	ExtraAlerts *[]uint        `json:"extra_alerts,omitempty"`
	Dismiss     []uint         `json:"dismiss,omitempty"`
}

// cgmCommandAliases are short names for the CGM commands in "refuse".
var cgmCommandAliases = map[string]string{
	"sensor_type":    "SetSensorTypeRequest",
	"transmitter_id": "SetG6TransmitterIdRequest",
	"g6_start":       "StartDexcomG6SensorSessionRequest",
	"g7_code":        "SetDexcomG7PairingCodeRequest",
	"stop":           "StopDexcomCGMSensorSessionRequest",
}

func cgmCommandName(name string) (string, error) {
	if full, ok := cgmCommandAliases[name]; ok {
		return full, nil
	}
	for _, c := range handler.CGMCommands {
		if c == name {
			return c, nil
		}
	}
	aliases := make([]string, 0, len(cgmCommandAliases))
	for a := range cgmCommandAliases {
		aliases = append(aliases, a)
	}
	sort.Strings(aliases)
	return "", fmt.Errorf("unknown CGM command %q (%s, or the request's name)", name, strings.Join(aliases, " | "))
}

// rawIntOrString reads a JSON number or numeric string.
func rawIntOrString(raw json.RawMessage) (int, string, error) {
	var n int
	if err := json.Unmarshal(raw, &n); err == nil {
		return n, strconv.Itoa(n), nil
	}
	var s string
	if err := json.Unmarshal(raw, &s); err != nil {
		return 0, "", fmt.Errorf("want a number or a string, got %s", raw)
	}
	n, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil {
		return 0, s, nil
	}
	return n, s, nil
}

var batteryNames = map[string]int{
	"unavailable": state.CGMBatteryUnavailable, "error": state.CGMBatteryError, "expired": state.CGMBatteryExpired,
	"ok": state.CGMBatteryOK, "okay": state.CGMBatteryOK, "out_of_range": state.CGMBatteryOutOfRange,
}

func applyCGMTiming(c *state.CGMState, u cgmUpdate) {
	if u.ResetTiming {
		c.Timing = state.CGMTiming{}
	}
	if u.Timing == nil {
		return
	}
	set := func(dst *time.Duration, v *float64) {
		if v != nil {
			*dst = secondsToDuration(*v)
		}
	}
	set(&c.Timing.Connect, u.Timing.ConnectSeconds)
	set(&c.Timing.Warmup, u.Timing.WarmupSeconds)
	set(&c.Timing.Lifetime, u.Timing.LifetimeSeconds)
	set(&c.Timing.Stop, u.Timing.StopSeconds)
	if g := u.Timing.GraceSeconds; g != nil {
		c.Timing.Grace, c.Timing.NoGrace = secondsToDuration(*g), *g == 0
	}
}

func applyCGMUpdate(c *state.CGMState, u cgmUpdate) ([]string, error) {
	var applied []string
	for _, apply := range []func(*state.CGMState, cgmUpdate) ([]string, error){
		applyCGMIdentity, applyCGMReadings, applyCGMControl, applyCGMAlerts,
	} {
		a, err := apply(c, u)
		if err != nil {
			return nil, err
		}
		applied = append(applied, a...)
	}
	return applied, nil
}

// applyCGMIdentity sets what the pump was configured with.
func applyCGMIdentity(c *state.CGMState, u cgmUpdate) ([]string, error) {
	var applied []string
	if u.SensorType != nil {
		_, name, err := rawIntOrString(*u.SensorType)
		if err != nil {
			return nil, fmt.Errorf("sensor_type: %w", err)
		}
		t, err := state.ParseCGMSensorType(name)
		if err != nil {
			return nil, err
		}
		c.SensorType = t
		applied = append(applied, "sensor_type")
	}
	if u.TransmitterID != nil {
		c.TransmitterID = strings.ToUpper(*u.TransmitterID)
		applied = append(applied, "transmitter_id")
	}
	if u.G6SensorCode != nil {
		c.G6SensorCode = *u.G6SensorCode
		applied = append(applied, "g6_sensor_code")
	}
	if u.G7PairingCode != nil {
		n, _, err := rawIntOrString(*u.G7PairingCode)
		if err != nil || n < -1 || n > 9999 {
			return nil, fmt.Errorf("g7_pairing_code: want 0-9999, or -1 for none")
		}
		c.G7PairingCode = n
		applied = append(applied, "g7_pairing_code")
	}
	return applied, nil
}

// applyCGMReadings sets what the session reports.
func applyCGMReadings(c *state.CGMState, u cgmUpdate) ([]string, error) {
	var applied []string
	if u.EGV != nil {
		c.CurrentEGV = *u.EGV
		applied = append(applied, "egv")
	}
	if u.TrendRate != nil {
		c.TrendRate = *u.TrendRate
		applied = append(applied, "trend_rate")
	}
	if u.TransmitterBattery != nil {
		n, name, err := rawIntOrString(*u.TransmitterBattery)
		if err != nil {
			return nil, fmt.Errorf("transmitter_battery: %w", err)
		}
		if b, ok := batteryNames[strings.ToLower(name)]; ok {
			n = b
		} else if _, numeric := strconv.Atoi(name); numeric != nil {
			return nil, fmt.Errorf("transmitter_battery: unknown %q (unavailable | error | expired | ok | out_of_range)", name)
		}
		c.TransmitterBattery = n
		applied = append(applied, "transmitter_battery")
	}
	return applied, nil
}

// applyCGMControl sets how the lifecycle runs: timers, the sensor worn, refusals.
func applyCGMControl(c *state.CGMState, u cgmUpdate) ([]string, error) {
	var applied []string
	if u.Auto != nil {
		c.Auto = *u.Auto
		applied = append(applied, "auto")
	}
	if u.Timing != nil || u.ResetTiming {
		applyCGMTiming(c, u)
		applied = append(applied, "timing")
	}
	if u.Body != nil {
		if u.Body.G7Code != nil {
			c.BodyG7Code = *u.Body.G7Code
		}
		if u.Body.TransmitterID != nil {
			c.BodyTransmitterID = strings.ToUpper(*u.Body.TransmitterID)
		}
		applied = append(applied, "body")
	}
	if u.Refuse != nil {
		refuse := map[string]int{}
		for name, status := range u.Refuse {
			full, err := cgmCommandName(name)
			if err != nil {
				return nil, err
			}
			if status != 0 {
				refuse[full] = status
			}
		}
		c.Refuse = refuse
		applied = append(applied, "refuse")
	}
	return applied, nil
}

// applyCGMAlerts raises and clears alert bits on top of the lifecycle's.
func applyCGMAlerts(c *state.CGMState, u cgmUpdate) ([]string, error) {
	var applied []string
	if u.ExtraAlerts != nil {
		c.ExtraAlerts = 0
		for _, bit := range *u.ExtraAlerts {
			if bit > 63 {
				return nil, fmt.Errorf("extra_alerts: bit %d is past 63", bit)
			}
			c.ExtraAlerts |= 1 << bit
			c.Dismissed &^= 1 << bit
		}
		applied = append(applied, "extra_alerts")
	}
	if len(u.Dismiss) > 0 {
		for _, bit := range u.Dismiss {
			if bit > 63 {
				return nil, fmt.Errorf("dismiss: bit %d is past 63", bit)
			}
			c.Dismissed |= 1 << bit
			c.ExtraAlerts &^= 1 << bit
		}
		applied = append(applied, "dismiss")
	}
	return applied, nil
}

// handleCGM serves /api/cgm and its actions.
func (h *Harness) handleCGM(w http.ResponseWriter, r *http.Request) {
	if h.pumpState == nil {
		writeError(w, http.StatusInternalServerError, "no pump state attached")
		return
	}
	action := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/cgm"), "/")
	switch {
	case action == "" && r.Method == http.MethodGet:
	case action == "" && (r.Method == http.MethodPut || r.Method == http.MethodPatch):
		var u cgmUpdate
		if err := decodeBody(r, &u); err != nil {
			writeError(w, http.StatusBadRequest, "%v", err)
			return
		}
		var applied []string
		err := h.pumpState.UpdateCGM(func(c *state.CGMState, _ time.Time) error {
			edited := *c
			var err error
			if applied, err = applyCGMUpdate(&edited, u); err == nil {
				*c = edited
			}
			return err
		})
		if err != nil {
			writeError(w, http.StatusBadRequest, "%v", err)
			return
		}
		h.tick()
		writeJSON(w, http.StatusOK, map[string]interface{}{"applied": applied, "cgm": newCGMView(h.pumpState.CGM())})
		return
	case action == "advance" && r.Method == http.MethodPost:
		h.cgmAdvance(w, r)
		return
	case action == "reset" && r.Method == http.MethodPost:
		h.cgmReset(w, r)
		return
	default:
		writeError(w, http.StatusNotFound, "no %s /api/cgm/%s (GET/PUT /api/cgm, POST /api/cgm/advance, POST /api/cgm/reset)", r.Method, action)
		return
	}
	writeJSON(w, http.StatusOK, newCGMView(h.pumpState.CGM()))
}

type cgmAdvanceBody struct {
	// To is a phase or ending (state.CGMTargets); "next" when empty.
	To string `json:"to"`
	// SessionAgeSeconds dates the session's start that far back, so a test
	// can land late in a session (about to expire, in grace).
	SessionAgeSeconds *float64 `json:"session_age_seconds,omitempty"`
}

func (h *Harness) cgmAdvance(w http.ResponseWriter, r *http.Request) {
	var body cgmAdvanceBody
	if err := decodeBody(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "%v", err)
		return
	}
	if body.To == "" {
		body.To = "next"
	}
	var age *time.Duration
	if body.SessionAgeSeconds != nil {
		d := secondsToDuration(*body.SessionAgeSeconds)
		age = &d
	}
	err := h.pumpState.UpdateCGM(func(c *state.CGMState, now time.Time) error {
		return c.MoveTo(body.To, now, age)
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, "%v", err)
		return
	}
	h.tick()
	writeJSON(w, http.StatusOK, newCGMView(h.pumpState.CGM()))
}

type cgmResetBody struct {
	SensorType string `json:"sensor_type"`
}

func (h *Harness) cgmReset(w http.ResponseWriter, r *http.Request) {
	var body cgmResetBody
	if err := decodeBody(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "%v", err)
		return
	}
	t, err := state.ParseCGMSensorType(body.SensorType)
	if err != nil {
		writeError(w, http.StatusBadRequest, "%v", err)
		return
	}
	h.pumpState.ResetCGM(t)
	h.tick()
	writeJSON(w, http.StatusOK, newCGMView(h.pumpState.CGM()))
}
