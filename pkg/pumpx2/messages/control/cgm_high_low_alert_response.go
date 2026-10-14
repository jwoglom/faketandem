package control

import "github.com/jwoglom/faketandem/pkg/pumpx2/messages"

// CgmHighLowAlertResponse is the pump's answer to a CgmHighLowAlertRequest:
// Status 0 is success.
type CgmHighLowAlertResponse struct {
	messages.Base
	Status int32 `pumpx2:"status"`
}

var cgmHighLowAlertResponseProps = messages.Props{OpCode: -61, Size: 1, Signed: true, Type: messages.Response, Characteristic: messages.Control, Request: "CgmHighLowAlertRequest"}

func init() {
	messages.Register("CgmHighLowAlertResponse", &CgmHighLowAlertResponse{}, cgmHighLowAlertResponseProps,
		messages.Constructor{Params: "int status", Func: NewCgmHighLowAlertResponse},
	)
}

// NewCgmHighLowAlertResponse builds the response.
func NewCgmHighLowAlertResponse(status int32) *CgmHighLowAlertResponse {
	m := &CgmHighLowAlertResponse{Status: status}
	m.SetCargo([]byte{byte(status)})
	return m
}

// Parse implements messages.Message.
func (m *CgmHighLowAlertResponse) Parse(raw []byte) error {
	raw = messages.RemoveSignedRequestHmacBytes(cgmHighLowAlertResponseProps, raw)
	if len(raw) != cgmHighLowAlertResponseProps.Size {
		return messages.InvalidLength("CgmHighLowAlertResponse", len(raw), cgmHighLowAlertResponseProps.Size)
	}
	m.SetCargo(raw)
	m.Status = int32(int8(raw[0]))
	return nil
}
