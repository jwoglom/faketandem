package state

import (
	"fmt"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
)

// The CGM model is a sensor session's lifecycle as the pump reports it, not a
// CGM: glucose is whatever the harness sets. Lengths follow Dexcom's published
// wear (and LoopKit/g7sim's presets for the G7); which session state the pump
// reports in each phase is faketandem's reading of pumpX2's enums, unconfirmed
// on pump firmware.

// CGM sensor types, as SetSensorTypeRequest and CgmStatusV2Response number them.
const (
	CGMSensorTypeNone     = 0
	CGMSensorTypeDexcomG6 = 1
	CGMSensorTypeFSL2     = 2
	CGMSensorTypeDexcomG7 = 3
)

// CGMPhase is where a sensor session is in its lifecycle.
type CGMPhase string

// The lifecycle: start_pending -> warmup -> active -> grace (G7 only) ->
// stopped (expired). stop_pending -> stopped is the user's stop; signal_loss is
// a running session that has stopped reading, which only the harness enters.
const (
	CGMPhaseStopped      CGMPhase = "stopped"
	CGMPhaseStartPending CGMPhase = "start_pending"
	CGMPhaseWarmup       CGMPhase = "warmup"
	CGMPhaseActive       CGMPhase = "active"
	CGMPhaseGrace        CGMPhase = "grace"
	CGMPhaseSignalLoss   CGMPhase = "signal_loss"
	CGMPhaseStopPending  CGMPhase = "stop_pending"
)

// Why the last session ended, while the phase is stopped.
const (
	CGMEndUser     = "user"
	CGMEndExpired  = "expired"
	CGMEndFailed   = "failed"
	CGMEndNotFound = "not_found"
)

// Session states, as CGMStatusResponse and CgmStatusV2Response carry them.
const (
	CGMSessionStopped      = 0
	CGMSessionStartPending = 1
	CGMSessionActive       = 2
	CGMSessionStopPending  = 3
)

// Transmitter battery states, as CgmStatusV2Response carries them.
const (
	CGMBatteryUnavailable = 0
	CGMBatteryError       = 1
	CGMBatteryExpired     = 2
	CGMBatteryOK          = 3
	CGMBatteryOutOfRange  = 4
)

// EGV statuses, as CurrentEGVGuiDataResponse carries them.
const (
	EGVStatusInvalid     = 0
	EGVStatusValid       = 1
	EGVStatusLow         = 2
	EGVStatusHigh        = 3
	EGVStatusUnavailable = 4
)

// CGM alert bits, as CGMAlertStatusResponse's mask carries them.
const (
	CGMAlertSensorFailed       = 11
	CGMAlertSensorExpiring     = 12
	CGMAlertSensorExpired      = 13
	CGMAlertOutOfRange         = 14
	CGMAlertFailedConnection   = 27
	CGMAlertGracePeriodStart   = 31
	CGMAlertGracePeriodExpires = 32
)

// cgmAlertNames are pumpX2's CGMAlert names, without its CGM_ALERT suffix.
var cgmAlertNames = map[uint]string{
	1: "FIXED_LOW", 2: "HIGH", 3: "LOW", 4: "CALIBRATION_REQUEST", 5: "RISE", 6: "RAPID_RISE", 7: "FALL",
	8: "RAPID_FALL", 9: "LOW_CALIBRATION_ERROR", 10: "HIGH_CALIBRATION_ERROR",
	CGMAlertSensorFailed: "SENSOR_FAILED", CGMAlertSensorExpiring: "SENSOR_EXPIRING",
	CGMAlertSensorExpired: "SENSOR_EXPIRED", CGMAlertOutOfRange: "OUT_OF_RANGE",
	16: "FIRST_START_CALIBRATION", 17: "SECOND_START_CALIBRATION", 18: "CALIBRATION_REQUIRED",
	19: "LOW_TRANSMITTER", 20: "TRANSMITTER", 22: "SENSOR_EXPIRING_2", 24: "URGENT_LOW_SOON", 25: "SENSOR_REUSE",
	26: "TEMPERATURE", CGMAlertFailedConnection: "FAILED_CONNECTION", 28: "CHECK_SENSOR", 29: "EGV_UNUSABLE",
	CGMAlertGracePeriodStart: "GRACE_PERIOD_START", CGMAlertGracePeriodExpires: "GRACE_PERIOD_EXPIRING_SOON",
	39: "TRANSMITTER_EXPIRED", 40: "PUMP_BLUETOOTH_ERROR", 45: "TRANSMITTER_EXPIRING", 46: "TRANSMITTER_EXPIRING_2",
	48: "UNAVAILABLE",
}

