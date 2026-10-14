package control

import "github.com/jwoglom/faketandem/pkg/pumpx2/messages"

// SetActiveIDPRequest activates an insulin delivery profile. IDPID is the
// profile's ID, not its slot; ProfileIndex is its slot (0-5) in
// ProfileStatusResponse.
type SetActiveIDPRequest struct {
	messages.Base
	IDPID        int32 `pumpx2:"idpId"`
	ProfileIndex int32 `pumpx2:"profileIndex"`
}

var setActiveIDPRequestProps = messages.Props{OpCode: -20, Size: 2, Signed: true, Type: messages.Request, Characteristic: messages.Control,
	ModifiesInsulinDelivery: true, SupportedDevices: "MOBI_ONLY", Response: "SetActiveIDPResponse"}

func init() {
	messages.Register("SetActiveIDPRequest", &SetActiveIDPRequest{}, setActiveIDPRequestProps,
		messages.Constructor{Params: "int idpId, int profileIndex", Func: NewSetActiveIDPRequest},
		messages.Constructor{Params: "int idpId", Func: NewSetActiveIDPRequestByID},
		messages.Constructor{Params: "byte[] raw", Func: NewSetActiveIDPRequestFromCargo},
	)
}

// NewSetActiveIDPRequest builds the request.
func NewSetActiveIDPRequest(idpID, profileIndex int32) *SetActiveIDPRequest {
	m := &SetActiveIDPRequest{IDPID: idpID, ProfileIndex: profileIndex}
	m.SetCargo([]byte{byte(idpID), byte(profileIndex)})
	return m
}

// NewSetActiveIDPRequestByID is pumpX2's deprecated constructor, with
// profile index 0.
func NewSetActiveIDPRequestByID(idpID int32) *SetActiveIDPRequest {
	return NewSetActiveIDPRequest(idpID, 0)
}

// NewSetActiveIDPRequestFromCargo parses a request's payload.
func NewSetActiveIDPRequestFromCargo(raw []byte) (*SetActiveIDPRequest, error) {
	m := &SetActiveIDPRequest{}
	return m, m.Parse(raw)
}

// Parse implements messages.Message.
func (m *SetActiveIDPRequest) Parse(raw []byte) error {
	raw = messages.RemoveSignedRequestHmacBytes(setActiveIDPRequestProps, raw)
	if len(raw) != setActiveIDPRequestProps.Size {
		return messages.InvalidLength("SetActiveIDPRequest", len(raw), setActiveIDPRequestProps.Size)
	}
	m.SetCargo(raw)
	m.IDPID = int32(int8(raw[0]))
	m.ProfileIndex = int32(int8(raw[1]))
	return nil
}
