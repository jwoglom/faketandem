package control

import (
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/helpers"
)

// CgmHighLowAlertRequest's alert types.
const (
	CgmHighLowAlertTypeHigh = 0
	CgmHighLowAlertTypeLow  = 1
)

// CgmHighLowAlertRequest sets the CGM high or low alert: Threshold in mg/dL,
// how often it repeats and whether it is on. Bitmask flags the fields that
// change: bit 0 the enable, bit 1 the threshold, bit 2 the repeat.
type CgmHighLowAlertRequest struct {
	messages.Base
	Threshold             int32 `pumpx2:"threshold"`
	RepeatDurationMinutes int32 `pumpx2:"repeatDurationMinutes"`
	// AlertType is CgmHighLowAlertTypeHigh or CgmHighLowAlertTypeLow.
	AlertType   int32 `pumpx2:"alertType"`
	EnableAlert bool  `pumpx2:"enableAlert"`
	Bitmask     int32 `pumpx2:"bitmask"`
}

var cgmHighLowAlertRequestProps = messages.Props{OpCode: -62, Size: 7, Signed: true, Type: messages.Request, Characteristic: messages.Control,
	Response: "CgmHighLowAlertResponse"}

func init() {
	messages.Register("CgmHighLowAlertRequest", &CgmHighLowAlertRequest{}, cgmHighLowAlertRequestProps,
		messages.Constructor{Params: "int alertType, int threshold, int repeatDurationMinutes, boolean enableAlert, int bitmask", Func: NewCgmHighLowAlertRequest},
	)
}

// NewCgmHighLowAlertRequest builds the request.
func NewCgmHighLowAlertRequest(alertType, threshold, repeatDurationMinutes int32, enableAlert bool, bitmask int32) *CgmHighLowAlertRequest {
	m := &CgmHighLowAlertRequest{
		Threshold:             threshold,
		RepeatDurationMinutes: repeatDurationMinutes,
		AlertType:             alertType,
		EnableAlert:           enableAlert,
		Bitmask:               bitmask,
	}
	m.SetCargo(helpers.Combine(
		helpers.FirstTwoBytesLittleEndian(threshold),
		helpers.FirstTwoBytesLittleEndian(repeatDurationMinutes),
		[]byte{byte(alertType), helpers.Bool(enableAlert), byte(bitmask)},
	))
	return m
}

// Parse implements messages.Message.
func (m *CgmHighLowAlertRequest) Parse(raw []byte) error {
	raw = messages.RemoveSignedRequestHmacBytes(cgmHighLowAlertRequestProps, raw)
	if len(raw) != cgmHighLowAlertRequestProps.Size {
		return messages.InvalidLength("CgmHighLowAlertRequest", len(raw), cgmHighLowAlertRequestProps.Size)
	}
	m.SetCargo(raw)
	m.Threshold = helpers.ReadShort(raw, 0)
	m.RepeatDurationMinutes = helpers.ReadShort(raw, 2)
	m.AlertType = int32(raw[4])
	m.EnableAlert = raw[5] != 0
	m.Bitmask = int32(raw[6])
	return nil
}