const (
	cgmExpiringWarning      = 24 * time.Hour
	cgmGraceExpiringWarning = 2 * time.Hour
	egvLowLimit             = 40
	egvHighLimit            = 400
	egvInterval             = 5 * time.Minute
)

// CGMTiming is how long each timed phase lasts. Zero fields take the sensor
// type's default (CGMDefaultTiming).
type CGMTiming struct {
	// Connect is start_pending's length: the pump finding the sensor.
	Connect time.Duration
	Warmup  time.Duration
	// Lifetime is the rated wear, from the session's start, warmup included.
	Lifetime time.Duration
	// Grace is how long a session reads past Lifetime; G6 has none.
	Grace time.Duration
	// Stop is stop_pending's length.
	Stop time.Duration
	// NoGrace makes Grace zero whatever the default, since a zero Grace means
	// "the default".
	NoGrace bool
}

// CGMDefaultTiming is a sensor type's wear: a G7's 27-minute warmup, 10 days
// and 12-hour grace period (g7sim's 10-day preset); a G6's 2-hour warmup and
// 10 days with no grace period.
func CGMDefaultTiming(sensorType int) CGMTiming {
	t := CGMTiming{Connect: 30 * time.Second, Lifetime: 10 * 24 * time.Hour, Stop: 5 * time.Second}
	switch sensorType {
	case CGMSensorTypeDexcomG7:
		t.Warmup, t.Grace = 27*time.Minute, 12*time.Hour
	default:
		t.Warmup = 2 * time.Hour
	}
	return t
}

func (t CGMTiming) over(d CGMTiming) CGMTiming {
	if t.Connect > 0 {
		d.Connect = t.Connect
	}
	if t.Warmup > 0 {
		d.Warmup = t.Warmup
	}
	if t.Lifetime > 0 {
		d.Lifetime = t.Lifetime
	}
	if t.Grace > 0 {
		d.Grace = t.Grace
	}
	if t.NoGrace {
		d.Grace = 0
	}
	if t.Stop > 0 {
		d.Stop = t.Stop
	}
	return d
}

// CGMState is the pump's CGM: its configuration, the sensor session's
// lifecycle, and the sensor on the body that a pairing has to match.
type CGMState struct {
	SensorType int
	// TransmitterID is the G6 transmitter the pump was given.
	TransmitterID string
	// G6SensorCode is the code the last G6 session started with (0 for none).
	G6SensorCode int
	// G7PairingCode is the code the last G7 session started with; -1 for none.
	G7PairingCode int

	Phase      CGMPhase
	PhaseSince time.Time
	// SessionStart is when the session's warmup began, on the pump clock.
	SessionStart time.Time
	EndReason    string

	// Auto moves timed phases on with the pump clock; off, only commands and
	// the harness move them.
	Auto bool
	// Timing overrides the sensor type's default lengths.
	Timing CGMTiming

	// BodyG7Code and BodyTransmitterID describe the sensor actually worn: a
	// pairing with another code or transmitter never finds it. Empty matches
	// anything.
	BodyG7Code        string
	BodyTransmitterID string

	CurrentEGV int
	TrendRate  int
	// TransmitterBattery is a G6 transmitter's; a G7 reports none.
	TransmitterBattery int

	// ExtraAlerts are CGM alert bits the harness raises on top of the
	// lifecycle's; Dismissed are bits the app or the harness cleared.
	ExtraAlerts uint64
	Dismissed   uint64

	// Refuse makes a command answer with a nonzero status, by request name.
	Refuse map[string]int

	notifiedPhase  CGMPhase
	notifiedAlerts uint64
}

func newCGMState(now time.Time) *CGMState {
	return &CGMState{
		SensorType:         CGMSensorTypeDexcomG6,
		TransmitterID:      "80AB12",
		G7PairingCode:      -1,
		Phase:              CGMPhaseActive,
		PhaseSince:         now.Add(-22 * time.Hour),
		SessionStart:       now.Add(-24 * time.Hour),
		Auto:               true,
		CurrentEGV:         120,
		TransmitterBattery: CGMBatteryOK,
		notifiedPhase:      CGMPhaseActive,
	}
}

