package control

import (
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/helpers"
)

// CgmRiseFallAlertRequest's alert types.
const (
	CgmRiseFallAlertTypeRise = 0
	CgmRiseFallAlertTypeFall = 1
)

// CgmRiseFallAlertRequest sets the CGM rise or fall rate alert: whether it is
// on and MgPerDl, the rate in mg/dL per minute. Bitmask flags the fields that
// change: bit 0 the enable, bit 1 the rate.
type CgmRiseFallAlertRequest struct {
	messages.Base
	// AlertType is CgmRiseFallAlertTypeRise or CgmRiseFallAlertTypeFall.
	AlertType int32 `pumpx2:"alertType"`
	Enable    bool  `pumpx2:"enable"`
	MgPerDl   int32 `pumpx2:"mgPerDl"`
	Bitmask   int32 `pumpx2:"bitmask"`
}

var cgmRiseFallAlertRequestProps = messages.Props{OpCode: -60, Size: 4, Signed: true, Type: messages.Request, Characteristic: messages.Control,
	Response: "CgmRiseFallAlertResponse"}

func init() {
	messages.Register("CgmRiseFallAlertRequest", &CgmRiseFallAlertRequest{}, cgmRiseFallAlertRequestProps,
		messages.Constructor{Params: "int alertType, boolean enable, int mgPerDl, int bitmask", Func: NewCgmRiseFallAlertRequest},
	)
}

// NewCgmRiseFallAlertRequest builds the request.
func NewCgmRiseFallAlertRequest(alertType int32, enable bool, mgPerDl, bitmask int32) *CgmRiseFallAlertRequest {
	m := &CgmRiseFallAlertRequest{AlertType: alertType, Enable: enable, MgPerDl: mgPerDl, Bitmask: bitmask}
	m.SetCargo([]byte{byte(alertType), helpers.Bool(enable), byte(mgPerDl), byte(bitmask)})
	return m
}

// Parse implements messages.Message.
func (m *CgmRiseFallAlertRequest) Parse(raw []byte) error {
	raw = messages.RemoveSignedRequestHmacBytes(cgmRiseFallAlertRequestProps, raw)
	if len(raw) != cgmRiseFallAlertRequestProps.Size {
		return messages.InvalidLength("CgmRiseFallAlertRequest", len(raw), cgmRiseFallAlertRequestProps.Size)
	}
	m.SetCargo(raw)
	m.AlertType = int32(raw[0])
	m.Enable = raw[1] != 0
	m.MgPerDl = int32(raw[2])
	m.Bitmask = int32(raw[3])
	return nil
}
