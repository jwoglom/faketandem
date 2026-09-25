package state

import (
	"time"

	log "github.com/sirupsen/logrus"
)

// One transition, one record.
//
// A delivery lifecycle event -- a bolus ending, a temp rate ending, delivery
// stopping or restarting -- used to be assembled out of two or three steps at
// each call site: read the current state, clear it, then write the history
// record. That left two kinds of holes.
//
//   - A path could clear the state and forget the record. A temp rate ended by
//     a driver's StopTempRate, or by a suspend commanded over the protocol,
//     wrote no TempRateCompleted at all, so a consumer building a timeline out
//     of the history log saw a temp rate that never ended.
//   - Two paths could both see the same thing running and both close it. The
//     simulator expiring a temp rate on a tick while the driver stops it, or a
//     bolus finishing on a tick while the driver cancels it, would write two
//     endings for one event.
//
// The functions below are the only way delivery ends. Each one takes the pump
// state lock, checks and clears the thing it ends in the same critical section,
// and writes exactly one record through the per-type writer in
// history_events.go -- so whichever path gets there first writes the record and
// the loser writes nothing. They deliberately raise no qualifying events: they
// report what they ended, and the caller (protocol router or harness action)
// emits the events its own transport owes the driver.

// EndTempRate ends a running temp rate and writes its one TempRateCompleted
// record, stamped at `when` on the pump's clock (zero means now). Pass the
// instant the temp rate actually ended -- the programmed end for a natural
// expiry, so a coarse simulator tick cannot move the recorded end second; the
// stop or replacement instant for a temp rate cut short.
//
// The returned snapshot is the temp rate as it was running, and the returned
// rate is the profile rate delivery has gone back to. ok is false when no temp
// rate was running, which is also how a path that loses a race to end the same
// temp rate learns that the winner already wrote the record.
func (ps *PumpState) EndTempRate(when time.Time) (temp TempRateSnapshot, profileRate float64, ok bool) {
	if when.IsZero() {
		when = ps.Now()
	}

	temp, profileRate, ok = ps.takeActiveTempRate()
	if !ok {
		return temp, profileRate, false
	}

	ps.RecordTempRateCompleted(temp, profileRate, when)
	log.Infof("Temp rate %d ended, back to profile %.3f U/hr", temp.TempRateID, profileRate)
	return temp, profileRate, true
}

// takeActiveTempRate clears a running temp rate and returns what was running,
// atomically. The temp rate's other fields are left as they were, so the state
// snapshot can still report the temp rate that just finished.
func (ps *PumpState) takeActiveTempRate() (TempRateSnapshot, float64, bool) {
	ps.mutex.Lock()
	defer ps.mutex.Unlock()

	profileRate := ps.Basal.CurrentRate
	if !ps.Basal.TempBasalActive {
		return TempRateSnapshot{}, profileRate, false
	}

	temp := TempRateSnapshot{
		Active:     true,
		Percent:    ps.Basal.TempBasalPercent,
		Rate:       ps.Basal.TempBasalRate,
		StartTime:  ps.Basal.TempBasalStart,
		EndTime:    ps.Basal.TempBasalEnd,
		TempRateID: ps.Basal.TempRateID,
	}
	ps.Basal.TempBasalActive = false
	return temp, profileRate, true
}

// EndBolus ends an in-progress bolus, writing both halves of its ending: the
// last-bolus record a driver's next LastBolusStatus query reports, and the one
// BolusCompleted history record. endReasonID is the wire's BolusStatus (see
// the BolusEndReason* constants) and lands in both.
//
// deliveredOverride, when non-nil, is the volume the bolus is declared to have
// delivered -- a harness saying "this bolus put in 1.4 of its 2 units before it
// stopped". With no override, a bolus ended as completed delivered everything
// it asked for (that is what completed means) and one cut short delivered
// whatever the simulator had counted so far. Insulin beyond what was already
// counted is taken out of the reservoir here, so neither an override nor a
// declared completion can invent insulin that never left the cartridge.
//
// ok is false when no bolus was in progress.
func (ps *PumpState) EndBolus(endReasonID int, deliveredOverride *float64) (LastBolusRecord, bool) {
	bolus, ok := ps.takeActiveBolus(endReasonID, deliveredOverride)
	if !ok {
		return LastBolusRecord{}, false
	}

	record := LastBolusRecord{
		BolusID:        bolus.BolusID,
		RequestedUnits: bolus.UnitsTotal,
		DeliveredUnits: bolus.UnitsDelivered,
		SourceID:       bolus.SourceID,
		TypeBitmask:    bolus.TypeBitmask,
		EndReasonID:    endReasonID,
		EndTime:        ps.Now(),
	}
	ps.RecordLastBolus(record)
	ps.RecordBolusCompleted(bolus.BolusID, bolus.UnitsDelivered, bolus.UnitsTotal, endReasonID)
	return record, true
}

