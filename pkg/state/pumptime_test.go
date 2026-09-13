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
	want := PumpTimeSeconds(time.Now())
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
