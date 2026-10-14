package control

import (
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/helpers"
)

// SetConfigurableFeatureRequest turns the pump feature at (GroupIndex,
// FeatureIndex) on or off. A t:slim X2 sends it to enable CGM support when
// setting up a FreeStyle Libre 3; the Mobi does not use it. Nothing reads a
// feature's state back. It is not SetRuntimeConfigRequest.
type SetConfigurableFeatureRequest struct {
	messages.Base
	GroupIndex   int32 `pumpx2:"groupIndex"`
	FeatureIndex int32 `pumpx2:"featureIndex"`
	FeatureState bool  `pumpx2:"featureState"`
}

var setConfigurableFeatureRequestProps = messages.Props{OpCode: 116, Size: 3, Signed: true, Type: messages.Request, Characteristic: messages.Control,
	Response: "SetConfigurableFeatureResponse"}

func init() {
	messages.Register("SetConfigurableFeatureRequest", &SetConfigurableFeatureRequest{}, setConfigurableFeatureRequestProps,
		messages.Constructor{Params: "int groupIndex, int featureIndex, boolean featureState", Func: NewSetConfigurableFeatureRequest},
	)
}

// NewSetConfigurableFeatureRequest builds the request.
func NewSetConfigurableFeatureRequest(groupIndex, featureIndex int32, featureState bool) *SetConfigurableFeatureRequest {
	m := &SetConfigurableFeatureRequest{GroupIndex: groupIndex, FeatureIndex: featureIndex, FeatureState: featureState}
	m.SetCargo([]byte{byte(groupIndex), byte(featureIndex), helpers.Bool(featureState)})
	return m
}

// Parse implements messages.Message.
func (m *SetConfigurableFeatureRequest) Parse(raw []byte) error {
	raw = messages.RemoveSignedRequestHmacBytes(setConfigurableFeatureRequestProps, raw)
	if len(raw) != setConfigurableFeatureRequestProps.Size {
		return messages.InvalidLength("SetConfigurableFeatureRequest", len(raw), setConfigurableFeatureRequestProps.Size)
	}
	m.SetCargo(raw)
	m.GroupIndex = int32(raw[0])
	m.FeatureIndex = int32(raw[1])
	m.FeatureState = raw[2] != 0
	return nil
}
