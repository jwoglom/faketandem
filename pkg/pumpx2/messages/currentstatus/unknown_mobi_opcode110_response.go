package currentstatus

import "github.com/jwoglom/faketandem/pkg/pumpx2/messages"

// UnknownMobiOpcode110Response is a Mobi's answer to
// UnknownMobiOpcode110Request: four bytes whose meaning is unknown.
type UnknownMobiOpcode110Response struct {
	messages.Base
}

var unknownMobiOpcode110ResponseProps = messages.Props{OpCode: 111, Size: 4, Type: messages.Response, MinAPI: "MOBI_API_V3_5", SupportedDevices: "MOBI_ONLY",
	Request: "UnknownMobiOpcode110Request"}

func init() {
	messages.Register("UnknownMobiOpcode110Response", &UnknownMobiOpcode110Response{}, unknownMobiOpcode110ResponseProps,
		messages.Constructor{Params: "byte[] raw", Func: NewUnknownMobiOpcode110ResponseFromCargo},
	)
}

// NewUnknownMobiOpcode110ResponseFromCargo parses a response's payload.
func NewUnknownMobiOpcode110ResponseFromCargo(raw []byte) (*UnknownMobiOpcode110Response, error) {
	m := &UnknownMobiOpcode110Response{}
	return m, m.Parse(raw)
}

// Parse implements messages.Message.
func (m *UnknownMobiOpcode110Response) Parse(raw []byte) error {
	if len(raw) != unknownMobiOpcode110ResponseProps.Size {
		return messages.InvalidLength("UnknownMobiOpcode110Response", len(raw), unknownMobiOpcode110ResponseProps.Size)
	}
	m.SetCargo(raw)
	return nil
}
