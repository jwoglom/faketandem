package control

import (
	"fmt"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/helpers"
)

// CancelBolusResponse is the pump's answer to a CancelBolusRequest:
// StatusID is a CancelStatus and ReasonID a CancelReason. A bolus already
// delivered, or an ID the pump does not know, fails with
// INVALID_OR_ALREADY_DELIVERED.
type CancelBolusResponse struct {
	messages.Base
	StatusID int32 `pumpx2:"statusId"`
	BolusID  int32 `pumpx2:"bolusId"`
	ReasonID int32 `pumpx2:"reasonId"`
}

var cancelBolusResponseProps = messages.Props{OpCode: -95, Size: 5, Signed: true, Type: messages.Response, Characteristic: messages.Control,
	MinAPI: "API_V2_5", Request: "CancelBolusRequest"}

func init() {
	messages.Register("CancelBolusResponse", &CancelBolusResponse{}, cancelBolusResponseProps,
		messages.Constructor{Params: "int statusId, int bolusId, int reasonId", Func: NewCancelBolusResponse},
	)
}

// NewCancelBolusResponse builds the response.
func NewCancelBolusResponse(statusID, bolusID, reasonID int32) *CancelBolusResponse {
	m := &CancelBolusResponse{StatusID: statusID, BolusID: bolusID, ReasonID: reasonID}
	m.SetCargo(helpers.Combine([]byte{byte(statusID)}, helpers.FirstTwoBytesLittleEndian(bolusID), []byte{byte(reasonID), 0}))
	return m
}

// Parse implements messages.Message.
func (m *CancelBolusResponse) Parse(raw []byte) error {
	raw = messages.RemoveSignedRequestHmacBytes(cancelBolusResponseProps, raw)
	if len(raw) != cancelBolusResponseProps.Size {
		return messages.InvalidLength("CancelBolusResponse", len(raw), cancelBolusResponseProps.Size)
	}
	m.SetCargo(raw)
	m.StatusID = int32(int8(raw[0]))
	m.BolusID = helpers.ReadShort(raw, 1)
	m.ReasonID = int32(int8(raw[3]))
	return nil
}

// Status is StatusID as a CancelStatus, or nil for an ID that is none.
func (m *CancelBolusResponse) Status() *CancelStatus { return CancelStatusFromID(m.StatusID) }

// Reason is ReasonID as a CancelReason, or nil for an ID that is none.
func (m *CancelBolusResponse) Reason() *CancelReason { return CancelReasonFromID(m.ReasonID) }

// WasCancelled reports whether the pump stopped the bolus.
func (m *CancelBolusResponse) WasCancelled() bool {
	s, r := m.Status(), m.Reason()
	return s != nil && *s == CancelStatusSuccess && r != nil && *r == CancelReasonNoError
}

// CancelStatus is whether the pump stopped a bolus.
type CancelStatus int32

// The cancel statuses, by their wire IDs.
const (
	CancelStatusSuccess CancelStatus = 0
	CancelStatusFailed  CancelStatus = 1
)

var cancelStatusNames = []string{"SUCCESS", "FAILED"}

// CancelStatusFromID is the status with wire ID id, or nil for an ID that is
// none.
func CancelStatusFromID(id int32) *CancelStatus {
	if id < 0 || int(id) >= len(cancelStatusNames) {
		return nil
	}
	s := CancelStatus(id)
	return &s
}

func (s CancelStatus) String() string {
	if s >= 0 && int(s) < len(cancelStatusNames) {
		return cancelStatusNames[s]
	}
	return fmt.Sprintf("CancelStatus(%d)", int32(s))
}

// CancelReason is why the pump did not stop a bolus, or NO_ERROR.
type CancelReason int32

// The cancel reasons, by their wire IDs.
const (
	CancelReasonNoError                   CancelReason = 0
	CancelReasonInvalidOrAlreadyDelivered CancelReason = 2
)

// CancelReasonFromID is the reason with wire ID id, or nil for an ID that is
// none.
func CancelReasonFromID(id int32) *CancelReason {
	switch r := CancelReason(id); r {
	case CancelReasonNoError, CancelReasonInvalidOrAlreadyDelivered:
		return &r
	}
	return nil
}

func (r CancelReason) String() string {
	switch r {
	case CancelReasonNoError:
		return "NO_ERROR"
	case CancelReasonInvalidOrAlreadyDelivered:
		return "INVALID_OR_ALREADY_DELIVERED"
	}
	return fmt.Sprintf("CancelReason(%d)", int32(r))
}
