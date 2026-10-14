package currentstatus

import (
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/helpers"
)

// CurrentBatteryV2Response is CurrentBatteryV1Response for a pump past API
// 2.1, with whether it is charging (ChargingStatus 1). Unknown1 to Unknown4
// are not understood and have always read 0.
type CurrentBatteryV2Response struct {
	messages.Base
	CurrentBatteryAbc int32 `pumpx2:"currentBatteryAbc"`
	CurrentBatteryIbc int32 `pumpx2:"currentBatteryIbc"`
	ChargingStatus    int32 `pumpx2:"chargingStatus"`
	Unknown1          int32 `pumpx2:"unknown1"`
	Unknown2          int32 `pumpx2:"unknown2"`
	Unknown3          int32 `pumpx2:"unknown3"`
	Unknown4          int32 `pumpx2:"unknown4"`
}

var currentBatteryV2ResponseProps = messages.Props{OpCode: -111, Size: 11, Type: messages.Response, MinAPI: "API_V2_5", Request: "CurrentBatteryV2Request"}

func init() {
	messages.Register("CurrentBatteryV2Response", &CurrentBatteryV2Response{}, currentBatteryV2ResponseProps,
		messages.Constructor{
			Params: "int currentBatteryAbc, int currentBatteryIbc, int chargingStatus, int unknown1, int unknown2, int unknown3, int unknown4",
			Func:   NewCurrentBatteryV2Response,
		},
	)
}

// NewCurrentBatteryV2Response builds the response.
func NewCurrentBatteryV2Response(currentBatteryAbc, currentBatteryIbc, chargingStatus, unknown1, unknown2, unknown3, unknown4 int32) *CurrentBatteryV2Response {
	m := &CurrentBatteryV2Response{
		CurrentBatteryAbc: currentBatteryAbc,
		CurrentBatteryIbc: currentBatteryIbc,
		ChargingStatus:    chargingStatus,
		Unknown1:          unknown1,
		Unknown2:          unknown2,
		Unknown3:          unknown3,
		Unknown4:          unknown4,
	}
	m.SetCargo(helpers.Combine(
		[]byte{byte(currentBatteryAbc), byte(currentBatteryIbc), byte(chargingStatus)},
		helpers.FirstTwoBytesLittleEndian(unknown1),
		helpers.FirstTwoBytesLittleEndian(unknown2),
		helpers.FirstTwoBytesLittleEndian(unknown3),
		helpers.FirstTwoBytesLittleEndian(unknown4),
	))
	return m
}

// Parse implements messages.Message.
func (m *CurrentBatteryV2Response) Parse(raw []byte) error {
	if len(raw) != currentBatteryV2ResponseProps.Size {
		return messages.InvalidLength("CurrentBatteryV2Response", len(raw), currentBatteryV2ResponseProps.Size)
	}
	m.SetCargo(raw)
	m.CurrentBatteryAbc = int32(int8(raw[0]))
	m.CurrentBatteryIbc = int32(int8(raw[1]))
	m.ChargingStatus = int32(int8(raw[2]))
	m.Unknown1 = helpers.ReadShort(raw, 3)
	m.Unknown2 = helpers.ReadShort(raw, 5)
	m.Unknown3 = helpers.ReadShort(raw, 7)
	m.Unknown4 = helpers.ReadShort(raw, 9)
	return nil
}

// BatteryPercent is the charge the pump displays.
func (m *CurrentBatteryV2Response) BatteryPercent() int32 { return m.CurrentBatteryIbc }

// Charging reports whether the pump is charging.
func (m *CurrentBatteryV2Response) Charging() bool { return m.ChargingStatus == 1 }
