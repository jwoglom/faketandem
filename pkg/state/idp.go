package state

import log "github.com/sirupsen/logrus"

// IDP (insulin delivery profile) limits a Tandem pump enforces.
const (
	MaxIDPProfiles = 6
	MaxIDPSegments = 16
)

// IDPStatusOK and IDPStatusRejected are the statuses the IDP write responses carry. A real pump
// gives no reason beyond a nonzero status.
const (
	IDPStatusOK       = 0
	IDPStatusRejected = 1
)

// IDPSegmentOperation is SetIDPSegmentRequest's operation byte.
type IDPSegmentOperation int

const (
	IDPModifySegment IDPSegmentOperation = 0
	IDPCreateSegment IDPSegmentOperation = 1
	IDPDeleteSegment IDPSegmentOperation = 2
)

// IDPSegment is one time segment of a profile, in the units the wire carries.
type IDPSegment struct {
	StartTime int    `json:"start_time"` // minutes after midnight, pump local time
	BasalRate int    `json:"basal_rate"` // milliunits/hour
	CarbRatio uint32 `json:"carb_ratio"` // pumpX2's encoding: 1000 x grams per unit
	TargetBG  int    `json:"target_bg"`  // mg/dL
	ISF       int    `json:"isf"`        // mg/dL per unit
}

// IDPProfile is one insulin delivery profile. The profile in slot 0 is the active one, as
// pumpX2's ProfileStatusResponse.getActiveIdpSlotId reads it.
type IDPProfile struct {
	ID              int          `json:"id"`
	Name            string       `json:"name"`
	InsulinDuration int          `json:"insulin_duration"` // minutes
	MaxBolus        int          `json:"max_bolus"`        // milliunits
	CarbEntry       bool         `json:"carb_entry"`
	Segments        []IDPSegment `json:"segments"`
}

// DefaultIDPProfiles is the profile a fresh emulator holds: one segment at the default basal rate.
func DefaultIDPProfiles() []IDPProfile {
	return []IDPProfile{{
		ID:              1,
		Name:            "Profile 1",
		InsulinDuration: 300,
		MaxBolus:        25000,
		CarbEntry:       true,
		Segments: []IDPSegment{
			{StartTime: 0, BasalRate: 850, CarbRatio: 10000, TargetBG: 110, ISF: 50},
		},
	}}
}

func copyIDPProfiles(profiles []IDPProfile) []IDPProfile {
	out := make([]IDPProfile, len(profiles))
	for i, p := range profiles {
		out[i] = p
		out[i].Segments = append([]IDPSegment(nil), p.Segments...)
	}
	return out
}

// IDPProfiles returns a copy of the profiles in slot order.
func (ps *PumpState) IDPProfiles() []IDPProfile {
	ps.mutex.RLock()
	defer ps.mutex.RUnlock()
	return copyIDPProfiles(ps.idpProfiles)
}

// IDPProfilesUnlocked is for callers already holding the state lock.
func (ps *PumpState) IDPProfilesUnlocked() []IDPProfile {
	return copyIDPProfiles(ps.idpProfiles)
}

// SetIDPProfiles stages the profiles directly, with none of the checks a request goes through, so
// a harness can start from a pump holding no profile or a given one.
func (ps *PumpState) SetIDPProfiles(profiles []IDPProfile) {
	ps.mutex.Lock()
	defer ps.mutex.Unlock()
	ps.idpProfiles = copyIDPProfiles(profiles)
}

// IDPProfile returns the profile with idpID.
func (ps *PumpState) IDPProfile(idpID int) (IDPProfile, bool) {
	ps.mutex.RLock()
	defer ps.mutex.RUnlock()
	if i := ps.idpProfileIndexUnlocked(idpID); i >= 0 {
		return copyIDPProfiles(ps.idpProfiles[i : i+1])[0], true
	}
	return IDPProfile{}, false
}

func (ps *PumpState) idpProfileIndexUnlocked(idpID int) int {
	for i, p := range ps.idpProfiles {
		if p.ID == idpID {
			return i
		}
	}
	return -1
}

