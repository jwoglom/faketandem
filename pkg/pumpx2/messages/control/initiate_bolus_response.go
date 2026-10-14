package control

import (
	"fmt"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/helpers"
)

// InitiateBolusResponse is the pump's answer to an InitiateBolusRequest:
// Status 0 and StatusTypeID 0 (SUCCESS) mean the bolus is pending.
type InitiateBolusResponse struct {
	messages.Base
	Status       int32 `pumpx2:"status"`
	BolusID      int32 `pumpx2:"bolusId"`
	StatusTypeID int32 `pumpx2:"statusTypeId"`
}

var initiateBolusResponseProps = messages.Props{OpCode: -97, Size: 6, Signed: true, Type: messages.Response, Characteristic: messages.Control,
	ModifiesInsulinDelivery: true, Request: "InitiateBolusRequest"}

func init() {
	messages.Register("InitiateBolusResponse", &InitiateBolusResponse{}, initiateBolusResponseProps,
		messages.Constructor{Params: "int status, int bolusId, int statusTypeId", Func: NewInitiateBolusResponse},
	)
}

// NewInitiateBolusResponse builds the response.
func NewInitiateBolusResponse(status, bolusID, statusTypeID int32) *InitiateBolusResponse {
	m := &InitiateBolusResponse{Status: status, BolusID: bolusID, StatusTypeID: statusTypeID}
	m.SetCargo(helpers.Combine([]byte{byte(status)}, helpers.FirstTwoBytesLittleEndian(bolusID), []byte{0, 0, byte(statusTypeID)}))
	return m
}

// Parse implements messages.Message.
func (m *InitiateBolusResponse) Parse(raw []byte) error {
	raw = messages.RemoveSignedRequestHmacBytes(initiateBolusResponseProps, raw)
	if len(raw) != initiateBolusResponseProps.Size {
		return messages.InvalidLength("InitiateBolusResponse", len(raw), initiateBolusResponseProps.Size)
	}
	m.SetCargo(raw)
	m.Status = int32(int8(raw[0]))
	m.BolusID = helpers.ReadShort(raw, 1)
	m.StatusTypeID = int32(int8(raw[5]))
	return nil
}

// StatusType is StatusTypeID as a BolusResponseStatus, or nil for an ID that
// is none.
func (m *InitiateBolusResponse) StatusType() *BolusResponseStatus {
	return BolusResponseStatusFromID(m.StatusTypeID)
}

// WasBolusInitiated reports whether the pump took the bolus.
func (m *InitiateBolusResponse) WasBolusInitiated() bool {
	s := m.StatusType()
	return m.Status == 0 && s != nil && *s == BolusResponseStatusSuccess
}

// BolusResponseStatus is why the pump took or refused a bolus.
type BolusResponseStatus int32

// The bolus response statuses, by their wire IDs.
const (
	BolusResponseStatusSuccess         BolusResponseStatus = 0
	BolusResponseStatusRevokedPriority BolusResponseStatus = 2
)

// BolusResponseStatusFromID is the status with wire ID id, or nil for an ID
// that is none.
func BolusResponseStatusFromID(id int32) *BolusResponseStatus {
	switch s := BolusResponseStatus(id); s {
	case BolusResponseStatusSuccess, BolusResponseStatusRevokedPriority:
		return &s
	}
	return nil
}

func (s BolusResponseStatus) String() string {
	switch s {
	case BolusResponseStatusSuccess:
		return "SUCCESS"
	case BolusResponseStatusRevokedPriority:
		return "REVOKED_PRIORITY"
	}
	return fmt.Sprintf("BolusResponseStatus(%d)", int32(s))
}
