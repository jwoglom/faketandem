package currentstatus

import "github.com/jwoglom/faketandem/pkg/pumpx2/messages"

// LocalizationRequest asks for the pump's glucose unit, language and region.
type LocalizationRequest struct {
	messages.Base
}

var localizationRequestProps = messages.Props{OpCode: -90, Type: messages.Request, Response: "LocalizationResponse"}

func init() {
	messages.Register("LocalizationRequest", &LocalizationRequest{}, localizationRequestProps,
		messages.Constructor{Func: NewLocalizationRequest},
	)
}

// NewLocalizationRequest is the request, which has no payload.
func NewLocalizationRequest() *LocalizationRequest {
	m := &LocalizationRequest{}
	m.SetCargo([]byte{})
	return m
}

// Parse implements messages.Message.
func (m *LocalizationRequest) Parse(raw []byte) error {
	if len(raw) != localizationRequestProps.Size {
		return messages.InvalidLength("LocalizationRequest", len(raw), localizationRequestProps.Size)
	}
	m.SetCargo(raw)
	return nil
}
