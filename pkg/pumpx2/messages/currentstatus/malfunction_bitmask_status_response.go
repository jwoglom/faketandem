package currentstatus

import (
	"fmt"
	"math/big"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/helpers"
)

// MalfunctionBitmaskStatusResponse is the pump's active malfunctions: a
// 64-bit mask with a MalfunctionType at each bit. pumpX2 also reads the mask
// as two 32-bit codes, CodeA (low) and CodeB (high).
type MalfunctionBitmaskStatusResponse struct {
	messages.Base
	Bitmask      *big.Int          `pumpx2:"bitmask"`
	Malfunctions []MalfunctionType `pumpx2:"malfunctions"`
}

var malfunctionBitmaskStatusResponseProps = messages.Props{OpCode: 119, Size: 8, Type: messages.Response, Request: "MalfunctionStatusRequest"}

func init() {
	messages.Register("MalfunctionBitmaskStatusResponse", &MalfunctionBitmaskStatusResponse{}, malfunctionBitmaskStatusResponseProps,
		messages.Constructor{Params: "long codeA, long codeB", Func: NewMalfunctionBitmaskStatusResponseFromCodes},
		messages.Constructor{Params: "BigInteger bitmask", Func: NewMalfunctionBitmaskStatusResponse},
	)
}

// NewMalfunctionBitmaskStatusResponse is the response with the low 64 bits
// of bitmask.
func NewMalfunctionBitmaskStatusResponse(bitmask *big.Int) *MalfunctionBitmaskStatusResponse {
	m := &MalfunctionBitmaskStatusResponse{}
	m.decode(helpers.ToUint64(helpers.LongValue(bitmask)))
	return m
}

// NewMalfunctionBitmaskStatusResponseFromCodes is the response whose mask's
// low 32 bits are codeA and high 32 bits codeB.
func NewMalfunctionBitmaskStatusResponseFromCodes(codeA, codeB int64) *MalfunctionBitmaskStatusResponse {
	m := &MalfunctionBitmaskStatusResponse{}
	m.decode(helpers.Combine(helpers.ToUint32(codeA), helpers.ToUint32(codeB)))
	return m
}

// Parse implements messages.Message.
func (m *MalfunctionBitmaskStatusResponse) Parse(raw []byte) error {
	if len(raw) != malfunctionBitmaskStatusResponseProps.Size {
		return messages.InvalidLength("MalfunctionBitmaskStatusResponse", len(raw), malfunctionBitmaskStatusResponseProps.Size)
	}
	m.decode(raw)
	return nil
}

func (m *MalfunctionBitmaskStatusResponse) decode(raw []byte) {
	m.SetCargo(raw)
	m.Bitmask = helpers.ReadUint64(raw, 0)
	m.Malfunctions = MalfunctionTypeFromBitmask(m.Bitmask)
}

// CodeA is the low 32 bits of the mask.
func (m *MalfunctionBitmaskStatusResponse) CodeA() int64 { return helpers.ReadUint32(m.Cargo(), 0) }

// CodeB is the high 32 bits of the mask.
func (m *MalfunctionBitmaskStatusResponse) CodeB() int64 { return helpers.ReadUint32(m.Cargo(), 4) }

// HasMalfunctionBit reports whether bit, 0 to 63, is set in the mask.
func (m *MalfunctionBitmaskStatusResponse) HasMalfunctionBit(bit int32) bool {
	return bit >= 0 && bit < 64 && m.Bitmask != nil && m.Bitmask.Bit(int(bit)) != 0
}

// HasMalfunction reports whether malfunction is active.
func (m *MalfunctionBitmaskStatusResponse) HasMalfunction(malfunction MalfunctionType) bool {
	return m.HasMalfunctionBit(int32(malfunction))
}

// Size is how many malfunctions are active.
func (m *MalfunctionBitmaskStatusResponse) Size() int { return len(m.Malfunctions) }

// NotificationIDs is empty: pumpX2 describes no malfunction.
func (m *MalfunctionBitmaskStatusResponse) NotificationIDs() []int32 { return []int32{} }

// MalfunctionType is a pump malfunction, as its bit in the malfunction
// mask.
type MalfunctionType int32

