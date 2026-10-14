package currentstatus

import "github.com/jwoglom/faketandem/pkg/pumpx2/messages"

// BasalIQSettingsRequest asks for the Basal-IQ settings.
type BasalIQSettingsRequest struct {
	messages.Base
}

var basalIQSettingsRequestProps = messages.Props{OpCode: 98, Type: messages.Request, Response: "BasalIQSettingsResponse"}

func init() {
	messages.Register("BasalIQSettingsRequest", &BasalIQSettingsRequest{}, basalIQSettingsRequestProps,
		messages.Constructor{Func: NewBasalIQSettingsRequest},
	)
}

// NewBasalIQSettingsRequest is the request, which has no payload.
func NewBasalIQSettingsRequest() *BasalIQSettingsRequest {
	m := &BasalIQSettingsRequest{}
	m.SetCargo([]byte{})
	return m
}

// Parse implements messages.Message.
func (m *BasalIQSettingsRequest) Parse(raw []byte) error {
	if len(raw) != basalIQSettingsRequestProps.Size {
		return messages.InvalidLength("BasalIQSettingsRequest", len(raw), basalIQSettingsRequestProps.Size)
	}
	m.SetCargo(raw)
	return nil
}
