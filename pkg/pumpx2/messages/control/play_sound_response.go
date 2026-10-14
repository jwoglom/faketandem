package control

import "github.com/jwoglom/faketandem/pkg/pumpx2/messages"

// PlaySoundResponse is the pump's answer to a PlaySoundRequest: Status 0
// means success.
type PlaySoundResponse struct {
	messages.Base
	Status int32 `pumpx2:"status"`
}

var playSoundResponseProps = messages.Props{OpCode: -11, Size: 1, Signed: true, Type: messages.Response, Characteristic: messages.Control,
	Request: "PlaySoundRequest"}

func init() {
	messages.Register("PlaySoundResponse", &PlaySoundResponse{}, playSoundResponseProps,
		messages.Constructor{Params: "byte[] raw", Func: NewPlaySoundResponseFromCargo},
	)
}

// NewPlaySoundResponseFromCargo parses a response's payload.
func NewPlaySoundResponseFromCargo(raw []byte) (*PlaySoundResponse, error) {
	m := &PlaySoundResponse{}
	return m, m.Parse(raw)
}

// Parse implements messages.Message.
func (m *PlaySoundResponse) Parse(raw []byte) error {
	raw = messages.RemoveSignedRequestHmacBytes(playSoundResponseProps, raw)
	if len(raw) != playSoundResponseProps.Size {
		return messages.InvalidLength("PlaySoundResponse", len(raw), playSoundResponseProps.Size)
	}
	m.SetCargo(raw)
	m.Status = int32(int8(raw[0]))
	return nil
}