// SessionState is the phase as the pump's status messages report it.
func (p CGMPhase) SessionState() int {
	switch p {
	case CGMPhaseStartPending:
		return CGMSessionStartPending
	case CGMPhaseWarmup, CGMPhaseActive, CGMPhaseGrace, CGMPhaseSignalLoss:
		return CGMSessionActive
	case CGMPhaseStopPending:
		return CGMSessionStopPending
	default:
		return CGMSessionStopped
	}
}

// Running is whether a session exists, from the start through a pending stop.
func (p CGMPhase) Running() bool { return p.SessionState() != CGMSessionStopped }

func (p CGMPhase) inSession() bool { return p.SessionState() == CGMSessionActive }

func (c *CGMState) timing() CGMTiming {
	return c.Timing.over(CGMDefaultTiming(c.SensorType))
}

func (c *CGMState) setPhase(p CGMPhase, at time.Time, endReason string) {
	if c.Phase != p || endReason != "" {
		log.Infof("CGM: %s -> %s %s", c.Phase, p, endReason)
	}
	c.Phase, c.PhaseSince = p, at
	if p == CGMPhaseStopped {
		c.EndReason = endReason
	} else {
		c.EndReason = ""
	}
}

func (c *CGMState) startSession(at time.Time) {
	c.SessionStart = at
	c.Dismissed = 0
	c.setPhase(CGMPhaseWarmup, at, "")
}

func (c *CGMState) found() bool {
	switch c.SensorType {
	case CGMSensorTypeDexcomG7:
		return c.BodyG7Code == "" || c.BodyG7Code == fmt.Sprintf("%04d", c.G7PairingCode)
	case CGMSensorTypeDexcomG6:
		return c.BodyTransmitterID == "" || strings.EqualFold(c.BodyTransmitterID, c.TransmitterID)
	}
	return true
}

// advance moves timed phases on to now, each transition dated when it fell due.
func (c *CGMState) advance(now time.Time) {
	if !c.Auto {
		return
	}
	t := c.timing()
	for i := 0; i < 8; i++ {
		due, timed := c.phaseEnds(t)
		if !timed || now.Before(due) {
			return
		}
		c.endPhase(due, t)
	}
}

// phaseEnds is when the current phase ends on its own, if it does.
func (c *CGMState) phaseEnds(t CGMTiming) (time.Time, bool) {
	switch c.Phase {
	case CGMPhaseStartPending:
		return c.PhaseSince.Add(t.Connect), true
	case CGMPhaseWarmup:
		return c.SessionStart.Add(t.Warmup), true
	case CGMPhaseActive, CGMPhaseSignalLoss:
		return c.SessionStart.Add(t.Lifetime), true
	case CGMPhaseGrace:
		return c.SessionStart.Add(t.Lifetime + t.Grace), true
	case CGMPhaseStopPending:
		return c.PhaseSince.Add(t.Stop), true
	}
	return time.Time{}, false
}

// endPhase moves on to the phase after the current one's timed end.
func (c *CGMState) endPhase(due time.Time, t CGMTiming) {
	switch c.Phase {
	case CGMPhaseStartPending:
		if c.found() {
			c.startSession(due)
		} else {
			c.setPhase(CGMPhaseStopped, due, CGMEndNotFound)
		}
	case CGMPhaseWarmup:
		c.setPhase(CGMPhaseActive, due, "")
	case CGMPhaseActive, CGMPhaseSignalLoss:
		if t.Grace > 0 {
			c.setPhase(CGMPhaseGrace, due, "")
		} else {
			c.setPhase(CGMPhaseStopped, due, CGMEndExpired)
		}
	case CGMPhaseGrace:
		c.setPhase(CGMPhaseStopped, due, CGMEndExpired)
	case CGMPhaseStopPending:
		c.setPhase(CGMPhaseStopped, due, CGMEndUser)
	}
}

