package state

import (
	"testing"
	"time"
)

func cgmPump(t *testing.T, sensorType int) (*PumpState, *ManualClock) {
	t.Helper()
	ps, clock := lifecyclePump(t)
	ps.ResetCGM(sensorType)
	ps.AdvanceCGM()
	return ps, clock
}

func wantPhase(t *testing.T, ps *PumpState, phase CGMPhase, sessionState int) CGMSnapshot {
	t.Helper()
	s := ps.CGM()
	if s.Phase != phase || s.SessionState != sessionState {
		t.Fatalf("phase %s (session state %d), want %s (%d); end reason %q", s.Phase, s.SessionState, phase, sessionState, s.EndReason)
	}
	return s
}

func hasAlert(s CGMSnapshot, bit uint) bool { return s.AlertBitmask&(1<<bit) != 0 }

func TestTheDefaultCGMIsAnActiveG6Session(t *testing.T) {
	ps, _ := lifecyclePump(t)
	ps.ResetCGM(CGMSensorTypeNone)
	fresh := NewPumpState()
	s := fresh.CGM()
	if s.Phase != CGMPhaseActive || s.SensorType != CGMSensorTypeDexcomG6 || s.EGVStatus != EGVStatusValid || s.TransmitterBattery != CGMBatteryOK {
		t.Fatalf("default CGM = %+v", s)
	}
	if s := ps.CGM(); s.Phase != CGMPhaseStopped || s.EndReason != "" || s.SessionState != CGMSessionStopped {
		t.Fatalf("reset CGM = %+v", s)
	}
}

func TestAG7PairingCodeStartsASessionAfterItsWarmup(t *testing.T) {
	ps, clock := cgmPump(t, CGMSensorTypeNone)

	if st := ps.CGMSetG7PairingCode(1234); st != CGMCommandRefused {
		t.Fatalf("a G7 code with no sensor type answered %d", st)
	}
	if st := ps.CGMSetSensorType(CGMSensorTypeDexcomG7); st != CGMCommandOK {
		t.Fatalf("SetSensorType(G7) = %d", st)
	}
	if st := ps.CGMSetG7PairingCode(1234); st != CGMCommandOK {
		t.Fatalf("SetDexcomG7PairingCode = %d", st)
	}
	paired := ps.Now()
	s := wantPhase(t, ps, CGMPhaseStartPending, CGMSessionStartPending)
	if s.G7PairingCode != 1234 || s.EGVStatus != EGVStatusUnavailable {
		t.Fatalf("start pending = %+v", s)
	}

	clock.Advance(30 * time.Second)
	s = wantPhase(t, ps, CGMPhaseWarmup, CGMSessionActive)
	if !s.SessionStart.Equal(paired.Add(30*time.Second)) || s.EGV != 0 || s.TransmitterBattery != CGMBatteryUnavailable {
		t.Fatalf("warmup = %+v", s)
	}

	clock.Advance(27 * time.Minute)
	s = wantPhase(t, ps, CGMPhaseActive, CGMSessionActive)
	if s.EGVStatus != EGVStatusValid || s.EGV != 120 || !s.EGVTime.Equal(s.WarmupEnds) {
		t.Fatalf("active = %+v", s)
	}
	if s.SessionDuration != 10*24*time.Hour || s.SessionRemaining != 10*24*time.Hour-27*time.Minute {
		t.Fatalf("duration %v remaining %v", s.SessionDuration, s.SessionRemaining)
	}
}

func TestAG7SessionExpiresAfterItsGracePeriod(t *testing.T) {
	ps, clock := cgmPump(t, CGMSensorTypeDexcomG7)
	ps.CGMSetG7PairingCode(1234)
	clock.Advance(30*time.Second + 9*24*time.Hour + 34*time.Minute)

	s := wantPhase(t, ps, CGMPhaseActive, CGMSessionActive)
	if !hasAlert(s, CGMAlertSensorExpiring) {
		t.Fatalf("a day before expiry the alerts are %v", s.AlertNames())
	}

	clock.Advance(24 * time.Hour)
	s = wantPhase(t, ps, CGMPhaseGrace, CGMSessionActive)
	if !s.GracePeriod || !hasAlert(s, CGMAlertGracePeriodStart) || hasAlert(s, CGMAlertSensorExpiring) {
		t.Fatalf("grace = %+v %v", s, s.AlertNames())
	}

	clock.Advance(12 * time.Hour)
	s = wantPhase(t, ps, CGMPhaseStopped, CGMSessionStopped)
	if s.EndReason != CGMEndExpired || !hasAlert(s, CGMAlertSensorExpired) || s.EGVStatus != EGVStatusUnavailable {
		t.Fatalf("expired = %+v %v", s, s.AlertNames())
	}
	if !s.SessionStart.IsZero() || s.SessionRemaining != 0 {
		t.Fatalf("a stopped session still reports its times: %+v", s)
	}
}

