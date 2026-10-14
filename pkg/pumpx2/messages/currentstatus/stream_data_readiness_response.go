package currentstatus

import (
	"fmt"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
)

// StreamDataReadinessResponse is whether the pump is ready to take a
// FreeStyle Libre 2 sensor context, for the stream data type
// StreamDataReadinessRequest asked about.
type StreamDataReadinessResponse struct {
	messages.Base
	// FreestyleLibre2ReadinessID is a FreestyleLibre2Readiness ID.
	FreestyleLibre2ReadinessID int32 `pumpx2:"freestyleLibre2ReadinessId"`
	// StreamDataTypeOrdinal is a StreamDataType ordinal.
	StreamDataTypeOrdinal int32 `pumpx2:"streamDataTypeOrdinal"`
}

var streamDataReadinessResponseProps = messages.Props{OpCode: -57, Size: 2, Type: messages.Response, Request: "StreamDataReadinessRequest"}

func init() {
	messages.Register("StreamDataReadinessResponse", &StreamDataReadinessResponse{}, streamDataReadinessResponseProps,
		messages.Constructor{Params: "int freestyleLibre2ReadinessId, int streamDataTypeOrdinal", Func: NewStreamDataReadinessResponse},
	)
}

// NewStreamDataReadinessResponse builds the response.
func NewStreamDataReadinessResponse(freestyleLibre2ReadinessID, streamDataTypeOrdinal int32) *StreamDataReadinessResponse {
	m := &StreamDataReadinessResponse{FreestyleLibre2ReadinessID: freestyleLibre2ReadinessID, StreamDataTypeOrdinal: streamDataTypeOrdinal}
	m.SetCargo([]byte{byte(freestyleLibre2ReadinessID), byte(streamDataTypeOrdinal)})
	return m
}

// Parse implements messages.Message.
func (m *StreamDataReadinessResponse) Parse(raw []byte) error {
	if len(raw) != streamDataReadinessResponseProps.Size {
		return messages.InvalidLength("StreamDataReadinessResponse", len(raw), streamDataReadinessResponseProps.Size)
	}
	m.SetCargo(raw)
	m.FreestyleLibre2ReadinessID = int32(raw[0])
	m.StreamDataTypeOrdinal = int32(raw[1])
	return nil
}

// FreestyleLibre2Readiness is the pump's readiness.
func (m *StreamDataReadinessResponse) FreestyleLibre2Readiness() FreestyleLibre2Readiness {
	return FreestyleLibre2ReadinessFromID(m.FreestyleLibre2ReadinessID)
}

// StreamDataType is the stream data type the response is about.
func (m *StreamDataReadinessResponse) StreamDataType() StreamDataType {
	return StreamDataTypeFromOrdinal(m.StreamDataTypeOrdinal)
}

// FreestyleLibre2Readiness is whether the pump is ready for a FreeStyle
// Libre 2 sensor context transfer, or why not, by the firmware's sparse IDs.
type FreestyleLibre2Readiness int32

// The readinesses, by their IDs.
const (
	FreestyleLibre2ReadinessPumpReadyForFSL2ContextTransfer                                  FreestyleLibre2Readiness = 0
	FreestyleLibre2ReadinessNotReadyAnotherCGMSessionIsAlreadyActive                         FreestyleLibre2Readiness = 2
	FreestyleLibre2ReadinessNotReadyPumpTimeIsInvalid                                        FreestyleLibre2Readiness = 5
	FreestyleLibre2ReadinessNotReadyPumpWorkflowDoesNotAllowFSL2Start                        FreestyleLibre2Readiness = 6
	FreestyleLibre2ReadinessFreestyleLibre2CGMIsUnavailable                                  FreestyleLibre2Readiness = 8
	FreestyleLibre2ReadinessNotReadyThePumpIsUnableToFulfillTheRequestDueToAnUnexpectedError FreestyleLibre2Readiness = 11
	FreestyleLibre2ReadinessUnknown                                                          FreestyleLibre2Readiness = -1
)

var freestyleLibre2ReadinessNames = map[FreestyleLibre2Readiness]string{
	FreestyleLibre2ReadinessPumpReadyForFSL2ContextTransfer:                                  "PUMP_READY_FOR_FSL2_CONTEXT_TRANSFER",
	FreestyleLibre2ReadinessNotReadyAnotherCGMSessionIsAlreadyActive:                         "NOT_READY_ANOTHER_CGM_SESSION_IS_ALREADY_ACTIVE",
	FreestyleLibre2ReadinessNotReadyPumpTimeIsInvalid:                                        "NOT_READY_PUMP_TIME_IS_INVALID",
	FreestyleLibre2ReadinessNotReadyPumpWorkflowDoesNotAllowFSL2Start:                        "NOT_READY_PUMP_WORKFLOW_DOES_NOT_ALLOW_FSL2_START",
	FreestyleLibre2ReadinessFreestyleLibre2CGMIsUnavailable:                                  "FREESTYLE_LIBRE_2_CGM_IS_UNAVAILABLE",
	FreestyleLibre2ReadinessNotReadyThePumpIsUnableToFulfillTheRequestDueToAnUnexpectedError: "NOT_READY_THE_PUMP_IS_UNABLE_TO_FULFILL_THE_REQUEST_DUE_TO_AN_UNEXPECTED_ERROR",
	FreestyleLibre2ReadinessUnknown:                                                          "UNKNOWN",
}

// FreestyleLibre2ReadinessFromID is the readiness with ID id, or Unknown.
func FreestyleLibre2ReadinessFromID(id int32) FreestyleLibre2Readiness {
	if _, ok := freestyleLibre2ReadinessNames[FreestyleLibre2Readiness(id)]; ok {
		return FreestyleLibre2Readiness(id)
	}
	return FreestyleLibre2ReadinessUnknown
}

// ID is the readiness's ID.
func (r FreestyleLibre2Readiness) ID() int32 { return int32(r) }

func (r FreestyleLibre2Readiness) String() string {
	if name, ok := freestyleLibre2ReadinessNames[r]; ok {
		return name
	}
	return fmt.Sprintf("FreestyleLibre2Readiness(%d)", int32(r))
}

// StreamDataType is a kind of data the pump streams, by its ordinal.
type StreamDataType int32

// The stream data types.
const (
	StreamDataTypeFreestyleLibre2SensorContext StreamDataType = 0
	StreamDataTypeUnknown                      StreamDataType = 1
)

// StreamDataTypeFromOrdinal is the type with ordinal ordinal, or Unknown.
func StreamDataTypeFromOrdinal(ordinal int32) StreamDataType {
	if ordinal == int32(StreamDataTypeFreestyleLibre2SensorContext) {
		return StreamDataTypeFreestyleLibre2SensorContext
	}
	return StreamDataTypeUnknown
}

// OrdinalValue is the type's ordinal.
func (s StreamDataType) OrdinalValue() int32 { return int32(s) }

func (s StreamDataType) String() string {
	switch s {
	case StreamDataTypeFreestyleLibre2SensorContext:
		return "FREESTYLE_LIBRE_2_SENSOR_CONTEXT"
	case StreamDataTypeUnknown:
		return "UNKNOWN"
	}
	return fmt.Sprintf("StreamDataType(%d)", int32(s))
}
