package currentstatus

import (
	"fmt"
	"math/big"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/helpers"
)

// PumpFeaturesV1Response is the features the pump supports, as a 64-bit
// bitmask of which pumpX2 names only some bits.
type PumpFeaturesV1Response struct {
	messages.Base
	// IntMap is the whole bitmask, unnamed bits included; pumpX2 leaves it
	// out of toString.
	IntMap   *big.Int
	Features []PumpFeatureType `pumpx2:"features"`
}

var pumpFeaturesV1ResponseProps = messages.Props{OpCode: 79, Size: 8, Type: messages.Response, Request: "PumpFeaturesV1Request"}

func init() {
	messages.Register("PumpFeaturesV1Response", &PumpFeaturesV1Response{}, pumpFeaturesV1ResponseProps,
		messages.Constructor{Params: "byte[] raw", Func: NewPumpFeaturesV1ResponseFromCargo},
		messages.Constructor{Params: "BigInteger intMap", Func: NewPumpFeaturesV1Response},
	)
}

// NewPumpFeaturesV1ResponseFromCargo parses a response's payload.
func NewPumpFeaturesV1ResponseFromCargo(raw []byte) (*PumpFeaturesV1Response, error) {
	m := &PumpFeaturesV1Response{}
	return m, m.Parse(raw)
}

// NewPumpFeaturesV1Response builds the response from its bitmask, whose low
// 64 bits it sends. It fails on a nil bitmask.
func NewPumpFeaturesV1Response(intMap *big.Int) (*PumpFeaturesV1Response, error) {
	if intMap == nil {
		return nil, messages.Invalid("PumpFeaturesV1Response: no intMap")
	}
	m := &PumpFeaturesV1Response{IntMap: intMap, Features: PumpFeatureTypeFromBitmask(intMap)}
	m.SetCargo(helpers.ToUint64(helpers.LongValue(intMap)))
	return m, nil
}

// Parse implements messages.Message.
func (m *PumpFeaturesV1Response) Parse(raw []byte) error {
	if len(raw) != pumpFeaturesV1ResponseProps.Size {
		return messages.InvalidLength("PumpFeaturesV1Response", len(raw), pumpFeaturesV1ResponseProps.Size)
	}
	m.SetCargo(raw)
	m.IntMap = helpers.ReadUint64(raw, 0)
	m.Features = PumpFeatureTypeFromBitmask(m.IntMap)
	return nil
}

// PumpFeatureType is a feature the pump supports, as its bit of
// PumpFeaturesV1Response's bitmask and of PumpFeaturesV2Response's for
// MAIN_FEATURES.
type PumpFeatureType int32

// The features pumpX2 names, by their bits.
const (
	PumpFeatureDexcomG5Supported             PumpFeatureType = 1 << 0
	PumpFeatureDexcomG6Supported             PumpFeatureType = 1 << 1
	PumpFeatureBasalIQSupported              PumpFeatureType = 1 << 2
	PumpFeatureControlIQSupported            PumpFeatureType = 1 << 10
	PumpFeatureWombatSupported               PumpFeatureType = 1 << 16
	PumpFeatureBasalLimitSupported           PumpFeatureType = 1 << 18
	PumpFeatureControlIQProSupported         PumpFeatureType = 1 << 23
	PumpFeatureAutoPopSupported              PumpFeatureType = 1 << 25
	PumpFeatureBLEPumpControlSupported       PumpFeatureType = 1 << 28
	PumpFeaturePumpSettingsInIDPGUISupported PumpFeatureType = 1 << 29
)

// PumpFeatureTypes are the features in pumpX2's declaration order.
var PumpFeatureTypes = []PumpFeatureType{
	PumpFeatureDexcomG5Supported, PumpFeatureDexcomG6Supported, PumpFeatureBasalIQSupported, PumpFeatureControlIQSupported,
	PumpFeatureWombatSupported, PumpFeatureBasalLimitSupported, PumpFeatureControlIQProSupported, PumpFeatureAutoPopSupported,
	PumpFeatureBLEPumpControlSupported, PumpFeaturePumpSettingsInIDPGUISupported,
}

var pumpFeatureTypeNames = map[PumpFeatureType]string{
	PumpFeatureDexcomG5Supported:             "DEXCOM_G5_SUPPORTED",
	PumpFeatureDexcomG6Supported:             "DEXCOM_G6_SUPPORTED",
	PumpFeatureBasalIQSupported:              "BASAL_IQ_SUPPORTED",
	PumpFeatureControlIQSupported:            "CONTROL_IQ_SUPPORTED",
	PumpFeatureWombatSupported:               "WOMBAT_SUPPORTED",
	PumpFeatureBasalLimitSupported:           "BASAL_LIMIT_SUPPORTED",
	PumpFeatureControlIQProSupported:         "CONTROL_IQ_PRO_SUPPORTED",
	PumpFeatureAutoPopSupported:              "AUTO_POP_SUPPORTED",
	PumpFeatureBLEPumpControlSupported:       "BLE_PUMP_CONTROL_SUPPORTED",
	PumpFeaturePumpSettingsInIDPGUISupported: "PUMP_SETTINGS_IN_IDP_GUI_SUPPORTED",
}

// PumpFeatureTypeFromBitmask is the named features whose bits mask sets, in
// declaration order; never nil. A negative mask is two's complement, as a
// BigInteger's is.
func PumpFeatureTypeFromBitmask(mask *big.Int) []PumpFeatureType {
	out := []PumpFeatureType{}
	bit := new(big.Int)
	for _, f := range PumpFeatureTypes {
		if bit.And(mask, big.NewInt(int64(f))).Sign() != 0 {
			out = append(out, f)
		}
	}
	return out
}

// PumpFeatureTypeToBitmask is pumpX2's PumpFeatureType.build: the bitmask
// with the features' bits set.
func PumpFeatureTypeToBitmask(features ...PumpFeatureType) *big.Int {
	mask := new(big.Int)
	for _, f := range features {
		mask.Or(mask, big.NewInt(int64(f)))
	}
	return mask
}

func (f PumpFeatureType) String() string {
	if name, ok := pumpFeatureTypeNames[f]; ok {
		return name
	}
	return fmt.Sprintf("PumpFeatureType(%d)", int32(f))
}