func TestAG6SessionNeedsATransmitterAndHasNoGracePeriod(t *testing.T) {
	ps, clock := cgmPump(t, CGMSensorTypeDexcomG6)

	if st := ps.CGMStartG6Session(0); st != CGMCommandRefused {
		t.Fatalf("a G6 start with no transmitter answered %d", st)
	}
	if st := ps.CGMSetG6TransmitterID("8G"); st != CGMCommandRefused {
		t.Fatalf("a short transmitter ID answered %d", st)
	}
	if st := ps.CGMSetG6TransmitterID("8gxyz1"); st != CGMCommandOK {
		t.Fatalf("SetG6TransmitterId = %d", st)
	}
	if st := ps.CGMStartG6Session(5678); st != CGMCommandOK {
		t.Fatalf("StartDexcomG6SensorSession = %d", st)
	}
	if st := ps.CGMStartG6Session(5678); st != CGMCommandRefused {
		t.Fatalf("a second start while one runs answered %d", st)
	}
	s := ps.CGM()
	if s.TransmitterID != "8GXYZ1" || s.G6SensorCode != 5678 || s.TransmitterBattery != CGMBatteryOK {
		t.Fatalf("after the start = %+v", s)
	}

	clock.Advance(30*time.Second + 2*time.Hour)
	wantPhase(t, ps, CGMPhaseActive, CGMSessionActive)
	clock.Advance(10*24*time.Hour - 2*time.Hour)
	s = wantPhase(t, ps, CGMPhaseStopped, CGMSessionStopped)
	if s.EndReason != CGMEndExpired {
		t.Fatalf("end reason %q", s.EndReason)
	}
}

func TestAPairingThatDoesNotMatchTheSensorWornNeverFindsIt(t *testing.T) {
	ps, clock := cgmPump(t, CGMSensorTypeDexcomG7)
	ps.CGMState.BodyG7Code = "4321"

	ps.CGMSetG7PairingCode(1234)
	clock.Advance(30 * time.Second)
	s := wantPhase(t, ps, CGMPhaseStopped, CGMSessionStopped)
	if s.EndReason != CGMEndNotFound || !hasAlert(s, CGMAlertFailedConnection) {
		t.Fatalf("not found = %+v %v", s, s.AlertNames())
	}

	ps.CGMSetG7PairingCode(4321)
	clock.Advance(30 * time.Second)
	s = wantPhase(t, ps, CGMPhaseWarmup, CGMSessionActive)
	if s.AlertBitmask != 0 {
		t.Fatalf("a new session kept the old alerts: %v", s.AlertNames())
	}
}

func TestStoppingASessionPassesThroughStopPending(t *testing.T) {
	ps, clock := cgmPump(t, CGMSensorTypeDexcomG7)
	ps.CGMSetG7PairingCode(1234)
	clock.Advance(time.Hour)
	wantPhase(t, ps, CGMPhaseActive, CGMSessionActive)

	if st := ps.CGMStopSession(); st != CGMCommandOK {
		t.Fatalf("StopDexcomCGMSensorSession = %d", st)
	}
	wantPhase(t, ps, CGMPhaseStopPending, CGMSessionStopPending)
	clock.Advance(5 * time.Second)
	s := wantPhase(t, ps, CGMPhaseStopped, CGMSessionStopped)
	if s.EndReason != CGMEndUser || s.AlertBitmask != 0 {
		t.Fatalf("stopped = %+v", s)
	}
	if st := ps.CGMStopSession(); st != CGMCommandOK {
		t.Fatalf("a stop with no session answered %d", st)
	}
}

