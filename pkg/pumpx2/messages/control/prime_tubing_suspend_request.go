package control

import "github.com/jwoglom/faketandem/pkg/pumpx2/messages"

// PrimeTubingSuspendRequest is the Tandem Mobi app's prime tubing suspend
// command.
type PrimeTubingSuspendRequest struct {
	messages.Base
}

var primeTubingSuspendRequestProps = messages.Props{OpCode: -18, Signed: true, Type: messages.Request, Characteristic: messages.Control,
	Response: "PrimeTubingSuspendResponse"}

func init() {
	messages.Register("PrimeTubingSuspendRequest", &PrimeTubingSuspendRequest{}, primeTubingSuspendRequestProps,
		messages.Constructor{Func: NewPrimeTubingSuspendRequest},
	)
}

// NewPrimeTubingSuspendRequest builds the request, which has no payload.
func NewPrimeTubingSuspendRequest() *PrimeTubingSuspendRequest {
	m := &PrimeTubingSuspendRequest{}
	m.SetCargo([]byte{})
	return m
}

// Parse implements messages.Message. Any payload parses.
func (m *PrimeTubingSuspendRequest) Parse(raw []byte) error {
	m.SetCargo(messages.RemoveSignedRequestHmacBytes(primeTubingSuspendRequestProps, raw))
	return nil
}
