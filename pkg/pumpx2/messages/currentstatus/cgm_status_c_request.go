package currentstatus

import "github.com/jwoglom/faketandem/pkg/pumpx2/messages"

// CGMStatusCRequest asks a Mobi at API 3.8 or later for its CGM session,
// answered by CGMStatusCResponse.
type CGMStatusCRequest struct {
	messages.Base
}

var cgmStatusCRequestProps = messages.Props{OpCode: -52, Type: messages.Request, MinAPI: "MOBI_API_V3_8", Response: "CGMStatusCResponse"}

func init() {
	messages.Register("CGMStatusCRequest", &CGMStatusCRequest{}, cgmStatusCRequestProps,
		messages.Constructor{Func: NewCGMStatusCRequest},
	)
}

// NewCGMStatusCRequest is the request, which has no payload.
func NewCGMStatusCRequest() *CGMStatusCRequest {
	m := &CGMStatusCRequest{}
	m.SetCargo([]byte{})
	return m
}

// Parse implements messages.Message.
func (m *CGMStatusCRequest) Parse(raw []byte) error {
	if len(raw) != cgmStatusCRequestProps.Size {
		return messages.InvalidLength("CGMStatusCRequest", len(raw), cgmStatusCRequestProps.Size)
	}
	m.SetCargo(raw)
	return nil
}
