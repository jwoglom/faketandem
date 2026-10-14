package currentstatus

import "github.com/jwoglom/faketandem/pkg/pumpx2/messages"

// ProfileStatusRequest asks which insulin delivery profiles the pump holds, in
// which slots, and which segment of the active one is in force.
type ProfileStatusRequest struct {
	messages.Base
}

var profileStatusRequestProps = messages.Props{OpCode: 62, Type: messages.Request, Response: "ProfileStatusResponse"}

func init() {
	messages.Register("ProfileStatusRequest", &ProfileStatusRequest{}, profileStatusRequestProps,
		messages.Constructor{Func: NewProfileStatusRequest},
	)
}

// NewProfileStatusRequest is the request, which has no payload.
func NewProfileStatusRequest() *ProfileStatusRequest {
	m := &ProfileStatusRequest{}
	m.SetCargo([]byte{})
	return m
}

// Parse implements messages.Message.
func (m *ProfileStatusRequest) Parse(raw []byte) error {
	if len(raw) != profileStatusRequestProps.Size {
		return messages.InvalidLength("ProfileStatusRequest", len(raw), profileStatusRequestProps.Size)
	}
	m.SetCargo(raw)
	return nil
}
