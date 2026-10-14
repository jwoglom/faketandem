package currentstatus

import (
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/helpers"
)

// BleSoftwareInfoResponse is the version of the Bluetooth stack (Nordic's
// SoftDevice) on the pump's BLE chip: ID 132, version 7.2.0 on the pumps
// pumpX2 has read.
type BleSoftwareInfoResponse struct {
	messages.Base
	SoftDeviceID            int32 `pumpx2:"softDeviceId"`
	SoftDeviceMajorVersion  int32 `pumpx2:"softDeviceMajorVersion"`
	SoftDeviceMinorVersion  int32 `pumpx2:"softDeviceMinorVersion"`
	SoftDeviceBugfixVersion int32 `pumpx2:"softDeviceBugfixVersion"`
	SoftDeviceVersion       int64 `pumpx2:"softDeviceVersion"`
	SoftDeviceSubVersion    int32 `pumpx2:"softDeviceSubVersion"`
}

var bleSoftwareInfoResponseProps = messages.Props{OpCode: -119, Size: 14, Type: messages.Response, Request: "BleSoftwareInfoRequest"}

func init() {
	messages.Register("BleSoftwareInfoResponse", &BleSoftwareInfoResponse{}, bleSoftwareInfoResponseProps,
		messages.Constructor{
			Params: "int softDeviceId, int softDeviceMajorVersion, int softDeviceMinorVersion, int softDeviceBugfixVersion, long softDeviceVersion, int softDeviceSubVersion",
			Func:   NewBleSoftwareInfoResponse,
		},
		messages.Constructor{Params: "byte[] raw", Func: NewBleSoftwareInfoResponseFromCargo},
	)
}

// NewBleSoftwareInfoResponse builds the response.
func NewBleSoftwareInfoResponse(softDeviceID, softDeviceMajorVersion, softDeviceMinorVersion, softDeviceBugfixVersion int32,
	softDeviceVersion int64, softDeviceSubVersion int32) *BleSoftwareInfoResponse {
	m := &BleSoftwareInfoResponse{
		SoftDeviceID:            softDeviceID,
		SoftDeviceMajorVersion:  softDeviceMajorVersion,
		SoftDeviceMinorVersion:  softDeviceMinorVersion,
		SoftDeviceBugfixVersion: softDeviceBugfixVersion,
		SoftDeviceVersion:       softDeviceVersion,
		SoftDeviceSubVersion:    softDeviceSubVersion,
	}
	m.SetCargo(helpers.Combine(
		helpers.FirstTwoBytesLittleEndian(softDeviceID),
		helpers.FirstTwoBytesLittleEndian(softDeviceMajorVersion),
		helpers.FirstTwoBytesLittleEndian(softDeviceMinorVersion),
		helpers.FirstTwoBytesLittleEndian(softDeviceBugfixVersion),
		helpers.ToUint32(softDeviceVersion),
		helpers.FirstTwoBytesLittleEndian(softDeviceSubVersion),
	))
	return m
}

// NewBleSoftwareInfoResponseFromCargo parses a response's payload.
func NewBleSoftwareInfoResponseFromCargo(raw []byte) (*BleSoftwareInfoResponse, error) {
	m := &BleSoftwareInfoResponse{}
	return m, m.Parse(raw)
}

// Parse implements messages.Message. Like pumpX2 it reads the first 14
// bytes of a payload of any length, failing only on a shorter one.
func (m *BleSoftwareInfoResponse) Parse(raw []byte) error {
	if len(raw) < bleSoftwareInfoResponseProps.Size {
		return messages.InvalidLength("BleSoftwareInfoResponse", len(raw), bleSoftwareInfoResponseProps.Size)
	}
	m.SetCargo(raw)
	m.SoftDeviceID = helpers.ReadShort(raw, 0)
	m.SoftDeviceMajorVersion = helpers.ReadShort(raw, 2)
	m.SoftDeviceMinorVersion = helpers.ReadShort(raw, 4)
	m.SoftDeviceBugfixVersion = helpers.ReadShort(raw, 6)
	m.SoftDeviceVersion = helpers.ReadUint32(raw, 8)
	m.SoftDeviceSubVersion = helpers.ReadShort(raw, 12)
	return nil
}
