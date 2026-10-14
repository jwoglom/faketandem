package control

import (
	"fmt"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
)

// SetSensorTypeResponse is the pump's answer to a SetSensorTypeRequest:
// Status 0 is ACK and 1 NACK, and StatusAcknowledgement, a
// StatusAcknowledgement code, says why.
type SetSensorTypeResponse struct {
	messages.Base
	Status                int32 `pumpx2:"status"`
	StatusAcknowledgement int32 `pumpx2:"statusAcknowledgement"`
}

var setSensorTypeResponseProps = messages.Props{OpCode: -63, Size: 2, Signed: true, Type: messages.Response, Characteristic: messages.Control,
	Request: "SetSensorTypeRequest"}

func init() {
	messages.Register("SetSensorTypeResponse", &SetSensorTypeResponse{}, setSensorTypeResponseProps,
		messages.Constructor{Params: "int status, int statusAcknowledgement", Func: NewSetSensorTypeResponse},
	)
}

// NewSetSensorTypeResponse builds the response.
func NewSetSensorTypeResponse(status, statusAcknowledgement int32) *SetSensorTypeResponse {
	m := &SetSensorTypeResponse{Status: status, StatusAcknowledgement: statusAcknowledgement}
	m.SetCargo([]byte{byte(status), byte(statusAcknowledgement)})
	return m
}

// Parse implements messages.Message.
func (m *SetSensorTypeResponse) Parse(raw []byte) error {
	raw = messages.RemoveSignedRequestHmacBytes(setSensorTypeResponseProps, raw)
	if len(raw) != setSensorTypeResponseProps.Size {
		return messages.InvalidLength("SetSensorTypeResponse", len(raw), setSensorTypeResponseProps.Size)
	}
	m.SetCargo(raw)
	m.Status = int32(int8(raw[0]))
	m.StatusAcknowledgement = int32(raw[1])
	return nil
}

// StatusAcknowledgementType is StatusAcknowledgement as a code.
func (m *SetSensorTypeResponse) StatusAcknowledgementType() StatusAcknowledgement {
	return StatusAcknowledgementFromCode(m.StatusAcknowledgement)
}

// StatusAcknowledgement is the Tandem app's reason for a sensor type
// change's ACK or NACK.
type StatusAcknowledgement int32

// The acknowledgement codes.
const (
	StatusAcknowledgementSuccess StatusAcknowledgement = iota
	StatusAcknowledgementCgmSessionAlreadyActive
	StatusAcknowledgementInvalidCgmType
	StatusAcknowledgementMalfunctionActive
	StatusAcknowledgementMissingOrInvalidCgmSupportPackage
	StatusAcknowledgementInvalidScreenState
	StatusAcknowledgementUnknown
)

var statusAcknowledgementNames = []string{
	"SUCCESS", "CGM_SESSION_ALREADY_ACTIVE", "INVALID_CGM_TYPE", "MALFUNCTION_ACTIVE",
	"MISSING_OR_INVALID_CGM_SUPPORT_PACKAGE", "INVALID_SCREEN_STATE", "UNKNOWN",
}

// StatusAcknowledgementFromCode is the acknowledgement with that code, or
// Unknown.
func StatusAcknowledgementFromCode(code int32) StatusAcknowledgement {
	if code >= 0 && int(code) < len(statusAcknowledgementNames) {
		return StatusAcknowledgement(code)
	}
	return StatusAcknowledgementUnknown
}

// Code is the acknowledgement's wire code.
func (s StatusAcknowledgement) Code() int32 { return int32(s) }

func (s StatusAcknowledgement) String() string {
	if s >= 0 && int(s) < len(statusAcknowledgementNames) {
		return statusAcknowledgementNames[s]
	}
	return fmt.Sprintf("StatusAcknowledgement(%d)", int32(s))
}
