package authentication

import (
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/helpers"
)

// Jpake3SessionKeyRequest asks for the pump's session key nonce once the
// EC-JPAKE rounds are done. The app always sends ChallengeParam 0.
type Jpake3SessionKeyRequest struct {
	messages.Base
	ChallengeParam int32 `pumpx2:"challengeParam"`
}

var jpake3SessionKeyRequestProps = messages.Props{OpCode: 38, Size: 2, Type: messages.Request, Characteristic: messages.Authorization, MinAPI: "API_V3_2", Response: "Jpake3SessionKeyResponse"}

func init() {
	messages.Register("Jpake3SessionKeyRequest", &Jpake3SessionKeyRequest{}, jpake3SessionKeyRequestProps,
		messages.Constructor{Params: "byte[] rawCargo", Func: NewJpake3SessionKeyRequestFromCargo},
		messages.Constructor{Params: "int challengeParam", Func: NewJpake3SessionKeyRequest},
	)
}

// NewJpake3SessionKeyRequestFromCargo parses a request's payload.
func NewJpake3SessionKeyRequestFromCargo(rawCargo []byte) *Jpake3SessionKeyRequest {
	m := &Jpake3SessionKeyRequest{}
	m.decode(rawCargo)
	return m
}

// NewJpake3SessionKeyRequest builds the request.
func NewJpake3SessionKeyRequest(challengeParam int32) *Jpake3SessionKeyRequest {
	m := &Jpake3SessionKeyRequest{ChallengeParam: challengeParam}
	m.SetCargo(helpers.FirstTwoBytesLittleEndian(challengeParam))
	return m
}

// Parse implements messages.Message. Like pumpX2's, it never fails: a
// payload shorter than two bytes reads as zero-padded.
func (m *Jpake3SessionKeyRequest) Parse(raw []byte) error {
	m.decode(raw)
	return nil
}

func (m *Jpake3SessionKeyRequest) decode(raw []byte) {
	m.SetCargo(raw)
	m.ChallengeParam = helpers.ReadShort(helpers.CopyOfRange(raw, 0, 2), 0)
}
