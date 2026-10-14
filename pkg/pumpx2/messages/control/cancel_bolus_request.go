package control

import (
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/helpers"
)

// CancelBolusRequest stops a bolus, extended ones included, whether the app
// or the pump started it. pumpX2 also calls it BolusTerminationRequest.
type CancelBolusRequest struct {
	messages.Base
	BolusID int32 `pumpx2:"bolusId"`
}

var cancelBolusRequestProps = messages.Props{OpCode: -96, Size: 4, Signed: true, Type: messages.Request, Characteristic: messages.Control,
	MinAPI: "API_V2_5", Response: "CancelBolusResponse"}

func init() {
	messages.Register("CancelBolusRequest", &CancelBolusRequest{}, cancelBolusRequestProps,
		messages.Constructor{Params: "int bolusId", Func: NewCancelBolusRequest},
	)
}

// NewCancelBolusRequest builds the request.
func NewCancelBolusRequest(bolusID int32) *CancelBolusRequest {
	m := &CancelBolusRequest{BolusID: bolusID}
	m.SetCargo(helpers.Combine(helpers.FirstTwoBytesLittleEndian(bolusID), []byte{0, 0}))
	return m
}

// Parse implements messages.Message.
func (m *CancelBolusRequest) Parse(raw []byte) error {
	raw = messages.RemoveSignedRequestHmacBytes(cancelBolusRequestProps, raw)
	if len(raw) != cancelBolusRequestProps.Size {
		return messages.InvalidLength("CancelBolusRequest", len(raw), cancelBolusRequestProps.Size)
	}
	m.SetCargo(raw)
	m.BolusID = helpers.ReadShort(raw, 0)
	return nil
}
