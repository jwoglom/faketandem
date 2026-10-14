package currentstatus

import (
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/helpers"
)

// commonSoftwareInfoMobiSize is the Mobi layout's size; Parse reads any other
// as the t:slim X2's.
const commonSoftwareInfoMobiSize = 51

// CommonSoftwareInfoResponse is the software version and part numbers of the
// application and bootloader images of one microcontroller, in one of two
// layouts:
//
//   - 51 bytes, the Tandem Mobi's: the requested MCU type, then the app
//     version (17 bytes), part number and part revision number, then the
//     bootloader's. Every Mobi capture so far is this layout, both versions
//     the 16-hex-digit software ID history log 449 records.
//   - 60 bytes, the t:slim X2's as its app reads it, unconfirmed by any
//     capture: no MCU type, the app version (18 bytes), part, dash and
//     revision numbers, then the bootloader's version (17 bytes) and numbers.
//     pumpX2 builds it as 59 bytes.
//
// The Mobi layout has no dash numbers: AppSoftwarePartDashNumber reads 0 and
// BootloaderPartDashNumber nil.
type CommonSoftwareInfoResponse struct {
	messages.Base
	// MCUTypeID is nil but in the Mobi layout.
	MCUTypeID                     *int32 `pumpx2:"mcuTypeId"`
	AppSoftwareVersion            string `pumpx2:"appSoftwareVersion"`
	AppSoftwarePartNumber         int64  `pumpx2:"appSoftwarePartNumber"`
	AppSoftwarePartDashNumber     int64  `pumpx2:"appSoftwarePartDashNumber"`
	AppSoftwarePartRevisionNumber int64  `pumpx2:"appSoftwarePartRevisionNumber"`
	BootloaderVersion             string `pumpx2:"bootloaderVersion"`
	BootloaderPartNumber          int64  `pumpx2:"bootloaderPartNumber"`
	BootloaderPartDashNumber      *int64 `pumpx2:"bootloaderPartDashNumber"`
	BootloaderPartRevisionNumber  *int64 `pumpx2:"bootloaderPartRevisionNumber"`
}

var commonSoftwareInfoResponseProps = messages.Props{OpCode: -113, Size: 60, VariableSize: true, Type: messages.Response, MinAPI: "API_FUTURE", Request: "CommonSoftwareInfoRequest"}

func init() {
	messages.Register("CommonSoftwareInfoResponse", &CommonSoftwareInfoResponse{}, commonSoftwareInfoResponseProps,
		messages.Constructor{
			Params: "String appSoftwareVersion, long appSoftwarePartNumber, long appSoftwarePartDashNumber, long appSoftwarePartRevisionNumber, " +
				"String bootloaderVersion, long bootloaderPartNumber",
			Func: NewTruncatedCommonSoftwareInfoResponse,
		},
		messages.Constructor{
			Params: "int mcuTypeId, String appSoftwareVersion, long appSoftwarePartNumber, long appSoftwarePartRevisionNumber, " +
				"String bootloaderVersion, long bootloaderPartNumber, long bootloaderPartRevisionNumber",
			Func: NewMobiCommonSoftwareInfoResponse,
		},
		messages.Constructor{
			Params: "String appSoftwareVersion, long appSoftwarePartNumber, long appSoftwarePartDashNumber, long appSoftwarePartRevisionNumber, " +
				"String bootloaderVersion, long bootloaderPartNumber, long bootloaderPartDashNumber, long bootloaderPartRevisionNumber",
			Func: NewCommonSoftwareInfoResponse,
		},
		messages.Constructor{Params: "byte[] raw", Func: NewCommonSoftwareInfoResponseFromCargo},
	)
}

// NewCommonSoftwareInfoResponse builds the t:slim X2 layout. It fails when a
// version does not fit its field.
func NewCommonSoftwareInfoResponse(appSoftwareVersion string, appSoftwarePartNumber, appSoftwarePartDashNumber, appSoftwarePartRevisionNumber int64,
	bootloaderVersion string, bootloaderPartNumber, bootloaderPartDashNumber, bootloaderPartRevisionNumber int64) (*CommonSoftwareInfoResponse, error) {
	m, err := newX2CommonSoftwareInfoResponse(appSoftwareVersion, appSoftwarePartNumber, appSoftwarePartDashNumber, appSoftwarePartRevisionNumber,
		bootloaderVersion, bootloaderPartNumber)
	if err != nil {
		return nil, err
	}
	m.BootloaderPartDashNumber = &bootloaderPartDashNumber
	m.BootloaderPartRevisionNumber = &bootloaderPartRevisionNumber
	m.SetCargo(helpers.Combine(m.Cargo(), helpers.ToUint32(bootloaderPartDashNumber), helpers.ToUint32(bootloaderPartRevisionNumber)))
	return m, nil
}

// NewMobiCommonSoftwareInfoResponse builds the Tandem Mobi layout. It fails
// when a version does not fit its field.
func NewMobiCommonSoftwareInfoResponse(mcuTypeID int32, appSoftwareVersion string, appSoftwarePartNumber, appSoftwarePartRevisionNumber int64,
	bootloaderVersion string, bootloaderPartNumber, bootloaderPartRevisionNumber int64) (*CommonSoftwareInfoResponse, error) {
	appVersion, err := helpers.WriteString(appSoftwareVersion, 17)
	if err != nil {
		return nil, messages.Invalid("CommonSoftwareInfoResponse: app software version: %v", err)
	}
	bootVersion, err := helpers.WriteString(bootloaderVersion, 17)
	if err != nil {
		return nil, messages.Invalid("CommonSoftwareInfoResponse: bootloader version: %v", err)
	}
	return NewCommonSoftwareInfoResponseFromCargo(helpers.Combine(
		[]byte{byte(mcuTypeID)},
		appVersion,
		helpers.ToUint32(appSoftwarePartNumber),
		helpers.ToUint32(appSoftwarePartRevisionNumber),
		bootVersion,
		helpers.ToUint32(bootloaderPartNumber),
		helpers.ToUint32(bootloaderPartRevisionNumber),
	))
}

