package control

import "github.com/jwoglom/faketandem/pkg/pumpx2/messages"

// CreateIDPResponse is the pump's answer to a CreateIDPRequest: Status 0 is
// success. pumpX2 marks NewIDPID, which should be the new profile's ID, as
// incorrect.
type CreateIDPResponse struct {
	messages.Base
	Status   int32 `pumpx2:"status"`
	NewIDPID int32 `pumpx2:"newIdpId"`
}

var createIDPResponseProps = messages.Props{OpCode: -25, Size: 2, Signed: true, Type: messages.Response, Characteristic: messages.Control,
	ModifiesInsulinDelivery: true, SupportedDevices: "MOBI_ONLY", Request: "CreateIDPRequest"}

func init() {
	messages.Register("CreateIDPResponse", &CreateIDPResponse{}, createIDPResponseProps,
		messages.Constructor{Params: "int status, int newIdpId", Func: NewCreateIDPResponse},
		messages.Constructor{Params: "byte[] raw", Func: NewCreateIDPResponseFromCargo},
	)
}

// NewCreateIDPResponse builds the response.
func NewCreateIDPResponse(status, newIDPID int32) *CreateIDPResponse {
	m := &CreateIDPResponse{Status: status, NewIDPID: newIDPID}
	m.SetCargo([]byte{byte(status), byte(newIDPID)})
	return m
}

// NewCreateIDPResponseFromCargo parses a response's payload.
func NewCreateIDPResponseFromCargo(raw []byte) (*CreateIDPResponse, error) {
	m := &CreateIDPResponse{}
	return m, m.Parse(raw)
}

// Parse implements messages.Message.
func (m *CreateIDPResponse) Parse(raw []byte) error {
	raw = messages.RemoveSignedRequestHmacBytes(createIDPResponseProps, raw)
	if len(raw) != createIDPResponseProps.Size {
		return messages.InvalidLength("CreateIDPResponse", len(raw), createIDPResponseProps.Size)
	}
	m.SetCargo(raw)
	m.Status = int32(int8(raw[0]))
	m.NewIDPID = int32(int8(raw[1]))
	return nil
}
