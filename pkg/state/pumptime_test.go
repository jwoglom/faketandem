package state

import (
	"testing"
	"time"
)

func TestPumpTimeSeconds_TandemEpoch(t *testing.T) {
	tests := []struct {
		name string
		in   time.Time
		want uint32
	}{
		{"the epoch itself", TandemEpoch, 0},
		{"one second after the epoch", TandemEpoch.Add(time.Second), 1},
		{"one day after the epoch", TandemEpoch.Add(24 * time.Hour), 86400},
		{
			// 2024-01-01T00:00:00Z is 16 years after the epoch, four of them leap years.
			"a realistic pump timestamp",
			time.Date(2024, time.January, 1, 0, 0, 0, 0, time.UTC),
			(16*365 + 4) * 86400,
		},
		{"before the epoch clamps to zero rather than wrapping", TandemEpoch.Add(-time.Hour), 0},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := PumpTimeSeconds(tc.in); got != tc.want {
				t.Errorf("PumpTimeSeconds(%v) = %d, want %d", tc.in, got, tc.want)
			}
		})
	}
}

func TestPumpTimeSeconds_IsNotUnixTime(t *testing.T) {
	// The whole point: a pump timestamp is ~1.2 billion seconds smaller than the
	// Unix timestamp for the same instant. Emitting the Unix value decodes on the
	// driver side as a date decades in the future.
	now := time.Now()
	if got, unix := int64(PumpTimeSeconds(now)), now.Unix(); unix-got != 1199145600 {
		t.Errorf("pump time %d vs unix %d: difference %d, want 1199145600", got, unix, unix-got)
	}
}

func TestPumpTimeRoundTrip(t *testing.T) {
	original := time.Date(2026, time.September, 13, 14, 30, 45, 0, time.UTC)
	if got := PumpTimeToWallClock(PumpTimeSeconds(original)); !got.Equal(original) {
		t.Errorf("round trip gave %v, want %v", got, original)
	}
}

func TestPumpTimeNow_TracksPumpStateClock(t *testing.T) {
	ps := NewPumpState()
	ps.UpdateTimeSinceReset()

	got := ps.PumpTimeNow()
	want := PumpTimeSecondsIn(time.Now(), ps.GetPumpTimeZone())
	if diff := int64(want) - int64(got); diff > 2 || diff < -2 {
		t.Errorf("PumpTimeNow() = %d, want within 2s of %d", got, want)
	}
}

func TestNewPumpState_DefaultsToMobiAPIVersion(t *testing.T) {
	ps := NewPumpState()
	if ps.GetAPIVersionMajor() != 3 || ps.GetAPIVersionMinor() != 5 {
		t.Errorf("default API version = %d.%d, want 3.5 (Tandem Mobi)",
			ps.GetAPIVersionMajor(), ps.GetAPIVersionMinor())
	}
	if ps.IsClosedLoopEnabled() {
		t.Error("closed loop should default to off, so temp basals and manual boluses stay enactable")
	}
}

func TestAPIVersionOverride(t *testing.T) {
	t.Setenv(apiVersionEnvVar, "2.5")

	ps := NewPumpState()
	if ps.GetAPIVersionMajor() != 2 || ps.GetAPIVersionMinor() != 5 {
		t.Errorf("overridden API version = %d.%d, want 2.5",
			ps.GetAPIVersionMajor(), ps.GetAPIVersionMinor())
	}
}

func TestAPIVersionOverride_InvalidValueFallsBackToDefault(t *testing.T) {
	t.Setenv(apiVersionEnvVar, "not-a-version")

	ps := NewPumpState()
	if ps.GetAPIVersionMajor() != defaultAPIVersionMajor || ps.GetAPIVersionMinor() != defaultAPIVersionMinor {
		t.Errorf("API version = %d.%d, want the %d.%d default after an invalid override",
			ps.GetAPIVersionMajor(), ps.GetAPIVersionMinor(), defaultAPIVersionMajor, defaultAPIVersionMinor)
	}
}

func TestGetNextBolusID_IsMonotonic(t *testing.T) {
	ps := NewPumpState()

	first := ps.GetNextBolusID()
	second := ps.GetNextBolusID()
	if second <= first {
		t.Errorf("bolus IDs %d then %d are not increasing", first, second)
	}

	// A bolus started with an externally supplied ID must push the allocator
	// past it, so a later permission grant cannot reissue an ID in use.
	ps.StartBolusWithSource(1.0, 5000, BolusSourceBluetoothRemote, 0)
	if next := ps.GetNextBolusID(); next <= 5000 {
		t.Errorf("next bolus ID %d should be greater than the in-use ID 5000", next)
	}
}

func TestRecordLastBolus(t *testing.T) {
	ps := NewPumpState()

	if _, ok := ps.GetLastBolus(); ok {
		t.Error("a fresh pump should report no last bolus")
	}

	ps.RecordLastBolus(LastBolusRecord{
		BolusID:        42,
		RequestedUnits: 2.5,
		DeliveredUnits: 2.4,
		EndReasonID:    BolusEndReasonStopped,
	})

	record, ok := ps.GetLastBolus()
	if !ok {
		t.Fatal("expected a last bolus record")
	}
	if record.BolusID != 42 || record.DeliveredUnits != 2.4 {
		t.Errorf("record = %+v, want bolusID 42 delivered 2.4", record)
	}
	if record.EndTime.IsZero() {
		t.Error("RecordLastBolus should stamp an end time when the caller leaves it zero")
	}
}