func TestChangingTheSensorTypeEndsTheSession(t *testing.T) {
	ps, _ := lifecyclePump(t)
	if st := ps.CGMSetSensorType(CGMSensorTypeDexcomG7); st != CGMCommandOK {
		t.Fatalf("SetSensorType = %d", st)
	}
	s := wantPhase(t, ps, CGMPhaseStopped, CGMSessionStopped)
	if s.SensorType != CGMSensorTypeDexcomG7 || s.EndReason != CGMEndUser {
		t.Fatalf("after the type change = %+v", s)
	}
	if st := ps.CGMSetSensorType(9); st != CGMCommandRefused {
		t.Fatalf("an unknown sensor type answered %d", st)
	}
}

func TestWithAutoOffOnlyMoveToAdvancesTheSession(t *testing.T) {
	ps, clock := cgmPump(t, CGMSensorTypeDexcomG7)
	ps.CGMState.Auto = false
	ps.CGMSetG7PairingCode(1234)
	clock.Advance(48 * time.Hour)
	wantPhase(t, ps, CGMPhaseStartPending, CGMSessionStartPending)

	move := func(target string, age *time.Duration) {
		t.Helper()
		if err := ps.UpdateCGM(func(c *CGMState, now time.Time) error { return c.MoveTo(target, now, age) }); err != nil {
			t.Fatalf("MoveTo(%s): %v", target, err)
		}
	}
	move("next", nil)
	wantPhase(t, ps, CGMPhaseWarmup, CGMSessionActive)
	move("next", nil)
	s := wantPhase(t, ps, CGMPhaseActive, CGMSessionActive)
	if !s.SessionStart.Equal(ps.Now().Add(-27 * time.Minute)) {
		t.Fatalf("an active session moved to started at %v", s.SessionStart)
	}

	age := 10*24*time.Hour - 3*time.Hour
	move("active", &age)
	s = ps.CGM()
	if s.SessionRemaining != 3*time.Hour || !hasAlert(s, CGMAlertSensorExpiring) {
		t.Fatalf("aged session = %+v %v", s, s.AlertNames())
	}
	move("signal_loss", nil)
	s = wantPhase(t, ps, CGMPhaseSignalLoss, CGMSessionActive)
	if s.SessionRemaining != 3*time.Hour || s.EGVStatus != EGVStatusUnavailable || !hasAlert(s, CGMAlertOutOfRange) {
		t.Fatalf("signal loss = %+v %v", s, s.AlertNames())
	}
	move("failed", nil)
	s = wantPhase(t, ps, CGMPhaseStopped, CGMSessionStopped)
	if s.EndReason != CGMEndFailed || !hasAlert(s, CGMAlertSensorFailed) {
		t.Fatalf("failed = %+v %v", s, s.AlertNames())
	}

	ps.CGMState.SensorType = CGMSensorTypeDexcomG6
	if err := ps.UpdateCGM(func(c *CGMState, now time.Time) error { return c.MoveTo("grace", now, nil) }); err == nil {
		t.Fatal("a G6 moved into a grace period")
	}
}

func TestAdvanceCGMReportsEachNewPhaseAndAlertOnce(t *testing.T) {
	ps, clock := cgmPump(t, CGMSensorTypeDexcomG7)

	if phase, alerts := ps.AdvanceCGM(); phase || alerts {
		t.Fatalf("nothing happened, AdvanceCGM = %v %v", phase, alerts)
	}
	ps.CGMSetG7PairingCode(1234)
	if phase, _ := ps.AdvanceCGM(); !phase {
		t.Fatal("start pending not reported")
	}
	if phase, _ := ps.AdvanceCGM(); phase {
		t.Fatal("start pending reported twice")
	}
	clock.Advance(10*24*time.Hour + time.Hour)
	if phase, alerts := ps.AdvanceCGM(); !phase || !alerts {
		t.Fatalf("grace = %v %v", phase, alerts)
	}

	ps.DismissCGMAlert(CGMAlertGracePeriodStart)
	if s := ps.CGM(); hasAlert(s, CGMAlertGracePeriodStart) {
		t.Fatalf("a dismissed alert stayed: %v", s.AlertNames())
	}
	if _, alerts := ps.AdvanceCGM(); alerts {
		t.Fatal("a dismissal was reported as a new alert")
	}
}
