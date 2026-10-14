package currentstatus

import "github.com/jwoglom/faketandem/pkg/pumpx2/messages"

// ControlIQSleepScheduleRequest asks for Control-IQ's sleep schedules.
type ControlIQSleepScheduleRequest struct {
	messages.Base
}

var controlIQSleepScheduleRequestProps = messages.Props{OpCode: 106, Type: messages.Request, Response: "ControlIQSleepScheduleResponse"}

func init() {
	messages.Register("ControlIQSleepScheduleRequest", &ControlIQSleepScheduleRequest{}, controlIQSleepScheduleRequestProps,
		messages.Constructor{Func: NewControlIQSleepScheduleRequest},
	)
}

// NewControlIQSleepScheduleRequest is the request, which has no payload.
func NewControlIQSleepScheduleRequest() *ControlIQSleepScheduleRequest {
	m := &ControlIQSleepScheduleRequest{}
	m.SetCargo([]byte{})
	return m
}

// Parse implements messages.Message.
func (m *ControlIQSleepScheduleRequest) Parse(raw []byte) error {
	if len(raw) != controlIQSleepScheduleRequestProps.Size {
		return messages.InvalidLength("ControlIQSleepScheduleRequest", len(raw), controlIQSleepScheduleRequestProps.Size)
	}
	m.SetCargo(raw)
	return nil
}
