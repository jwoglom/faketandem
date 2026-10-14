package currentstatus

import "github.com/jwoglom/faketandem/pkg/pumpx2/messages"

// LastBolusStatusV2Request asks for the last bolus and the volume requested
// for it.
type LastBolusStatusV2Request struct {
	messages.Base
}

var lastBolusStatusV2RequestProps = messages.Props{OpCode: -92, Type: messages.Request, MinAPI: "API_V2_5", Response: "LastBolusStatusV2Response"}

func init() {
	messages.Register("LastBolusStatusV2Request", &LastBolusStatusV2Request{}, lastBolusStatusV2RequestProps,
		messages.Constructor{Func: NewLastBolusStatusV2Request},
	)
}

// NewLastBolusStatusV2Request is the request, which has no payload.
func NewLastBolusStatusV2Request() *LastBolusStatusV2Request {
	m := &LastBolusStatusV2Request{}
	m.SetCargo([]byte{})
	return m
}

// Parse implements messages.Message.
func (m *LastBolusStatusV2Request) Parse(raw []byte) error {
	if len(raw) != lastBolusStatusV2RequestProps.Size {
		return messages.InvalidLength("LastBolusStatusV2Request", len(raw), lastBolusStatusV2RequestProps.Size)
	}
	m.SetCargo(raw)
	return nil
}
