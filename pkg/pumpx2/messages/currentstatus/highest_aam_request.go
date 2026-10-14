package currentstatus

import "github.com/jwoglom/faketandem/pkg/pumpx2/messages"

// HighestAamRequest asks for the highest-priority active alert, alarm or malfunction
// (AAM).
type HighestAamRequest struct {
	messages.Base
}

var highestAamRequestProps = messages.Props{OpCode: 120, Type: messages.Request, Response: "HighestAamResponse"}

func init() {
	messages.Register("HighestAamRequest", &HighestAamRequest{}, highestAamRequestProps,
		messages.Constructor{Func: NewHighestAamRequest},
	)
}

// NewHighestAamRequest is the request, which has no payload.
func NewHighestAamRequest() *HighestAamRequest {
	m := &HighestAamRequest{}
	m.SetCargo([]byte{})
	return m
}

// Parse implements messages.Message.
func (m *HighestAamRequest) Parse(raw []byte) error {
	if len(raw) != highestAamRequestProps.Size {
		return messages.InvalidLength("HighestAamRequest", len(raw), highestAamRequestProps.Size)
	}
	m.SetCargo(raw)
	return nil
}
