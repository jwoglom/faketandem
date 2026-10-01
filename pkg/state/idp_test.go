package state

import (
	"reflect"
	"testing"
)

func seg(startTime, basalRate int) IDPSegment {
	return IDPSegment{StartTime: startTime, BasalRate: basalRate, CarbRatio: 10000, TargetBG: 110, ISF: 50}
}

func profileWith(segments ...IDPSegment) []IDPProfile {
	return []IDPProfile{{ID: 1, Name: "P", InsulinDuration: 300, MaxBolus: 25000, Segments: segments}}
}

func TestApplyIDPSegmentFollowsThePumpsRules(t *testing.T) {
	cases := []struct {
		name    string
		start   []IDPSegment
		op      IDPSegmentOperation
		index   int
		segment IDPSegment
		status  int
		want    []IDPSegment
	}{
		{"modify a rate", []IDPSegment{seg(0, 800)}, IDPModifySegment, 0, seg(0, 900), IDPStatusOK, []IDPSegment{seg(0, 900)}},
		{"segment 0 stays at midnight", []IDPSegment{seg(0, 800)}, IDPModifySegment, 0, seg(60, 800), IDPStatusRejected, []IDPSegment{seg(0, 800)}},
		{"modify onto another segment's start", []IDPSegment{seg(0, 800), seg(60, 700), seg(120, 600)}, IDPModifySegment, 1, seg(120, 700), IDPStatusRejected, []IDPSegment{seg(0, 800), seg(60, 700), seg(120, 600)}},
		{"modify a segment that is not there", []IDPSegment{seg(0, 800)}, IDPModifySegment, 1, seg(60, 700), IDPStatusRejected, []IDPSegment{seg(0, 800)}},
		{"create appends", []IDPSegment{seg(0, 800)}, IDPCreateSegment, 0, seg(360, 700), IDPStatusOK, []IDPSegment{seg(0, 800), seg(360, 700)}},
		{"create onto a taken start", []IDPSegment{seg(0, 800)}, IDPCreateSegment, 0, seg(0, 700), IDPStatusRejected, []IDPSegment{seg(0, 800)}},
		{"delete shifts the rest down", []IDPSegment{seg(0, 800), seg(60, 700), seg(120, 600)}, IDPDeleteSegment, 1, seg(60, 700), IDPStatusOK, []IDPSegment{seg(0, 800), seg(120, 600)}},
		{"segment 0 cannot be deleted", []IDPSegment{seg(0, 800), seg(60, 700)}, IDPDeleteSegment, 0, seg(0, 800), IDPStatusRejected, []IDPSegment{seg(0, 800), seg(60, 700)}},
		{"unknown operation", []IDPSegment{seg(0, 800)}, IDPSegmentOperation(7), 0, seg(0, 900), IDPStatusRejected, []IDPSegment{seg(0, 800)}},
	}
	for _, c := range cases {
		ps := NewPumpState()
		ps.SetIDPProfiles(profileWith(c.start...))

		if got := ps.ApplyIDPSegment(1, c.op, c.index, c.segment); got != c.status {
			t.Errorf("%s: status %d, want %d", c.name, got, c.status)
		}
		profile, _ := ps.IDPProfile(1)
		if !reflect.DeepEqual(profile.Segments, c.want) {
			t.Errorf("%s: segments %v, want %v", c.name, profile.Segments, c.want)
		}
	}
}

func TestAProfileHoldsAtMostSixteenSegments(t *testing.T) {
	ps := NewPumpState()
	ps.SetIDPProfiles(profileWith(seg(0, 800)))

	for i := 1; i < MaxIDPSegments; i++ {
		if status := ps.ApplyIDPSegment(1, IDPCreateSegment, 0, seg(i*60, 800)); status != IDPStatusOK {
			t.Fatalf("segment %d refused", i+1)
		}
	}
	if status := ps.ApplyIDPSegment(1, IDPCreateSegment, 0, seg(MaxIDPSegments*60, 800)); status != IDPStatusRejected {
		t.Errorf("a 17th segment was accepted")
	}
}

func TestWritesToAProfileThePumpDoesNotHoldAreRefused(t *testing.T) {
	ps := NewPumpState()
	ps.SetIDPProfiles(nil)

	if status := ps.ApplyIDPSegment(1, IDPModifySegment, 0, seg(0, 800)); status != IDPStatusRejected {
		t.Errorf("status %d, want %d", status, IDPStatusRejected)
	}
}

func TestCreateIDPOnAnEmptyPumpMakesTheActiveProfile(t *testing.T) {
	ps := NewPumpState()
	ps.SetIDPProfiles(nil)

	id, ok := ps.CreateIDP("TandemKitProfile", seg(30, 650), 240, false, -1)

	if !ok {
		t.Fatal("CreateIDP refused on an empty pump")
	}
	profiles := ps.IDPProfiles()
	if len(profiles) != 1 || profiles[0].ID != id {
		t.Fatalf("profiles %v, want only the new one in slot 0", profiles)
	}
	if want := []IDPSegment{seg(0, 650)}; !reflect.DeepEqual(profiles[0].Segments, want) {
		t.Errorf("segments %v, want %v: one segment, from midnight", profiles[0].Segments, want)
	}
}

func TestCreateIDPDuplicatesASourceProfile(t *testing.T) {
	ps := NewPumpState()
	ps.SetIDPProfiles(profileWith(seg(0, 800), seg(360, 700)))

	id, ok := ps.CreateIDP("Copy", IDPSegment{}, 0, false, 1)

	copied, found := ps.IDPProfile(id)
	if !ok || !found || id == 1 {
		t.Fatalf("CreateIDP returned %d, %v", id, ok)
	}
	if want := []IDPSegment{seg(0, 800), seg(360, 700)}; !reflect.DeepEqual(copied.Segments, want) {
		t.Errorf("segments %v, want %v", copied.Segments, want)
	}
}

func TestCreateIDPRefusesASeventhProfile(t *testing.T) {
	ps := NewPumpState()
	ps.SetIDPProfiles(nil)
	for i := 0; i < MaxIDPProfiles; i++ {
		if _, ok := ps.CreateIDP("P", seg(0, 800), 300, false, -1); !ok {
			t.Fatalf("profile %d refused", i+1)
		}
	}

	if _, ok := ps.CreateIDP("P", seg(0, 800), 300, false, -1); ok {
		t.Error("a seventh profile was accepted")
	}
}

func TestStagedProfilesAreCopies(t *testing.T) {
	ps := NewPumpState()
	staged := profileWith(seg(0, 800))
	ps.SetIDPProfiles(staged)

	staged[0].Segments[0].BasalRate = 1
	read := ps.IDPProfiles()
	read[0].Segments[0].BasalRate = 2

	if profile, _ := ps.IDPProfile(1); profile.Segments[0].BasalRate != 800 {
		t.Errorf("basal rate %d, want 800", profile.Segments[0].BasalRate)
	}
}
