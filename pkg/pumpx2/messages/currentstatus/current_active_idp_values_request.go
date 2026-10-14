package currentstatus

import "github.com/jwoglom/faketandem/pkg/pumpx2/messages"

// CurrentActiveIdpValuesRequest asks for the values of the active insulin
// delivery profile in force now: carb ratio, target BG, insulin duration
// and ISF.
type CurrentActiveIdpValuesRequest struct {
	messages.Base
}

var currentActiveIdpValuesRequestProps = messages.Props{OpCode: -106, Type: messages.Request, Response: "CurrentActiveIdpValuesResponse"}

func init() {
	messages.Register("CurrentActiveIdpValuesRequest", &CurrentActiveIdpValuesRequest{}, currentActiveIdpValuesRequestProps,
		messages.Constructor{Func: NewCurrentActiveIdpValuesRequest},
	)
}

// NewCurrentActiveIdpValuesRequest is the request, which has no payload.
func NewCurrentActiveIdpValuesRequest() *CurrentActiveIdpValuesRequest {
	m := &CurrentActiveIdpValuesRequest{}
	m.SetCargo([]byte{})
	return m
}

// Parse implements messages.Message. pumpX2 takes any payload.
func (m *CurrentActiveIdpValuesRequest) Parse(raw []byte) error {
	m.SetCargo(raw)
	return nil
}
