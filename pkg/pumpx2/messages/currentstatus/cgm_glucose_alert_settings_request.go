package currentstatus

import "github.com/jwoglom/faketandem/pkg/pumpx2/messages"

// CGMGlucoseAlertSettingsRequest asks for the CGM high and low glucose alert
// settings.
type CGMGlucoseAlertSettingsRequest struct {
	messages.Base
}

var cgmGlucoseAlertSettingsRequestProps = messages.Props{OpCode: 90, Type: messages.Request, Response: "CGMGlucoseAlertSettingsResponse"}

func init() {
	messages.Register("CGMGlucoseAlertSettingsRequest", &CGMGlucoseAlertSettingsRequest{}, cgmGlucoseAlertSettingsRequestProps,
		messages.Constructor{Func: NewCGMGlucoseAlertSettingsRequest},
	)
}

// NewCGMGlucoseAlertSettingsRequest is the request, which has no payload.
func NewCGMGlucoseAlertSettingsRequest() *CGMGlucoseAlertSettingsRequest {
	m := &CGMGlucoseAlertSettingsRequest{}
	m.SetCargo([]byte{})
	return m
}

// Parse implements messages.Message.
func (m *CGMGlucoseAlertSettingsRequest) Parse(raw []byte) error {
	if len(raw) != cgmGlucoseAlertSettingsRequestProps.Size {
		return messages.InvalidLength("CGMGlucoseAlertSettingsRequest", len(raw), cgmGlucoseAlertSettingsRequestProps.Size)
	}
	m.SetCargo(raw)
	return nil
}
