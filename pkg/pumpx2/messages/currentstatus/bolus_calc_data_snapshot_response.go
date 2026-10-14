package currentstatus

import (
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/helpers"
)

// BolusCalcDataSnapshotResponse is what the bolus calculator starts from.
// Bytes 26 to 36 and 38 to 45 are not decoded.
type BolusCalcDataSnapshotResponse struct {
	messages.Base
	// IsUnacked is true when the pump could not gather the data, or it is
	// blank.
	IsUnacked bool `pumpx2:"isUnacked"`
	// CorrectionFactor is, despite its name, the CGM's current BG in mg/dL,
	// when there is one.
	CorrectionFactor int32 `pumpx2:"correctionFactor"`
	// IOB is in milliunits.
	IOB int64 `pumpx2:"iob"`
	// CartridgeRemainingInsulin is in units.
	CartridgeRemainingInsulin int32 `pumpx2:"cartridgeRemainingInsulin"`
	// TargetBG is the profile's target, in mg/dL.
	TargetBG int32 `pumpx2:"targetBg"`
	// ISF is the profile's correction factor: mg/dL per unit.
	ISF              int32 `pumpx2:"isf"`
	CarbEntryEnabled bool  `pumpx2:"carbEntryEnabled"`
	// CarbRatio is grams per unit, times 1000.
	CarbRatio int64 `pumpx2:"carbRatio"`
	// MaxBolusAmount is the most one bolus may deliver, in milliunits.
	MaxBolusAmount int32 `pumpx2:"maxBolusAmount"`
	// MaxBolusHourlyTotal is the most boluses may deliver in an hour, or 0
	// when not set.
	MaxBolusHourlyTotal    int64 `pumpx2:"maxBolusHourlyTotal"`
	MaxBolusEventsExceeded bool  `pumpx2:"maxBolusEventsExceeded"`
	MaxIOBEventsExceeded   bool  `pumpx2:"maxIobEventsExceeded"`
	// IsAutopopAllowed is whether the calculator may fill in the CGM's BG
	// (CorrectionFactor) to add a correction.
	IsAutopopAllowed bool `pumpx2:"isAutopopAllowed"`
}

var bolusCalcDataSnapshotResponseProps = messages.Props{OpCode: 115, Size: 46, Type: messages.Response, MinAPI: "API_V2_5", Request: "BolusCalcDataSnapshotRequest"}

func init() {
	messages.Register("BolusCalcDataSnapshotResponse", &BolusCalcDataSnapshotResponse{}, bolusCalcDataSnapshotResponseProps,
		messages.Constructor{
			Params: "boolean isUnacked, int correctionFactor, long iob, int cartridgeRemainingInsulin, int targetBg, int isf, boolean carbEntryEnabled, " +
				"long carbRatio, int maxBolusAmount, long maxBolusHourlyTotal, boolean maxBolusEventsExceeded, boolean maxIobEventsExceeded, " +
				"boolean isAutopopAllowed, byte[] unknown11bytes, byte[] unknown8bytes",
			Func: NewBolusCalcDataSnapshotResponseWithUnknownBytes,
		},
		messages.Constructor{
			Params: "boolean isUnacked, int correctionFactor, long iob, int cartridgeRemainingInsulin, int targetBg, int isf, boolean carbEntryEnabled, " +
				"long carbRatio, int maxBolusAmount, long maxBolusHourlyTotal, boolean maxBolusEventsExceeded, boolean maxIobEventsExceeded, " +
				"boolean isAutopopAllowed",
			Func: NewBolusCalcDataSnapshotResponse,
		},
	)
}

// NewBolusCalcDataSnapshotResponse builds the response with its undecoded
// bytes zero.
func NewBolusCalcDataSnapshotResponse(isUnacked bool, correctionFactor int32, iob int64, cartridgeRemainingInsulin, targetBG, isf int32,
	carbEntryEnabled bool, carbRatio int64, maxBolusAmount int32, maxBolusHourlyTotal int64, maxBolusEventsExceeded, maxIOBEventsExceeded,
	isAutopopAllowed bool) *BolusCalcDataSnapshotResponse {
	return NewBolusCalcDataSnapshotResponseWithUnknownBytes(isUnacked, correctionFactor, iob, cartridgeRemainingInsulin, targetBG, isf,
		carbEntryEnabled, carbRatio, maxBolusAmount, maxBolusHourlyTotal, maxBolusEventsExceeded, maxIOBEventsExceeded, isAutopopAllowed,
		make([]byte, 11), make([]byte, 8))
}

