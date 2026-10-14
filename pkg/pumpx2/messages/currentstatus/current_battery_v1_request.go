package currentstatus

import "github.com/jwoglom/faketandem/pkg/pumpx2/messages"

// CurrentBatteryV1Request asks a pump at API 2.1 for its battery charge; later
// pumps take CurrentBatteryV2Request.
type CurrentBatteryV1Request struct {
	messages.Base
}

var currentBatteryV1RequestProps = messages.Props{OpCode: 52, Type: messages.Request, Response: "CurrentBatteryV1Response"}

func init() {
	messages.Register("CurrentBatteryV1Request", &CurrentBatteryV1Request{}, currentBatteryV1RequestProps,
		messages.Constructor{Func: NewCurrentBatteryV1Request},
	)
}

// NewCurrentBatteryV1Request is the request, which has no payload.
func NewCurrentBatteryV1Request() *CurrentBatteryV1Request {
	m := &CurrentBatteryV1Request{}
	m.SetCargo([]byte{})
	return m
}

// Parse implements messages.Message.
func (m *CurrentBatteryV1Request) Parse(raw []byte) error {
	if len(raw) != currentBatteryV1RequestProps.Size {
		return messages.InvalidLength("CurrentBatteryV1Request", len(raw), currentBatteryV1RequestProps.Size)
	}
	m.SetCargo(raw)
	return nil
}