// takeActiveBolus clears an in-progress bolus and returns what was running,
// atomically, settling the delivered volume on the way out.
func (ps *PumpState) takeActiveBolus(endReasonID int, deliveredOverride *float64) (BolusState, bool) {
	ps.mutex.Lock()
	defer ps.mutex.Unlock()

	if !ps.Bolus.Active {
		return BolusState{}, false
	}

	delivered := deliveredOverride
	if delivered == nil && endReasonID == BolusEndReasonCompleted {
		delivered = &ps.Bolus.UnitsTotal
	}

	if delivered != nil {
		if extra := *delivered - ps.Bolus.UnitsDelivered; extra > 0 {
			ps.Reservoir.CurrentUnits -= extra
			if ps.Reservoir.CurrentUnits < 0 {
				ps.Reservoir.CurrentUnits = 0
			}
			ps.IOB += extra
			ps.TDD += extra
		}
		ps.Bolus.UnitsDelivered = *delivered
	}

	ps.Bolus.Active = false
	ps.Bolus.Stalled = false
	return *ps.Bolus, true
}

// SuspendOutcome reports what stopping delivery actually ended, so the caller
// can raise the qualifying events its transport owes the driver. The history
// records are already written by the time it is returned.
type SuspendOutcome struct {
	// Changed is false when delivery was already suspended. Nothing was ended
	// and nothing was recorded: a second suspend command is not a second
	// transition, and must not write a second PumpingSuspended record.
	Changed bool
	// Bolus is the bolus the suspend cut short, or nil if none was running.
	Bolus *LastBolusRecord
	// TempRate is the temp rate the suspend ended, or nil if none was running.
	TempRate *TempRateSnapshot
	// PreviousRate is the rate that was being delivered before the stop, and
	// ProfileRate the scheduled rate delivery returns to on resume.
	PreviousRate float64
	ProfileRate  float64
}

// SuspendDelivery stops delivery for the given reason (see SuspendReasonID),
// ending anything in progress on the way down.
//
// A real stop is not just a flag: it halts an in-progress bolus and any running
// temp rate, and each of those endings is its own history record, written
// before the PumpingSuspended record that caused them so the log reads in the
// order it happened. The protocol path used to set only the flag, which left a
// temp rate running forever in the log and a bolus with a start and no end.
func (ps *PumpState) SuspendDelivery(reason string) SuspendOutcome {
	outcome := SuspendOutcome{
		PreviousRate: ps.GetBasalRate(),
		ProfileRate:  ps.GetProfileBasalRate(),
	}

	if !ps.compareAndSetSuspended(true, reason) {
		return outcome
	}
	outcome.Changed = true

	if record, ok := ps.EndBolus(BolusEndReasonStopped, nil); ok {
		outcome.Bolus = &record
	}
	if temp, profileRate, ok := ps.EndTempRate(time.Time{}); ok {
		outcome.TempRate = &temp
		outcome.ProfileRate = profileRate
	}

	ps.RecordPumpingSuspended(reason)
	log.Infof("Delivery suspended (%s)", reason)
	return outcome
}

// ResumeDelivery restarts delivery, writing one PumpingResumed record. It
// reports false when delivery was not suspended, in which case nothing was
// recorded -- a resume that resumes nothing is not a transition.
func (ps *PumpState) ResumeDelivery() bool {
	if !ps.compareAndSetSuspended(false, "") {
		return false
	}

	ps.RecordPumpingResumed()
	log.Info("Delivery resumed")
	return true
}

// compareAndSetSuspended moves the suspend flag and reports whether it actually
// moved. Checking and setting in one critical section is what makes "exactly
// one record per transition" hold when two paths suspend at once.
func (ps *PumpState) compareAndSetSuspended(suspended bool, reason string) bool {
	ps.mutex.Lock()
	defer ps.mutex.Unlock()

	if ps.PumpingSuspended == suspended {
		return false
	}
	ps.PumpingSuspended = suspended
	if suspended {
		ps.suspendReason = reason
	} else {
		ps.suspendReason = ""
	}
	return true
}
