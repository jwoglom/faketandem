package control

import (
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/currentstatus"
)

// SetSensorTypeRequest sets the kind of CGM sensor the pump pairs with,
// which a change of sensor kind needs before pairing. CgmSensorType is a
// currentstatus.CgmStatusV2SensorType ID.
type SetSensorTypeRequest struct {
	messages.Base
	CgmSensorType int32 `pumpx2:"cgmSensorType"`
}

var setSensorTypeRequestProps = messages.Props{OpCode: -64, Size: 1, Signed: true, Type: messages.Request, Characteristic: messages.Control,
	Response: "SetSensorTypeResponse"}

func init() {
	messages.Register("SetSensorTypeRequest", &SetSensorTypeRequest{}, setSensorTypeRequestProps,
		messages.Constructor{Params: "CgmStatusV2Response.CgmSensorType sensorType", Func: NewSetSensorTypeRequest},
		messages.Constructor{Params: "int cgmSensorType", Func: NewSetSensorTypeRequestFromID},
	)
}

// NewSetSensorTypeRequest builds the request.
func NewSetSensorTypeRequest(sensorType currentstatus.CgmStatusV2SensorType) *SetSensorTypeRequest {
	return NewSetSensorTypeRequestFromID(sensorType.ID())
}

// NewSetSensorTypeRequestFromID builds the request from a sensor type ID.
func NewSetSensorTypeRequestFromID(cgmSensorType int32) *SetSensorTypeRequest {
	m := &SetSensorTypeRequest{CgmSensorType: cgmSensorType}
	m.SetCargo([]byte{byte(cgmSensorType)})
	return m
}

// Parse implements messages.Message.
func (m *SetSensorTypeRequest) Parse(raw []byte) error {
	raw = messages.RemoveSignedRequestHmacBytes(setSensorTypeRequestProps, raw)
	if len(raw) != setSensorTypeRequestProps.Size {
		return messages.InvalidLength("SetSensorTypeRequest", len(raw), setSensorTypeRequestProps.Size)
	}
	m.SetCargo(raw)
	m.CgmSensorType = int32(raw[0])
	return nil
}

// SensorType is CgmSensorType as a sensor type, Unknown for an ID that is
// none.
func (m *SetSensorTypeRequest) SensorType() currentstatus.CgmStatusV2SensorType {
	return currentstatus.CgmStatusV2SensorTypeFromID(m.CgmSensorType)
}
