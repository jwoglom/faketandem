package currentstatus

import (
	"fmt"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
)

// CommonSoftwareInfoRequest asks for the software version and part numbers of
// one of the pump's microcontrollers.
type CommonSoftwareInfoRequest struct {
	messages.Base
	MCUTypeID int32   `pumpx2:"mcuTypeId"`
	MCUType   MCUType `pumpx2:"mcuType"`
}

var commonSoftwareInfoRequestProps = messages.Props{OpCode: -114, Size: 1, Type: messages.Request, MinAPI: "API_FUTURE", Response: "CommonSoftwareInfoResponse"}

func init() {
	messages.Register("CommonSoftwareInfoRequest", &CommonSoftwareInfoRequest{}, commonSoftwareInfoRequestProps,
		messages.Constructor{Params: "byte[] raw", Func: NewCommonSoftwareInfoRequestFromCargo},
		messages.Constructor{Params: "CommonSoftwareInfoRequest.MCUType mcuType", Func: NewCommonSoftwareInfoRequest},
		messages.Constructor{Params: "int mcuTypeId", Func: NewCommonSoftwareInfoRequestFromID},
		messages.Constructor{Func: func() *CommonSoftwareInfoRequest { return NewCommonSoftwareInfoRequest(MCUTypeARM) }},
	)
}

// NewCommonSoftwareInfoRequestFromCargo parses a request's payload.
func NewCommonSoftwareInfoRequestFromCargo(raw []byte) (*CommonSoftwareInfoRequest, error) {
	m := &CommonSoftwareInfoRequest{}
	return m, m.Parse(raw)
}

// NewCommonSoftwareInfoRequest asks after mcuType.
func NewCommonSoftwareInfoRequest(mcuType MCUType) *CommonSoftwareInfoRequest {
	return NewCommonSoftwareInfoRequestFromID(mcuType.ID())
}

// NewCommonSoftwareInfoRequestFromID asks after the microcontroller with wire
// ID mcuTypeID.
func NewCommonSoftwareInfoRequestFromID(mcuTypeID int32) *CommonSoftwareInfoRequest {
	m := &CommonSoftwareInfoRequest{MCUTypeID: mcuTypeID, MCUType: MCUTypeFromID(mcuTypeID)}
	m.SetCargo([]byte{byte(mcuTypeID)})
	return m
}

// Parse implements messages.Message.
func (m *CommonSoftwareInfoRequest) Parse(raw []byte) error {
	if len(raw) != commonSoftwareInfoRequestProps.Size {
		return messages.InvalidLength("CommonSoftwareInfoRequest", len(raw), commonSoftwareInfoRequestProps.Size)
	}
	m.SetCargo(raw)
	m.MCUTypeID = int32(raw[0])
	m.MCUType = MCUTypeFromID(m.MCUTypeID)
	return nil
}

// MCUType is one of the pump's microcontrollers.
type MCUType int32

// The microcontrollers.
const (
	MCUTypeARM MCUType = iota
	MCUTypeBLE
	MCUTypeMSP
	MCUTypeUnknown
)

var mcuTypeNames = []string{"ARM", "BLE", "MSP", "UNKNOWN"}

// MCUTypeFromID is the microcontroller with wire ID id, or Unknown.
func MCUTypeFromID(id int32) MCUType {
	if id >= 0 && int(id) < len(mcuTypeNames) {
		return MCUType(id)
	}
	return MCUTypeUnknown
}

// ID is the microcontroller's wire ID.
func (t MCUType) ID() int32 { return int32(t) }

func (t MCUType) String() string {
	if t >= 0 && int(t) < len(mcuTypeNames) {
		return mcuTypeNames[t]
	}
	return fmt.Sprintf("MCUType(%d)", int32(t))
}
