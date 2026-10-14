package currentstatus

import (
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/helpers"
)

// BolusPermissionChangeReasonRequest asks why the bolus permission for a
// bolus ID last changed. The pumps pumpX2 was tried on answer it with an
// ErrorResponse.
type BolusPermissionChangeReasonRequest struct {
	messages.Base
	BolusID int32 `pumpx2:"bolusId"`
}

var bolusPermissionChangeReasonRequestProps = messages.Props{OpCode: -88, Size: 2, Type: messages.Request, MinAPI: "API_FUTURE", Response: "BolusPermissionChangeReasonResponse"}

func init() {
	messages.Register("BolusPermissionChangeReasonRequest", &BolusPermissionChangeReasonRequest{}, bolusPermissionChangeReasonRequestProps,
		messages.Constructor{Params: "int bolusId", Func: NewBolusPermissionChangeReasonRequest},
	)
}

// NewBolusPermissionChangeReasonRequest asks about bolusID.
func NewBolusPermissionChangeReasonRequest(bolusID int32) *BolusPermissionChangeReasonRequest {
	m := &BolusPermissionChangeReasonRequest{BolusID: bolusID}
	m.SetCargo(helpers.FirstTwoBytesLittleEndian(bolusID))
	return m
}

// Parse implements messages.Message.
func (m *BolusPermissionChangeReasonRequest) Parse(raw []byte) error {
	if len(raw) != bolusPermissionChangeReasonRequestProps.Size {
		return messages.InvalidLength("BolusPermissionChangeReasonRequest", len(raw), bolusPermissionChangeReasonRequestProps.Size)
	}
	m.SetCargo(raw)
	m.BolusID = helpers.ReadShort(raw, 0)
	return nil
}
