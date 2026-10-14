package currentstatus

import (
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/helpers"
)

// InsulinStatusResponse is the insulin left in the cartridge, in whole
// units, whether that amount is an estimate (IsEstimate 1), and the amount,
// in units, below which the pump warns of low insulin.
type InsulinStatusResponse struct {
	messages.Base
	CurrentInsulinAmount int32 `pumpx2:"currentInsulinAmount"`
	IsEstimate           int32 `pumpx2:"isEstimate"`
	InsulinLowAmount     int32 `pumpx2:"insulinLowAmount"`
}

var insulinStatusResponseProps = messages.Props{OpCode: 37, Size: 4, Type: messages.Response, Request: "InsulinStatusRequest"}

func init() {
	messages.Register("InsulinStatusResponse", &InsulinStatusResponse{}, insulinStatusResponseProps,
		messages.Constructor{Params: "int currentInsulinAmount, int isEstimate, int insulinLowAmount", Func: NewInsulinStatusResponse},
	)
}

// NewInsulinStatusResponse builds the response.
func NewInsulinStatusResponse(currentInsulinAmount, isEstimate, insulinLowAmount int32) *InsulinStatusResponse {
	m := &InsulinStatusResponse{CurrentInsulinAmount: currentInsulinAmount, IsEstimate: isEstimate, InsulinLowAmount: insulinLowAmount}
	m.SetCargo(helpers.Combine(helpers.FirstTwoBytesLittleEndian(currentInsulinAmount), []byte{byte(isEstimate), byte(insulinLowAmount)}))
	return m
}

// Parse implements messages.Message.
func (m *InsulinStatusResponse) Parse(raw []byte) error {
	if len(raw) != insulinStatusResponseProps.Size {
		return messages.InvalidLength("InsulinStatusResponse", len(raw), insulinStatusResponseProps.Size)
	}
	m.SetCargo(raw)
	m.CurrentInsulinAmount = helpers.ReadShort(raw, 0)
	m.IsEstimate = int32(int8(raw[2]))
	m.InsulinLowAmount = int32(int8(raw[3]))
	return nil
}
