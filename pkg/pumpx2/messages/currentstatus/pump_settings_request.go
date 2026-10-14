package currentstatus

import "github.com/jwoglom/faketandem/pkg/pumpx2/messages"

// PumpSettingsRequest asks for the pump's low insulin, cannula fill, auto-off,
// feature lock and screen timeout settings.
type PumpSettingsRequest struct {
	messages.Base
}

var pumpSettingsRequestProps = messages.Props{OpCode: 82, Type: messages.Request, Response: "PumpSettingsResponse"}

func init() {
	messages.Register("PumpSettingsRequest", &PumpSettingsRequest{}, pumpSettingsRequestProps)
}

// NewPumpSettingsRequest is the request, which has no payload.
func NewPumpSettingsRequest() *PumpSettingsRequest {
	m := &PumpSettingsRequest{}
	m.SetCargo([]byte{})
	return m
}

// Parse implements messages.Message.
func (m *PumpSettingsRequest) Parse(raw []byte) error {
	if len(raw) != pumpSettingsRequestProps.Size {
		return messages.InvalidLength("PumpSettingsRequest", len(raw), pumpSettingsRequestProps.Size)
	}
	m.SetCargo(raw)
	return nil
}
