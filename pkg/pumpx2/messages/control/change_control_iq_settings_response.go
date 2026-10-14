package control

import "github.com/jwoglom/faketandem/pkg/pumpx2/messages"

// ChangeControlIQSettingsResponse is the pump's answer to a
// ChangeControlIQSettingsRequest: Status, the first of its three bytes, is 0
// on success.
type ChangeControlIQSettingsResponse struct {
	messages.Base
	Status int32 `pumpx2:"status"`
}

var changeControlIQSettingsResponseProps = messages.Props{OpCode: -53, Size: 3, Signed: true, Type: messages.Response, Characteristic: messages.Control,
	Request: "ChangeControlIQSettingsRequest"}

func init() {
	messages.Register("ChangeControlIQSettingsResponse", &ChangeControlIQSettingsResponse{}, changeControlIQSettingsResponseProps,
		messages.Constructor{Params: "byte[] raw", Func: NewChangeControlIQSettingsResponseFromCargo},
	)
}

// NewChangeControlIQSettingsResponseFromCargo parses a response's payload.
func NewChangeControlIQSettingsResponseFromCargo(raw []byte) (*ChangeControlIQSettingsResponse, error) {
	m := &ChangeControlIQSettingsResponse{}
	return m, m.Parse(raw)
}

// Parse implements messages.Message.
func (m *ChangeControlIQSettingsResponse) Parse(raw []byte) error {
	raw = messages.RemoveSignedRequestHmacBytes(changeControlIQSettingsResponseProps, raw)
	if len(raw) != changeControlIQSettingsResponseProps.Size {
		return messages.InvalidLength("ChangeControlIQSettingsResponse", len(raw), changeControlIQSettingsResponseProps.Size)
	}
	m.SetCargo(raw)
	m.Status = int32(int8(raw[0]))
	return nil
}
