package state

import "time"

// The one writer per record type.
//
// A history record used to be assembled at each call site, so the same event
// reached a driver with different field names depending on which path produced
// it: the protocol handler wrote BolusActivated with `bolusSize` while the
// harness wrote `units`, TempRateActivated with `durationMilliseconds` versus
// `minutes`, PumpingSuspended with `reasonId` versus `reason`, and the
// simulator's BolusCompleted carried no end reason at all. A consumer then had
// to know which path had produced a record before it could read it.
//
// Every path now goes through the writers below, which settle two things:
//
//   - Wire fields are named exactly as pumpX2 and TandemKit name them, and
//     nothing else goes in `Data`. The Swift class for each record
//     (Sources/TandemCore/Messages/HistoryLog/*.swift) is the reference: if its
//     `init(cargo:)` does not read a field, that field is not a record field.
//   - Pump-side context a 26-byte record cannot hold -- a temp rate's delivered
//     volume, the string reason behind a suspend, the alert type behind an
//     alarm -- goes in `Extra`, which is reported in JSON and never encoded.
//
// Old field names stay accepted as decoding aliases in encodeHistoryPayload, so
// a scenario still holding an input file with `units` in it keeps working.

// Suspend reason IDs, matching pumpX2/TandemKit's
// PumpingSuspendedHistoryLog.SuspendReason.
const (
	// SuspendReasonIDUserAborted is a person stopping delivery on the pump.
	SuspendReasonIDUserAborted = 0
	// SuspendReasonIDAlarm is an alarm stopping delivery.
	SuspendReasonIDAlarm = 1
	// SuspendReasonIDMalfunction is a hardware fault stopping delivery. An
	// occlusion is reported here: the enum has no occlusion member of its own,
	// and an occlusion is the pump detecting that it physically cannot deliver.
	SuspendReasonIDMalfunction = 2
	// SuspendReasonIDPredictiveLowGlucose is Basal-IQ/Control-IQ suspending.
	SuspendReasonIDPredictiveLowGlucose = 6
)

// SuspendReasonID maps the emulator's suspend reason strings ("user",
// "occlusion", "alarm", "plgs") onto the wire's reason id. An unrecognized
// reason is reported as user-aborted, the value a real pump uses for every
// observed user-initiated suspend.
func SuspendReasonID(reason string) int {
	switch reason {
	case "occlusion", "malfunction":
		return SuspendReasonIDMalfunction
	case "alarm":
		return SuspendReasonIDAlarm
	case "plgs", "predictive_low_glucose":
		return SuspendReasonIDPredictiveLowGlucose
	default:
		return SuspendReasonIDUserAborted
	}
}

// RecordBolusActivated writes the BolusActivated record for a bolus that has
// just begun, from whichever path started it.
//
// Wire fields (BolusActivatedHistoryLog): bolusId, selectedIob, iob, bolusSize.
// The bolus *source* is not one of them -- it rides on BolusDeliveryHistoryLog
// -- so it is reported as context only.
func (ps *PumpState) RecordBolusActivated(bolusID uint32, units float64, sourceID int) uint32 {
	return ps.AddHistoryLogEntryWithExtra(HistoryBolusActivated, "BolusActivated",
		map[string]interface{}{
			"bolusId":     bolusID,
			"selectedIob": 0,
			"iob":         ps.GetIOB(),
			"bolusSize":   units,
		},
		map[string]interface{}{
			"bolusSourceId": sourceID,
		})
}

// RecordBolusCompleted writes the BolusCompleted record for a bolus that has
// just ended, however it ended.
//
// Wire fields (BolusCompletedHistoryLog): completionStatusId, bolusId, iob,
// insulinDelivered, insulinRequested. completionStatusId IS the end reason --
// it decodes as LastBolusStatusAbstractResponse.BolusStatus, the same enum
// LastBolusStatus reports -- so there is no separate endReasonId on the wire
// and every path must set it. The simulator's completion used to omit it,
// which made a completed bolus indistinguishable from a canceled one in the
// log.
func (ps *PumpState) RecordBolusCompleted(bolusID uint32, delivered, requested float64, endReasonID int) uint32 {
	return ps.AddHistoryLogEntryWithExtra(HistoryBolusCompleted, "BolusCompleted",
		map[string]interface{}{
			"completionStatusId": endReasonID,
			"bolusId":            bolusID,
			"iob":                ps.GetIOB(),
			"insulinDelivered":   delivered,
			"insulinRequested":   requested,
		},
		nil)
}

// RecordTempRateActivated writes the TempRateActivated record for a temp rate
// that has just started.
//
// Wire fields (TempRateActivatedHistoryLog): percent, durationMilliseconds,
// tempRateId. The absolute rates are context: the record carries a percentage,
// and what that percentage resolves to depends on the profile the pump was
// running.
func (ps *PumpState) RecordTempRateActivated(basal *BasalState, profileRate float64) uint32 {
	duration := basal.TempBasalEnd.Sub(basal.TempBasalStart)
	return ps.AddHistoryLogEntryWithExtra(HistoryTempRateActivated, "TempRateActivated",
		map[string]interface{}{
			"percent":              basal.TempBasalPercent,
			"durationMilliseconds": duration.Seconds() * 1000,
			"tempRateId":           basal.TempRateID,
		},
		map[string]interface{}{
			"tempRate":        basal.TempBasalRate,
			"normalRate":      profileRate,
			"durationSeconds": duration.Seconds(),
			"scheduledEndPumpSeconds": ps.PumpTimeFor(basal.TempBasalEnd),
		})
}

