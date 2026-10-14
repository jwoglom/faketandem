package currentstatus

import (
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/helpers"
)

// IDPSettingsRequest asks for an insulin delivery profile's settings,
// including how many time segments it has, by the profile's ID (not its
// slot).
type IDPSettingsRequest struct {
	messages.Base
	IDPID int32 `pumpx2:"idpId"`
}

var idpSettingsRequestProps = messages.Props{OpCode: 64, Size: 1, Type: messages.Request, Response: "IDPSettingsResponse"}

func init() {
	messages.Register("IDPSettingsRequest", &IDPSettingsRequest{}, idpSettingsRequestProps,
		messages.Constructor{Params: "int idpId", Func: NewIDPSettingsRequest},
	)
}

// NewIDPSettingsRequest builds the request.
func NewIDPSettingsRequest(idpID int32) *IDPSettingsRequest {
	m := &IDPSettingsRequest{IDPID: idpID}
	m.SetCargo(helpers.FirstByteLittleEndian(idpID))
	return m
}

// Parse implements messages.Message.
func (m *IDPSettingsRequest) Parse(raw []byte) error {
	if len(raw) != idpSettingsRequestProps.Size {
		return messages.InvalidLength("IDPSettingsRequest", len(raw), idpSettingsRequestProps.Size)
	}
	m.SetCargo(raw)
	m.IDPID = int32(int8(raw[0]))
	return nil
}
