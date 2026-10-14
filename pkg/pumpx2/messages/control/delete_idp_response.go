package control

import "github.com/jwoglom/faketandem/pkg/pumpx2/messages"

// DeleteIDPResponse is the pump's answer to a DeleteIDPRequest: Status 0 is
// success.
type DeleteIDPResponse struct {
	messages.Base
	Status       int32 `pumpx2:"status"`
	DeletedIDPID int32 `pumpx2:"deletedIdpId"`
}

var deleteIDPResponseProps = messages.Props{OpCode: -81, Size: 2, Signed: true, Type: messages.Response, Characteristic: messages.Control,
	ModifiesInsulinDelivery: true, SupportedDevices: "MOBI_ONLY", Request: "DeleteIDPRequest"}

func init() {
	messages.Register("DeleteIDPResponse", &DeleteIDPResponse{}, deleteIDPResponseProps,
		messages.Constructor{Params: "int status, int deletedIdpId", Func: NewDeleteIDPResponse},
		messages.Constructor{Params: "int status", Func: NewDeleteIDPResponseWithStatus},
	)
}

// NewDeleteIDPResponse builds the response.
func NewDeleteIDPResponse(status, deletedIDPID int32) *DeleteIDPResponse {
	m := &DeleteIDPResponse{Status: status, DeletedIDPID: deletedIDPID}
	m.SetCargo([]byte{byte(status), byte(deletedIDPID)})
	return m
}

// NewDeleteIDPResponseWithStatus is the response with DeletedIDPID 0.
func NewDeleteIDPResponseWithStatus(status int32) *DeleteIDPResponse {
	return NewDeleteIDPResponse(status, 0)
}

// Parse implements messages.Message.
func (m *DeleteIDPResponse) Parse(raw []byte) error {
	raw = messages.RemoveSignedRequestHmacBytes(deleteIDPResponseProps, raw)
	if len(raw) != deleteIDPResponseProps.Size {
		return messages.InvalidLength("DeleteIDPResponse", len(raw), deleteIDPResponseProps.Size)
	}
	m.SetCargo(raw)
	m.Status = int32(int8(raw[0]))
	m.DeletedIDPID = int32(int8(raw[1]))
	return nil
}