// ActiveIDPSegmentIndex is the index of the active profile's segment in force at the pump's local
// time of day, or 0 with no profile.
func (ps *PumpState) ActiveIDPSegmentIndex() int {
	now := ps.PumpNow().In(ps.GetPumpTimeZone())
	minutes := now.Hour()*60 + now.Minute()

	ps.mutex.RLock()
	defer ps.mutex.RUnlock()
	if len(ps.idpProfiles) == 0 {
		return 0
	}
	active := 0
	for i, s := range ps.idpProfiles[0].Segments {
		if s.StartTime <= minutes && s.StartTime >= ps.idpProfiles[0].Segments[active].StartTime {
			active = i
		}
	}
	return active
}

// CreateIDP adds a profile with one segment at midnight, or a copy of sourceID's segments when
// sourceID names an existing profile, and returns its id. The first profile a pump holds becomes
// the active one, since slot 0 is.
func (ps *PumpState) CreateIDP(name string, first IDPSegment, insulinDuration int, carbEntry bool, sourceID int) (int, bool) {
	ps.mutex.Lock()
	defer ps.mutex.Unlock()

	if len(ps.idpProfiles) >= MaxIDPProfiles {
		log.Warnf("Refusing CreateIDP: the pump already holds %d profiles", len(ps.idpProfiles))
		return 0, false
	}

	id := 1
	for _, p := range ps.idpProfiles {
		if p.ID >= id {
			id = p.ID + 1
		}
	}

	profile := IDPProfile{ID: id, Name: name, InsulinDuration: insulinDuration, MaxBolus: 25000, CarbEntry: carbEntry}
	if i := ps.idpProfileIndexUnlocked(sourceID); i >= 0 {
		source := copyIDPProfiles(ps.idpProfiles[i : i+1])[0]
		profile.InsulinDuration, profile.MaxBolus, profile.CarbEntry = source.InsulinDuration, source.MaxBolus, source.CarbEntry
		profile.Segments = source.Segments
	} else {
		first.StartTime = 0
		profile.Segments = []IDPSegment{first}
	}

	ps.idpProfiles = append(ps.idpProfiles, profile)
	log.Infof("Created IDP %d %q with %d segment(s)", id, name, len(profile.Segments))
	return id, true
}

// ApplyIDPSegment applies one SetIDPSegmentRequest and returns the status the pump answers with.
//
// A create appends, as TandemKit's segment ordering assumes; whether firmware re-sorts by start
// time is not confirmed. No two segments may share a start time, segment 0 stays at midnight and
// cannot be deleted, and a profile holds at most MaxIDPSegments.
func (ps *PumpState) ApplyIDPSegment(idpID int, op IDPSegmentOperation, segmentIndex int, segment IDPSegment) int {
	ps.mutex.Lock()
	defer ps.mutex.Unlock()

	pi := ps.idpProfileIndexUnlocked(idpID)
	if pi < 0 {
		log.Warnf("Refusing SetIDPSegment: no profile %d", idpID)
		return IDPStatusRejected
	}
	segments, ok := applyIDPSegmentOperation(ps.idpProfiles[pi].Segments, op, segmentIndex, segment)
	if !ok {
		log.Warnf("Refusing SetIDPSegment operation %d on segment %d of profile %d (start %d, %d segments)",
			op, segmentIndex, idpID, segment.StartTime, len(ps.idpProfiles[pi].Segments))
		return IDPStatusRejected
	}
	ps.idpProfiles[pi].Segments = segments
	return IDPStatusOK
}

func applyIDPSegmentOperation(segments []IDPSegment, op IDPSegmentOperation, index int, segment IDPSegment) ([]IDPSegment, bool) {
	collides := func(startTime, except int) bool {
		for i, s := range segments {
			if i != except && s.StartTime == startTime {
				return true
			}
		}
		return false
	}

	switch op {
	case IDPModifySegment:
		if index >= len(segments) || collides(segment.StartTime, index) || (index == 0 && segment.StartTime != 0) {
			return nil, false
		}
		updated := append([]IDPSegment(nil), segments...)
		updated[index] = segment
		return updated, true
	case IDPCreateSegment:
		if len(segments) >= MaxIDPSegments || collides(segment.StartTime, -1) {
			return nil, false
		}
		return append(append([]IDPSegment(nil), segments...), segment), true
	case IDPDeleteSegment:
		if index == 0 || index >= len(segments) {
			return nil, false
		}
		updated := append([]IDPSegment(nil), segments[:index]...)
		return append(updated, segments[index+1:]...), true
	default:
		return nil, false
	}
}
