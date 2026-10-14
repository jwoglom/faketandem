package control

import "github.com/jwoglom/faketandem/pkg/pumpx2/messages"

// CgmOutOfRangeAlertResponse is the pump's answer to a
// CgmOutOfRangeAlertRequest: Status 0 is success.
type CgmOutOfRangeAlertResponse struct {
	messages.Base
	Status int32 `pumpx2:"status"`
}

var cgmOutOfRangeAlertResponseProps = messages.Props{OpCode: -57, Size: 1, Signed: true, Type: messages.Response, Characteristic: messages.Control, Request: "CgmOutOfRangeAlertRequest"}

func init() {
	messages.Register("CgmOutOfRangeAlertResponse", &CgmOutOfRangeAlertResponse{}, cgmOutOfRangeAlertResponseProps,
		messages.Constructor{Params: "int status", Func: NewCgmOutOfRangeAlertResponse},
	)
}

// NewCgmOutOfRangeAlertResponse builds the response.
func NewCgmOutOfRangeAlertResponse(status int32) *CgmOutOfRangeAlertResponse {
	m := &CgmOutOfRangeAlertResponse{Status: status}
	m.SetCargo([]byte{byte(status)})
	return m
}

// Parse implements messages.Message.
func (m *CgmOutOfRangeAlertResponse) Parse(raw []byte) error {
	raw = messages.RemoveSignedRequestHmacBytes(cgmOutOfRangeAlertResponseProps, raw)
	if len(raw) != cgmOutOfRangeAlertResponseProps.Size {
		return messages.InvalidLength("CgmOutOfRangeAlertResponse", len(raw), cgmOutOfRangeAlertResponseProps.Size)
	}
	m.SetCargo(raw)
	m.Status = int32(int8(raw[0]))
	return nil
}