// NewBolusCalcDataSnapshotResponseWithUnknownBytes builds the response with
// the undecoded bytes: unknown11bytes before IsAutopopAllowed and
// unknown8bytes after it. pumpX2 does not check their lengths.
func NewBolusCalcDataSnapshotResponseWithUnknownBytes(isUnacked bool, correctionFactor int32, iob int64, cartridgeRemainingInsulin, targetBG, isf int32,
	carbEntryEnabled bool, carbRatio int64, maxBolusAmount int32, maxBolusHourlyTotal int64, maxBolusEventsExceeded, maxIOBEventsExceeded,
	isAutopopAllowed bool, unknown11bytes, unknown8bytes []byte) *BolusCalcDataSnapshotResponse {
	m := &BolusCalcDataSnapshotResponse{
		IsUnacked:                 isUnacked,
		CorrectionFactor:          correctionFactor,
		IOB:                       iob,
		CartridgeRemainingInsulin: cartridgeRemainingInsulin,
		TargetBG:                  targetBG,
		ISF:                       isf,
		CarbEntryEnabled:          carbEntryEnabled,
		CarbRatio:                 carbRatio,
		MaxBolusAmount:            maxBolusAmount,
		MaxBolusHourlyTotal:       maxBolusHourlyTotal,
		MaxBolusEventsExceeded:    maxBolusEventsExceeded,
		MaxIOBEventsExceeded:      maxIOBEventsExceeded,
		IsAutopopAllowed:          isAutopopAllowed,
	}
	m.SetCargo(helpers.Combine(
		[]byte{helpers.Bool(isUnacked)},
		helpers.FirstTwoBytesLittleEndian(correctionFactor),
		helpers.ToUint32(iob),
		helpers.FirstTwoBytesLittleEndian(cartridgeRemainingInsulin),
		helpers.FirstTwoBytesLittleEndian(targetBG),
		helpers.FirstTwoBytesLittleEndian(isf),
		[]byte{helpers.Bool(carbEntryEnabled)},
		helpers.ToUint32(carbRatio),
		helpers.FirstTwoBytesLittleEndian(maxBolusAmount),
		helpers.ToUint32(maxBolusHourlyTotal),
		[]byte{helpers.Bool(maxBolusEventsExceeded), helpers.Bool(maxIOBEventsExceeded)},
		unknown11bytes,
		[]byte{helpers.Bool(isAutopopAllowed)},
		unknown8bytes,
	))
	return m
}

// Parse implements messages.Message.
func (m *BolusCalcDataSnapshotResponse) Parse(raw []byte) error {
	if len(raw) != bolusCalcDataSnapshotResponseProps.Size {
		return messages.InvalidLength("BolusCalcDataSnapshotResponse", len(raw), bolusCalcDataSnapshotResponseProps.Size)
	}
	m.SetCargo(raw)
	m.IsUnacked = raw[0] != 0
	m.CorrectionFactor = helpers.ReadShort(raw, 1)
	m.IOB = helpers.ReadUint32(raw, 3)
	m.CartridgeRemainingInsulin = helpers.ReadShort(raw, 7)
	m.TargetBG = helpers.ReadShort(raw, 9)
	m.ISF = helpers.ReadShort(raw, 11)
	m.CarbEntryEnabled = raw[13] != 0
	m.CarbRatio = helpers.ReadUint32(raw, 14)
	m.MaxBolusAmount = helpers.ReadShort(raw, 18)
	m.MaxBolusHourlyTotal = helpers.ReadUint32(raw, 20)
	m.MaxBolusEventsExceeded = raw[24] != 0
	m.MaxIOBEventsExceeded = raw[25] != 0
	m.IsAutopopAllowed = raw[37] != 0
	return nil
}
