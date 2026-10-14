package control

import (
	"fmt"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
)

// SendTipsControlGenericTestResponse is the pump's answer to a
// SendTipsControlGenericTestRequest. Status is a StatusCode; 0 (ACK) means
// success.
type SendTipsControlGenericTestResponse struct {
	messages.Base
	Status int32 `pumpx2:"status"`
}

var sendTipsControlGenericTestResponseProps = messages.Props{OpCode: 119, Size: 1, Signed: true, Type: messages.Response, Characteristic: messages.Control,
	Request: "SendTipsControlGenericTestRequest"}

func init() {
	messages.Register("SendTipsControlGenericTestResponse", &SendTipsControlGenericTestResponse{}, sendTipsControlGenericTestResponseProps,
		messages.Constructor{Params: "int status", Func: NewSendTipsControlGenericTestResponse},
	)
}

// NewSendTipsControlGenericTestResponse builds the response.
func NewSendTipsControlGenericTestResponse(status int32) *SendTipsControlGenericTestResponse {
	m := &SendTipsControlGenericTestResponse{Status: status}
	m.SetCargo([]byte{byte(status)})
	return m
}

// Parse implements messages.Message.
func (m *SendTipsControlGenericTestResponse) Parse(raw []byte) error {
	raw = messages.RemoveSignedRequestHmacBytes(sendTipsControlGenericTestResponseProps, raw)
	if len(raw) != sendTipsControlGenericTestResponseProps.Size {
		return messages.InvalidLength("SendTipsControlGenericTestResponse", len(raw), sendTipsControlGenericTestResponseProps.Size)
	}
	m.SetCargo(raw)
	m.Status = int32(raw[0])
	return nil
}

// StatusCode is a SendTipsControlGenericTestResponse status.
type StatusCode int32

// The status codes, by their wire values.
const (
	StatusCodeAck                   StatusCode = 0
	StatusCodeInvalidTestFeatureID  StatusCode = 1
	StatusCodeInvalidFeatureCommand StatusCode = 2
	StatusCodeUnknown               StatusCode = 255
)

// StatusCodeFromCode is the status with wire value code, or Unknown.
func StatusCodeFromCode(code int32) StatusCode {
	switch s := StatusCode(code); s {
	case StatusCodeAck, StatusCodeInvalidTestFeatureID, StatusCodeInvalidFeatureCommand, StatusCodeUnknown:
		return s
	}
	return StatusCodeUnknown
}

func (s StatusCode) String() string {
	switch s {
	case StatusCodeAck:
		return "ACK"
	case StatusCodeInvalidTestFeatureID:
		return "INVALID_TEST_FEATURE_ID"
	case StatusCodeInvalidFeatureCommand:
		return "INVALID_FEATURE_COMMAND"
	case StatusCodeUnknown:
		return "UNKNOWN"
	}
	return fmt.Sprintf("StatusCode(%d)", int32(s))
}