// alertBits is the CGM alert mask the lifecycle raises now, before dismissals.
func (c *CGMState) alertBits(now time.Time) uint64 {
	var bits uint64
	t := c.timing()
	switch c.Phase {
	case CGMPhaseActive, CGMPhaseSignalLoss:
		if c.SessionStart.Add(t.Lifetime).Sub(now) <= cgmExpiringWarning {
			bits |= 1 << CGMAlertSensorExpiring
		}
	case CGMPhaseGrace:
		bits |= 1 << CGMAlertGracePeriodStart
		if c.SessionStart.Add(t.Lifetime+t.Grace).Sub(now) <= cgmGraceExpiringWarning {
			bits |= 1 << CGMAlertGracePeriodExpires
		}
	case CGMPhaseStopped:
		switch c.EndReason {
		case CGMEndExpired:
			bits |= 1 << CGMAlertSensorExpired
		case CGMEndFailed:
			bits |= 1 << CGMAlertSensorFailed
		case CGMEndNotFound:
			bits |= 1 << CGMAlertFailedConnection
		}
	}
	if c.Phase == CGMPhaseSignalLoss {
		bits |= 1 << CGMAlertOutOfRange
	}
	return bits
}

// CGMSnapshot is the CGM as the pump reports it at one instant.
type CGMSnapshot struct {
	SensorType         int
	TransmitterID      string
	G6SensorCode       int
	G7PairingCode      int
	Phase              CGMPhase
	PhaseSince         time.Time
	EndReason          string
	SessionState       int
	SessionStart       time.Time // zero when no session has run
	WarmupEnds         time.Time
	LifetimeEnds       time.Time
	GraceEnds          time.Time
	SessionDuration    time.Duration
	SessionRemaining   time.Duration
	GracePeriod        bool
	TransmitterBattery int
	EGV                int
	EGVStatus          int
	EGVTime            time.Time // zero when there is no reading
	TrendRate          int
	AlertBitmask       uint64
	Auto               bool
	Timing             CGMTiming
	BodyG7Code         string
	BodyTransmitterID  string
	Refuse             map[string]int
}

// AlertNames names the bits of the alert mask.
func (s CGMSnapshot) AlertNames() []string {
	names := []string{}
	for bit := uint(0); bit < 64; bit++ {
		if s.AlertBitmask&(1<<bit) == 0 {
			continue
		}
		if name, ok := cgmAlertNames[bit]; ok {
			names = append(names, name)
		} else {
			names = append(names, fmt.Sprintf("BIT_%d", bit))
		}
	}
	return names
}

func (c *CGMState) snapshot(now time.Time) CGMSnapshot {
	t := c.timing()
	s := CGMSnapshot{
		SensorType: c.SensorType, TransmitterID: c.TransmitterID, G6SensorCode: c.G6SensorCode,
		G7PairingCode: c.G7PairingCode, Phase: c.Phase, PhaseSince: c.PhaseSince, EndReason: c.EndReason,
		SessionState: c.Phase.SessionState(), TrendRate: c.TrendRate, Auto: c.Auto, Timing: t,
		BodyG7Code: c.BodyG7Code, BodyTransmitterID: c.BodyTransmitterID, EGVStatus: EGVStatusUnavailable,
	}
	s.Refuse = make(map[string]int, len(c.Refuse))
	for k, v := range c.Refuse {
		s.Refuse[k] = v
	}
	s.AlertBitmask = (c.alertBits(now) | c.ExtraAlerts) &^ c.Dismissed
	if c.SensorType == CGMSensorTypeDexcomG6 {
		s.TransmitterBattery = c.TransmitterBattery
	}
	if !c.Phase.inSession() && c.Phase != CGMPhaseStopPending {
		return s
	}
	s.SessionStart = c.SessionStart
	s.WarmupEnds = c.SessionStart.Add(t.Warmup)
	s.LifetimeEnds = c.SessionStart.Add(t.Lifetime)
	if t.Grace > 0 {
		s.GraceEnds = s.LifetimeEnds.Add(t.Grace)
	}
	s.SessionDuration = t.Lifetime
	end := s.LifetimeEnds
	if c.Phase == CGMPhaseGrace {
		end, s.GracePeriod = s.GraceEnds, true
	}
	s.SessionRemaining = clampDuration(end.Sub(now), 0, t.Lifetime+t.Grace)
	if c.Phase != CGMPhaseActive && c.Phase != CGMPhaseGrace {
		return s
	}
	s.EGV = c.CurrentEGV
	s.EGVStatus = EGVStatusValid
	if c.CurrentEGV < egvLowLimit {
		s.EGVStatus = EGVStatusLow
	} else if c.CurrentEGV > egvHighLimit {
		s.EGVStatus = EGVStatusHigh
	}
	since := now.Sub(s.WarmupEnds)
	if since < 0 {
		since = 0
	}
	s.EGVTime = s.WarmupEnds.Add(since - since%egvInterval)
	return s
}

