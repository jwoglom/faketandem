package currentstatus

import "github.com/jwoglom/faketandem/pkg/pumpx2/messages"

// NonControlIQIOBRequest asks a pump without Control-IQ for its insulin on
// board; ControlIQIOBRequest replaces it on a pump with Control-IQ.
type NonControlIQIOBRequest struct {
	messages.Base
}

var nonControlIQIOBRequestProps = messages.Props{OpCode: 38, Type: messages.Request, Response: "NonControlIQIOBResponse"}

func init() {
	messages.Register("NonControlIQIOBRequest", &NonControlIQIOBRequest{}, nonControlIQIOBRequestProps)
}

// NewNonControlIQIOBRequest is the request, which has no payload.
func NewNonControlIQIOBRequest() *NonControlIQIOBRequest {
	m := &NonControlIQIOBRequest{}
	m.SetCargo([]byte{})
	return m
}

// Parse implements messages.Message. pumpX2 checks no length here.
func (m *NonControlIQIOBRequest) Parse(raw []byte) error {
	m.SetCargo(raw)
	return nil
}
