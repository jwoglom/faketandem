package currentstatus

import (
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/helpers"
)

// PumpVersionBResponse is the pump's software name ("Control-IQ+ 7.9.0.2"),
// configuration bits, serial and model numbers and revision, and its circuit
// board's part, serial and revision numbers. The software name is a string
// of at most 20 bytes and the revisions of at most 8.
type PumpVersionBResponse struct {
	messages.Base
	SoftwareName       string `pumpx2:"softwareName"`
	ConfigurationBitsA int64  `pumpx2:"configurationBitsA"`
	ConfigurationBitsB int64  `pumpx2:"configurationBitsB"`
	SerialNumber       int64  `pumpx2:"serialNumber"`
	ModelNumber        int64  `pumpx2:"modelNumber"`
	PumpRevision       string `pumpx2:"pumpRevision"`
	PcbPartNumberA     int64  `pumpx2:"pcbPartNumberA"`
	PcbSerialNumberA   int64  `pumpx2:"pcbSerialNumberA"`
	PcbRevisionNumberA string `pumpx2:"pcbRevisionNumberA"`
}

var pumpVersionBResponseProps = messages.Props{OpCode: -123, Size: 60, Type: messages.Response, Request: "PumpVersionBRequest"}

func init() {
	messages.Register("PumpVersionBResponse", &PumpVersionBResponse{}, pumpVersionBResponseProps,
		messages.Constructor{
			Params: "String softwareName, long configurationBitsA, long configurationBitsB, long serialNumber, long modelNumber, " +
				"String pumpRevision, long pcbPartNumberA, long pcbSerialNumberA, String pcbRevisionNumberA",
			Func: NewPumpVersionBResponse,
		},
	)
}

// NewPumpVersionBResponse builds the response. It fails when a string does
// not fit its field.
func NewPumpVersionBResponse(softwareName string, configurationBitsA, configurationBitsB, serialNumber, modelNumber int64,
	pumpRevision string, pcbPartNumberA, pcbSerialNumberA int64, pcbRevisionNumberA string) (*PumpVersionBResponse, error) {
	softwareNameBytes, err := helpers.WriteString(softwareName, 20)
	if err != nil {
		return nil, messages.Invalid("PumpVersionBResponse: softwareName: %v", err)
	}
	pumpRevisionBytes, err := helpers.WriteString(pumpRevision, 8)
	if err != nil {
		return nil, messages.Invalid("PumpVersionBResponse: pumpRevision: %v", err)
	}
	pcbRevisionNumberABytes, err := helpers.WriteString(pcbRevisionNumberA, 8)
	if err != nil {
		return nil, messages.Invalid("PumpVersionBResponse: pcbRevisionNumberA: %v", err)
	}
	m := &PumpVersionBResponse{
		SoftwareName:       softwareName,
		ConfigurationBitsA: configurationBitsA,
		ConfigurationBitsB: configurationBitsB,
		SerialNumber:       serialNumber,
		ModelNumber:        modelNumber,
		PumpRevision:       pumpRevision,
		PcbPartNumberA:     pcbPartNumberA,
		PcbSerialNumberA:   pcbSerialNumberA,
		PcbRevisionNumberA: pcbRevisionNumberA,
	}
	m.SetCargo(helpers.Combine(
		softwareNameBytes,
		helpers.ToUint32(configurationBitsA),
		helpers.ToUint32(configurationBitsB),
		helpers.ToUint32(serialNumber),
		helpers.ToUint32(modelNumber),
		pumpRevisionBytes,
		helpers.ToUint32(pcbPartNumberA),
		helpers.ToUint32(pcbSerialNumberA),
		pcbRevisionNumberABytes,
	))
	return m, nil
}

// Parse implements messages.Message.
func (m *PumpVersionBResponse) Parse(raw []byte) error {
	if len(raw) != pumpVersionBResponseProps.Size {
		return messages.InvalidLength("PumpVersionBResponse", len(raw), pumpVersionBResponseProps.Size)
	}
	m.SetCargo(raw)
	m.SoftwareName = helpers.ReadString(raw, 0, 20)
	m.ConfigurationBitsA = helpers.ReadUint32(raw, 20)
	m.ConfigurationBitsB = helpers.ReadUint32(raw, 24)
	m.SerialNumber = helpers.ReadUint32(raw, 28)
	m.ModelNumber = helpers.ReadUint32(raw, 32)
	m.PumpRevision = helpers.ReadString(raw, 36, 8)
	m.PcbPartNumberA = helpers.ReadUint32(raw, 44)
	m.PcbSerialNumberA = helpers.ReadUint32(raw, 48)
	m.PcbRevisionNumberA = helpers.ReadString(raw, 52, 8)
	return nil
}
