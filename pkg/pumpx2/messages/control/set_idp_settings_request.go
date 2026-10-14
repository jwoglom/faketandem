package control

import (
	"fmt"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/helpers"
)

// SetIDPSettingsRequest changes one setting of an insulin delivery profile,
// the one ChangeType names. IDPID is the profile's ID, not its slot;
// ProfileIndex is its slot (0-5) in ProfileStatusResponse.
// ProfileInsulinDuration is in minutes, and ProfileCarbEntry is 1 when
// boluses take carbs.
type SetIDPSettingsRequest struct {
	messages.Base
	IDPID                  int32       `pumpx2:"idpId"`
	ProfileIndex           int32       `pumpx2:"profileIndex"`
	ProfileInsulinDuration int32       `pumpx2:"profileInsulinDuration"`
	ProfileCarbEntry       int32       `pumpx2:"profileCarbEntry"`
	ChangeTypeID           int32       `pumpx2:"changeTypeId"`
	ChangeType             *ChangeType `pumpx2:"changeType"`
}

var setIDPSettingsRequestProps = messages.Props{OpCode: -84, Size: 6, Signed: true, Type: messages.Request, Characteristic: messages.Control,
	Response: "SetIDPSettingsResponse"}

func init() {
	messages.Register("SetIDPSettingsRequest", &SetIDPSettingsRequest{}, setIDPSettingsRequestProps,
		messages.Constructor{
			Params: "int idpId, int profileIndex, int profileInsulinDuration, int profileCarbEntry, SetIDPSettingsRequest.ChangeType changeType",
			Func:   NewSetIDPSettingsRequest,
		},
		messages.Constructor{
			Params: "int idpId, int profileInsulinDuration, int profileCarbEntry, SetIDPSettingsRequest.ChangeType changeType",
			Func:   NewSetIDPSettingsRequestByID,
		},
	)
}

// NewSetIDPSettingsRequest builds the request.
func NewSetIDPSettingsRequest(idpID, profileIndex, profileInsulinDuration, profileCarbEntry int32, changeType ChangeType) *SetIDPSettingsRequest {
	m := &SetIDPSettingsRequest{
		IDPID:                  idpID,
		ProfileIndex:           profileIndex,
		ProfileInsulinDuration: profileInsulinDuration,
		ProfileCarbEntry:       profileCarbEntry,
		ChangeTypeID:           int32(changeType),
		ChangeType:             ChangeTypeFromID(int32(changeType)),
	}
	m.SetCargo(helpers.Combine(
		[]byte{byte(idpID), byte(profileIndex)},
		helpers.FirstTwoBytesLittleEndian(profileInsulinDuration),
		[]byte{byte(profileCarbEntry), byte(changeType)},
	))
	return m
}

// NewSetIDPSettingsRequestByID is pumpX2's deprecated constructor, with
// profile index 0.
func NewSetIDPSettingsRequestByID(idpID, profileInsulinDuration, profileCarbEntry int32, changeType ChangeType) *SetIDPSettingsRequest {
	return NewSetIDPSettingsRequest(idpID, 0, profileInsulinDuration, profileCarbEntry, changeType)
}

// Parse implements messages.Message.
func (m *SetIDPSettingsRequest) Parse(raw []byte) error {
	raw = messages.RemoveSignedRequestHmacBytes(setIDPSettingsRequestProps, raw)
	if len(raw) != setIDPSettingsRequestProps.Size {
		return messages.InvalidLength("SetIDPSettingsRequest", len(raw), setIDPSettingsRequestProps.Size)
	}
	m.SetCargo(raw)
	m.IDPID = int32(int8(raw[0]))
	m.ProfileIndex = int32(int8(raw[1]))
	m.ProfileInsulinDuration = helpers.ReadShort(raw, 2)
	m.ProfileCarbEntry = int32(int8(raw[4]))
	m.ChangeTypeID = int32(int8(raw[5]))
	m.ChangeType = ChangeTypeFromID(m.ChangeTypeID)
	return nil
}

// ChangeType is the profile setting a SetIDPSettingsRequest changes.
type ChangeType int32

// The change types, by their wire IDs.
const (
	ChangeTypeChangeInsulinDuration ChangeType = 1
	ChangeTypeChangeCarbEntry       ChangeType = 4
)

// ChangeTypeFromID is the change type with wire ID id, or nil for an ID that
// is none.
func ChangeTypeFromID(id int32) *ChangeType {
	switch c := ChangeType(id); c {
	case ChangeTypeChangeInsulinDuration, ChangeTypeChangeCarbEntry:
		return &c
	}
	return nil
}

func (c ChangeType) String() string {
	switch c {
	case ChangeTypeChangeInsulinDuration:
		return "CHANGE_INSULIN_DURATION"
	case ChangeTypeChangeCarbEntry:
		return "CHANGE_CARB_ENTRY"
	}
	return fmt.Sprintf("ChangeType(%d)", int32(c))
}
