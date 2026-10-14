package currentstatus

import "github.com/jwoglom/faketandem/pkg/pumpx2/messages"

// APIVersionRequest asks for the pump's API version, which the app reads
// first on every connection.
type APIVersionRequest struct {
	messages.Base
}

var apiVersionRequestProps = messages.Props{OpCode: 32, Size: 2, VariableSize: true, Type: messages.Request, Response: "ApiVersionResponse"}

func init() {
	messages.Register("ApiVersionRequest", &APIVersionRequest{}, apiVersionRequestProps,
		messages.Constructor{Func: NewAPIVersionRequest},
		messages.Constructor{Params: "byte[] cargo", Func: NewAPIVersionRequestWithCargo},
	)
}

// NewAPIVersionRequest is the request as the app sends it, with no payload.
func NewAPIVersionRequest() *APIVersionRequest {
	m := &APIVersionRequest{}
	m.SetCargo([]byte{})
	return m
}

// NewAPIVersionRequestWithCargo is the request with the given payload.
func NewAPIVersionRequestWithCargo(cargo []byte) *APIVersionRequest {
	m := &APIVersionRequest{}
	m.SetCargo(cargo)
	return m
}

// Parse implements messages.Message. Only an empty payload parses: pumpX2
// drops three bytes from the two its size allows, which always fails.
func (m *APIVersionRequest) Parse(raw []byte) error {
	if len(raw) == 0 {
		return nil
	}
	if len(raw) != apiVersionRequestProps.Size {
		return messages.InvalidLength("ApiVersionRequest", len(raw), apiVersionRequestProps.Size)
	}
	return messages.Invalid("ApiVersionRequest: cannot drop the first 3 of %d bytes", len(raw))
}
