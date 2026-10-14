package currentstatus

import "github.com/jwoglom/faketandem/pkg/pumpx2/messages"

// GlobalMaxBolusSettingsRequest asks for the pump's maximum bolus setting
// and its default.
type GlobalMaxBolusSettingsRequest struct {
	messages.Base
}

var globalMaxBolusSettingsRequestProps = messages.Props{OpCode: -116, Type: messages.Request, Response: "GlobalMaxBolusSettingsResponse"}

func init() {
	messages.Register("GlobalMaxBolusSettingsRequest", &GlobalMaxBolusSettingsRequest{}, globalMaxBolusSettingsRequestProps,
		messages.Constructor{Func: NewGlobalMaxBolusSettingsRequest},
	)
}

// NewGlobalMaxBolusSettingsRequest is the request, which has no payload.
func NewGlobalMaxBolusSettingsRequest() *GlobalMaxBolusSettingsRequest {
	m := &GlobalMaxBolusSettingsRequest{}
	m.SetCargo([]byte{})
	return m
}

// Parse implements messages.Message.
func (m *GlobalMaxBolusSettingsRequest) Parse(raw []byte) error {
	if len(raw) != globalMaxBolusSettingsRequestProps.Size {
		return messages.InvalidLength("GlobalMaxBolusSettingsRequest", len(raw), globalMaxBolusSettingsRequestProps.Size)
	}
	m.SetCargo(raw)
	return nil
}
