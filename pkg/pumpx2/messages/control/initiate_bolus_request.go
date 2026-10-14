package control

import (
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/helpers"
)

// Bolus volume floors InitiateBolusRequest enforces, in milliunits.
const (
	MinBolusMilliunits         = 50
	MinExtendedBolusMilliunits = 400
)

// InitiateBolusRequest starts a bolus, under the bolus ID a
// BolusPermissionRequest granted. Volumes are in milliunits, BolusBG in
// mg/dL and BolusCarbs in grams; an extended bolus delivers ExtendedVolume
// over ExtendedSeconds.
type InitiateBolusRequest struct {
	messages.Base
	TotalVolume      int64 `pumpx2:"totalVolume"`
	BolusID          int32 `pumpx2:"bolusID"`
	BolusTypeBitmask int32 `pumpx2:"bolusTypeBitmask"`
	FoodVolume       int64 `pumpx2:"foodVolume"`
	CorrectionVolume int64 `pumpx2:"correctionVolume"`
	BolusCarbs       int32 `pumpx2:"bolusCarbs"`
	BolusBG          int32 `pumpx2:"bolusBG"`
	BolusIOB         int64 `pumpx2:"bolusIOB"`
	ExtendedVolume   int64 `pumpx2:"extendedVolume"`
	ExtendedSeconds  int64 `pumpx2:"extendedSeconds"`
	Extended3        int64 `pumpx2:"extended3"`
}

var initiateBolusRequestProps = messages.Props{OpCode: -98, Size: 37, Signed: true, Type: messages.Request, Characteristic: messages.Control,
	ModifiesInsulinDelivery: true, Response: "InitiateBolusResponse"}

func init() {
	messages.Register("InitiateBolusRequest", &InitiateBolusRequest{}, initiateBolusRequestProps,
		messages.Constructor{
			Params: "long totalVolume, int bolusID, int bolusTypeBitmask, long foodVolume, long correctionVolume, int bolusCarbs, int bolusBG, long bolusIOB, " +
				"long extendedVolume, long extendedSeconds, long extended3",
			Func: NewExtendedInitiateBolusRequest,
		},
		messages.Constructor{
			Params: "long totalVolume, int bolusID, int bolusTypeBitmask, long foodVolume, long correctionVolume, int bolusCarbs, int bolusBG, long bolusIOB",
			Func:   NewInitiateBolusRequest,
		},
		messages.Constructor{Params: "byte[] raw", Func: NewInitiateBolusRequestFromCargo},
	)
}

// NewInitiateBolusRequestFromCargo parses a request's payload.
func NewInitiateBolusRequestFromCargo(raw []byte) (*InitiateBolusRequest, error) {
	m := &InitiateBolusRequest{}
	return m, m.Parse(raw)
}

// NewInitiateBolusRequest is a standard bolus.
func NewInitiateBolusRequest(totalVolume int64, bolusID, bolusTypeBitmask int32, foodVolume, correctionVolume int64,
	bolusCarbs, bolusBG int32, bolusIOB int64) (*InitiateBolusRequest, error) {
	return NewExtendedInitiateBolusRequest(totalVolume, bolusID, bolusTypeBitmask, foodVolume, correctionVolume, bolusCarbs, bolusBG, bolusIOB, 0, 0, 0)
}

// NewExtendedInitiateBolusRequest is a bolus with an extended part. It fails
// below the minimum bolus, or on a non-positive bolus ID or a negative
// volume, carb count, BG or IOB.
func NewExtendedInitiateBolusRequest(totalVolume int64, bolusID, bolusTypeBitmask int32, foodVolume, correctionVolume int64,
	bolusCarbs, bolusBG int32, bolusIOB, extendedVolume, extendedSeconds, extended3 int64) (*InitiateBolusRequest, error) {
	switch {
	case totalVolume < MinBolusMilliunits && totalVolume+extendedVolume < MinExtendedBolusMilliunits:
		return nil, messages.Invalid("InitiateBolusRequest: %d milliunits is below the minimum bolus", totalVolume)
	case bolusID <= 0:
		return nil, messages.Invalid("InitiateBolusRequest: bolus ID %d", bolusID)
	case foodVolume < 0 || correctionVolume < 0 || bolusCarbs < 0 || bolusBG < 0 || bolusIOB < 0:
		return nil, messages.Invalid("InitiateBolusRequest: a negative food or correction volume, carbs, BG or IOB")
	}
	m := &InitiateBolusRequest{
		TotalVolume:      totalVolume,
		BolusID:          bolusID,
		BolusTypeBitmask: bolusTypeBitmask,
		FoodVolume:       foodVolume,
		CorrectionVolume: correctionVolume,
		BolusCarbs:       bolusCarbs,
		BolusBG:          bolusBG,
		BolusIOB:         bolusIOB,
		ExtendedVolume:   extendedVolume,
		ExtendedSeconds:  extendedSeconds,
		Extended3:        extended3,
	}
	m.SetCargo(helpers.Combine(
		helpers.ToUint32(totalVolume),
		helpers.FirstTwoBytesLittleEndian(bolusID),
		[]byte{0, 0},
		[]byte{byte(bolusTypeBitmask)},
		helpers.ToUint32(foodVolume),
		helpers.ToUint32(correctionVolume),
		helpers.FirstTwoBytesLittleEndian(bolusCarbs),
		helpers.FirstTwoBytesLittleEndian(bolusBG),
		helpers.ToUint32(bolusIOB),
		helpers.ToUint32(extendedVolume),
		helpers.ToUint32(extendedSeconds),
		helpers.ToUint32(extended3),
	))
	return m, nil
}

// Parse implements messages.Message.
func (m *InitiateBolusRequest) Parse(raw []byte) error {
	raw = messages.RemoveSignedRequestHmacBytes(initiateBolusRequestProps, raw)
	if len(raw) != initiateBolusRequestProps.Size {
		return messages.InvalidLength("InitiateBolusRequest", len(raw), initiateBolusRequestProps.Size)
	}
	m.SetCargo(raw)
	m.TotalVolume = helpers.ReadUint32(raw, 0)
	m.BolusID = helpers.ReadShort(raw, 4)
	m.BolusTypeBitmask = int32(int8(raw[8]))
	m.FoodVolume = helpers.ReadUint32(raw, 9)
	m.CorrectionVolume = helpers.ReadUint32(raw, 13)
	m.BolusCarbs = helpers.ReadShort(raw, 17)
	m.BolusBG = helpers.ReadShort(raw, 19)
	m.BolusIOB = helpers.ReadUint32(raw, 21)
	m.ExtendedVolume = helpers.ReadUint32(raw, 25)
	m.ExtendedSeconds = helpers.ReadUint32(raw, 29)
	m.Extended3 = helpers.ReadUint32(raw, 33)
	return nil
}
