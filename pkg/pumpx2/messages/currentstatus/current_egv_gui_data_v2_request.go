package currentstatus

import "github.com/jwoglom/faketandem/pkg/pumpx2/messages"

// CurrentEgvGuiDataV2Request asks for the latest CGM reading, answered by
// CurrentEgvGuiDataV2Response.
type CurrentEgvGuiDataV2Request struct {
	messages.Base
}

var currentEgvGuiDataV2RequestProps = messages.Props{OpCode: -64, Type: messages.Request, MinAPI: "API_FUTURE", Response: "CurrentEgvGuiDataV2Response"}

func init() {
	messages.Register("CurrentEgvGuiDataV2Request", &CurrentEgvGuiDataV2Request{}, currentEgvGuiDataV2RequestProps,
		messages.Constructor{Func: NewCurrentEgvGuiDataV2Request},
	)
}

// NewCurrentEgvGuiDataV2Request is the request, which has no payload.
func NewCurrentEgvGuiDataV2Request() *CurrentEgvGuiDataV2Request {
	m := &CurrentEgvGuiDataV2Request{}
	m.SetCargo([]byte{})
	return m
}

// Parse implements messages.Message. Any payload parses: pumpX2 does not
// check its size.
func (m *CurrentEgvGuiDataV2Request) Parse(raw []byte) error {
	m.SetCargo(raw)
	return nil
}
