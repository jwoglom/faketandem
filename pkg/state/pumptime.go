package state

import "time"

// TandemEpoch is the epoch every Tandem pump timestamp on the wire counts from:
// 2008-01-01 00:00:00 UTC, i.e. Unix second 1199145600.
//
// pumpX2 (helpers/Dates.java) and TandemKit (TandemCore/Common/Dates.swift)
// both decode pump timestamps -- CurrentBolusStatusResponse.timestamp,
// TimeSinceResetResponse.currentTime, LastBolusStatus*.timestamp,
// TempRateResponse.startTimeRaw, every history-log record's pumpTimeSec --
// as seconds since this instant. Emitting a Unix timestamp instead (as this
// emulator used to) decodes on the driver side as a date roughly 38 years in
// the future, which drivers then silently clamp to "now"; the bug is invisible
// until something depends on the actual delivery start or end time.
var TandemEpoch = time.Date(2008, time.January, 1, 0, 0, 0, 0, time.UTC)

// tandemEpochUnix is TandemEpoch expressed in Unix seconds (1199145600).
const tandemEpochUnix int64 = 1199145600

// PumpTimeSeconds converts a wall-clock time to the pump's own timestamp
// representation: seconds elapsed since TandemEpoch. Times before the epoch
// clamp to 0 rather than wrapping around the unsigned range.
//
// Note that TandemKit's Dates.swift additionally shifts by the phone's local
// UTC offset when decoding, on the theory that a real pump reports its clock in
// local time with no zone attached. This helper deliberately does NOT apply an
// offset of its own: the pump-clock model lives in PumpState (Clock plus
// pumpClockOffset, applied by PumpTimeFor/PumpNow), and baking a host-local
// offset in here would make that model's behavior depend on the machine
// running the emulator.
func PumpTimeSeconds(t time.Time) uint32 {
	secs := t.UTC().Unix() - tandemEpochUnix
	if secs < 0 {
		return 0
	}
	return uint32(secs)
}

// PumpTimeToWallClock is the inverse of PumpTimeSeconds, for tests and for
// rendering stored pump timestamps back to humans.
func PumpTimeToWallClock(pumpSeconds uint32) time.Time {
	return time.Unix(tandemEpochUnix+int64(pumpSeconds), 0).UTC()
}

// PumpTimeNow returns the pump's current clock as a pump-epoch timestamp.
//
// It reads the pump's own clock (PumpNow: the controllable Clock plus the
// configurable pump-clock skew) rather than time.Now(), so a harness that
// freezes, steps or skews the pump moves every emitted timestamp with it.
func (ps *PumpState) PumpTimeNow() uint32 {
	return PumpTimeSeconds(ps.PumpNow())
}
