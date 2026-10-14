package currentstatus

import "github.com/jwoglom/faketandem/pkg/pumpx2/messages"

// ControlIQInfoV2Request asks a pump past API 2.1 for Control-IQ's state and
// settings, and the exercise activity.
type ControlIQInfoV2Request struct {
	messages.Base
}

var controlIQInfoV2RequestProps = messages.Props{OpCode: -78, Type: messages.Request, Response: "ControlIQInfoV2Response"}

func init() {
	messages.Register("ControlIQInfoV2Request", &ControlIQInfoV2Request{}, controlIQInfoV2RequestProps,
		messages.Constructor{Func: NewControlIQInfoV2Request},
	)
}

// NewControlIQInfoV2Request is the request, which has no payload.
func NewControlIQInfoV2Request() *ControlIQInfoV2Request {
	m := &ControlIQInfoV2Request{}
	m.SetCargo([]byte{})
	return m
}

// Parse implements messages.Message.
func (m *ControlIQInfoV2Request) Parse(raw []byte) error {
	if len(raw) != controlIQInfoV2RequestProps.Size {
		return messages.InvalidLength("ControlIQInfoV2Request", len(raw), controlIQInfoV2RequestProps.Size)
	}
	m.SetCargo(raw)
	return nil
}