// The malfunctions, by their bits.
const (
	MalfunctionTypeSoftware         MalfunctionType = 0
	MalfunctionTypeCPUCore          MalfunctionType = 1
	MalfunctionTypeArmMspCom        MalfunctionType = 2
	MalfunctionTypeSensor           MalfunctionType = 3
	MalfunctionTypeLipo             MalfunctionType = 4
	MalfunctionTypeTouchscreen      MalfunctionType = 5
	MalfunctionTypeNvm              MalfunctionType = 6
	MalfunctionTypeDisplay          MalfunctionType = 7
	MalfunctionTypeMsp              MalfunctionType = 8
	MalfunctionTypeMotor            MalfunctionType = 9
	MalfunctionTypeExternalBins     MalfunctionType = 10
	MalfunctionTypeSwInit           MalfunctionType = 11
	MalfunctionTypeVibe             MalfunctionType = 12
	MalfunctionTypePeriphPower      MalfunctionType = 13
	MalfunctionTypeP2               MalfunctionType = 14
	MalfunctionTypeDatalog          MalfunctionType = 15
	MalfunctionTypeSpeaker          MalfunctionType = 16
	MalfunctionTypeMspSupply        MalfunctionType = 17
	MalfunctionTypeCalData          MalfunctionType = 18
	MalfunctionTypeBtle             MalfunctionType = 19
	MalfunctionTypeAp               MalfunctionType = 20
	MalfunctionTypeRtc              MalfunctionType = 21
	MalfunctionTypeArmBLECom        MalfunctionType = 22
	MalfunctionTypeOvertravelStall  MalfunctionType = 23
	MalfunctionTypeUndertravelStall MalfunctionType = 24
	MalfunctionTypePushoffStall     MalfunctionType = 25
	MalfunctionTypeHwCryptoFail     MalfunctionType = 26
)

// MalfunctionTypes are the malfunctions in pumpX2's declaration order.
var MalfunctionTypes = []MalfunctionType{
	MalfunctionTypeSoftware, MalfunctionTypeCPUCore, MalfunctionTypeArmMspCom, MalfunctionTypeSensor, MalfunctionTypeLipo,
	MalfunctionTypeTouchscreen, MalfunctionTypeNvm, MalfunctionTypeDisplay, MalfunctionTypeMsp, MalfunctionTypeMotor,
	MalfunctionTypeExternalBins, MalfunctionTypeSwInit, MalfunctionTypeVibe, MalfunctionTypePeriphPower,
	MalfunctionTypeP2, MalfunctionTypeDatalog, MalfunctionTypeSpeaker, MalfunctionTypeMspSupply, MalfunctionTypeCalData,
	MalfunctionTypeBtle, MalfunctionTypeAp, MalfunctionTypeRtc, MalfunctionTypeArmBLECom, MalfunctionTypeOvertravelStall,
	MalfunctionTypeUndertravelStall, MalfunctionTypePushoffStall, MalfunctionTypeHwCryptoFail,
}

var malfunctionTypeNames = []string{
	"SOFTWARE", "CPU_CORE", "ARM_MSP_COM", "SENSOR", "LIPO", "TOUCHSCREEN", "NVM", "DISPLAY", "MSP", "MOTOR",
	"EXTERNAL_BINS", "SW_INIT", "VIBE", "PERIPH_POWER", "P2", "DATALOG", "SPEAKER", "MSP_SUPPLY", "CAL_DATA", "BTLE",
	"AP", "RTC", "ARM_BLE_COM", "OVERTRAVEL_STALL", "UNDERTRAVEL_STALL", "PUSHOFF_STALL", "HW_CRYPTO_FAIL",
}

// MalfunctionTypeFromBitmask is the malfunctions whose bits bitmask sets, in
// declaration order; never nil.
func MalfunctionTypeFromBitmask(bitmask *big.Int) []MalfunctionType {
	out := []MalfunctionType{}
	for _, t := range MalfunctionTypes {
		if bitmask.Bit(int(t)) != 0 {
			out = append(out, t)
		}
	}
	return out
}

func (t MalfunctionType) String() string {
	if t >= 0 && int(t) < len(malfunctionTypeNames) {
		return malfunctionTypeNames[t]
	}
	return fmt.Sprintf("MalfunctionType(%d)", int32(t))
}