func clampDuration(d, lo, hi time.Duration) time.Duration {
	if d < lo {
		return lo
	}
	if d > hi {
		return hi
	}
	return d
}

// ---------------------------------------------------------------------------
// PumpState API
// ---------------------------------------------------------------------------

// CGM is the CGM as the pump reports it now, after moving timed phases on.
func (ps *PumpState) CGM() CGMSnapshot {
	now := ps.Now()
	ps.mutex.Lock()
	defer ps.mutex.Unlock()
	ps.CGMState.advance(now)
	return ps.CGMState.snapshot(now)
}

// AdvanceCGM moves timed phases on to the pump clock's now and reports whether
// the phase or the alert mask changed since it last reported, which is when the
// pump raises CGM_CHANGE and CGM_ALERT qualifying events.
func (ps *PumpState) AdvanceCGM() (phaseChanged, alertsChanged bool) {
	now := ps.Now()
	ps.mutex.Lock()
	defer ps.mutex.Unlock()
	c := ps.CGMState
	c.advance(now)
	alerts := c.snapshot(now).AlertBitmask
	phaseChanged = c.Phase != c.notifiedPhase
	alertsChanged = alerts&^c.notifiedAlerts != 0
	c.notifiedPhase, c.notifiedAlerts = c.Phase, alerts
	return phaseChanged, alertsChanged
}

// UpdateCGM runs fn on the CGM with the state locked, after moving timed phases
// on, so a harness edit lands on the current phase.
func (ps *PumpState) UpdateCGM(fn func(c *CGMState, now time.Time) error) error {
	now := ps.Now()
	ps.mutex.Lock()
	defer ps.mutex.Unlock()
	ps.CGMState.advance(now)
	return fn(ps.CGMState, now)
}

// CGMCommandStatus is the status a CGM command answers with: 0 when the pump
// takes it, otherwise the reason it refused.
type CGMCommandStatus int

// Statuses a CGM command can answer with. The pump's own codes are unknown;
// any nonzero status is a refusal to TandemKit and pumpX2.
const (
	CGMCommandOK      CGMCommandStatus = 0
	CGMCommandRefused CGMCommandStatus = 1
)

func (c *CGMState) refused(request string) (CGMCommandStatus, bool) {
	if s, ok := c.Refuse[request]; ok && s != 0 {
		log.Infof("CGM: %s refused with status %d (harness)", request, s)
		return CGMCommandStatus(s), true
	}
	return CGMCommandOK, false
}

// CGMSetSensorType is SetSensorTypeRequest: a different type ends the session.
func (ps *PumpState) CGMSetSensorType(sensorType int) CGMCommandStatus {
	var status CGMCommandStatus
	_ = ps.UpdateCGM(func(c *CGMState, now time.Time) error {
		if s, ok := c.refused("SetSensorTypeRequest"); ok {
			status = s
			return nil
		}
		if sensorType < CGMSensorTypeNone || sensorType > CGMSensorTypeDexcomG7 {
			status = CGMCommandRefused
			return nil
		}
		if sensorType != c.SensorType && c.Phase.Running() {
			c.setPhase(CGMPhaseStopped, now, CGMEndUser)
		}
		c.SensorType = sensorType
		return nil
	})
	return status
}

// CGMSetG6TransmitterID is SetG6TransmitterIdRequest. A new transmitter ends
// the session the old one ran.
func (ps *PumpState) CGMSetG6TransmitterID(txID string) CGMCommandStatus {
	var status CGMCommandStatus
	_ = ps.UpdateCGM(func(c *CGMState, now time.Time) error {
		if s, ok := c.refused("SetG6TransmitterIdRequest"); ok {
			status = s
			return nil
		}
		txID = strings.ToUpper(strings.TrimRight(txID, "\x00 "))
		if c.SensorType != CGMSensorTypeDexcomG6 || !validTransmitterID(txID) {
			status = CGMCommandRefused
			return nil
		}
		if txID != c.TransmitterID && c.Phase.Running() {
			c.setPhase(CGMPhaseStopped, now, CGMEndUser)
		}
		c.TransmitterID = txID
		return nil
	})
	return status
}

