package currentstatus

import (
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/helpers"
)

// PumpVersionResponse is the pump's ARM and MSP software versions,
// configuration bits, serial, part and model numbers, and its own and its
// circuit board's revision and serial number. The revisions are strings of
// at most 8 bytes.
type PumpVersionResponse struct {
	messages.Base
	ArmSwVer    int64  `pumpx2:"armSwVer"`
	MspSwVer    int64  `pumpx2:"mspSwVer"`
	ConfigABits int64  `pumpx2:"configABits"`
	ConfigBBits int64  `pumpx2:"configBBits"`
	SerialNum   int64  `pumpx2:"serialNum"`
	PartNum     int64  `pumpx2:"partNum"`
	PumpRev     string `pumpx2:"pumpRev"`
	PcbaSN      int64  `pumpx2:"pcbaSN"`
	PcbaRev     string `pumpx2:"pcbaRev"`
	ModelNum    int64  `pumpx2:"modelNum"`
}

var pumpVersionResponseProps = messages.Props{OpCode: 85, Size: 48, Type: messages.Response, Request: "PumpVersionRequest"}

func init() {
	messages.Register("PumpVersionResponse", &PumpVersionResponse{}, pumpVersionResponseProps,
		messages.Constructor{
			Params: "long armSwVer, long mspSwVer, long configABits, long configBBits, long serialNum, long partNum, String pumpRev, long pcbaSN, " +
				"String pcbaRev, long modelNum",
			Func: NewPumpVersionResponse,
		},
	)
}

// NewPumpVersionResponse builds the response. It fails when a revision
// does not fit its 8 bytes.
func NewPumpVersionResponse(armSwVer, mspSwVer, configABits, configBBits, serialNum, partNum int64, pumpRev string, pcbaSN int64,
	pcbaRev string, modelNum int64) (*PumpVersionResponse, error) {
	pumpRevBytes, err := helpers.WriteString(pumpRev, 8)
	if err != nil {
		return nil, messages.Invalid("PumpVersionResponse: pumpRev: %v", err)
	}
	pcbaRevBytes, err := helpers.WriteString(pcbaRev, 8)
	if err != nil {
		return nil, messages.Invalid("PumpVersionResponse: pcbaRev: %v", err)
	}
	m := &PumpVersionResponse{
		ArmSwVer:    armSwVer,
		MspSwVer:    mspSwVer,
		ConfigABits: configABits,
		ConfigBBits: configBBits,
		SerialNum:   serialNum,
		PartNum:     partNum,
		PumpRev:     pumpRev,
		PcbaSN:      pcbaSN,
		PcbaRev:     pcbaRev,
		ModelNum:    modelNum,
	}
	m.SetCargo(helpers.Combine(
		helpers.ToUint32(armSwVer),
		helpers.ToUint32(mspSwVer),
		helpers.ToUint32(configABits),
		helpers.ToUint32(configBBits),
		helpers.ToUint32(serialNum),
		helpers.ToUint32(partNum),
		pumpRevBytes,
		helpers.ToUint32(pcbaSN),
		pcbaRevBytes,
		helpers.ToUint32(modelNum),
	))
	return m, nil
}

// Parse implements messages.Message.
func (m *PumpVersionResponse) Parse(raw []byte) error {
	if len(raw) != pumpVersionResponseProps.Size {
		return messages.InvalidLength("PumpVersionResponse", len(raw), pumpVersionResponseProps.Size)
	}
	m.SetCargo(raw)
	m.ArmSwVer = helpers.ReadUint32(raw, 0)
	m.MspSwVer = helpers.ReadUint32(raw, 4)
	m.ConfigABits = helpers.ReadUint32(raw, 8)
	m.ConfigBBits = helpers.ReadUint32(raw, 12)
	m.SerialNum = helpers.ReadUint32(raw, 16)
	m.PartNum = helpers.ReadUint32(raw, 20)
	m.PumpRev = helpers.ReadString(raw, 24, 8)
	m.PcbaSN = helpers.ReadUint32(raw, 32)
	m.PcbaRev = helpers.ReadString(raw, 36, 8)
	m.ModelNum = helpers.ReadUint32(raw, 44)
	return nil
}
