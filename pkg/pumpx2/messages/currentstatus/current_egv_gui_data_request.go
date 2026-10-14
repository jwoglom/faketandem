package currentstatus

import "github.com/jwoglom/faketandem/pkg/pumpx2/messages"

// CurrentEGVGuiDataRequest asks for the latest CGM reading.
type CurrentEGVGuiDataRequest struct {
	messages.Base
}

var currentEGVGuiDataRequestProps = messages.Props{OpCode: 34, Type: messages.Request, Response: "CurrentEGVGuiDataResponse"}

func init() {
	messages.Register("CurrentEGVGuiDataRequest", &CurrentEGVGuiDataRequest{}, currentEGVGuiDataRequestProps,
		messages.Constructor{Func: NewCurrentEGVGuiDataRequest},
	)
}

// NewCurrentEGVGuiDataRequest is the request, which has no payload.
func NewCurrentEGVGuiDataRequest() *CurrentEGVGuiDataRequest {
	m := &CurrentEGVGuiDataRequest{}
	m.SetCargo([]byte{})
	return m
}

// Parse implements messages.Message.
func (m *CurrentEGVGuiDataRequest) Parse(raw []byte) error {
	if len(raw) != currentEGVGuiDataRequestProps.Size {
		return messages.InvalidLength("CurrentEGVGuiDataRequest", len(raw), currentEGVGuiDataRequestProps.Size)
	}
	m.SetCargo(raw)
	return nil
}
