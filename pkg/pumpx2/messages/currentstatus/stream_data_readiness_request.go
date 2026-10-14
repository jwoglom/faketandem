package currentstatus

import "github.com/jwoglom/faketandem/pkg/pumpx2/messages"

// StreamDataReadinessRequest asks whether the pump is ready to take a kind
// of stream data, by its StreamDataType ordinal; pumpX2's no-argument
// constructor asks about a FreeStyle Libre 2 sensor context.
type StreamDataReadinessRequest struct {
	messages.Base
	StreamDataTypeID int32          `pumpx2:"streamDataTypeId"`
	StreamDataType   StreamDataType `pumpx2:"streamDataType"`
}

var streamDataReadinessRequestProps = messages.Props{OpCode: -58, Size: 1, Type: messages.Request, Response: "StreamDataReadinessResponse"}

func init() {
	messages.Register("StreamDataReadinessRequest", &StreamDataReadinessRequest{}, streamDataReadinessRequestProps,
		messages.Constructor{Params: "StreamDataReadinessResponse.StreamDataType streamDataType", Func: NewStreamDataReadinessRequest},
		messages.Constructor{Params: "int streamDataTypeId", Func: NewStreamDataReadinessRequestFromID},
	)
}

// NewStreamDataReadinessRequest asks about streamDataType.
func NewStreamDataReadinessRequest(streamDataType StreamDataType) *StreamDataReadinessRequest {
	return NewStreamDataReadinessRequestFromID(streamDataType.OrdinalValue())
}

// NewStreamDataReadinessRequestFromID asks about the stream data type with
// ordinal streamDataTypeID.
func NewStreamDataReadinessRequestFromID(streamDataTypeID int32) *StreamDataReadinessRequest {
	m := &StreamDataReadinessRequest{StreamDataTypeID: streamDataTypeID, StreamDataType: StreamDataTypeFromOrdinal(streamDataTypeID)}
	m.SetCargo([]byte{byte(streamDataTypeID)})
	return m
}

// Parse implements messages.Message.
func (m *StreamDataReadinessRequest) Parse(raw []byte) error {
	if len(raw) != streamDataReadinessRequestProps.Size {
		return messages.InvalidLength("StreamDataReadinessRequest", len(raw), streamDataReadinessRequestProps.Size)
	}
	m.SetCargo(raw)
	m.StreamDataTypeID = int32(raw[0])
	m.StreamDataType = StreamDataTypeFromOrdinal(m.StreamDataTypeID)
	return nil
}
