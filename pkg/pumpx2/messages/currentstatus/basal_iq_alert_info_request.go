package currentstatus

import "github.com/jwoglom/faketandem/pkg/pumpx2/messages"

// BasalIQAlertInfoRequest asks for the Basal-IQ alert the pump is showing.
type BasalIQAlertInfoRequest struct {
	messages.Base
}

var basalIQAlertInfoRequestProps = messages.Props{OpCode: 102, Type: messages.Request, Response: "BasalIQAlertInfoResponse"}

func init() {
	messages.Register("BasalIQAlertInfoRequest", &BasalIQAlertInfoRequest{}, basalIQAlertInfoRequestProps,
		messages.Constructor{Func: NewBasalIQAlertInfoRequest},
	)
}

// NewBasalIQAlertInfoRequest is the request, which has no payload.
func NewBasalIQAlertInfoRequest() *BasalIQAlertInfoRequest {
	m := &BasalIQAlertInfoRequest{}
	m.SetCargo([]byte{})
	return m
}

// Parse implements messages.Message.
func (m *BasalIQAlertInfoRequest) Parse(raw []byte) error {
	if len(raw) != basalIQAlertInfoRequestProps.Size {
		return messages.InvalidLength("BasalIQAlertInfoRequest", len(raw), basalIQAlertInfoRequestProps.Size)
	}
	m.SetCargo(raw)
	return nil
}
