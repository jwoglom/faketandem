package control

import "github.com/jwoglom/faketandem/pkg/pumpx2/messages"

// RenameIDPResponse is the pump's answer to a RenameIDPRequest: Status 0
// means success. pumpX2 takes the second byte to be the number of profiles,
// unconfirmed.
type RenameIDPResponse struct {
	messages.Base
	Status           int32 `pumpx2:"status"`
	NumberOfProfiles int32 `pumpx2:"numberOfProfiles"`
}

var renameIDPResponseProps = messages.Props{OpCode: -87, Size: 2, Signed: true, Type: messages.Response, Characteristic: messages.Control,
	ModifiesInsulinDelivery: true, SupportedDevices: "MOBI_ONLY", Request: "RenameIDPRequest"}

func init() {
	messages.Register("RenameIDPResponse", &RenameIDPResponse{}, renameIDPResponseProps,
		messages.Constructor{Params: "int status, int numberOfProfiles", Func: NewRenameIDPResponse},
	)
}

// NewRenameIDPResponse builds the response.
func NewRenameIDPResponse(status, numberOfProfiles int32) *RenameIDPResponse {
	m := &RenameIDPResponse{Status: status, NumberOfProfiles: numberOfProfiles}
	m.SetCargo([]byte{byte(status), byte(numberOfProfiles)})
	return m
}

// Parse implements messages.Message.
func (m *RenameIDPResponse) Parse(raw []byte) error {
	raw = messages.RemoveSignedRequestHmacBytes(renameIDPResponseProps, raw)
	if len(raw) != renameIDPResponseProps.Size {
		return messages.InvalidLength("RenameIDPResponse", len(raw), renameIDPResponseProps.Size)
	}
	m.SetCargo(raw)
	m.Status = int32(int8(raw[0]))
	m.NumberOfProfiles = int32(int8(raw[1]))
	return nil
}