func validTransmitterID(id string) bool {
	if len(id) != 6 {
		return false
	}
	for _, r := range id {
		if (r < '0' || r > '9') && (r < 'A' || r > 'Z') {
			return false
		}
	}
	return true
}

// CGMStartG6Session is StartDexcomG6SensorSessionRequest: the pump starts
// looking for the transmitter. A G6 session already running refuses it.
func (ps *PumpState) CGMStartG6Session(sensorCode int) CGMCommandStatus {
	var status CGMCommandStatus
	_ = ps.UpdateCGM(func(c *CGMState, now time.Time) error {
		if s, ok := c.refused("StartDexcomG6SensorSessionRequest"); ok {
			status = s
			return nil
		}
		if c.SensorType != CGMSensorTypeDexcomG6 || c.TransmitterID == "" || c.Phase.Running() {
			status = CGMCommandRefused
			return nil
		}
		c.G6SensorCode = sensorCode
		c.setPhase(CGMPhaseStartPending, now, "")
		return nil
	})
	return status
}

// CGMSetG7PairingCode is SetDexcomG7PairingCodeRequest: a new G7 sensor, which
// ends any session running.
func (ps *PumpState) CGMSetG7PairingCode(code int) CGMCommandStatus {
	var status CGMCommandStatus
	_ = ps.UpdateCGM(func(c *CGMState, now time.Time) error {
		if s, ok := c.refused("SetDexcomG7PairingCodeRequest"); ok {
			status = s
			return nil
		}
		if c.SensorType != CGMSensorTypeDexcomG7 || code < 0 || code > 9999 {
			status = CGMCommandRefused
			return nil
		}
		c.G7PairingCode = code
		c.setPhase(CGMPhaseStartPending, now, "")
		return nil
	})
	return status
}

// CGMStopSession is StopDexcomCGMSensorSessionRequest. With no session there is
// nothing to stop, and the pump still takes it.
func (ps *PumpState) CGMStopSession() CGMCommandStatus {
	var status CGMCommandStatus
	_ = ps.UpdateCGM(func(c *CGMState, now time.Time) error {
		if s, ok := c.refused("StopDexcomCGMSensorSessionRequest"); ok {
			status = s
			return nil
		}
		if c.Phase.Running() && c.Phase != CGMPhaseStopPending {
			c.setPhase(CGMPhaseStopPending, now, "")
		}
		return nil
	})
	return status
}

// DismissCGMAlert clears one CGM alert bit, as acknowledging it does. A
// condition that persists stays cleared until the next session.
func (ps *PumpState) DismissCGMAlert(bit uint) {
	if bit > 63 {
		return
	}
	_ = ps.UpdateCGM(func(c *CGMState, _ time.Time) error {
		c.Dismissed |= 1 << bit
		c.ExtraAlerts &^= 1 << bit
		return nil
	})
}

// CGMTargets are the phases and endings the harness can move a session to.
var CGMTargets = []string{
	"next", string(CGMPhaseStartPending), string(CGMPhaseWarmup), string(CGMPhaseActive), string(CGMPhaseGrace),
	string(CGMPhaseSignalLoss), string(CGMPhaseStopPending), string(CGMPhaseStopped),
	CGMEndExpired, CGMEndFailed, CGMEndNotFound,
}

// MoveTo puts the session in a phase now, as if it had got there on its own:
// a session phase dates the session's start so that the timed phases after it
// fall where they would (sessionAge, when given, sets it instead). "next" is
// the phase after this one; "expired", "failed" and "not_found" end the
// session that way.
func (c *CGMState) MoveTo(target string, now time.Time, sessionAge *time.Duration) error {
	t := c.timing()
	if target == "next" {
		target = c.nextPhase(t)
	}
	if c.SensorType == CGMSensorTypeNone && target != string(CGMPhaseStopped) {
		return fmt.Errorf("no sensor type set: PUT /api/cgm {\"sensor_type\": \"g6\"|\"g7\"} first")
	}
	switch target {
	case string(CGMPhaseWarmup), string(CGMPhaseActive), string(CGMPhaseSignalLoss), string(CGMPhaseGrace):
		return c.moveIntoSession(CGMPhase(target), now, sessionAge, t)
	case string(CGMPhaseStartPending):
		c.setPhase(CGMPhaseStartPending, now, "")
	case string(CGMPhaseStopPending):
		if !c.Phase.Running() {
			return fmt.Errorf("no session to stop")
		}
		c.setPhase(CGMPhaseStopPending, now, "")
	case string(CGMPhaseStopped):
		c.setPhase(CGMPhaseStopped, now, CGMEndUser)
	case CGMEndExpired, CGMEndFailed, CGMEndNotFound:
		c.Dismissed = 0
		c.setPhase(CGMPhaseStopped, now, target)
	default:
		return fmt.Errorf("unknown CGM phase %q (%s)", target, strings.Join(CGMTargets, " | "))
	}
	return nil
}

