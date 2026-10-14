package currentstatus

import "github.com/jwoglom/faketandem/pkg/pumpx2/messages"

// BasalIQStatusRequest asks what Basal-IQ is doing.
type BasalIQStatusRequest struct {
	messages.Base
}

var basalIQStatusRequestProps = messages.Props{OpCode: 112, Type: messages.Request, Response: "BasalIQStatusResponse"}

func init() {
	messages.Register("BasalIQStatusRequest", &BasalIQStatusRequest{}, basalIQStatusRequestProps,
		messages.Constructor{Func: NewBasalIQStatusRequest},
	)
}

// NewBasalIQStatusRequest is the request, which has no payload.
func NewBasalIQStatusRequest() *BasalIQStatusRequest {
	m := &BasalIQStatusRequest{}
	m.SetCargo([]byte{})
	return m
}

// Parse implements messages.Message.
func (m *BasalIQStatusRequest) Parse(raw []byte) error {
	if len(raw) != basalIQStatusRequestProps.Size {
		return messages.InvalidLength("BasalIQStatusRequest", len(raw), basalIQStatusRequestProps.Size)
	}
	m.SetCargo(raw)
	return nil
}
