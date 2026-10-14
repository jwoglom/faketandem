package control

import (
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/helpers"
)

// RenameIDPRequest renames an insulin delivery profile. IDPID is the
// profile's ID, not its slot; ProfileIndex is its slot (0-5) in
// ProfileStatusResponse.
type RenameIDPRequest struct {
	messages.Base
	IDPID        int32  `pumpx2:"idpId"`
	ProfileIndex int32  `pumpx2:"profileIndex"`
	ProfileName  string `pumpx2:"profileName"`
}

var renameIDPRequestProps = messages.Props{OpCode: -88, Size: 19, Signed: true, Type: messages.Request, Characteristic: messages.Control,
	ModifiesInsulinDelivery: true, SupportedDevices: "MOBI_ONLY", Response: "RenameIDPResponse"}

func init() {
	messages.Register("RenameIDPRequest", &RenameIDPRequest{}, renameIDPRequestProps,
		messages.Constructor{Params: "int idpId, int profileIndex, String profileName", Func: NewRenameIDPRequest},
		messages.Constructor{Params: "int idpId, String profileName", Func: NewRenameIDPRequestByID},
	)
}

// NewRenameIDPRequest builds the request. It fails when profileName is more
// than 16 bytes of UTF-8.
func NewRenameIDPRequest(idpID, profileIndex int32, profileName string) (*RenameIDPRequest, error) {
	name, err := helpers.WriteString(profileName, 16)
	if err != nil {
		return nil, messages.Invalid("RenameIDPRequest: %v", err)
	}
	m := &RenameIDPRequest{IDPID: idpID, ProfileIndex: profileIndex, ProfileName: profileName}
	m.SetCargo(helpers.Combine([]byte{byte(idpID), byte(profileIndex)}, name, []byte{0}))
	return m, nil
}

// NewRenameIDPRequestByID is pumpX2's deprecated constructor, with profile
// index 0.
func NewRenameIDPRequestByID(idpID int32, profileName string) (*RenameIDPRequest, error) {
	return NewRenameIDPRequest(idpID, 0, profileName)
}

// Parse implements messages.Message.
func (m *RenameIDPRequest) Parse(raw []byte) error {
	raw = messages.RemoveSignedRequestHmacBytes(renameIDPRequestProps, raw)
	if len(raw) != renameIDPRequestProps.Size {
		return messages.InvalidLength("RenameIDPRequest", len(raw), renameIDPRequestProps.Size)
	}
	m.SetCargo(raw)
	m.IDPID = int32(int8(raw[0]))
	m.ProfileIndex = int32(int8(raw[1]))
	m.ProfileName = helpers.ReadString(raw, 2, 16)
	return nil
}
