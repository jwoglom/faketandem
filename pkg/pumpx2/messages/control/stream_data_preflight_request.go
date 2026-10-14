package control

import (
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/helpers"
)

// StreamDataPreflightRequest precedes streaming CGM data to the pump.
// StreamType is a PreflightStreamType ID; a 16-bit Length and the
// variable-length Hmac follow it on the wire.
type StreamDataPreflightRequest struct {
	messages.Base
	Hmac       []byte `pumpx2:"hmac"`
	Length     int32  `pumpx2:"length"`
	StreamType int32  `pumpx2:"streamType"`
}

var streamDataPreflightRequestProps = messages.Props{OpCode: -126, Size: 3, VariableSize: true, Signed: true, Type: messages.Request, Characteristic: messages.Control,
	MinAPI: "API_FUTURE", Response: "StreamDataPreflightResponse"}

func init() {
	messages.Register("StreamDataPreflightRequest", &StreamDataPreflightRequest{}, streamDataPreflightRequestProps,
		messages.Constructor{Params: "int streamType, int length, byte[] hmac", Func: NewStreamDataPreflightRequest},
	)
}

// NewStreamDataPreflightRequest builds the request.
func NewStreamDataPreflightRequest(streamType, length int32, hmac []byte) *StreamDataPreflightRequest {
	m := &StreamDataPreflightRequest{StreamType: streamType, Length: length, Hmac: hmac}
	m.SetCargo(helpers.Combine([]byte{byte(streamType)}, helpers.FirstTwoBytesLittleEndian(length), hmac))
	return m
}

// Parse implements messages.Message. Its length is not checked, and the
// signature trailer is dropped only from a payload of exactly 27 bytes, so
// a longer one leaves it in Hmac.
func (m *StreamDataPreflightRequest) Parse(raw []byte) error {
	raw = messages.RemoveSignedRequestHmacBytes(streamDataPreflightRequestProps, raw)
	m.SetCargo(raw)
	m.StreamType = int32(raw[0])
	m.Length = helpers.ReadShort(raw, 1)
	m.Hmac = helpers.CopyOfRange(raw, 3, len(raw))
	return nil
}
