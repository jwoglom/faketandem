package control

import (
	"fmt"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
)

// StreamDataPreflightResponse is the pump's answer to a
// StreamDataPreflightRequest: Status 0 is ACK, StatusTypeID a
// PreflightStatusType ID and StreamTypeID a PreflightStreamType ID.
type StreamDataPreflightResponse struct {
	messages.Base
	Status       int32 `pumpx2:"status"`
	StatusTypeID int32 `pumpx2:"statusTypeId"`
	StreamTypeID int32 `pumpx2:"streamTypeId"`
}

var streamDataPreflightResponseProps = messages.Props{OpCode: -125, Size: 3, Signed: true, Type: messages.Response, Characteristic: messages.Control,
	MinAPI: "API_FUTURE", Request: "StreamDataPreflightRequest"}

func init() {
	messages.Register("StreamDataPreflightResponse", &StreamDataPreflightResponse{}, streamDataPreflightResponseProps,
		messages.Constructor{Params: "int status, int statusTypeId, int streamTypeId", Func: NewStreamDataPreflightResponse},
	)
}

// NewStreamDataPreflightResponse builds the response.
func NewStreamDataPreflightResponse(status, statusTypeID, streamTypeID int32) *StreamDataPreflightResponse {
	m := &StreamDataPreflightResponse{Status: status, StatusTypeID: statusTypeID, StreamTypeID: streamTypeID}
	m.SetCargo([]byte{byte(status), byte(statusTypeID), byte(streamTypeID)})
	return m
}

// Parse implements messages.Message.
func (m *StreamDataPreflightResponse) Parse(raw []byte) error {
	raw = messages.RemoveSignedRequestHmacBytes(streamDataPreflightResponseProps, raw)
	if len(raw) != streamDataPreflightResponseProps.Size {
		return messages.InvalidLength("StreamDataPreflightResponse", len(raw), streamDataPreflightResponseProps.Size)
	}
	m.SetCargo(raw)
	m.Status = int32(int8(raw[0]))
	m.StatusTypeID = int32(raw[1])
	m.StreamTypeID = int32(raw[2])
	return nil
}

// StatusType is StatusTypeID as a status, Unknown for an ID that is none.
func (m *StreamDataPreflightResponse) StatusType() PreflightStatusType {
	return PreflightStatusTypeFromID(m.StatusTypeID)
}

// StreamType is StreamTypeID as a stream type, Unknown for an ID that is
// none.
func (m *StreamDataPreflightResponse) StreamType() PreflightStreamType {
	return PreflightStreamTypeFromID(m.StreamTypeID)
}

// PreflightStatusType is why the pump took or refused a CGM data stream.
type PreflightStatusType int32

// The preflight statuses, by their wire IDs.
const (
	PreflightStatusTypeSuccess PreflightStatusType = iota
	PreflightStatusTypeSensorContextTooBig
	PreflightStatusTypeAnotherCgmActive
	PreflightStatusTypeIncompatibleSensor
	PreflightStatusTypeInvalidSensorContext
	PreflightStatusTypeInvalidPumpTime
	PreflightStatusTypePumpIsBusy
	PreflightStatusTypeSensorHasAlreadyBeenUsed
	PreflightStatusTypeInternalError
	PreflightStatusTypeEngineeringUseOnly
	PreflightStatusTypeTimeout
	PreflightStatusTypeSensorUnavailable
	PreflightStatusTypeInvalidSensorTimeParams
	PreflightStatusTypeUnknown
)

var preflightStatusTypeNames = []string{
	"SUCCESS", "SENSOR_CONTEXT_TOO_BIG", "ANOTHER_CGM_ACTIVE", "INCOMPATIBLE_SENSOR", "INVALID_SENSOR_CONTEXT",
	"INVALID_PUMP_TIME", "PUMP_IS_BUSY", "SENSOR_HAS_ALREADY_BEEN_USED", "INTERNAL_ERROR", "ENGINEERING_USE_ONLY",
	"TIMEOUT", "SENSOR_UNAVAILABLE", "INVALID_SENSOR_TIME_PARAMS", "UNKNOWN",
}

// PreflightStatusTypeFromID is the status with wire ID id, or Unknown.
func PreflightStatusTypeFromID(id int32) PreflightStatusType {
	if id >= 0 && int(id) < len(preflightStatusTypeNames) {
		return PreflightStatusType(id)
	}
	return PreflightStatusTypeUnknown
}

// ID is the status's wire ID.
func (s PreflightStatusType) ID() int32 { return int32(s) }

func (s PreflightStatusType) String() string {
	if s >= 0 && int(s) < len(preflightStatusTypeNames) {
		return preflightStatusTypeNames[s]
	}
	return fmt.Sprintf("PreflightStatusType(%d)", int32(s))
}

// PreflightStreamType is the kind of CGM data stream.
type PreflightStreamType int32

// The stream types, by their wire IDs.
const (
	PreflightStreamTypeFreestyleLibre2 PreflightStreamType = iota
	PreflightStreamTypeUnknown
)

var preflightStreamTypeNames = []string{"FREESTYLE_LIBRE_2", "UNKNOWN"}

// PreflightStreamTypeFromID is the stream type with wire ID id, or Unknown.
func PreflightStreamTypeFromID(id int32) PreflightStreamType {
	if id >= 0 && int(id) < len(preflightStreamTypeNames) {
		return PreflightStreamType(id)
	}
	return PreflightStreamTypeUnknown
}

// ID is the stream type's wire ID.
func (s PreflightStreamType) ID() int32 { return int32(s) }

func (s PreflightStreamType) String() string {
	if s >= 0 && int(s) < len(preflightStreamTypeNames) {
		return preflightStreamTypeNames[s]
	}
	return fmt.Sprintf("PreflightStreamType(%d)", int32(s))
}
