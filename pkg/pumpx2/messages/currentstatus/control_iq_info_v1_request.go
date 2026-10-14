package currentstatus

import "github.com/jwoglom/faketandem/pkg/pumpx2/messages"

// ControlIQInfoV1Request asks a pump at API 2.1 for Control-IQ's state and
// settings; later pumps take ControlIQInfoV2Request.
type ControlIQInfoV1Request struct {
	messages.Base
}

var controlIQInfoV1RequestProps = messages.Props{OpCode: 104, Type: messages.Request, Response: "ControlIQInfoV1Response"}

func init() {
	messages.Register("ControlIQInfoV1Request", &ControlIQInfoV1Request{}, controlIQInfoV1RequestProps,
		messages.Constructor{Func: NewControlIQInfoV1Request},
	)
}

// NewControlIQInfoV1Request is the request, which has no payload.
func NewControlIQInfoV1Request() *ControlIQInfoV1Request {
	m := &ControlIQInfoV1Request{}
	m.SetCargo([]byte{})
	return m
}

// Parse implements messages.Message.
func (m *ControlIQInfoV1Request) Parse(raw []byte) error {
	if len(raw) != controlIQInfoV1RequestProps.Size {
		return messages.InvalidLength("ControlIQInfoV1Request", len(raw), controlIQInfoV1RequestProps.Size)
	}
	m.SetCargo(raw)
	return nil
}
