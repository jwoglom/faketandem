package currentstatus

import "github.com/jwoglom/faketandem/pkg/pumpx2/messages"

// ExtendedBolusStatusV2Request asks for the extended bolus in progress, if any, with
// the pump's time since reset.
type ExtendedBolusStatusV2Request struct {
	messages.Base
}

var extendedBolusStatusV2RequestProps = messages.Props{OpCode: -74, Type: messages.Request, Response: "ExtendedBolusStatusV2Response"}

func init() {
	messages.Register("ExtendedBolusStatusV2Request", &ExtendedBolusStatusV2Request{}, extendedBolusStatusV2RequestProps,
		messages.Constructor{Func: NewExtendedBolusStatusV2Request},
	)
}

// NewExtendedBolusStatusV2Request is the request, which has no payload.
func NewExtendedBolusStatusV2Request() *ExtendedBolusStatusV2Request {
	m := &ExtendedBolusStatusV2Request{}
	m.SetCargo([]byte{})
	return m
}

// Parse implements messages.Message. Any payload parses: pumpX2 does not
// check its size.
func (m *ExtendedBolusStatusV2Request) Parse(raw []byte) error {
	m.SetCargo(raw)
	return nil
}
