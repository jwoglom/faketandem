package control

import "github.com/jwoglom/faketandem/pkg/pumpx2/messages"

// SuspendPumpingRequest suspends insulin delivery.
type SuspendPumpingRequest struct {
	messages.Base
}

var suspendPumpingRequestProps = messages.Props{OpCode: -100, Signed: true, Type: messages.Request, Characteristic: messages.Control,
	ModifiesInsulinDelivery: true, MinAPI: "MOBI_API_V3_5", SupportedDevices: "MOBI_ONLY", Response: "SuspendPumpingResponse"}

func init() {
	messages.Register("SuspendPumpingRequest", &SuspendPumpingRequest{}, suspendPumpingRequestProps,
		messages.Constructor{Func: NewSuspendPumpingRequest},
	)
}

// NewSuspendPumpingRequest is the request, with no payload.
func NewSuspendPumpingRequest() *SuspendPumpingRequest {
	m := &SuspendPumpingRequest{}
	m.SetCargo([]byte{})
	return m
}

// Parse implements messages.Message.
func (m *SuspendPumpingRequest) Parse(raw []byte) error {
	raw = messages.RemoveSignedRequestHmacBytes(suspendPumpingRequestProps, raw)
	if len(raw) != suspendPumpingRequestProps.Size {
		return messages.InvalidLength("SuspendPumpingRequest", len(raw), suspendPumpingRequestProps.Size)
	}
	m.SetCargo(raw)
	return nil
}
