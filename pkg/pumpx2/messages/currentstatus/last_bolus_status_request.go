package currentstatus

import "github.com/jwoglom/faketandem/pkg/pumpx2/messages"

// LastBolusStatusRequest asks for the last bolus. pumpX2 asks a pump past
// API 2.1 LastBolusStatusV2Request instead, whose answer adds the volume
// requested.
type LastBolusStatusRequest struct {
	messages.Base
}

var lastBolusStatusRequestProps = messages.Props{OpCode: 48, Type: messages.Request, Response: "LastBolusStatusResponse"}

func init() {
	messages.Register("LastBolusStatusRequest", &LastBolusStatusRequest{}, lastBolusStatusRequestProps,
		messages.Constructor{Func: NewLastBolusStatusRequest},
	)
}

// NewLastBolusStatusRequest is the request, which has no payload.
func NewLastBolusStatusRequest() *LastBolusStatusRequest {
	m := &LastBolusStatusRequest{}
	m.SetCargo([]byte{})
	return m
}

// Parse implements messages.Message.
func (m *LastBolusStatusRequest) Parse(raw []byte) error {
	if len(raw) != lastBolusStatusRequestProps.Size {
		return messages.InvalidLength("LastBolusStatusRequest", len(raw), lastBolusStatusRequestProps.Size)
	}
	m.SetCargo(raw)
	return nil
}