// RecordTempRateCompleted writes the TempRateCompleted record for a temp rate
// that has just ended, at the instant `when` on the pump's Clock.
//
// Wire fields (TempRateCompletedHistoryLog): tempRateId and timeLeft, and
// nothing else -- the record's `init(cargo:)` reads a short at cargo 12 and a
// uint32 at cargo 14, so the rate, the profile rate, the start and the
// delivered volume have nowhere on the wire to go and are reported as context.
// timeLeft is how much of the programmed duration was still to run, in
// seconds: zero for a temp rate that expired on its own, positive for one cut
// short by a stop or a suspend.
func (ps *PumpState) RecordTempRateCompleted(temp TempRateSnapshot, profileRate float64, when time.Time) uint32 {
	if when.IsZero() {
		when = ps.Now()
	}

	timeLeft := 0.0
	if !temp.EndTime.IsZero() && temp.EndTime.After(when) {
		timeLeft = temp.EndTime.Sub(when).Seconds()
	}

	delivered := 0.0
	durationSeconds := 0.0
	if !temp.StartTime.IsZero() && when.After(temp.StartTime) {
		durationSeconds = when.Sub(temp.StartTime).Seconds()
		delivered = temp.Rate * durationSeconds / 3600.0
	}

	extra := map[string]interface{}{
		"tempRate":        temp.Rate,
		"normalRate":      profileRate,
		"percent":         temp.Percent,
		"durationSeconds": durationSeconds,
		"deliveredUnits":  delivered,
		"endedEarly":      timeLeft > 0,
	}
	if !temp.StartTime.IsZero() {
		extra["startPumpSeconds"] = ps.PumpTimeFor(temp.StartTime)
		extra["startTime"] = temp.StartTime.UTC().Format(time.RFC3339Nano)
	}

	return ps.AddHistoryLogEntryAtWithExtra(HistoryTempRateCompleted, "TempRateCompleted", when,
		map[string]interface{}{
			"tempRateId": temp.TempRateID,
			"timeLeft":   int(timeLeft),
		},
		extra)
}

// RecordPumpingSuspended writes the PumpingSuspended record for a stop, from
// whichever path stopped delivery.
//
// Wire fields (PumpingSuspendedHistoryLog): preSuspendState, insulinAmount,
// reasonId, rpaTimeout. `reason` is a string this emulator uses internally and
// is reported as context; the wire carries only the id.
func (ps *PumpState) RecordPumpingSuspended(reason string) uint32 {
	return ps.AddHistoryLogEntryWithExtra(HistoryPumpingSuspended, "PumpingSuspended",
		map[string]interface{}{
			"preSuspendState": 106,
			"insulinAmount":   int(ps.GetReservoirLevel()),
			"reasonId":        SuspendReasonID(reason),
			"rpaTimeout":      15,
		},
		map[string]interface{}{
			"reason": reason,
		})
}

// RecordPumpingResumed writes the PumpingResumed record for a restart.
//
// Wire fields (PumpingResumedHistoryLog): preResumeState, insulinAmount.
func (ps *PumpState) RecordPumpingResumed() uint32 {
	return ps.AddHistoryLogEntryWithExtra(HistoryPumpingResumed, "PumpingResumed",
		map[string]interface{}{
			"preResumeState": 100,
			"insulinAmount":  int(ps.GetReservoirLevel()),
		},
		nil)
}

// RecordAlarmActivated writes the AlarmActivated record for an alarm the pump
// has just raised.
//
// Wire fields (AlarmActivatedHistoryLog): alarmId, faultLocatorData, param1,
// param2. alarmId is the alarm's own id, which the matching AlarmCleared
// repeats -- that pairing is what lets a timeline close an alarm. The alert
// *type* behind it is this emulator's own classification, so it is reported as
// context (alertTypeId / alertTypeName) rather than squeezed into a wire field
// that means something else.
func (ps *PumpState) RecordAlarmActivated(alert Alert, reason string) uint32 {
	return ps.AddHistoryLogEntryAtWithExtra(HistoryAlarmActivated, "AlarmActivated", alert.Timestamp,
		map[string]interface{}{
			"alarmId": alert.ID,
		},
		map[string]interface{}{
			"alertTypeId":   int(alert.Type),
			"alertTypeName": alert.Type.String(),
			"priority":      int(alert.Priority),
			"reason":        reason,
			"message":       alert.Message,
		})
}

// RecordAlarmCleared writes the AlarmCleared record for one alarm that has
// stopped standing, at the instant `when`.
//
// Wire fields (AlarmClearedHistoryLog): alarmId, matching the AlarmActivated it
// closes. One record per alarm: a single record saying "3 alarms went away"
// (which is what the harness used to write) is not a record format a pump has.
func (ps *PumpState) RecordAlarmCleared(alert Alert, when time.Time) uint32 {
	return ps.AddHistoryLogEntryAtWithExtra(HistoryAlarmCleared, "AlarmCleared", when,
		map[string]interface{}{
			"alarmId": alert.ID,
		},
		map[string]interface{}{
			"alertTypeId":   int(alert.Type),
			"alertTypeName": alert.Type.String(),
			"message":       alert.Message,
		})
}

// String names the alert type, for the JSON context on alarm records.
func (t AlertType) String() string {
	switch t {
	case AlertLowReservoir:
		return "LowReservoir"
	case AlertLowBattery:
		return "LowBattery"
	case AlertCartridgeExpired:
		return "CartridgeExpired"
	case AlertOcclusion:
		return "Occlusion"
	case AlertBasalSuspended:
		return "BasalSuspended"
	default:
		return "Unknown"
	}
}
