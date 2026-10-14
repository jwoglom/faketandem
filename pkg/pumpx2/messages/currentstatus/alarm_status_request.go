package currentstatus

import "github.com/jwoglom/faketandem/pkg/pumpx2/messages"

// AlarmStatusRequest asks which alarms are active.
type AlarmStatusRequest struct {
	messages.Base
}

var alarmStatusRequestProps = messages.Props{OpCode: 70, Type: messages.Request, Response: "AlarmStatusResponse"}

func init() {
	messages.Register("AlarmStatusRequest", &AlarmStatusRequest{}, alarmStatusRequestProps,
		messages.Constructor{Func: NewAlarmStatusRequest},
	)
}

// NewAlarmStatusRequest is the request, which has no payload.
func NewAlarmStatusRequest() *AlarmStatusRequest {
	m := &AlarmStatusRequest{}
	m.SetCargo([]byte{})
	return m
}

// Parse implements messages.Message. pumpX2 takes any payload.
func (m *AlarmStatusRequest) Parse(raw []byte) error {
	m.SetCargo(raw)
	return nil
}
