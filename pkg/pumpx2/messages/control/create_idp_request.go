package control

import (
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/helpers"
)

// CreateIDPRequest creates a Mobi insulin delivery profile, either new with
// one segment starting at midnight, or a copy of profile SourceIDPID.
type CreateIDPRequest struct {
	messages.Base
	// ProfileName is written in 17 bytes of UTF-8 but read back from at
	// most 16.
	ProfileName string `pumpx2:"profileName"`
	// FirstSegmentProfileCarbRatio is grams per unit, times 1000.
	FirstSegmentProfileCarbRatio int32 `pumpx2:"firstSegmentProfileCarbRatio"`
	// FirstSegmentProfileStartTime is minutes since midnight.
	FirstSegmentProfileStartTime int32 `pumpx2:"firstSegmentProfileStartTime"`
	// FirstSegmentProfileBasalRate is in milliunits per hour.
	FirstSegmentProfileBasalRate int32 `pumpx2:"firstSegmentProfileBasalRate"`
	// FirstSegmentProfileTargetBG is in mg/dL.
	FirstSegmentProfileTargetBG int32 `pumpx2:"firstSegmentProfileTargetBG"`
	// FirstSegmentProfileISF is mg/dL per unit.
	FirstSegmentProfileISF int32 `pumpx2:"firstSegmentProfileISF"`
	// ProfileInsulinDuration is in minutes.
	ProfileInsulinDuration int32 `pumpx2:"profileInsulinDuration"`
	// TimeSegmentBitmask flags the segment fields set: bit 0 basal rate,
	// 1 carb ratio, 2 target BG, 3 correction factor, 4 start time. A new
	// profile sets all five (31), a copy none.
	TimeSegmentBitmask int32 `pumpx2:"timeSegmentBitmask"`
	// BolusSettingsBitmask flags the bolus settings set: bit 0 insulin
	// duration, 1 max bolus, 2 carb entry. A new profile sets 5, a copy 0.
	BolusSettingsBitmask int32 `pumpx2:"bolusSettingsBitmask"`
	// ProfileCarbEntry is 1 when the profile takes carbs.
	ProfileCarbEntry int32 `pumpx2:"profileCarbEntry"`
	// SourceIDPID is the profile copied, or -1 for a new one.
	SourceIDPID int32 `pumpx2:"sourceIdpId"`
}

var createIDPRequestProps = messages.Props{OpCode: -26, Size: 35, Signed: true, Type: messages.Request, Characteristic: messages.Control,
	ModifiesInsulinDelivery: true, SupportedDevices: "MOBI_ONLY", Response: "CreateIDPResponse"}

func init() {
	messages.Register("CreateIDPRequest", &CreateIDPRequest{}, createIDPRequestProps,
		messages.Constructor{Params: "String profileName, int sourceIdpId", Func: NewCreateIDPRequestCopying},
		messages.Constructor{
			Params: "String profileName, int firstSegmentProfileCarbRatio, int firstSegmentProfileBasalRate, int firstSegmentProfileTargetBG, " +
				"int firstSegmentProfileISF, int profileInsulinDuration, int profileCarbEntry, int timeSegmentBitmask, int bolusSettingsBitmask",
			Func: NewCreateIDPRequestWithBitmasks,
		},
		messages.Constructor{
			Params: "String profileName, int firstSegmentProfileCarbRatio, int firstSegmentProfileBasalRate, int firstSegmentProfileTargetBG, " +
				"int firstSegmentProfileISF, int profileInsulinDuration, int profileCarbEntry",
			Func: NewCreateIDPRequest,
		},
	)
}

// NewCreateIDPRequest is a new profile with every segment field and the
// insulin duration and carb entry settings set. It fails on a name longer
// than 17 bytes.
func NewCreateIDPRequest(profileName string, firstSegmentProfileCarbRatio, firstSegmentProfileBasalRate, firstSegmentProfileTargetBG,
	firstSegmentProfileISF, profileInsulinDuration, profileCarbEntry int32) (*CreateIDPRequest, error) {
	return NewCreateIDPRequestWithBitmasks(profileName, firstSegmentProfileCarbRatio, firstSegmentProfileBasalRate, firstSegmentProfileTargetBG,
		firstSegmentProfileISF, profileInsulinDuration, profileCarbEntry, 31, 5)
}

