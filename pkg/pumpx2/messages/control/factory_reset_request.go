package control

import (
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/helpers"
)

// FactoryResetRequest resets the pump to its factory settings. It carries a
// Key and the pump's SerialNumber, each a uint32.
type FactoryResetRequest struct {
	messages.Base
	Key          int64 `pumpx2:"key"`
	SerialNumber int64 `pumpx2:"serialNumber"`
}

var factoryResetRequestProps = messages.Props{OpCode: -24, Size: 8, Signed: true, Type: messages.Request, Characteristic: messages.Control,
	Response: "FactoryResetResponse"}

func init() {
	messages.Register("FactoryResetRequest", &FactoryResetRequest{}, factoryResetRequestProps,
		messages.Constructor{Params: "long key, long serialNumber", Func: NewFactoryResetRequest},
	)
}

// NewFactoryResetRequest builds the request.
func NewFactoryResetRequest(key, serialNumber int64) *FactoryResetRequest {
	m := &FactoryResetRequest{Key: key, SerialNumber: serialNumber}
	m.SetCargo(helpers.Combine(helpers.ToUint32(key), helpers.ToUint32(serialNumber)))
	return m
}

// Parse implements messages.Message.
func (m *FactoryResetRequest) Parse(raw []byte) error {
	raw = messages.RemoveSignedRequestHmacBytes(factoryResetRequestProps, raw)
	if len(raw) != factoryResetRequestProps.Size {
		return messages.InvalidLength("FactoryResetRequest", len(raw), factoryResetRequestProps.Size)
	}
	m.SetCargo(raw)
	m.Key = helpers.ReadUint32(raw, 0)
	m.SerialNumber = helpers.ReadUint32(raw, 4)
	return nil
}
