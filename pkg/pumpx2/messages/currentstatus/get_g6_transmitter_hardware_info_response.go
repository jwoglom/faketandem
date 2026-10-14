package currentstatus

import (
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/helpers"
)

// GetG6TransmitterHardwareInfoResponse is the paired Dexcom G6 transmitter's
// details, each a NUL-padded 16-byte string, empty when the pump has none.
type GetG6TransmitterHardwareInfoResponse struct {
	messages.Base
	TransmitterFirmwareVersion  string `pumpx2:"transmitterFirmwareVersion"`
	TransmitterHardwareRevision string `pumpx2:"transmitterHardwareRevision"`
	TransmitterBleHardwareID    string `pumpx2:"transmitterBleHardwareId"`
	TransmitterSoftwareNumber   string `pumpx2:"transmitterSoftwareNumber"`
	TransmitterPairingCode      string `pumpx2:"transmitterPairingCode"`
	TransmitterSerialNumber     string `pumpx2:"transmitterSerialNumber"`
}

var getG6TransmitterHardwareInfoResponseProps = messages.Props{OpCode: -59, Size: 96, Type: messages.Response, Request: "GetG6TransmitterHardwareInfoRequest"}

func init() {
	messages.Register("GetG6TransmitterHardwareInfoResponse", &GetG6TransmitterHardwareInfoResponse{}, getG6TransmitterHardwareInfoResponseProps,
		messages.Constructor{Params: "byte[] raw", Func: NewGetG6TransmitterHardwareInfoResponseFromCargo},
	)
}

// NewGetG6TransmitterHardwareInfoResponseFromCargo parses a response's
// payload.
func NewGetG6TransmitterHardwareInfoResponseFromCargo(raw []byte) (*GetG6TransmitterHardwareInfoResponse, error) {
	m := &GetG6TransmitterHardwareInfoResponse{}
	return m, m.Parse(raw)
}

// Parse implements messages.Message.
func (m *GetG6TransmitterHardwareInfoResponse) Parse(raw []byte) error {
	if len(raw) != getG6TransmitterHardwareInfoResponseProps.Size {
		return messages.InvalidLength("GetG6TransmitterHardwareInfoResponse", len(raw), getG6TransmitterHardwareInfoResponseProps.Size)
	}
	m.SetCargo(raw)
	m.TransmitterFirmwareVersion = helpers.ReadString(raw, 0, 16)
	m.TransmitterHardwareRevision = helpers.ReadString(raw, 16, 16)
	m.TransmitterBleHardwareID = helpers.ReadString(raw, 32, 16)
	m.TransmitterSoftwareNumber = helpers.ReadString(raw, 48, 16)
	m.TransmitterPairingCode = helpers.ReadString(raw, 64, 16)
	m.TransmitterSerialNumber = helpers.ReadString(raw, 80, 16)
	return nil
}
