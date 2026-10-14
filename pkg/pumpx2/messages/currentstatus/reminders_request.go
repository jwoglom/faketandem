package currentstatus

import "github.com/jwoglom/faketandem/pkg/pumpx2/messages"

// RemindersRequest asks for the reminders configured on the pump.
type RemindersRequest struct {
	messages.Base
}

var remindersRequestProps = messages.Props{OpCode: 88, Type: messages.Request, Response: "RemindersResponse"}

func init() {
	messages.Register("RemindersRequest", &RemindersRequest{}, remindersRequestProps)
}

// NewRemindersRequest is the request, which has no payload.
func NewRemindersRequest() *RemindersRequest {
	m := &RemindersRequest{}
	m.SetCargo([]byte{})
	return m
}

// Parse implements messages.Message.
func (m *RemindersRequest) Parse(raw []byte) error {
	if len(raw) != remindersRequestProps.Size {
		return messages.InvalidLength("RemindersRequest", len(raw), remindersRequestProps.Size)
	}
	m.SetCargo(raw)
	return nil
}
