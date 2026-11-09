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
			"tempRate":                basal.TempBasalRate,
			"normalRate":              profileRate,
			"durationSeconds":         duration.Seconds(),
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
// milliseconds counted in whole minutes, as Mobi exports carry it: zero for a
// temp rate that expired on its own, positive for one cut short by a stop or a
// suspend.
func (ps *PumpState) RecordTempRateCompleted(temp TempRateSnapshot, profileRate float64, when time.Time) uint32 {
	if when.IsZero() {
		when = ps.Now()
	}

	timeLeft := time.Duration(0)
	if !temp.EndTime.IsZero() && temp.EndTime.After(when) {
		timeLeft = temp.EndTime.Sub(when).Truncate(time.Minute)
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
			"timeLeft":   int(timeLeft.Milliseconds()),
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

// notificationRecords are the records a category's activation and clearing
// write, and the wire field that names the notification in both. A reminder
// writes none; a malfunction is never cleared.
var notificationRecords = map[NotificationCategory]struct {
	activatedID int
	activated   string
	clearedID   int
	cleared     string
	idField     string
}{
	NotificationAlert:       {HistoryAlertActivated, "AlertActivated", HistoryAlertCleared, "AlertCleared", "alertId"},
	NotificationAlarm:       {HistoryAlarmActivated, "AlarmActivated", HistoryAlarmCleared, "AlarmCleared", "alarmId"},
	NotificationCGMAlert:    {HistoryCGMAlertActivated, "CgmAlertActivated", HistoryCGMAlertCleared, "CgmAlertCleared", "alertId"},
	NotificationMalfunction: {HistoryMalfunction, "Malfunction", 0, "", "malfId"},
}

// notificationExtra is the JSON-only context on a notification record.
func notificationExtra(alert Alert) map[string]interface{} {
	return map[string]interface{}{
		"alertTypeId":   int(alert.Type),
		"alertTypeName": alert.TypeName(),
		"category":      alert.Category.String(),
		"name":          alert.Name(),
		"message":       alert.Message,
	}
}

// RecordNotificationActivated writes the record a pump writes when it raises
// the notification: AlertActivated, AlarmActivated, CgmAlertActivated or
// Malfunction, at the alert's timestamp, naming it by its bit -- the
// AlertResponseType / AlarmResponseType / MalfunctionType value pumpX2 and
// TandemKit read the id field as. A malfunction carries its fault code as
// faultLocatorData. Returns the sequence, 0 when the category writes none.
func (ps *PumpState) RecordNotificationActivated(alert Alert, reason string) uint32 {
	records, ok := notificationRecords[alert.Category]
	if !ok {
		return 0
	}
	data := map[string]interface{}{records.idField: alert.Bit}
	if alert.Category == NotificationMalfunction {
		data["faultLocatorData"] = alert.FaultID
	}
	extra := notificationExtra(alert)
	extra["alertId"] = alert.ID
	extra["priority"] = int(alert.Priority)
	extra["reason"] = reason
	return ps.AddHistoryLogEntryAtWithExtra(records.activatedID, records.activated, alert.Timestamp, data, extra)
}

// RecordNotificationCleared writes the record a pump writes when the
// notification stops standing, at the instant `when`: AlertCleared,
// AlarmCleared or CgmAlertCleared, naming it as its activation did. One record
// per notification. Returns the sequence, 0 when the category writes none.
func (ps *PumpState) RecordNotificationCleared(alert Alert, when time.Time) uint32 {
	records, ok := notificationRecords[alert.Category]
	if !ok || records.cleared == "" {
		return 0
	}
	extra := notificationExtra(alert)
	extra["alertId"] = alert.ID
	return ps.AddHistoryLogEntryAtWithExtra(records.clearedID, records.cleared, when,
		map[string]interface{}{records.idField: alert.Bit}, extra)
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
	case AlertNotification:
		return "Notification"
	default:
		return "Unknown"
	}
}
