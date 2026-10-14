package control

import (
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/helpers"
)

// JoinFSL3Request has the pump pair with a FreeStyle Libre 3 sensor the phone
// has activated over NFC, handing it what it needs for its own BLE session
// with the sensor.
type JoinFSL3Request struct {
	messages.Base
	// BLEAddress is the sensor's 6-byte MAC address.
	BLEAddress             []byte `pumpx2:"bleAddress"`
	SensorCode             []byte `pumpx2:"sensorCode"`
	ActivationTime         []byte `pumpx2:"activationTime"`
	FirmwareVersion        []byte `pumpx2:"firmwareVersion"`
	WarmupTime             int32  `pumpx2:"warmupTime"`
	CompressedSerialNumber []byte `pumpx2:"compressedSerialNumber"`
	// NFCUID is the sensor's 8-byte NFC UID.
	NFCUID []byte `pumpx2:"nfcUid"`
}

var joinFSL3RequestProps = messages.Props{OpCode: 120, Size: 36, Signed: true, Type: messages.Request, Characteristic: messages.Control,
	Response: "JoinFSL3Response"}

func init() {
	messages.Register("JoinFSL3Request", &JoinFSL3Request{}, joinFSL3RequestProps,
		messages.Constructor{
			Params: "byte[] bleAddress, byte[] sensorCode, byte[] activationTime, byte[] firmwareVersion, int warmupTime, " +
				"byte[] compressedSerialNumber, byte[] nfcUid",
			Func: NewJoinFSL3Request,
		},
	)
}

// NewJoinFSL3Request builds the request. It fails unless bleAddress is 6
// bytes, sensorCode, activationTime and firmwareVersion 4, compressedSerialNumber
// 9 and nfcUID 8. WarmupTime keeps only warmupTime's low byte.
func NewJoinFSL3Request(bleAddress, sensorCode, activationTime, firmwareVersion []byte, warmupTime int32,
	compressedSerialNumber, nfcUID []byte) (*JoinFSL3Request, error) {
	if len(bleAddress) != 6 || len(sensorCode) != 4 || len(activationTime) != 4 || len(firmwareVersion) != 4 ||
		len(compressedSerialNumber) != 9 || len(nfcUID) != 8 {
		return nil, messages.Invalid("JoinFSL3Request: a field of the wrong length")
	}
	m := &JoinFSL3Request{}
	if err := m.Parse(helpers.Combine(bleAddress, sensorCode, activationTime, firmwareVersion, []byte{byte(warmupTime)}, compressedSerialNumber, nfcUID)); err != nil {
		return nil, err
	}
	return m, nil
}

// Parse implements messages.Message.
func (m *JoinFSL3Request) Parse(raw []byte) error {
	raw = messages.RemoveSignedRequestHmacBytes(joinFSL3RequestProps, raw)
	if len(raw) != joinFSL3RequestProps.Size {
		return messages.InvalidLength("JoinFSL3Request", len(raw), joinFSL3RequestProps.Size)
	}
	m.SetCargo(raw)
	m.BLEAddress = helpers.CopyOfRange(raw, 0, 6)
	m.SensorCode = helpers.CopyOfRange(raw, 6, 10)
	m.ActivationTime = helpers.CopyOfRange(raw, 10, 14)
	m.FirmwareVersion = helpers.CopyOfRange(raw, 14, 18)
	m.WarmupTime = int32(raw[18])
	m.CompressedSerialNumber = helpers.CopyOfRange(raw, 19, 28)
	m.NFCUID = helpers.CopyOfRange(raw, 28, 36)
	return nil
}
