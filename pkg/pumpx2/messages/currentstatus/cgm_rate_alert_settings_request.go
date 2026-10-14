package currentstatus

import "github.com/jwoglom/faketandem/pkg/pumpx2/messages"

// CGMRateAlertSettingsRequest asks for the CGM rise and fall rate alert settings.
type CGMRateAlertSettingsRequest struct {
	messages.Base
}

var cgmRateAlertSettingsRequestProps = messages.Props{OpCode: 92, Type: messages.Request, Response: "CGMRateAlertSettingsResponse"}

func init() {
	messages.Register("CGMRateAlertSettingsRequest", &CGMRateAlertSettingsRequest{}, cgmRateAlertSettingsRequestProps,
		messages.Constructor{Func: NewCGMRateAlertSettingsRequest},
	)
}

// NewCGMRateAlertSettingsRequest is the request, which has no payload.
func NewCGMRateAlertSettingsRequest() *CGMRateAlertSettingsRequest {
	m := &CGMRateAlertSettingsRequest{}
	m.SetCargo([]byte{})
	return m
}

// Parse implements messages.Message.
func (m *CGMRateAlertSettingsRequest) Parse(raw []byte) error {
	if len(raw) != cgmRateAlertSettingsRequestProps.Size {
		return messages.InvalidLength("CGMRateAlertSettingsRequest", len(raw), cgmRateAlertSettingsRequestProps.Size)
	}
	m.SetCargo(raw)
	return nil
}
