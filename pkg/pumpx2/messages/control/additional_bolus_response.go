package control

import (
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/helpers"
)

// AdditionalBolusResponse is the pump's answer to an AdditionalBolusRequest:
// Status 0 means it took the bolus.
type AdditionalBolusResponse struct {
	messages.Base
	Status  int32 `pumpx2:"status"`
	BolusID int32 `pumpx2:"bolusId"`
	Reserve int32 `pumpx2:"reserve"`
}

var additionalBolusResponseProps = messages.Props{OpCode: -5, Size: 5, Signed: true, Type: messages.Response, Characteristic: messages.Control,
	ModifiesInsulinDelivery: true, Request: "AdditionalBolusRequest"}

func init() {
	messages.Register("AdditionalBolusResponse", &AdditionalBolusResponse{}, additionalBolusResponseProps,
		messages.Constructor{Params: "int status, int bolusId, int reserve", Func: NewAdditionalBolusResponse},
	)
}

// NewAdditionalBolusResponse builds the response.
func NewAdditionalBolusResponse(status, bolusID, reserve int32) *AdditionalBolusResponse {
	m := &AdditionalBolusResponse{Status: status, BolusID: bolusID, Reserve: reserve}
	m.SetCargo(helpers.Combine([]byte{byte(status)}, helpers.FirstTwoBytesLittleEndian(bolusID), helpers.FirstTwoBytesLittleEndian(reserve)))
	return m
}

// Parse implements messages.Message.
func (m *AdditionalBolusResponse) Parse(raw []byte) error {
	raw = messages.RemoveSignedRequestHmacBytes(additionalBolusResponseProps, raw)
	if len(raw) != additionalBolusResponseProps.Size {
		return messages.InvalidLength("AdditionalBolusResponse", len(raw), additionalBolusResponseProps.Size)
	}
	m.SetCargo(raw)
	m.Status = int32(int8(raw[0]))
	m.BolusID = helpers.ReadShort(raw, 1)
	m.Reserve = helpers.ReadShort(raw, 3)
	return nil
}

// WasAccepted reports whether the pump took the additional bolus.
func (m *AdditionalBolusResponse) WasAccepted() bool { return m.Status == 0 }
