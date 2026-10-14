package currentstatus

import "github.com/jwoglom/faketandem/pkg/pumpx2/messages"

// CurrentBatteryV2Request asks a pump past API 2.1 for its battery charge and
// whether it is charging.
type CurrentBatteryV2Request struct {
	messages.Base
}

var currentBatteryV2RequestProps = messages.Props{OpCode: -112, Type: messages.Request, MinAPI: "API_V2_5", Response: "CurrentBatteryV2Response"}

func init() {
	messages.Register("CurrentBatteryV2Request", &CurrentBatteryV2Request{}, currentBatteryV2RequestProps,
		messages.Constructor{Func: NewCurrentBatteryV2Request},
	)
}

// NewCurrentBatteryV2Request is the request, which has no payload.
func NewCurrentBatteryV2Request() *CurrentBatteryV2Request {
	m := &CurrentBatteryV2Request{}
	m.SetCargo([]byte{})
	return m
}

// Parse implements messages.Message.
func (m *CurrentBatteryV2Request) Parse(raw []byte) error {
	if len(raw) != currentBatteryV2RequestProps.Size {
		return messages.InvalidLength("CurrentBatteryV2Request", len(raw), currentBatteryV2RequestProps.Size)
	}
	m.SetCargo(raw)
	return nil
}
