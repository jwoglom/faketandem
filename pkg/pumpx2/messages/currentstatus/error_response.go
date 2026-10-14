package currentstatus

import (
	"fmt"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/helpers"
)

// ErrorResponse is the pump's refusal of a request, on any characteristic:
// the request's opcode (RequestCodeID) and why (an ErrorCode ID). It is 2
// bytes on CURRENT_STATUS and 26 after a signed request (Tandem's
// ErrorResponseControl), whose last 24 are RemainingBytes. pumpX2 pairs it
// with NonexistentErrorRequest, a name with no class.
type ErrorResponse struct {
	messages.Base
	RequestCodeID  int32      `pumpx2:"requestCodeId"`
	ErrorCodeID    int32      `pumpx2:"errorCodeId"`
	RemainingBytes []byte     `pumpx2:"remainingBytes"`
	ErrorCode      *ErrorCode `pumpx2:"errorCode"`
}

var errorResponseProps = messages.Props{OpCode: 77, Size: 2, Type: messages.Response, Request: "NonexistentErrorRequest"}

func init() {
	messages.Register("ErrorResponse", &ErrorResponse{}, errorResponseProps,
		messages.Constructor{Params: "int requestCodeId, int errorCodeId", Func: NewErrorResponse},
	)
}

// NewErrorResponse is the refusal of the request with opcode requestCodeID,
// for the ErrorCode with ID errorCodeID.
func NewErrorResponse(requestCodeID, errorCodeID int32) *ErrorResponse {
	m := &ErrorResponse{}
	m.decode([]byte{byte(requestCodeID), byte(errorCodeID)})
	return m
}

// Parse implements messages.Message. It takes any size from 2 bytes up, as
// the response has two; a payload of exactly 2 leaves RemainingBytes as it
// was.
func (m *ErrorResponse) Parse(raw []byte) error {
	if len(raw) < errorResponseProps.Size {
		return messages.InvalidLength("ErrorResponse", len(raw), errorResponseProps.Size)
	}
	m.decode(raw)
	return nil
}

func (m *ErrorResponse) decode(raw []byte) {
	m.RequestCodeID = int32(int8(raw[0]))
	m.ErrorCodeID = int32(int8(raw[1]))
	m.ErrorCode = ErrorCodeFromID(m.ErrorCodeID)
	if len(raw) > 2 {
		m.RemainingBytes = helpers.DropFirstN(raw, 2)
	}
	m.SetCargo(raw)
}

// ErrorCode is why the pump refused a request.
type ErrorCode int32

// The error codes, by their wire IDs.
const (
	ErrorCodeUndefinedError             ErrorCode = 0
	ErrorCodeCRCMismatch                ErrorCode = 1
	ErrorCodeTransactionIDMismatch      ErrorCode = 3
	ErrorCodeBadCargoLength             ErrorCode = 4
	ErrorCodeBadOpcode                  ErrorCode = 6
	ErrorCodeInvalidRequiredParameter   ErrorCode = 7
	ErrorCodeMessageBufferFull          ErrorCode = 8
	ErrorCodeInvalidAuthenticationError ErrorCode = 9
)

var errorCodeNames = map[ErrorCode]string{
	ErrorCodeUndefinedError:             "UNDEFINED_ERROR",
	ErrorCodeCRCMismatch:                "CRC_MISMATCH",
	ErrorCodeTransactionIDMismatch:      "TRANSACTION_ID_MISMATCH",
	ErrorCodeBadCargoLength:             "BAD_CARGO_LENGTH",
	ErrorCodeBadOpcode:                  "BAD_OPCODE",
	ErrorCodeInvalidRequiredParameter:   "INVALID_REQUIRED_PARAMETER",
	ErrorCodeMessageBufferFull:          "MESSAGE_BUFFER_FULL",
	ErrorCodeInvalidAuthenticationError: "INVALID_AUTHENTICATION_ERROR",
}

// ErrorCodeFromID is the error code with wire ID id, or nil for an ID that
// is none.
func ErrorCodeFromID(id int32) *ErrorCode {
	c := ErrorCode(id)
	if _, ok := errorCodeNames[c]; !ok {
		return nil
	}
	return &c
}

// ID is the error code's wire ID.
func (c ErrorCode) ID() int32 { return int32(c) }

func (c ErrorCode) String() string {
	if name, ok := errorCodeNames[c]; ok {
		return name
	}
	return fmt.Sprintf("ErrorCode(%d)", int32(c))
}
