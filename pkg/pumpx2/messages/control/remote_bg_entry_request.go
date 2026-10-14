package control

import (
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/helpers"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/models"
)

// RemoteBgEntryRequest records the BG entered in the bolus window against a
// bolus in progress: one between its BolusPermissionRequest and its
// InitiateBolusRequest. BG is in mg/dL, and PumpTime is the pump's seconds
// since boot (TimeSinceResetResponse's pump time).
type RemoteBgEntryRequest struct {
	messages.Base
	BG int32 `pumpx2:"bg"`
	// UseForCgmCalibration calibrates the active G6 or G7 sensor with BG.
	UseForCgmCalibration bool                             `pumpx2:"useForCgmCalibration"`
	EntryType            models.BloodGlucoseReadingType   `pumpx2:"entryType"`
	Source               models.BloodGlucoseReadingSource `pumpx2:"source"`
	PumpTime             int64                            `pumpx2:"pumpTime"`
	BolusID              int32                            `pumpx2:"bolusId"`
}

var remoteBgEntryRequestProps = messages.Props{OpCode: -74, Size: 11, Signed: true, Type: messages.Request, Characteristic: messages.Control,
	MinAPI: "API_V2_5", Response: "RemoteBgEntryResponse"}

func init() {
	messages.Register("RemoteBgEntryRequest", &RemoteBgEntryRequest{}, remoteBgEntryRequestProps,
		messages.Constructor{
			Params: "int bg, boolean useForCgmCalibration, boolean isAutopopBg, long pumpTimeSecondsSinceBoot, int bolusId",
			Func:   NewManualRemoteBgEntryRequest,
		},
		messages.Constructor{
			Params: "int bg, boolean useForCgmCalibration, BloodGlucoseReadingType entryType, BloodGlucoseReadingSource source, " +
				"long pumpTimeSecondsSinceBoot, int bolusId",
			Func: NewRemoteBgEntryRequest,
		},
	)
}

// NewRemoteBgEntryRequest builds the request, for the bolus ID a
// BolusPermissionResponse granted.
func NewRemoteBgEntryRequest(bg int32, useForCgmCalibration bool, entryType models.BloodGlucoseReadingType, source models.BloodGlucoseReadingSource,
	pumpTimeSecondsSinceBoot int64, bolusID int32) *RemoteBgEntryRequest {
	m := &RemoteBgEntryRequest{
		BG:                   bg,
		UseForCgmCalibration: useForCgmCalibration,
		EntryType:            entryType,
		Source:               source,
		PumpTime:             pumpTimeSecondsSinceBoot,
		BolusID:              bolusID,
	}
	m.SetCargo(helpers.Combine(
		helpers.FirstTwoBytesLittleEndian(bg),
		[]byte{helpers.Bool(useForCgmCalibration), byte(entryType.ID()), byte(source.ID())},
		helpers.ToUint32(pumpTimeSecondsSinceBoot),
		helpers.FirstTwoBytesLittleEndian(bolusID),
	))
	return m
}

// NewManualRemoteBgEntryRequest is pumpX2's deprecated constructor: a MANUAL
// reading whose source is REMOTE when isAutopopBg, else PUMP.
func NewManualRemoteBgEntryRequest(bg int32, useForCgmCalibration, isAutopopBg bool, pumpTimeSecondsSinceBoot int64, bolusID int32) *RemoteBgEntryRequest {
	source := models.BloodGlucoseReadingSourcePump
	if isAutopopBg {
		source = models.BloodGlucoseReadingSourceRemote
	}
	return NewRemoteBgEntryRequest(bg, useForCgmCalibration, models.BloodGlucoseReadingTypeManual, source, pumpTimeSecondsSinceBoot, bolusID)
}

// Parse implements messages.Message.
func (m *RemoteBgEntryRequest) Parse(raw []byte) error {
	raw = messages.RemoveSignedRequestHmacBytes(remoteBgEntryRequestProps, raw)
	if len(raw) != remoteBgEntryRequestProps.Size {
		return messages.InvalidLength("RemoteBgEntryRequest", len(raw), remoteBgEntryRequestProps.Size)
	}
	m.SetCargo(raw)
	m.BG = helpers.ReadShort(raw, 0)
	m.UseForCgmCalibration = raw[2] == 1
	m.EntryType = models.BloodGlucoseReadingTypeFromID(int32(int8(raw[3])))
	m.Source = models.BloodGlucoseReadingSourceFromID(int32(int8(raw[4])))
	m.PumpTime = helpers.ReadUint32(raw, 5)
	m.BolusID = helpers.ReadShort(raw, 9)
	return nil
}

// IsAutopopBG reports whether Source is REMOTE, as pumpX2's deprecated
// isAutopopBg does.
func (m *RemoteBgEntryRequest) IsAutopopBG() bool {
	return m.Source == models.BloodGlucoseReadingSourceRemote
}
