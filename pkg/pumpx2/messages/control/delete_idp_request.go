package control

import "github.com/jwoglom/faketandem/pkg/pumpx2/messages"

// DeleteIDPRequest deletes a Mobi insulin delivery profile: IDPID is the
// profile's ID, not its slot, and ProfileIndex its slot (0-5) in
// ProfileStatusResponse.
type DeleteIDPRequest struct {
	messages.Base
	IDPID        int32 `pumpx2:"idpId"`
	ProfileIndex int32 `pumpx2:"profileIndex"`
}

var deleteIDPRequestProps = messages.Props{OpCode: -82, Size: 2, Signed: true, Type: messages.Request, Characteristic: messages.Control,
	ModifiesInsulinDelivery: true, SupportedDevices: "MOBI_ONLY", Response: "DeleteIDPResponse"}

func init() {
	messages.Register("DeleteIDPRequest", &DeleteIDPRequest{}, deleteIDPRequestProps,
		messages.Constructor{Params: "int idpId, int profileIndex", Func: NewDeleteIDPRequest},
		messages.Constructor{Params: "int idpId", Func: NewDeleteIDPRequestInSlot0},
	)
}

// NewDeleteIDPRequest builds the request.
func NewDeleteIDPRequest(idpID, profileIndex int32) *DeleteIDPRequest {
	m := &DeleteIDPRequest{IDPID: idpID, ProfileIndex: profileIndex}
	m.SetCargo([]byte{byte(idpID), byte(profileIndex)})
	return m
}

// NewDeleteIDPRequestInSlot0 is the request with ProfileIndex 0, which
// pumpX2 deprecates: it is right only for the profile in the first slot.
func NewDeleteIDPRequestInSlot0(idpID int32) *DeleteIDPRequest {
	return NewDeleteIDPRequest(idpID, 0)
}

// Parse implements messages.Message.
func (m *DeleteIDPRequest) Parse(raw []byte) error {
	raw = messages.RemoveSignedRequestHmacBytes(deleteIDPRequestProps, raw)
	if len(raw) != deleteIDPRequestProps.Size {
		return messages.InvalidLength("DeleteIDPRequest", len(raw), deleteIDPRequestProps.Size)
	}
	m.SetCargo(raw)
	m.IDPID = int32(int8(raw[0]))
	m.ProfileIndex = int32(int8(raw[1]))
	return nil
}
