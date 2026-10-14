package currentstatus

import (
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/helpers"
)

// PumpFeaturesV2Request asks for one of the pump's feature bitmasks, by its
// SupportedFeatureIndex ID, the payload's one byte. pumpX2's no-argument
// constructor asks for CONTROL_FEATURES.
type PumpFeaturesV2Request struct {
	messages.Base
}

var pumpFeaturesV2RequestProps = messages.Props{OpCode: -96, Size: 1, Type: messages.Request, MinAPI: "API_V2_5", Response: "PumpFeaturesV2Response"}

func init() {
	messages.Register("PumpFeaturesV2Request", &PumpFeaturesV2Request{}, pumpFeaturesV2RequestProps,
		messages.Constructor{Params: "PumpFeaturesV2Response.SupportedFeatureIndex supportedFeature", Func: NewPumpFeaturesV2Request},
		messages.Constructor{Params: "int input", Func: NewPumpFeaturesV2RequestFromID},
	)
}

// NewPumpFeaturesV2Request asks for the bitmask supportedFeature.
func NewPumpFeaturesV2Request(supportedFeature SupportedFeatureIndex) *PumpFeaturesV2Request {
	return NewPumpFeaturesV2RequestFromID(supportedFeature.ID())
}

// NewPumpFeaturesV2RequestFromID asks for the bitmask with ID input.
func NewPumpFeaturesV2RequestFromID(input int32) *PumpFeaturesV2Request {
	m := &PumpFeaturesV2Request{}
	m.SetCargo(helpers.FirstByteLittleEndian(input))
	return m
}

// Parse implements messages.Message.
func (m *PumpFeaturesV2Request) Parse(raw []byte) error {
	if len(raw) != pumpFeaturesV2RequestProps.Size {
		return messages.InvalidLength("PumpFeaturesV2Request", len(raw), pumpFeaturesV2RequestProps.Size)
	}
	m.SetCargo(raw)
	return nil
}
