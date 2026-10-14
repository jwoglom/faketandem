package currentstatus

import "github.com/jwoglom/faketandem/pkg/pumpx2/messages"

// AlertStatusRequest asks which alerts are active.
type AlertStatusRequest struct {
	messages.Base
}

var alertStatusRequestProps = messages.Props{OpCode: 68, Type: messages.Request, Response: "AlertStatusResponse"}

func init() {
	messages.Register("AlertStatusRequest", &AlertStatusRequest{}, alertStatusRequestProps,
		messages.Constructor{Func: NewAlertStatusRequest},
	)
}

// NewAlertStatusRequest is the request, which has no payload.
func NewAlertStatusRequest() *AlertStatusRequest {
	m := &AlertStatusRequest{}
	m.SetCargo([]byte{})
	return m
}

// Parse implements messages.Message. pumpX2 takes any payload.
func (m *AlertStatusRequest) Parse(raw []byte) error {
	m.SetCargo(raw)
	return nil
}
