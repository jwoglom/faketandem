package currentstatus

import "github.com/jwoglom/faketandem/pkg/pumpx2/messages"

// ControlIQIOBRequest asks a pump with Control-IQ for insulin on board;
// pumpX2 sends NonControlIQIOBRequest to one without.
type ControlIQIOBRequest struct {
	messages.Base
}

var controlIQIOBRequestProps = messages.Props{OpCode: 108, Type: messages.Request, Response: "ControlIQIOBResponse"}

func init() {
	messages.Register("ControlIQIOBRequest", &ControlIQIOBRequest{}, controlIQIOBRequestProps,
		messages.Constructor{Func: NewControlIQIOBRequest},
	)
}

// NewControlIQIOBRequest is the request, which has no payload.
func NewControlIQIOBRequest() *ControlIQIOBRequest {
	m := &ControlIQIOBRequest{}
	m.SetCargo([]byte{})
	return m
}

// Parse implements messages.Message. Any payload parses: pumpX2 does not
// check its size.
func (m *ControlIQIOBRequest) Parse(raw []byte) error {
	m.SetCargo(raw)
	return nil
}
