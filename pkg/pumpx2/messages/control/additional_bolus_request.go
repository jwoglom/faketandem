package control

import (
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/helpers"
)

// AdditionalBolusRequest asks for an additional bolus under a bolus ID that a
// BolusPermissionRequest granted.
type AdditionalBolusRequest struct {
	messages.Base
	BolusID int32 `pumpx2:"bolusID"`
	Reserve int32 `pumpx2:"reserve"`
}

var additionalBolusRequestProps = messages.Props{OpCode: -6, Size: 4, Signed: true, Type: messages.Request, Characteristic: messages.Control,
	ModifiesInsulinDelivery: true, Response: "AdditionalBolusResponse"}

func init() {
	messages.Register("AdditionalBolusRequest", &AdditionalBolusRequest{}, additionalBolusRequestProps,
		messages.Constructor{Params: "int bolusID, int reserve", Func: NewAdditionalBolusRequestWithReserve},
		messages.Constructor{Params: "int bolusID", Func: NewAdditionalBolusRequest},
	)
}

// NewAdditionalBolusRequest is the request with Reserve 0.
func NewAdditionalBolusRequest(bolusID int32) *AdditionalBolusRequest {
	return NewAdditionalBolusRequestWithReserve(bolusID, 0)
}

// NewAdditionalBolusRequestWithReserve builds the request.
func NewAdditionalBolusRequestWithReserve(bolusID, reserve int32) *AdditionalBolusRequest {
	m := &AdditionalBolusRequest{BolusID: bolusID, Reserve: reserve}
	m.SetCargo(helpers.Combine(helpers.FirstTwoBytesLittleEndian(bolusID), helpers.FirstTwoBytesLittleEndian(reserve)))
	return m
}

// Parse implements messages.Message.
func (m *AdditionalBolusRequest) Parse(raw []byte) error {
	raw = messages.RemoveSignedRequestHmacBytes(additionalBolusRequestProps, raw)
	if len(raw) != additionalBolusRequestProps.Size {
		return messages.InvalidLength("AdditionalBolusRequest", len(raw), additionalBolusRequestProps.Size)
	}
	m.SetCargo(raw)
	m.BolusID = helpers.ReadShort(raw, 0)
	m.Reserve = helpers.ReadShort(raw, 2)
	return nil
}
