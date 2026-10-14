package currentstatus

import "github.com/jwoglom/faketandem/pkg/pumpx2/messages"

// GetG6TransmitterHardwareInfoRequest asks for the paired Dexcom G6
// transmitter's firmware and hardware details, pairing code and serial
// number.
type GetG6TransmitterHardwareInfoRequest struct {
	messages.Base
}

var getG6TransmitterHardwareInfoRequestProps = messages.Props{OpCode: -60, Type: messages.Request, Response: "GetG6TransmitterHardwareInfoResponse"}

func init() {
	messages.Register("GetG6TransmitterHardwareInfoRequest", &GetG6TransmitterHardwareInfoRequest{}, getG6TransmitterHardwareInfoRequestProps,
		messages.Constructor{Func: NewGetG6TransmitterHardwareInfoRequest},
	)
}

// NewGetG6TransmitterHardwareInfoRequest is the request, which has no payload.
func NewGetG6TransmitterHardwareInfoRequest() *GetG6TransmitterHardwareInfoRequest {
	m := &GetG6TransmitterHardwareInfoRequest{}
	m.SetCargo([]byte{})
	return m
}

// Parse implements messages.Message.
func (m *GetG6TransmitterHardwareInfoRequest) Parse(raw []byte) error {
	if len(raw) != getG6TransmitterHardwareInfoRequestProps.Size {
		return messages.InvalidLength("GetG6TransmitterHardwareInfoRequest", len(raw), getG6TransmitterHardwareInfoRequestProps.Size)
	}
	m.SetCargo(raw)
	return nil
}
