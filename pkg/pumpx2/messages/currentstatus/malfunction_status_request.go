package currentstatus

import "github.com/jwoglom/faketandem/pkg/pumpx2/messages"

// MalfunctionStatusRequest asks for the pump's malfunctions, which it
// answers with a MalfunctionBitmaskStatusResponse.
type MalfunctionStatusRequest struct {
	messages.Base
}

var malfunctionStatusRequestProps = messages.Props{OpCode: 118, Type: messages.Request, Response: "MalfunctionBitmaskStatusResponse"}

func init() {
	messages.Register("MalfunctionStatusRequest", &MalfunctionStatusRequest{}, malfunctionStatusRequestProps)
}

// NewMalfunctionStatusRequest is the request, which has no payload.
func NewMalfunctionStatusRequest() *MalfunctionStatusRequest {
	m := &MalfunctionStatusRequest{}
	m.SetCargo([]byte{})
	return m
}

// Parse implements messages.Message.
func (m *MalfunctionStatusRequest) Parse(raw []byte) error {
	if len(raw) != malfunctionStatusRequestProps.Size {
		return messages.InvalidLength("MalfunctionStatusRequest", len(raw), malfunctionStatusRequestProps.Size)
	}
	m.SetCargo(raw)
	return nil
}
