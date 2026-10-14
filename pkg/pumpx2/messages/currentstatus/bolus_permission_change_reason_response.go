package currentstatus

import (
	"fmt"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/helpers"
)

// BolusPermissionChangeReasonResponse is why the bolus permission for a
// bolus ID last changed (a ChangeReason ID), whether that change was
// acknowledged, and whether the asker holds the permission now.
type BolusPermissionChangeReasonResponse struct {
	messages.Base
	BolusID                 int32 `pumpx2:"bolusId"`
	IsAcked                 bool  `pumpx2:"isAcked"`
	LastChangeReasonID      int32 `pumpx2:"lastChangeReasonId"`
	CurrentPermissionHolder bool  `pumpx2:"currentPermissionHolder"`
}

var bolusPermissionChangeReasonResponseProps = messages.Props{OpCode: -87, Size: 5, Type: messages.Response, MinAPI: "API_FUTURE", Request: "BolusPermissionChangeReasonRequest"}

func init() {
	messages.Register("BolusPermissionChangeReasonResponse", &BolusPermissionChangeReasonResponse{}, bolusPermissionChangeReasonResponseProps,
		messages.Constructor{Params: "int bolusId, boolean isAcked, int lastChangeReasonId, boolean currentPermissionHolder", Func: NewBolusPermissionChangeReasonResponse},
	)
}

// NewBolusPermissionChangeReasonResponse builds the response.
func NewBolusPermissionChangeReasonResponse(bolusID int32, isAcked bool, lastChangeReasonID int32, currentPermissionHolder bool) *BolusPermissionChangeReasonResponse {
	m := &BolusPermissionChangeReasonResponse{
		BolusID:                 bolusID,
		IsAcked:                 isAcked,
		LastChangeReasonID:      lastChangeReasonID,
		CurrentPermissionHolder: currentPermissionHolder,
	}
	m.SetCargo(helpers.Combine(
		helpers.FirstTwoBytesLittleEndian(bolusID),
		[]byte{helpers.Bool(isAcked), byte(lastChangeReasonID), helpers.Bool(currentPermissionHolder)},
	))
	return m
}

// Parse implements messages.Message.
func (m *BolusPermissionChangeReasonResponse) Parse(raw []byte) error {
	if len(raw) != bolusPermissionChangeReasonResponseProps.Size {
		return messages.InvalidLength("BolusPermissionChangeReasonResponse", len(raw), bolusPermissionChangeReasonResponseProps.Size)
	}
	m.SetCargo(raw)
	m.BolusID = helpers.ReadShort(raw, 0)
	m.IsAcked = raw[2] != 0
	m.LastChangeReasonID = int32(int8(raw[3]))
	m.CurrentPermissionHolder = raw[4] != 0
	return nil
}

// LastChangeReason is the reason LastChangeReasonID names, or nil.
func (m *BolusPermissionChangeReasonResponse) LastChangeReason() *ChangeReason {
	return ChangeReasonFromID(m.LastChangeReasonID)
}

// ChangeReason is why a bolus permission changed.
type ChangeReason int32

// The change reasons, by their wire IDs.
const (
	ChangeReasonGranted                ChangeReason = 0
	ChangeReasonReleased               ChangeReason = 1
	ChangeReasonRevokedPriority        ChangeReason = 2
	ChangeReasonRevokedTimeout         ChangeReason = 3
	ChangeReasonRevokedSettingsChanged ChangeReason = 4
	ChangeReasonRevokedPumpSuspend     ChangeReason = 5
	ChangeReasonRevokedInvalidRequest  ChangeReason = 6
	ChangeReasonUnknown                ChangeReason = 7
)

var changeReasonNames = []string{
	"GRANTED", "RELEASED", "REVOKED_PRIORITY", "REVOKED_TIMEOUT", "REVOKED_SETTINGS_CHANGED", "REVOKED_PUMP_SUSPEND",
	"REVOKED_INVALID_REQUEST", "UNKNOWN",
}

// ChangeReasonFromID is the reason with wire ID id, or nil.
func ChangeReasonFromID(id int32) *ChangeReason {
	if id < 0 || int(id) >= len(changeReasonNames) {
		return nil
	}
	r := ChangeReason(id)
	return &r
}

func (r ChangeReason) String() string {
	if r >= 0 && int(r) < len(changeReasonNames) {
		return changeReasonNames[r]
	}
	return fmt.Sprintf("ChangeReason(%d)", int32(r))
}
