package currentstatus

import "github.com/jwoglom/faketandem/pkg/pumpx2/messages"

// LastBolusStatusV3Request asks a Mobi for its last standard and last extended
// bolus.
type LastBolusStatusV3Request struct {
	messages.Base
}

var lastBolusStatusV3RequestProps = messages.Props{OpCode: -70, Type: messages.Request, MinAPI: "MOBI_API_V3_5", Response: "LastBolusStatusV3Response"}

func init() {
	messages.Register("LastBolusStatusV3Request", &LastBolusStatusV3Request{}, lastBolusStatusV3RequestProps,
		messages.Constructor{Func: NewLastBolusStatusV3Request},
	)
}

// NewLastBolusStatusV3Request is the request, which has no payload.
func NewLastBolusStatusV3Request() *LastBolusStatusV3Request {
	m := &LastBolusStatusV3Request{}
	m.SetCargo([]byte{})
	return m
}

// Parse implements messages.Message. pumpX2 takes any payload.
func (m *LastBolusStatusV3Request) Parse(raw []byte) error {
	m.SetCargo(raw)
	return nil
}
