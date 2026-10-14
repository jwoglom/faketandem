package control

import (
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/helpers"
)

// CgmOutOfRangeAlertRequest sets the CGM out-of-range alert: whether it is on
// and AlertDelay, the minutes out of range before it sounds. Bitmask flags
// the fields that change: bit 0 the enable, bit 1 the delay.
type CgmOutOfRangeAlertRequest struct {
	messages.Base
	Enable     bool  `pumpx2:"enable"`
	AlertDelay int32 `pumpx2:"alertDelay"`
	Bitmask    int32 `pumpx2:"bitmask"`
}

var cgmOutOfRangeAlertRequestProps = messages.Props{OpCode: -58, Size: 3, Signed: true, Type: messages.Request, Characteristic: messages.Control,
	Response: "CgmOutOfRangeAlertResponse"}

func init() {
	messages.Register("CgmOutOfRangeAlertRequest", &CgmOutOfRangeAlertRequest{}, cgmOutOfRangeAlertRequestProps,
		messages.Constructor{Params: "boolean enable, int alertDelay, int bitmask", Func: NewCgmOutOfRangeAlertRequest},
	)
}

// NewCgmOutOfRangeAlertRequest builds the request.
func NewCgmOutOfRangeAlertRequest(enable bool, alertDelay, bitmask int32) *CgmOutOfRangeAlertRequest {
	m := &CgmOutOfRangeAlertRequest{Enable: enable, AlertDelay: alertDelay, Bitmask: bitmask}
	m.SetCargo([]byte{helpers.Bool(enable), byte(alertDelay), byte(bitmask)})
	return m
}

// Parse implements messages.Message.
func (m *CgmOutOfRangeAlertRequest) Parse(raw []byte) error {
	raw = messages.RemoveSignedRequestHmacBytes(cgmOutOfRangeAlertRequestProps, raw)
	if len(raw) != cgmOutOfRangeAlertRequestProps.Size {
		return messages.InvalidLength("CgmOutOfRangeAlertRequest", len(raw), cgmOutOfRangeAlertRequestProps.Size)
	}
	m.SetCargo(raw)
	m.Enable = raw[0] != 0
	m.AlertDelay = int32(raw[1])
	m.Bitmask = int32(raw[2])
	return nil
}
