package control

import (
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/helpers"
)

// BolusPermissionReleaseRequest gives back the bolus permission a
// BolusPermissionRequest was granted under BolusID, without bolusing.
type BolusPermissionReleaseRequest struct {
	messages.Base
	BolusID int32 `pumpx2:"bolusID"`
	Reserve int32 `pumpx2:"reserve"`
}

var bolusPermissionReleaseRequestProps = messages.Props{OpCode: -16, Size: 4, Signed: true, Type: messages.Request, Characteristic: messages.Control,
	MinAPI: "API_V2_5", Response: "BolusPermissionReleaseResponse"}

func init() {
	messages.Register("BolusPermissionReleaseRequest", &BolusPermissionReleaseRequest{}, bolusPermissionReleaseRequestProps,
		messages.Constructor{Params: "int bolusID, int reserve", Func: NewBolusPermissionReleaseRequestWithReserve},
		messages.Constructor{Params: "int bolusID", Func: NewBolusPermissionReleaseRequest},
	)
}

// NewBolusPermissionReleaseRequest is the request with Reserve 0.
func NewBolusPermissionReleaseRequest(bolusID int32) *BolusPermissionReleaseRequest {
	return NewBolusPermissionReleaseRequestWithReserve(bolusID, 0)
}

// NewBolusPermissionReleaseRequestWithReserve builds the request.
func NewBolusPermissionReleaseRequestWithReserve(bolusID, reserve int32) *BolusPermissionReleaseRequest {
	m := &BolusPermissionReleaseRequest{BolusID: bolusID, Reserve: reserve}
	m.SetCargo(helpers.Combine(helpers.FirstTwoBytesLittleEndian(bolusID), helpers.FirstTwoBytesLittleEndian(reserve)))
	return m
}

// Parse implements messages.Message.
func (m *BolusPermissionReleaseRequest) Parse(raw []byte) error {
	raw = messages.RemoveSignedRequestHmacBytes(bolusPermissionReleaseRequestProps, raw)
	if len(raw) != bolusPermissionReleaseRequestProps.Size {
		return messages.InvalidLength("BolusPermissionReleaseRequest", len(raw), bolusPermissionReleaseRequestProps.Size)
	}
	m.SetCargo(raw)
	m.BolusID = helpers.ReadShort(raw, 0)
	m.Reserve = helpers.ReadShort(raw, 2)
	return nil
}
