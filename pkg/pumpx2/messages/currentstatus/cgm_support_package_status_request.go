package currentstatus

import (
	"fmt"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
)

// CgmSupportPackageStatusRequest asks whether the pump's support package for
// a kind of CGM is valid.
type CgmSupportPackageStatusRequest struct {
	messages.Base
	DeviceTypeID int32      `pumpx2:"deviceTypeId"`
	DeviceType   DeviceType `pumpx2:"deviceType"`
}

var cgmSupportPackageStatusRequestProps = messages.Props{OpCode: -56, Size: 1, Type: messages.Request, MinAPI: "API_FUTURE", Response: "CgmSupportPackageStatusResponse"}

func init() {
	messages.Register("CgmSupportPackageStatusRequest", &CgmSupportPackageStatusRequest{}, cgmSupportPackageStatusRequestProps,
		messages.Constructor{Params: "CgmSupportPackageStatusRequest.DeviceType deviceType", Func: NewCgmSupportPackageStatusRequest},
		messages.Constructor{Params: "int deviceTypeId", Func: NewCgmSupportPackageStatusRequestFromID},
		messages.Constructor{Func: func() *CgmSupportPackageStatusRequest {
			return NewCgmSupportPackageStatusRequest(DeviceTypeDexcomG7ICGM)
		}},
	)
}

// NewCgmSupportPackageStatusRequest asks after deviceType's support package.
func NewCgmSupportPackageStatusRequest(deviceType DeviceType) *CgmSupportPackageStatusRequest {
	return NewCgmSupportPackageStatusRequestFromID(deviceType.ID())
}

// NewCgmSupportPackageStatusRequestFromID asks after the support package of
// the device type with wire ID deviceTypeID.
func NewCgmSupportPackageStatusRequestFromID(deviceTypeID int32) *CgmSupportPackageStatusRequest {
	m := &CgmSupportPackageStatusRequest{DeviceTypeID: deviceTypeID, DeviceType: DeviceTypeFromID(deviceTypeID)}
	m.SetCargo([]byte{byte(deviceTypeID)})
	return m
}

// Parse implements messages.Message.
func (m *CgmSupportPackageStatusRequest) Parse(raw []byte) error {
	if len(raw) != cgmSupportPackageStatusRequestProps.Size {
		return messages.InvalidLength("CgmSupportPackageStatusRequest", len(raw), cgmSupportPackageStatusRequestProps.Size)
	}
	m.SetCargo(raw)
	m.DeviceTypeID = int32(raw[0])
	m.DeviceType = DeviceTypeFromID(m.DeviceTypeID)
	return nil
}

// DeviceType is a kind of CGM with a support package on the pump.
type DeviceType int32

// The device types.
const (
	DeviceTypeDexcomG7ICGM DeviceType = iota
	DeviceTypeUnknown
)

var deviceTypeNames = []string{"DEXCOM_G7_ICGM", "UNKNOWN"}

// DeviceTypeFromID is the device type with wire ID id, or Unknown.
func DeviceTypeFromID(id int32) DeviceType {
	if id >= 0 && int(id) < len(deviceTypeNames) {
		return DeviceType(id)
	}
	return DeviceTypeUnknown
}

// ID is the device type's wire ID.
func (d DeviceType) ID() int32 { return int32(d) }

func (d DeviceType) String() string {
	if d >= 0 && int(d) < len(deviceTypeNames) {
		return deviceTypeNames[d]
	}
	return fmt.Sprintf("DeviceType(%d)", int32(d))
}
