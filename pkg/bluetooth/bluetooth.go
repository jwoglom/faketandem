package bluetooth

// Service UUID for the Tandem pump
const (
	PumpServiceUUID = "0000fdfb-0000-1000-8000-00805f9b34fb"
)

// Characteristic UUIDs
const (
	CurrentStatusCharUUID    = "7B83FFF6-9F77-4E5C-8064-AAE2C24838B9"
	QualifyingEventsCharUUID = "7B83FFF7-9F77-4E5C-8064-AAE2C24838B9"
	HistoryLogCharUUID       = "7B83FFF8-9F77-4E5C-8064-AAE2C24838B9"
	AuthorizationCharUUID    = "7B83FFF9-9F77-4E5C-8064-AAE2C24838B9"
	ControlCharUUID          = "7B83FFFC-9F77-4E5C-8064-AAE2C24838B9"
	ControlStreamCharUUID    = "7B83FFFD-9F77-4E5C-8064-AAE2C24838B9"
)

// CharacteristicType identifies which characteristic received data
type CharacteristicType int

// Characteristic type constants
const (
	CharCurrentStatus CharacteristicType = iota
	CharQualifyingEvents
	CharHistoryLog
	CharAuthorization
	CharControl
	CharControlStream
)

func (c CharacteristicType) String() string {
	switch c {
	case CharCurrentStatus:
		return "CurrentStatus"
	case CharQualifyingEvents:
		return "QualifyingEvents"
	case CharHistoryLog:
		return "HistoryLog"
	case CharAuthorization:
		return "Authorization"
	case CharControl:
		return "Control"
	case CharControlStream:
		return "ControlStream"
	default:
		return "Unknown"
	}
}

// ToBtChar returns the btChar name used by pumpX2 cliparser
func (c CharacteristicType) ToBtChar() string {
	switch c {
	case CharCurrentStatus:
		return "currentStatus"
	case CharQualifyingEvents:
		return "qualifyingEvents"
	case CharHistoryLog:
		return "historyLog"
	case CharAuthorization:
		return "authentication"
	case CharControl:
		return "control"
	case CharControlStream:
		return "controlStream"
	default:
		return "currentStatus"
	}
}

// WriteHandler is called when data is written to a characteristic
type WriteHandler func(charType CharacteristicType, data []byte)

// ReadHandler is called when data is read from a characteristic
type ReadHandler func(charType CharacteristicType) []byte
