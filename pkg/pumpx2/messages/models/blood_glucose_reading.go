package models

// BloodGlucoseReadingType is whether a BG reading was entered by hand or
// filled in from the CGM: byte 3 of a RemoteBgEntryRequest.
type BloodGlucoseReadingType int32

// The reading types, by their wire IDs.
const (
	BloodGlucoseReadingTypeManual  BloodGlucoseReadingType = 0
	BloodGlucoseReadingTypeAuto    BloodGlucoseReadingType = 1
	BloodGlucoseReadingTypeUnknown BloodGlucoseReadingType = 2
)

// BloodGlucoseReadingTypeFromID is the type with wire ID id, or Unknown.
func BloodGlucoseReadingTypeFromID(id int32) BloodGlucoseReadingType {
	switch t := BloodGlucoseReadingType(id); t {
	case BloodGlucoseReadingTypeManual, BloodGlucoseReadingTypeAuto, BloodGlucoseReadingTypeUnknown:
		return t
	}
	return BloodGlucoseReadingTypeUnknown
}

// ID is the type's wire ID.
func (t BloodGlucoseReadingType) ID() int32 { return int32(t) }

func (t BloodGlucoseReadingType) String() string {
	switch t {
	case BloodGlucoseReadingTypeManual:
		return "MANUAL"
	case BloodGlucoseReadingTypeAuto:
		return "AUTO"
	}
	return "UNKNOWN"
}

// BloodGlucoseReadingSource is whether a BG reading was entered on the pump
// or sent to it: byte 4 of a RemoteBgEntryRequest.
type BloodGlucoseReadingSource int32

// The reading sources, by their wire IDs.
const (
	BloodGlucoseReadingSourcePump    BloodGlucoseReadingSource = 0
	BloodGlucoseReadingSourceRemote  BloodGlucoseReadingSource = 1
	BloodGlucoseReadingSourceUnknown BloodGlucoseReadingSource = 2
)

// BloodGlucoseReadingSourceFromID is the source with wire ID id, or Unknown.
func BloodGlucoseReadingSourceFromID(id int32) BloodGlucoseReadingSource {
	switch s := BloodGlucoseReadingSource(id); s {
	case BloodGlucoseReadingSourcePump, BloodGlucoseReadingSourceRemote, BloodGlucoseReadingSourceUnknown:
		return s
	}
	return BloodGlucoseReadingSourceUnknown
}

// ID is the source's wire ID.
func (s BloodGlucoseReadingSource) ID() int32 { return int32(s) }

func (s BloodGlucoseReadingSource) String() string {
	switch s {
	case BloodGlucoseReadingSourcePump:
		return "PUMP"
	case BloodGlucoseReadingSourceRemote:
		return "REMOTE"
	}
	return "UNKNOWN"
}
