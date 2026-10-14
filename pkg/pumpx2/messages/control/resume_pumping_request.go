package control

import "github.com/jwoglom/faketandem/pkg/pumpx2/messages"

// ResumePumpingRequest resumes insulin delivery on a suspended Mobi.
type ResumePumpingRequest struct {
	messages.Base
}

var resumePumpingRequestProps = messages.Props{OpCode: -102, Signed: true, Type: messages.Request, Characteristic: messages.Control,
	ModifiesInsulinDelivery: true, SupportedDevices: "MOBI_ONLY", Response: "ResumePumpingResponse"}

func init() {
	messages.Register("ResumePumpingRequest", &ResumePumpingRequest{}, resumePumpingRequestProps,
		messages.Constructor{Func: NewResumePumpingRequest},
	)
}

// NewResumePumpingRequest builds the request, which has no payload.
func NewResumePumpingRequest() *ResumePumpingRequest {
	m := &ResumePumpingRequest{}
	m.SetCargo([]byte{})
	return m
}

// Parse implements messages.Message.
func (m *ResumePumpingRequest) Parse(raw []byte) error {
	raw = messages.RemoveSignedRequestHmacBytes(resumePumpingRequestProps, raw)
	if len(raw) != resumePumpingRequestProps.Size {
		return messages.InvalidLength("ResumePumpingRequest", len(raw), resumePumpingRequestProps.Size)
	}
	m.SetCargo(raw)
	return nil
}
