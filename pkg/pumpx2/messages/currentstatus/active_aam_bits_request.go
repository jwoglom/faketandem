package currentstatus

import "github.com/jwoglom/faketandem/pkg/pumpx2/messages"

// ActiveAamBitsRequest asks a Mobi which alerts, alarms or malfunctions are
// active. Its one byte of payload is not documented.
type ActiveAamBitsRequest struct {
	messages.Base
}

var activeAamBitsRequestProps = messages.Props{OpCode: -110, Size: 1, Type: messages.Request, MinAPI: "MOBI_API_V3_5", Response: "ActiveAamBitsResponse"}

func init() {
	messages.Register("ActiveAamBitsRequest", &ActiveAamBitsRequest{}, activeAamBitsRequestProps,
		messages.Constructor{Func: NewActiveAamBitsRequest},
		messages.Constructor{Params: "byte[] raw", Func: NewActiveAamBitsRequestFromCargo},
	)
}

// NewActiveAamBitsRequest is the request with no payload, which pumpX2
// builds although it parses only a one-byte one.
func NewActiveAamBitsRequest() *ActiveAamBitsRequest {
	m := &ActiveAamBitsRequest{}
	m.SetCargo([]byte{})
	return m
}

// NewActiveAamBitsRequestFromCargo parses a request's payload.
func NewActiveAamBitsRequestFromCargo(raw []byte) (*ActiveAamBitsRequest, error) {
	m := &ActiveAamBitsRequest{}
	return m, m.Parse(raw)
}

// Parse implements messages.Message.
func (m *ActiveAamBitsRequest) Parse(raw []byte) error {
	if len(raw) != activeAamBitsRequestProps.Size {
		return messages.InvalidLength("ActiveAamBitsRequest", len(raw), activeAamBitsRequestProps.Size)
	}
	m.SetCargo(raw)
	return nil
}
