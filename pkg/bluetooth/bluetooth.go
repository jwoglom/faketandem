package bluetooth

// Service UUID for the Tandem pump
const (
	PumpServiceUUID = "0000fdfb-0000-1000-8000-00805f9b34fb"
)

// Standard service UUIDs.
const (
	GenericAttributeServiceUUID = "1801"
	GenericAccessServiceUUID    = "1800"
	DeviceInformationServiceUUID = "180A"
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

// Standard characteristic UUIDs.
const (
	ServiceChangedCharUUID          = "2A05"
	DeviceNameCharUUID              = "2A00"
	AppearanceCharUUID              = "2A01"
	PeripheralPreferredConnectionParametersCharUUID = "2A04"
	CentralAddressResolutionCharUUID = "2AA6"
	ManufacturerNameStringCharUUID  = "2A29"
	ModelNumberStringCharUUID       = "2A24"
	SerialNumberStringCharUUID      = "2A25"
	SoftwareRevisionStringCharUUID  = "2A28"
)

// Additional characteristic UUIDs observed from the Tandem Mobi pump.
const (
	UnknownCharFFE7UUID = "7B83FFE7-9F77-4E5C-8064-AAE2C24838B9"
	UnknownCharFFE8UUID = "7B83FFE8-9F77-4E5C-8064-AAE2C24838B9"
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

// ToBtChar returns the pumpX2 cliparser Characteristic enum constant name for
// c, used to set the PUMPX2_CHARACTERISTIC environment variable so cliparser's
// "parse" command can disambiguate an opcode that maps to more than one
// characteristic (see CharacteristicGuesser.filterKnownPossibilities in
// pumpX2) instead of guessing via its own fixed CONTROL/AUTHORIZATION/
// CURRENT_STATUS precedence, which can pick the wrong (and differently
// cargo-sized) message class. Returns "" for characteristics pumpX2 has no
// enum constant for (QualifyingEvents is notify-only and never parsed), so
// the caller knows not to set the environment variable at all.
func (c CharacteristicType) ToBtChar() string {
	switch c {
	case CharCurrentStatus:
		return "CURRENT_STATUS"
	case CharHistoryLog:
		return "HISTORY_LOG"
	case CharAuthorization:
		return "AUTHORIZATION"
	case CharControl:
		return "CONTROL"
	case CharControlStream:
		return "CONTROL_STREAM"
	default:
		return ""
	}
}

// WriteHandler is called when data is written to a characteristic
type WriteHandler func(charType CharacteristicType, data []byte)

// ReadHandler is called when data is read from a characteristic
type ReadHandler func(charType CharacteristicType) []byte

// ConnectionHandler is called when a central device connects or disconnects
type ConnectionHandler func(connected bool)
