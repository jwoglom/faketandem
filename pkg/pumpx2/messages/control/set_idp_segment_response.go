package control

import "github.com/jwoglom/faketandem/pkg/pumpx2/messages"

// SetIDPSegmentResponse is the pump's answer to a SetIDPSegmentRequest:
// Status 0 means success. Unknown has been 2 in every capture.
type SetIDPSegmentResponse struct {
	messages.Base
	Status  int32 `pumpx2:"status"`
	Unknown int32 `pumpx2:"unknown"`
}

var setIDPSegmentResponseProps = messages.Props{OpCode: -85, Size: 2, Signed: true, Type: messages.Response, Characteristic: messages.Control,
	Request: "SetIDPSegmentRequest"}

func init() {
	messages.Register("SetIDPSegmentResponse", &SetIDPSegmentResponse{}, setIDPSegmentResponseProps,
		messages.Constructor{Params: "byte[] raw", Func: NewSetIDPSegmentResponseFromCargo},
	)
}

// NewSetIDPSegmentResponseFromCargo parses a response's payload.
func NewSetIDPSegmentResponseFromCargo(raw []byte) (*SetIDPSegmentResponse, error) {
	m := &SetIDPSegmentResponse{}
	return m, m.Parse(raw)
}

// Parse implements messages.Message.
func (m *SetIDPSegmentResponse) Parse(raw []byte) error {
	raw = messages.RemoveSignedRequestHmacBytes(setIDPSegmentResponseProps, raw)
	if len(raw) != setIDPSegmentResponseProps.Size {
		return messages.InvalidLength("SetIDPSegmentResponse", len(raw), setIDPSegmentResponseProps.Size)
	}
	m.SetCargo(raw)
	m.Status = int32(int8(raw[0]))
	m.Unknown = int32(int8(raw[1]))
	return nil
}
