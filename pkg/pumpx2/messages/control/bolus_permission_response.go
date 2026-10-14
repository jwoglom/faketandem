package control

import (
	"fmt"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/helpers"
)

// BolusPermissionResponse is the pump's answer to a BolusPermissionRequest:
// with Status 0 and NackReasonID 0 (PERMISSION_GRANTED) the app may bolus
// under BolusID; otherwise NackReasonID, a NackReason, says why not.
type BolusPermissionResponse struct {
	messages.Base
	Status       int32 `pumpx2:"status"`
	BolusID      int32 `pumpx2:"bolusId"`
	NackReasonID int32 `pumpx2:"nackReasonId"`
}

var bolusPermissionResponseProps = messages.Props{OpCode: -93, Size: 6, Signed: true, Type: messages.Response, Characteristic: messages.Control,
	MinAPI: "API_V2_5", Request: "BolusPermissionRequest"}

func init() {
	messages.Register("BolusPermissionResponse", &BolusPermissionResponse{}, bolusPermissionResponseProps,
		messages.Constructor{Params: "int status, int bolusId, int nackReasonId", Func: NewBolusPermissionResponse},
	)
}

// NewBolusPermissionResponse builds the response.
func NewBolusPermissionResponse(status, bolusID, nackReasonID int32) *BolusPermissionResponse {
	m := &BolusPermissionResponse{Status: status, BolusID: bolusID, NackReasonID: nackReasonID}
	m.SetCargo(helpers.Combine([]byte{byte(status)}, helpers.FirstTwoBytesLittleEndian(bolusID), []byte{0, 0, byte(nackReasonID)}))
	return m
}

// Parse implements messages.Message.
func (m *BolusPermissionResponse) Parse(raw []byte) error {
	raw = messages.RemoveSignedRequestHmacBytes(bolusPermissionResponseProps, raw)
	if len(raw) != bolusPermissionResponseProps.Size {
		return messages.InvalidLength("BolusPermissionResponse", len(raw), bolusPermissionResponseProps.Size)
	}
	m.SetCargo(raw)
	m.Status = int32(int8(raw[0]))
	m.BolusID = helpers.ReadShort(raw, 1)
	m.NackReasonID = int32(int8(raw[5]))
	return nil
}

// NackReason is NackReasonID as a NackReason, or nil for an ID that is none.
func (m *BolusPermissionResponse) NackReason() *NackReason {
	return NackReasonFromID(m.NackReasonID)
}

// IsPermissionGranted reports whether the app may bolus under BolusID.
func (m *BolusPermissionResponse) IsPermissionGranted() bool {
	r := m.NackReason()
	return m.Status == 0 && r != nil && *r == NackReasonPermissionGranted
}

// NackReason is why the pump refused bolus permission, or
// PERMISSION_GRANTED.
type NackReason int32

// The nack reasons, by their wire IDs.
const (
	NackReasonPermissionGranted          NackReason = 0
	NackReasonInvalidPumpingState        NackReason = 1
	NackReasonPumpHasPermission          NackReason = 3
	NackReasonMPPStateWaitingForResponse NackReason = -1
	NackReasonMPPStateUnknownNackReason  NackReason = -3
)

// NackReasonFromID is the reason with wire ID id, or nil for an ID that is
// none.
func NackReasonFromID(id int32) *NackReason {
	switch r := NackReason(id); r {
	case NackReasonPermissionGranted, NackReasonInvalidPumpingState, NackReasonPumpHasPermission,
		NackReasonMPPStateWaitingForResponse, NackReasonMPPStateUnknownNackReason:
		return &r
	}
	return nil
}

func (r NackReason) String() string {
	switch r {
	case NackReasonPermissionGranted:
		return "PERMISSION_GRANTED"
	case NackReasonInvalidPumpingState:
		return "INVALID_PUMPING_STATE"
	case NackReasonPumpHasPermission:
		return "PUMP_HAS_PERMISSION"
	case NackReasonMPPStateWaitingForResponse:
		return "MPP_STATE_WAITING_FOR_RESPONSE"
	case NackReasonMPPStateUnknownNackReason:
		return "MPP_STATE_UNKNOWN_NACK_REASON"
	}
	return fmt.Sprintf("NackReason(%d)", int32(r))
}
