package control

import (
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/helpers"
)

// FillCannulaRequest primes a Mobi's cannula with PrimeSizeMilliUnits
// milliunits of insulin. Delivery must be suspended.
type FillCannulaRequest struct {
	messages.Base
	PrimeSizeMilliUnits int32 `pumpx2:"primeSizeMilliUnits"`
}

var fillCannulaRequestProps = messages.Props{OpCode: -104, Size: 2, Signed: true, Type: messages.Request, Characteristic: messages.Control,
	ModifiesInsulinDelivery: true, MinAPI: "MOBI_API_V3_5", SupportedDevices: "MOBI_ONLY", Response: "FillCannulaResponse"}

func init() {
	messages.Register("FillCannulaRequest", &FillCannulaRequest{}, fillCannulaRequestProps,
		messages.Constructor{Params: "int primeSizeMilliUnits", Func: NewFillCannulaRequest},
	)
}

// NewFillCannulaRequest builds the request. It fails unless the prime is
// more than 0 and at most 3000 milliunits, pumpX2's untested sanity bound.
func NewFillCannulaRequest(primeSizeMilliUnits int32) (*FillCannulaRequest, error) {
	if primeSizeMilliUnits <= 0 || primeSizeMilliUnits > 3000 {
		return nil, messages.Invalid("FillCannulaRequest: a prime of %d milliunits", primeSizeMilliUnits)
	}
	m := &FillCannulaRequest{PrimeSizeMilliUnits: primeSizeMilliUnits}
	m.SetCargo(helpers.FirstTwoBytesLittleEndian(primeSizeMilliUnits))
	return m, nil
}

// Parse implements messages.Message.
func (m *FillCannulaRequest) Parse(raw []byte) error {
	raw = messages.RemoveSignedRequestHmacBytes(fillCannulaRequestProps, raw)
	if len(raw) != fillCannulaRequestProps.Size {
		return messages.InvalidLength("FillCannulaRequest", len(raw), fillCannulaRequestProps.Size)
	}
	m.SetCargo(raw)
	m.PrimeSizeMilliUnits = helpers.ReadShort(raw, 0)
	return nil
}
