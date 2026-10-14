package control

import (
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/helpers"
)

// SetAutoOffAlertRequest sets the Auto-Off alert. AutoOffDuration is in
// minutes. Bitmask says which settings change: bit 0 EnableAutoOff, bit 1
// AutoOffDuration.
type SetAutoOffAlertRequest struct {
	messages.Base
	EnableAutoOff   bool  `pumpx2:"enableAutoOff"`
	AutoOffDuration int32 `pumpx2:"autoOffDuration"`
	Bitmask         int32 `pumpx2:"bitmask"`
}

var setAutoOffAlertRequestProps = messages.Props{OpCode: -32, Size: 4, Signed: true, Type: messages.Request, Characteristic: messages.Control,
	Response: "SetAutoOffAlertResponse"}

func init() {
	messages.Register("SetAutoOffAlertRequest", &SetAutoOffAlertRequest{}, setAutoOffAlertRequestProps,
		messages.Constructor{Params: "boolean enableAutoOff, int autoOffDuration, int bitmask", Func: NewSetAutoOffAlertRequest},
	)
}

// NewSetAutoOffAlertRequest builds the request.
func NewSetAutoOffAlertRequest(enableAutoOff bool, autoOffDuration, bitmask int32) *SetAutoOffAlertRequest {
	m := &SetAutoOffAlertRequest{EnableAutoOff: enableAutoOff, AutoOffDuration: autoOffDuration, Bitmask: bitmask}
	m.SetCargo(helpers.Combine([]byte{helpers.Bool(enableAutoOff)}, helpers.FirstTwoBytesLittleEndian(autoOffDuration), []byte{byte(bitmask)}))
	return m
}

// Parse implements messages.Message.
func (m *SetAutoOffAlertRequest) Parse(raw []byte) error {
	raw = messages.RemoveSignedRequestHmacBytes(setAutoOffAlertRequestProps, raw)
	if len(raw) != setAutoOffAlertRequestProps.Size {
		return messages.InvalidLength("SetAutoOffAlertRequest", len(raw), setAutoOffAlertRequestProps.Size)
	}
	m.SetCargo(raw)
	m.EnableAutoOff = raw[0] != 0
	m.AutoOffDuration = helpers.ReadShort(raw, 1)
	m.Bitmask = int32(raw[3])
	return nil
}
