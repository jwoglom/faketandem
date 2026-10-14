package currentstatus

import "github.com/jwoglom/faketandem/pkg/pumpx2/messages"

// LastBGRequest asks for the last BG entered in the bolus calculator. It
// is not the last CGM reading, which only the history log has.
type LastBGRequest struct {
	messages.Base
}

var lastBGRequestProps = messages.Props{OpCode: 50, Type: messages.Request, Response: "LastBGResponse"}

func init() {
	messages.Register("LastBGRequest", &LastBGRequest{}, lastBGRequestProps,
		messages.Constructor{Func: NewLastBGRequest},
	)
}

// NewLastBGRequest is the request, which has no payload.
func NewLastBGRequest() *LastBGRequest {
	m := &LastBGRequest{}
	m.SetCargo([]byte{})
	return m
}

// Parse implements messages.Message.
func (m *LastBGRequest) Parse(raw []byte) error {
	if len(raw) != lastBGRequestProps.Size {
		return messages.InvalidLength("LastBGRequest", len(raw), lastBGRequestProps.Size)
	}
	m.SetCargo(raw)
	return nil
}
