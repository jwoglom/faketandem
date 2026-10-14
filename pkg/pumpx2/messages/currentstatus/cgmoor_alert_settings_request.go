package currentstatus

import "github.com/jwoglom/faketandem/pkg/pumpx2/messages"

// CGMOORAlertSettingsRequest asks for the CGM out-of-range alert settings.
type CGMOORAlertSettingsRequest struct {
	messages.Base
}

var cgmoorAlertSettingsRequestProps = messages.Props{OpCode: 94, Type: messages.Request, Response: "CGMOORAlertSettingsResponse"}

func init() {
	messages.Register("CGMOORAlertSettingsRequest", &CGMOORAlertSettingsRequest{}, cgmoorAlertSettingsRequestProps,
		messages.Constructor{Func: NewCGMOORAlertSettingsRequest},
	)
}

// NewCGMOORAlertSettingsRequest is the request, which has no payload.
func NewCGMOORAlertSettingsRequest() *CGMOORAlertSettingsRequest {
	m := &CGMOORAlertSettingsRequest{}
	m.SetCargo([]byte{})
	return m
}

// Parse implements messages.Message.
func (m *CGMOORAlertSettingsRequest) Parse(raw []byte) error {
	if len(raw) != cgmoorAlertSettingsRequestProps.Size {
		return messages.InvalidLength("CGMOORAlertSettingsRequest", len(raw), cgmoorAlertSettingsRequestProps.Size)
	}
	m.SetCargo(raw)
	return nil
}