// swiftDecodeJan12008 is TandemKit's decoder, ported verbatim:
//
//	Dates.fromJan12008ToUnixEpochSeconds(s) =
//	    s + JANUARY_1_2008_UNIX_EPOCH - TimeZone.current.secondsFromGMT()
//
// zoneOffsetSeconds stands in for `TimeZone.current.secondsFromGMT()` on the
// phone -- which is what /api/clock reports as pump_timezone_offset_seconds.
func swiftDecodeJan12008(pumpSeconds uint32, zoneOffsetSeconds int) time.Time {
	return time.Unix(int64(pumpSeconds)+tandemEpochUnix-int64(zoneOffsetSeconds), 0).UTC()
}

// TestPumpTimeFor_RoundTripsThroughTandemKitsDecoder is the pin on the whole
// pump-local-time convention: a wall-clock instant encoded by this emulator
// must decode, through the formula TandemKit actually uses, back to that same
// instant. It used to land one UTC offset away (four hours in EDT) because the
// emulator stamped UTC while the decoder assumed local time.
func TestPumpTimeFor_RoundTripsThroughTandemKitsDecoder(t *testing.T) {
	zones := []string{"UTC", "America/New_York", "Europe/Berlin", "Asia/Kolkata", "Pacific/Auckland"}
	instants := []time.Time{
		time.Date(2024, time.January, 15, 9, 30, 0, 0, time.UTC),  // northern winter
		time.Date(2024, time.July, 4, 21, 5, 45, 0, time.UTC),     // northern summer (DST)
		time.Date(2024, time.March, 10, 11, 0, 0, 0, time.UTC),    // US spring-forward day
		time.Date(2024, time.November, 3, 5, 30, 0, 0, time.UTC),  // US fall-back day
		time.Date(2026, time.September, 13, 14, 30, 45, 0, time.UTC),
	}

	for _, zone := range zones {
		loc, err := time.LoadLocation(zone)
		if err != nil {
			t.Fatalf("LoadLocation(%q): %v", zone, err)
		}
		for _, instant := range instants {
			t.Run(zone+"/"+instant.Format(time.RFC3339), func(t *testing.T) {
				ps := NewPumpState()
				ps.SetClock(NewFrozenClock(instant))
				ps.SetPumpTimeZone(loc)

				wire := ps.PumpTimeFor(instant)
				offset := ps.PumpTimeZoneOffsetSeconds(instant)

				if got := swiftDecodeJan12008(wire, offset); !got.Equal(instant) {
					t.Errorf("%d decoded with offset %d = %v, want the original %v", wire, offset, got, instant)
				}
				// And the emulator's own inverse agrees with the driver's.
				if got := ps.WallClockForPumpTime(wire); !got.Equal(instant) {
					t.Errorf("WallClockForPumpTime(%d) = %v, want %v", wire, got, instant)
				}
			})
		}
	}
}

// TestPumpTimeFor_SkewIsAddedOnTopOfTheZone pins the two offsets as separate
// things: the zone is the pump's local-time convention and is not a lie, the
// skew is a pump whose clock is set wrong.
func TestPumpTimeFor_SkewIsAddedOnTopOfTheZone(t *testing.T) {
	loc, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Fatalf("LoadLocation: %v", err)
	}
	instant := time.Date(2024, time.July, 4, 21, 5, 45, 0, time.UTC)

	ps := NewPumpState()
	ps.SetClock(NewFrozenClock(instant))
	ps.SetPumpTimeZone(loc)

	// -0400 in July: local time runs four hours behind UTC, so the wire value
	// is 4 h smaller than the UTC-based encoding.
	if got, want := ps.PumpTimeZoneOffsetSeconds(instant), -4*3600; got != want {
		t.Fatalf("zone offset = %d, want %d", got, want)
	}
	if got, want := ps.PumpTimeFor(instant), PumpTimeSeconds(instant)-4*3600; got != want {
		t.Errorf("PumpTimeFor = %d, want %d", got, want)
	}

	ps.SetPumpClockOffset(8 * time.Second)
	if got, want := ps.PumpTimeFor(instant), PumpTimeSeconds(instant)-4*3600+8; got != want {
		t.Errorf("PumpTimeFor with an 8 s skew = %d, want %d", got, want)
	}
	// A driver in the same zone decodes the pump's lie, not the true instant.
	decoded := swiftDecodeJan12008(ps.PumpTimeFor(instant), ps.PumpTimeZoneOffsetSeconds(instant))
	if want := instant.Add(8 * time.Second); !decoded.Equal(want) {
		t.Errorf("decoded = %v, want the skewed %v", decoded, want)
	}
	// The emulator's inverse removes the skew again, recovering the instant.
	if got := ps.WallClockForPumpTime(ps.PumpTimeFor(instant)); !got.Equal(instant) {
		t.Errorf("WallClockForPumpTime = %v, want the true %v", got, instant)
	}
	// ... and the pump's own sense of time is untouched by either.
	if got := ps.Now(); !got.Equal(instant) {
		t.Errorf("Now() = %v, want the unskewed %v", got, instant)
	}
}

func TestPumpTimeZone_DefaultsToTheHostZone(t *testing.T) {
	ps := NewPumpState()
	if got := ps.GetPumpTimeZone(); got != time.Local {
		t.Errorf("default pump time zone = %v, want the host's local zone", got)
	}
	ps.SetPumpTimeZone(nil)
	if got := ps.GetPumpTimeZone(); got != time.UTC {
		t.Errorf("a nil zone = %v, want UTC", got)
	}
}