// NewCreateIDPRequestWithBitmasks is a new profile with the given fields
// flagged as set. It fails on a name longer than 17 bytes.
func NewCreateIDPRequestWithBitmasks(profileName string, firstSegmentProfileCarbRatio, firstSegmentProfileBasalRate, firstSegmentProfileTargetBG,
	firstSegmentProfileISF, profileInsulinDuration, profileCarbEntry, timeSegmentBitmask, bolusSettingsBitmask int32) (*CreateIDPRequest, error) {
	return buildCreateIDPRequest(CreateIDPRequest{
		ProfileName:                  profileName,
		FirstSegmentProfileCarbRatio: firstSegmentProfileCarbRatio,
		FirstSegmentProfileBasalRate: firstSegmentProfileBasalRate,
		FirstSegmentProfileTargetBG:  firstSegmentProfileTargetBG,
		FirstSegmentProfileISF:       firstSegmentProfileISF,
		ProfileInsulinDuration:       profileInsulinDuration,
		TimeSegmentBitmask:           timeSegmentBitmask,
		BolusSettingsBitmask:         bolusSettingsBitmask,
		ProfileCarbEntry:             profileCarbEntry,
		SourceIDPID:                  -1,
	})
}

// NewCreateIDPRequestCopying is a copy of profile sourceIDPID under a new
// name. It fails on a name longer than 17 bytes.
func NewCreateIDPRequestCopying(profileName string, sourceIDPID int32) (*CreateIDPRequest, error) {
	return buildCreateIDPRequest(CreateIDPRequest{ProfileName: profileName, SourceIDPID: sourceIDPID})
}

func buildCreateIDPRequest(m CreateIDPRequest) (*CreateIDPRequest, error) {
	name, err := helpers.WriteString(m.ProfileName, 17)
	if err != nil {
		return nil, messages.Invalid("CreateIDPRequest: %v", err)
	}
	m.SetCargo(helpers.Combine(
		name,
		helpers.ToUint32(int64(m.FirstSegmentProfileCarbRatio)),
		helpers.FirstTwoBytesLittleEndian(m.FirstSegmentProfileStartTime),
		helpers.FirstTwoBytesLittleEndian(m.FirstSegmentProfileBasalRate),
		helpers.FirstTwoBytesLittleEndian(m.FirstSegmentProfileTargetBG),
		helpers.FirstTwoBytesLittleEndian(m.FirstSegmentProfileISF),
		helpers.FirstTwoBytesLittleEndian(m.ProfileInsulinDuration),
		[]byte{byte(m.TimeSegmentBitmask), byte(m.BolusSettingsBitmask), byte(m.SourceIDPID), byte(m.ProfileCarbEntry)},
	))
	return &m, nil
}

// Parse implements messages.Message.
func (m *CreateIDPRequest) Parse(raw []byte) error {
	raw = messages.RemoveSignedRequestHmacBytes(createIDPRequestProps, raw)
	if len(raw) != createIDPRequestProps.Size {
		return messages.InvalidLength("CreateIDPRequest", len(raw), createIDPRequestProps.Size)
	}
	m.SetCargo(raw)
	m.ProfileName = helpers.ReadString(raw, 0, 16)
	m.FirstSegmentProfileCarbRatio = int32(helpers.ReadUint32(raw, 17))
	m.FirstSegmentProfileStartTime = helpers.ReadShort(raw, 21)
	m.FirstSegmentProfileBasalRate = helpers.ReadShort(raw, 23)
	m.FirstSegmentProfileTargetBG = helpers.ReadShort(raw, 25)
	m.FirstSegmentProfileISF = helpers.ReadShort(raw, 27)
	m.ProfileInsulinDuration = helpers.ReadShort(raw, 29)
	m.TimeSegmentBitmask = int32(raw[31])
	m.BolusSettingsBitmask = int32(raw[32])
	m.SourceIDPID = int32(int8(raw[33]))
	m.ProfileCarbEntry = int32(int8(raw[34]))
	return nil
}
