package control

import "github.com/jwoglom/faketandem/pkg/pumpx2/messages"

// ResumePumpingResponse is the pump's answer to a ResumePumpingRequest:
// Status 0 means delivery resumed.
type ResumePumpingResponse struct {
	messages.Base
	Status int32 `pumpx2:"status"`
}

var resumePumpingResponseProps = messages.Props{OpCode: -101, Size: 1, Signed: true, Type: messages.Response, Characteristic: messages.Control,
	ModifiesInsulinDelivery: true, SupportedDevices: "MOBI_ONLY", Request: "ResumePumpingRequest"}

func init() {
	messages.Register("ResumePumpingResponse", &ResumePumpingResponse{}, resumePumpingResponseProps,
		messages.Constructor{Params: "int status", Func: NewResumePumpingResponse},
		messages.Constructor{Params: "byte[] cargo", Func: NewResumePumpingResponseFromCargo},
	)
}

// NewResumePumpingResponse builds the response. pumpX2 reads Status back
// from the byte it writes, so it is status's low byte, signed.
func NewResumePumpingResponse(status int32) *ResumePumpingResponse {
	m := &ResumePumpingResponse{}
	m.decode([]byte{byte(status)})
	return m
}

// NewResumePumpingResponseFromCargo parses a response's payload.
func NewResumePumpingResponseFromCargo(cargo []byte) (*ResumePumpingResponse, error) {
	m := &ResumePumpingResponse{}
	return m, m.Parse(cargo)
}

// Parse implements messages.Message.
func (m *ResumePumpingResponse) Parse(raw []byte) error {
	raw = messages.RemoveSignedRequestHmacBytes(resumePumpingResponseProps, raw)
	if len(raw) != resumePumpingResponseProps.Size {
		return messages.InvalidLength("ResumePumpingResponse", len(raw), resumePumpingResponseProps.Size)
	}
	m.decode(raw)
	return nil
}

func (m *ResumePumpingResponse) decode(raw []byte) {
	m.SetCargo(raw)
	m.Status = int32(int8(raw[0]))
}
