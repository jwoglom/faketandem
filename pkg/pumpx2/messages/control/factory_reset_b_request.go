package control

import (
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/helpers"
)

// FactoryResetBRequest factory-resets the pump whose serial number it names,
// optionally putting it into shelf mode afterwards.
type FactoryResetBRequest struct {
	messages.Base
	Key             int64 `pumpx2:"key"`
	SerialNumber    int64 `pumpx2:"serialNumber"`
	EnableShelfMode bool  `pumpx2:"enableShelfMode"`
}

var factoryResetBRequestProps = messages.Props{OpCode: 124, Size: 9, Signed: true, Type: messages.Request, Characteristic: messages.Control,
	Response: "FactoryResetBResponse"}

func init() {
	messages.Register("FactoryResetBRequest", &FactoryResetBRequest{}, factoryResetBRequestProps,
		messages.Constructor{Params: "long key, long serialNumber, boolean enableShelfMode", Func: NewFactoryResetBRequest},
	)
}

// NewFactoryResetBRequest builds the request.
func NewFactoryResetBRequest(key, serialNumber int64, enableShelfMode bool) *FactoryResetBRequest {
	m := &FactoryResetBRequest{Key: key, SerialNumber: serialNumber, EnableShelfMode: enableShelfMode}
	m.SetCargo(helpers.Combine(helpers.ToUint32(key), helpers.ToUint32(serialNumber), []byte{helpers.Bool(enableShelfMode)}))
	return m
}

// Parse implements messages.Message. pumpX2 checks no length here, so a
// short payload fails only on the read past its end.
func (m *FactoryResetBRequest) Parse(raw []byte) error {
	raw = messages.RemoveSignedRequestHmacBytes(factoryResetBRequestProps, raw)
	m.SetCargo(raw)
	m.Key = helpers.ReadUint32(raw, 0)
	m.SerialNumber = helpers.ReadUint32(raw, 4)
	m.EnableShelfMode = raw[8] != 0
	return nil
}
