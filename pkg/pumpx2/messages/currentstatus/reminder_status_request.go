package currentstatus

import "github.com/jwoglom/faketandem/pkg/pumpx2/messages"

// ReminderStatusRequest asks which reminders the pump is showing.
type ReminderStatusRequest struct {
	messages.Base
}

var reminderStatusRequestProps = messages.Props{OpCode: 72, Type: messages.Request, Response: "ReminderStatusResponse"}

func init() {
	messages.Register("ReminderStatusRequest", &ReminderStatusRequest{}, reminderStatusRequestProps)
}

// NewReminderStatusRequest is the request, which has no payload.
func NewReminderStatusRequest() *ReminderStatusRequest {
	m := &ReminderStatusRequest{}
	m.SetCargo([]byte{})
	return m
}

// Parse implements messages.Message.
func (m *ReminderStatusRequest) Parse(raw []byte) error {
	if len(raw) != reminderStatusRequestProps.Size {
		return messages.InvalidLength("ReminderStatusRequest", len(raw), reminderStatusRequestProps.Size)
	}
	m.SetCargo(raw)
	return nil
}