// moveIntoSession puts a running session in phase p, its start dated sessionAge
// back, or where p's own start would put it. Between active and signal_loss the
// session goes on.
func (c *CGMState) moveIntoSession(p CGMPhase, now time.Time, sessionAge *time.Duration, t CGMTiming) error {
	age := map[CGMPhase]time.Duration{CGMPhaseWarmup: 0, CGMPhaseActive: t.Warmup, CGMPhaseSignalLoss: t.Warmup, CGMPhaseGrace: t.Lifetime}[p]
	if sessionAge != nil {
		age = *sessionAge
	}
	if p == CGMPhaseGrace && t.Grace == 0 {
		return fmt.Errorf("a sensor of type %d has no grace period", c.SensorType)
	}
	ongoing := (p == CGMPhaseActive || p == CGMPhaseSignalLoss) && (c.Phase == CGMPhaseActive || c.Phase == CGMPhaseSignalLoss)
	if sessionAge != nil || !ongoing {
		c.startSession(now.Add(-age))
	}
	c.setPhase(p, now, "")
	return nil
}

func (c *CGMState) nextPhase(t CGMTiming) string {
	switch c.Phase {
	case CGMPhaseStopped:
		return string(CGMPhaseStartPending)
	case CGMPhaseStartPending:
		return string(CGMPhaseWarmup)
	case CGMPhaseWarmup, CGMPhaseSignalLoss:
		return string(CGMPhaseActive)
	case CGMPhaseActive:
		if t.Grace > 0 {
			return string(CGMPhaseGrace)
		}
		return CGMEndExpired
	case CGMPhaseGrace:
		return CGMEndExpired
	default:
		return string(CGMPhaseStopped)
	}
}

// ResetCGM replaces the CGM with one that has never run a session, of the
// given sensor type, with the harness's settings cleared.
func (ps *PumpState) ResetCGM(sensorType int) {
	now := ps.Now()
	ps.mutex.Lock()
	defer ps.mutex.Unlock()
	old := ps.CGMState
	fresh := newCGMState(now)
	fresh.SensorType, fresh.TransmitterID = sensorType, ""
	fresh.Phase, fresh.PhaseSince, fresh.SessionStart = CGMPhaseStopped, now, time.Time{}
	fresh.notifiedPhase, fresh.notifiedAlerts = old.notifiedPhase, old.notifiedAlerts
	ps.CGMState = fresh
}

// ParseCGMSensorType reads a sensor type by name (g6, g7, fsl2, none) or number.
func ParseCGMSensorType(s string) (int, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "none", "0", "":
		return CGMSensorTypeNone, nil
	case "g6", "dexcom_g6", "dexcomg6", "1":
		return CGMSensorTypeDexcomG6, nil
	case "fsl2", "libre2", "libre", "2":
		return CGMSensorTypeFSL2, nil
	case "g7", "dexcom_g7", "dexcomg7", "3":
		return CGMSensorTypeDexcomG7, nil
	}
	return 0, fmt.Errorf("unknown CGM sensor type %q (none | g6 | fsl2 | g7)", s)
}

// CGMSensorTypeName is a sensor type's name, as ParseCGMSensorType reads it.
func CGMSensorTypeName(t int) string {
	switch t {
	case CGMSensorTypeNone:
		return "none"
	case CGMSensorTypeDexcomG6:
		return "g6"
	case CGMSensorTypeFSL2:
		return "fsl2"
	case CGMSensorTypeDexcomG7:
		return "g7"
	}
	return fmt.Sprintf("unknown(%d)", t)
}
