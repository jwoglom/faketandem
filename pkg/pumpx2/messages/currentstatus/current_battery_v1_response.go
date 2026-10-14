package currentstatus

import "github.com/jwoglom/faketandem/pkg/pumpx2/messages"

// CurrentBatteryV1Response is the battery charge of a pump at API 2.1.
// CurrentBatteryIbc is the charge left, 0-100, as the pump displays it;
// CurrentBatteryAbc is a second 0-100 figure, not the displayed one.
type CurrentBatteryV1Response struct {
	messages.Base
	CurrentBatteryAbc int32 `pumpx2:"currentBatteryAbc"`
	CurrentBatteryIbc int32 `pumpx2:"currentBatteryIbc"`
}

var currentBatteryV1ResponseProps = messages.Props{OpCode: 53, Size: 2, Type: messages.Response, Request: "CurrentBatteryV1Request"}

func init() {
	messages.Register("CurrentBatteryV1Response", &CurrentBatteryV1Response{}, currentBatteryV1ResponseProps,
		messages.Constructor{Params: "int currentBatteryAbc, int currentBatteryIbc", Func: NewCurrentBatteryV1Response},
	)
}

// NewCurrentBatteryV1Response builds the response.
func NewCurrentBatteryV1Response(currentBatteryAbc, currentBatteryIbc int32) *CurrentBatteryV1Response {
	m := &CurrentBatteryV1Response{CurrentBatteryAbc: currentBatteryAbc, CurrentBatteryIbc: currentBatteryIbc}
	m.SetCargo([]byte{byte(currentBatteryAbc), byte(currentBatteryIbc)})
	return m
}

// Parse implements messages.Message.
func (m *CurrentBatteryV1Response) Parse(raw []byte) error {
	if len(raw) != currentBatteryV1ResponseProps.Size {
		return messages.InvalidLength("CurrentBatteryV1Response", len(raw), currentBatteryV1ResponseProps.Size)
	}
	m.SetCargo(raw)
	m.CurrentBatteryAbc = int32(int8(raw[0]))
	m.CurrentBatteryIbc = int32(int8(raw[1]))
	return nil
}

// BatteryPercent is the charge the pump displays.
func (m *CurrentBatteryV1Response) BatteryPercent() int32 { return m.CurrentBatteryIbc }
