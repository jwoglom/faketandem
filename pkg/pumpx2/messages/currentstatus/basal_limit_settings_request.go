package currentstatus

import "github.com/jwoglom/faketandem/pkg/pumpx2/messages"

// BasalLimitSettingsRequest asks for the pump's maximum basal rate setting.
type BasalLimitSettingsRequest struct {
	messages.Base
}

var basalLimitSettingsRequestProps = messages.Props{OpCode: -118, Type: messages.Request, Response: "BasalLimitSettingsResponse"}

func init() {
	messages.Register("BasalLimitSettingsRequest", &BasalLimitSettingsRequest{}, basalLimitSettingsRequestProps,
		messages.Constructor{Func: NewBasalLimitSettingsRequest},
	)
}

// NewBasalLimitSettingsRequest is the request, which has no payload.
func NewBasalLimitSettingsRequest() *BasalLimitSettingsRequest {
	m := &BasalLimitSettingsRequest{}
	m.SetCargo([]byte{})
	return m
}

// Parse implements messages.Message.
func (m *BasalLimitSettingsRequest) Parse(raw []byte) error {
	if len(raw) != basalLimitSettingsRequestProps.Size {
		return messages.InvalidLength("BasalLimitSettingsRequest", len(raw), basalLimitSettingsRequestProps.Size)
	}
	m.SetCargo(raw)
	return nil
}
