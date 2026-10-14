package control

import "github.com/jwoglom/faketandem/pkg/pumpx2/messages"

// SetRuntimeConfigRequest sets a t:slim X2 runtime configuration value, two
// bytes whose meaning is unknown, as part of the FreeStyle Libre 3 setup.
// The Tandem app only ever sends 0, 0, and nothing reads the value back.
type SetRuntimeConfigRequest struct {
	messages.Base
	Config0 int32 `pumpx2:"config0"`
	Config1 int32 `pumpx2:"config1"`
}

var setRuntimeConfigRequestProps = messages.Props{OpCode: 122, Size: 2, Signed: true, Type: messages.Request, Characteristic: messages.Control,
	Response: "SetRuntimeConfigResponse"}

func init() {
	messages.Register("SetRuntimeConfigRequest", &SetRuntimeConfigRequest{}, setRuntimeConfigRequestProps,
		messages.Constructor{Params: "int config0, int config1", Func: NewSetRuntimeConfigRequest},
	)
}

// NewSetRuntimeConfigRequest builds the request.
func NewSetRuntimeConfigRequest(config0, config1 int32) *SetRuntimeConfigRequest {
	m := &SetRuntimeConfigRequest{Config0: config0, Config1: config1}
	m.SetCargo([]byte{byte(config0), byte(config1)})
	return m
}

// Parse implements messages.Message.
func (m *SetRuntimeConfigRequest) Parse(raw []byte) error {
	raw = messages.RemoveSignedRequestHmacBytes(setRuntimeConfigRequestProps, raw)
	if len(raw) != setRuntimeConfigRequestProps.Size {
		return messages.InvalidLength("SetRuntimeConfigRequest", len(raw), setRuntimeConfigRequestProps.Size)
	}
	m.SetCargo(raw)
	m.Config0 = int32(raw[0])
	m.Config1 = int32(raw[1])
	return nil
}
