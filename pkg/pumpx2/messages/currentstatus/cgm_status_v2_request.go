package currentstatus

import "github.com/jwoglom/faketandem/pkg/pumpx2/messages"

// CgmStatusV2Request asks a Mobi for its CGM session, answered by
// CgmStatusV2Response.
type CgmStatusV2Request struct {
	messages.Base
}

var cgmStatusV2RequestProps = messages.Props{OpCode: -66, Type: messages.Request, MinAPI: "MOBI_API_V3_5", SupportedDevices: "MOBI_ONLY", Response: "CgmStatusV2Response"}

func init() {
	messages.Register("CgmStatusV2Request", &CgmStatusV2Request{}, cgmStatusV2RequestProps,
		messages.Constructor{Func: NewCgmStatusV2Request},
	)
}

// NewCgmStatusV2Request is the request, which has no payload.
func NewCgmStatusV2Request() *CgmStatusV2Request {
	m := &CgmStatusV2Request{}
	m.SetCargo([]byte{})
	return m
}

// Parse implements messages.Message.
func (m *CgmStatusV2Request) Parse(raw []byte) error {
	if len(raw) != cgmStatusV2RequestProps.Size {
		return messages.InvalidLength("CgmStatusV2Request", len(raw), cgmStatusV2RequestProps.Size)
	}
	m.SetCargo(raw)
	return nil
}
