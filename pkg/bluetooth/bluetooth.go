package bluetooth

// Service UUID for the Tandem pump
const (
	PumpServiceUUID = "0000fdfb-0000-1000-8000-00805f9b34fb"
)

// Standard service UUIDs.
const (
	GenericAttributeServiceUUID = "1801"
	GenericAccessServiceUUID    = "1800"
	HeartRateServiceUUID        = "180D"
	UserDataServiceUUID         = "181C"
)

// Additional service UUIDs observed from the Tandem Mobi pump.
const (
	UnknownServiceD15AUUID = "0000d15a-0000-1000-8000-aabbccddeeff"
	UnknownServiceAAA0UUID = "0000aaa0-0000-1000-8000-aabbccddeeff"
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
	CentralAddressResolutionCharUUID = "2AA6"
	FirstNameCharUUID               = "2A8A"
	LastNameCharUUID                = "2A90"
	GenderCharUUID                  = "2A8C"
)

// Additional characteristic UUIDs observed from the Tandem Mobi pump.
const (
	UnknownCharAAA1UUID = "0000aaa1-0000-1000-8000-aabbccddeeff"
	UnknownCharAAA2UUID = "0000aaa2-0000-1000-8000-aabbccddeeff"
)

// Descriptor UUIDs.
const (
	CustomDescriptorAAB0UUID        = "0000aab0-0000-1000-8000-aabbccddeeff"
	CharacteristicUserDescriptionUUID = "2901"
	CharacteristicPresentationFormatUUID = "2904"
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
