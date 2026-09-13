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

// A Tandem pump keeps its clock in *local time* and attaches no zone to it:
// the pump-epoch seconds on the wire count from 2008-01-01 00:00:00 as read on
// the pump's own wall clock, not in UTC. Every consumer decodes on that
// assumption. TandemKit's Dates.swift is explicit about it:
//
//	fromJan12008ToUnixEpochSeconds(s) = s + 1199145600 - TimeZone.current.secondsFromGMT()
//
// so the encoding a driver will decode back to the original instant is the
// inverse of that, with the pump's zone offset standing in for the phone's:
//
//	pumpSeconds(t) = t.Unix() - 1199145600 + offsetOf(pumpZone, at t)
//
// PumpTimeSecondsIn is that formula, and PumpTimeToWallClockIn its inverse.
// Emitting UTC-based seconds instead (as this emulator used to) puts every
// wire timestamp one UTC offset away from the pump's own record -- four hours
// in EDT -- which is invisible until a test compares a dose's second against
// what the pump says.

// PumpTimeSecondsIn converts a wall-clock instant to the pump's own timestamp
// representation for a pump whose clock is kept in loc: seconds elapsed since
// TandemEpoch as read on that zone's wall clock. A nil loc means UTC. Times
// before the epoch clamp to 0 rather than wrapping around the unsigned range.
func PumpTimeSecondsIn(t time.Time, loc *time.Location) uint32 {
	if loc == nil {
		loc = time.UTC
	}
	_, offset := t.In(loc).Zone()
	secs := t.Unix() - tandemEpochUnix + int64(offset)
	if secs < 0 {
		return 0
	}
	return uint32(secs)
}

// PumpTimeToWallClockIn is the inverse of PumpTimeSecondsIn: the true instant a
// pump keeping its clock in loc meant by pumpSeconds.
//
// The offset has to be looked up at the answer rather than at the argument, so
// the lookup is iterated to a fixed point; two rounds settle every real zone,
// and the DST transition hours where no fixed point exists resolve to the
// offset in force on the near side of the jump.
func PumpTimeToWallClockIn(pumpSeconds uint32, loc *time.Location) time.Time {
	if loc == nil {
		loc = time.UTC
	}
	// naive is the reading interpreted as if the pump's clock were UTC.
	naive := tandemEpochUnix + int64(pumpSeconds)
	guess := time.Unix(naive, 0).UTC()
	for i := 0; i < 2; i++ {
		_, offset := guess.In(loc).Zone()
		next := time.Unix(naive-int64(offset), 0).UTC()
		if next.Equal(guess) {
			break
		}
		guess = next
	}
	return guess
}

// PumpTimeSeconds is PumpTimeSecondsIn for a pump whose clock is UTC.
//
// It is the zoneless form, kept for callers that genuinely mean "seconds since
// the epoch" with no pump attached (the encoder's own tests, fixture data).
// Anything emitting a timestamp for *this* pump must go through
// PumpState.PumpTimeFor, which applies both the pump's zone and its skew.
func PumpTimeSeconds(t time.Time) uint32 {
	return PumpTimeSecondsIn(t, time.UTC)
}

// PumpTimeToWallClock is the inverse of PumpTimeSeconds, for a pump whose
// clock is UTC.
func PumpTimeToWallClock(pumpSeconds uint32) time.Time {
	return PumpTimeToWallClockIn(pumpSeconds, time.UTC)
}

// PumpTimeNow returns the pump's current clock as a pump-epoch timestamp.
//
// It reads the pump's own clock (PumpNow: the controllable Clock plus the
// configurable pump-clock skew) rather than time.Now(), so a harness that
// freezes, steps or skews the pump moves every emitted timestamp with it, and
// encodes it in the pump's configured time zone.
func (ps *PumpState) PumpTimeNow() uint32 {
	return ps.PumpTimeFor(ps.Now())
}
