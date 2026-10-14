package currentstatus

import "github.com/jwoglom/faketandem/pkg/pumpx2/messages"

// TimeSinceResetRequest asks for the pump's clock and how long it has run
// since it was reset, which pumpX2 reads on every connection.
type TimeSinceResetRequest struct {
	messages.Base
}

var timeSinceResetRequestProps = messages.Props{OpCode: 54, Type: messages.Request, Response: "TimeSinceResetResponse"}

func init() {
	messages.Register("TimeSinceResetRequest", &TimeSinceResetRequest{}, timeSinceResetRequestProps)
}

// NewTimeSinceResetRequest is the request, which has no payload.
func NewTimeSinceResetRequest() *TimeSinceResetRequest {
	m := &TimeSinceResetRequest{}
	m.SetCargo([]byte{})
	return m
}

// Parse implements messages.Message.
func (m *TimeSinceResetRequest) Parse(raw []byte) error {
	if len(raw) != timeSinceResetRequestProps.Size {
		return messages.InvalidLength("TimeSinceResetRequest", len(raw), timeSinceResetRequestProps.Size)
	}
	m.SetCargo(raw)
	return nil
}
