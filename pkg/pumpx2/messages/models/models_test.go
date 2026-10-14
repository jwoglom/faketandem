package models

import (
	"reflect"
	"testing"
)

func TestMinsTime(t *testing.T) {
	m := NewMinsTime(605)
	if m.Hour != 10 || m.Min != 5 || m.Encode() != 605 || m.String() != "10:05" {
		t.Errorf("NewMinsTime(605) = %+v %s", m, m)
	}
}

func TestMultiDayFromBitmaskKeepsDaysWhoseBitDividesIt(t *testing.T) {
	if got := MultiDayFromBitmask(6); !reflect.DeepEqual(got, []MultiDay{Monday, Tuesday}) {
		t.Errorf("MultiDayFromBitmask(6) = %v", got)
	}
	if got := MultiDayToBitmask(Monday, Sunday); got != 65 {
		t.Errorf("MultiDayToBitmask = %d", got)
	}
}

func TestBloodGlucoseReadingFromIDFallsBackToUnknown(t *testing.T) {
	if BloodGlucoseReadingTypeFromID(7) != BloodGlucoseReadingTypeUnknown || BloodGlucoseReadingSourceFromID(-1) != BloodGlucoseReadingSourceUnknown {
		t.Error("an unknown ID is not Unknown")
	}
	if BloodGlucoseReadingTypeFromID(1).String() != "AUTO" || BloodGlucoseReadingSourceFromID(1).String() != "REMOTE" {
		t.Error("known IDs do not print their names")
	}
}
