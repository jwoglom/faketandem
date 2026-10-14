package currentstatus

import "github.com/jwoglom/faketandem/pkg/pumpx2/messages"

// UnknownMobiOpcode110Request is a Mobi request whose meaning is unknown.
type UnknownMobiOpcode110Request struct {
	messages.Base
}

var unknownMobiOpcode110RequestProps = messages.Props{OpCode: 110, Type: messages.Request, MinAPI: "MOBI_API_V3_5", SupportedDevices: "MOBI_ONLY", Response: "UnknownMobiOpcode110Response"}

func init() {
	messages.Register("UnknownMobiOpcode110Request", &UnknownMobiOpcode110Request{}, unknownMobiOpcode110RequestProps)
}

// NewUnknownMobiOpcode110Request is the request, which has no payload.
func NewUnknownMobiOpcode110Request() *UnknownMobiOpcode110Request {
	m := &UnknownMobiOpcode110Request{}
	m.SetCargo([]byte{})
	return m
}

// Parse implements messages.Message.
func (m *UnknownMobiOpcode110Request) Parse(raw []byte) error {
	if len(raw) != unknownMobiOpcode110RequestProps.Size {
		return messages.InvalidLength("UnknownMobiOpcode110Request", len(raw), unknownMobiOpcode110RequestProps.Size)
	}
	m.SetCargo(raw)
	return nil
}
