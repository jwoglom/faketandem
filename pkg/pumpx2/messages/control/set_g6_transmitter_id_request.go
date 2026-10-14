package control

import (
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/helpers"
)

// G6TransmitterIDLength is the most bytes a Dexcom G6 transmitter ID takes in
// a SetG6TransmitterIDRequest.
const G6TransmitterIDLength = 6

// SetG6TransmitterIDRequest gives the pump a Dexcom G6 transmitter ID, NUL
// padded to 6 bytes and followed by 10 bytes of zero.
type SetG6TransmitterIDRequest struct {
	messages.Base
	TxID string `pumpx2:"txId"`
}

var setG6TransmitterIDRequestProps = messages.Props{OpCode: -80, Size: 16, Signed: true, Type: messages.Request, Characteristic: messages.Control,
	MinAPI: "MOBI_API_V3_5", Response: "SetG6TransmitterIdResponse"}

func init() {
	messages.Register("SetG6TransmitterIdRequest", &SetG6TransmitterIDRequest{}, setG6TransmitterIDRequestProps,
		messages.Constructor{Params: "String txId", Func: NewSetG6TransmitterIDRequest},
		messages.Constructor{Params: "byte[] raw", Func: NewSetG6TransmitterIDRequestFromCargo},
	)
}

// NewSetG6TransmitterIDRequest builds the request. It fails when txID is
// more than G6TransmitterIDLength bytes of UTF-8.
func NewSetG6TransmitterIDRequest(txID string) (*SetG6TransmitterIDRequest, error) {
	id, err := helpers.WriteString(txID, G6TransmitterIDLength)
	if err != nil {
		return nil, messages.Invalid("SetG6TransmitterIdRequest: %v", err)
	}
	m := &SetG6TransmitterIDRequest{TxID: txID}
	m.SetCargo(helpers.Combine(id, make([]byte, 10)))
	return m, nil
}

// NewSetG6TransmitterIDRequestFromCargo parses a request's payload.
func NewSetG6TransmitterIDRequestFromCargo(raw []byte) (*SetG6TransmitterIDRequest, error) {
	m := &SetG6TransmitterIDRequest{}
	return m, m.Parse(raw)
}

// Parse implements messages.Message.
func (m *SetG6TransmitterIDRequest) Parse(raw []byte) error {
	raw = messages.RemoveSignedRequestHmacBytes(setG6TransmitterIDRequestProps, raw)
	if len(raw) != setG6TransmitterIDRequestProps.Size {
		return messages.InvalidLength("SetG6TransmitterIdRequest", len(raw), setG6TransmitterIDRequestProps.Size)
	}
	m.SetCargo(raw)
	m.TxID = helpers.ReadString(raw, 0, G6TransmitterIDLength)
	return nil
}
