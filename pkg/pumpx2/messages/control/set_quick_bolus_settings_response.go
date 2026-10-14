package control

import "github.com/jwoglom/faketandem/pkg/pumpx2/messages"

// SetQuickBolusSettingsResponse is the pump's answer to a
// SetQuickBolusSettingsRequest: Status 0 is success.
type SetQuickBolusSettingsResponse struct {
	messages.Base
	Status int32 `pumpx2:"status"`
}

var setQuickBolusSettingsResponseProps = messages.Props{OpCode: -45, Size: 1, Signed: true, Type: messages.Response, Characteristic: messages.Control,
	Request: "SetQuickBolusSettingsRequest"}

func init() {
	messages.Register("SetQuickBolusSettingsResponse", &SetQuickBolusSettingsResponse{}, setQuickBolusSettingsResponseProps,
		messages.Constructor{Params: "byte[] raw", Func: NewSetQuickBolusSettingsResponseFromCargo},
	)
}

// NewSetQuickBolusSettingsResponseFromCargo parses a response's payload,
// which is how pumpX2 builds one.
func NewSetQuickBolusSettingsResponseFromCargo(raw []byte) (*SetQuickBolusSettingsResponse, error) {
	m := &SetQuickBolusSettingsResponse{}
	return m, m.Parse(raw)
}

// Parse implements messages.Message.
func (m *SetQuickBolusSettingsResponse) Parse(raw []byte) error {
	raw = messages.RemoveSignedRequestHmacBytes(setQuickBolusSettingsResponseProps, raw)
	if len(raw) != setQuickBolusSettingsResponseProps.Size {
		return messages.InvalidLength("SetQuickBolusSettingsResponse", len(raw), setQuickBolusSettingsResponseProps.Size)
	}
	m.SetCargo(raw)
	m.Status = int32(int8(raw[0]))
	return nil
}
