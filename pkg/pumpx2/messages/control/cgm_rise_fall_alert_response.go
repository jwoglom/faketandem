package control

import "github.com/jwoglom/faketandem/pkg/pumpx2/messages"

// CgmRiseFallAlertResponse is the pump's answer to a CgmRiseFallAlertRequest:
// Status 0 is success.
type CgmRiseFallAlertResponse struct {
	messages.Base
	Status int32 `pumpx2:"status"`
}

var cgmRiseFallAlertResponseProps = messages.Props{OpCode: -59, Size: 1, Signed: true, Type: messages.Response, Characteristic: messages.Control, Request: "CgmRiseFallAlertRequest"}

func init() {
	messages.Register("CgmRiseFallAlertResponse", &CgmRiseFallAlertResponse{}, cgmRiseFallAlertResponseProps,
		messages.Constructor{Params: "int status", Func: NewCgmRiseFallAlertResponse},
	)
}

// NewCgmRiseFallAlertResponse builds the response.
func NewCgmRiseFallAlertResponse(status int32) *CgmRiseFallAlertResponse {
	m := &CgmRiseFallAlertResponse{Status: status}
	m.SetCargo([]byte{byte(status)})
	return m
}

// Parse implements messages.Message.
func (m *CgmRiseFallAlertResponse) Parse(raw []byte) error {
	raw = messages.RemoveSignedRequestHmacBytes(cgmRiseFallAlertResponseProps, raw)
	if len(raw) != cgmRiseFallAlertResponseProps.Size {
		return messages.InvalidLength("CgmRiseFallAlertResponse", len(raw), cgmRiseFallAlertResponseProps.Size)
	}
	m.SetCargo(raw)
	m.Status = int32(int8(raw[0]))
	return nil
}