// NewTruncatedCommonSoftwareInfoResponse builds the t:slim X2 layout cut off
// after the bootloader part number.
//
// Deprecated: no pump sends the 51 bytes this builds, and Parse reads them as
// the Mobi layout. Use NewMobiCommonSoftwareInfoResponse or
// NewCommonSoftwareInfoResponse.
func NewTruncatedCommonSoftwareInfoResponse(appSoftwareVersion string, appSoftwarePartNumber, appSoftwarePartDashNumber, appSoftwarePartRevisionNumber int64,
	bootloaderVersion string, bootloaderPartNumber int64) (*CommonSoftwareInfoResponse, error) {
	return newX2CommonSoftwareInfoResponse(appSoftwareVersion, appSoftwarePartNumber, appSoftwarePartDashNumber, appSoftwarePartRevisionNumber,
		bootloaderVersion, bootloaderPartNumber)
}

func newX2CommonSoftwareInfoResponse(appSoftwareVersion string, appSoftwarePartNumber, appSoftwarePartDashNumber, appSoftwarePartRevisionNumber int64,
	bootloaderVersion string, bootloaderPartNumber int64) (*CommonSoftwareInfoResponse, error) {
	appVersion, err := helpers.WriteString(appSoftwareVersion, 18)
	if err != nil {
		return nil, messages.Invalid("CommonSoftwareInfoResponse: app software version: %v", err)
	}
	bootVersion, err := helpers.WriteString(bootloaderVersion, 17)
	if err != nil {
		return nil, messages.Invalid("CommonSoftwareInfoResponse: bootloader version: %v", err)
	}
	m := &CommonSoftwareInfoResponse{
		AppSoftwareVersion:            appSoftwareVersion,
		AppSoftwarePartNumber:         appSoftwarePartNumber,
		AppSoftwarePartDashNumber:     appSoftwarePartDashNumber,
		AppSoftwarePartRevisionNumber: appSoftwarePartRevisionNumber,
		BootloaderVersion:             bootloaderVersion,
		BootloaderPartNumber:          bootloaderPartNumber,
	}
	m.SetCargo(helpers.Combine(
		appVersion,
		helpers.ToUint32(appSoftwarePartNumber),
		helpers.ToUint32(appSoftwarePartDashNumber),
		helpers.ToUint32(appSoftwarePartRevisionNumber),
		bootVersion,
		helpers.ToUint32(bootloaderPartNumber),
	))
	return m, nil
}

// NewCommonSoftwareInfoResponseFromCargo parses a response's payload.
func NewCommonSoftwareInfoResponseFromCargo(raw []byte) (*CommonSoftwareInfoResponse, error) {
	m := &CommonSoftwareInfoResponse{}
	return m, m.Parse(raw)
}

// Parse implements messages.Message. A t:slim X2 payload shorter than 59
// bytes leaves the bootloader's dash and revision numbers as they were.
func (m *CommonSoftwareInfoResponse) Parse(raw []byte) error {
	if len(raw) < commonSoftwareInfoMobiSize {
		return messages.Invalid("CommonSoftwareInfoResponse: payload of %d bytes, want %d or more", len(raw), commonSoftwareInfoMobiSize)
	}
	m.SetCargo(raw)
	if len(raw) == commonSoftwareInfoMobiSize {
		mcuTypeID := int32(raw[0])
		bootloaderPartRevisionNumber := helpers.ReadUint32(raw, 47)
		m.MCUTypeID = &mcuTypeID
		m.AppSoftwareVersion = helpers.ReadString(raw, 1, 17)
		m.AppSoftwarePartNumber = helpers.ReadUint32(raw, 18)
		m.AppSoftwarePartDashNumber = 0
		m.AppSoftwarePartRevisionNumber = helpers.ReadUint32(raw, 22)
		m.BootloaderVersion = helpers.ReadString(raw, 26, 17)
		m.BootloaderPartNumber = helpers.ReadUint32(raw, 43)
		m.BootloaderPartDashNumber = nil
		m.BootloaderPartRevisionNumber = &bootloaderPartRevisionNumber
		return nil
	}
	m.MCUTypeID = nil
	m.AppSoftwareVersion = helpers.ReadString(raw, 0, 18)
	m.AppSoftwarePartNumber = helpers.ReadUint32(raw, 18)
	m.AppSoftwarePartDashNumber = helpers.ReadUint32(raw, 22)
	m.AppSoftwarePartRevisionNumber = helpers.ReadUint32(raw, 26)
	m.BootloaderVersion = helpers.ReadString(raw, 30, 17)
	m.BootloaderPartNumber = helpers.ReadUint32(raw, 47)
	if len(raw) >= 59 {
		bootloaderPartDashNumber := helpers.ReadUint32(raw, 51)
		bootloaderPartRevisionNumber := helpers.ReadUint32(raw, 55)
		m.BootloaderPartDashNumber = &bootloaderPartDashNumber
		m.BootloaderPartRevisionNumber = &bootloaderPartRevisionNumber
	}
	return nil
}

// MCUType is the microcontroller the response describes, or nil outside the
// Mobi layout.
func (m *CommonSoftwareInfoResponse) MCUType() *MCUType {
	if m.MCUTypeID == nil {
		return nil
	}
	t := MCUTypeFromID(*m.MCUTypeID)
	return &t
}
