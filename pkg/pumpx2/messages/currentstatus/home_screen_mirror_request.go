package currentstatus

import "github.com/jwoglom/faketandem/pkg/pumpx2/messages"

// HomeScreenMirrorRequest asks for the icons the pump's home screen shows: CGM
// trend and alert, status, bolus, basal and Control-IQ state.
type HomeScreenMirrorRequest struct {
	messages.Base
}

var homeScreenMirrorRequestProps = messages.Props{OpCode: 56, Type: messages.Request, Response: "HomeScreenMirrorResponse"}

func init() {
	messages.Register("HomeScreenMirrorRequest", &HomeScreenMirrorRequest{}, homeScreenMirrorRequestProps,
		messages.Constructor{Func: NewHomeScreenMirrorRequest},
	)
}

// NewHomeScreenMirrorRequest is the request, which has no payload.
func NewHomeScreenMirrorRequest() *HomeScreenMirrorRequest {
	m := &HomeScreenMirrorRequest{}
	m.SetCargo([]byte{})
	return m
}

// Parse implements messages.Message.
func (m *HomeScreenMirrorRequest) Parse(raw []byte) error {
	if len(raw) != homeScreenMirrorRequestProps.Size {
		return messages.InvalidLength("HomeScreenMirrorRequest", len(raw), homeScreenMirrorRequestProps.Size)
	}
	m.SetCargo(raw)
	return nil
}
