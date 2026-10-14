package control

import "github.com/jwoglom/faketandem/pkg/pumpx2/messages"

// PlaySoundRequest has the pump play a sound.
type PlaySoundRequest struct {
	messages.Base
}

var playSoundRequestProps = messages.Props{OpCode: -12, Signed: true, Type: messages.Request, Characteristic: messages.Control,
	Response: "PlaySoundResponse"}

func init() {
	messages.Register("PlaySoundRequest", &PlaySoundRequest{}, playSoundRequestProps,
		messages.Constructor{Func: NewPlaySoundRequest},
	)
}

// NewPlaySoundRequest builds the request, which has no payload.
func NewPlaySoundRequest() *PlaySoundRequest {
	m := &PlaySoundRequest{}
	m.SetCargo([]byte{})
	return m
}

// Parse implements messages.Message.
func (m *PlaySoundRequest) Parse(raw []byte) error {
	raw = messages.RemoveSignedRequestHmacBytes(playSoundRequestProps, raw)
	if len(raw) != playSoundRequestProps.Size {
		return messages.InvalidLength("PlaySoundRequest", len(raw), playSoundRequestProps.Size)
	}
	m.SetCargo(raw)
	return nil
}
