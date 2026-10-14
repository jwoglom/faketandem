package currentstatus

import (
	"fmt"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/helpers"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/models"
)

// IDPSegmentResponse is one segment of an insulin delivery profile. A
// request for a segment the profile does not have is answered with an
// ErrorResponse and the pump drops the connection.
type IDPSegmentResponse struct {
	messages.Base
	IDPID        int32 `pumpx2:"idpId"`
	SegmentIndex int32 `pumpx2:"segmentIndex"`
	// ProfileStartTime is minutes since midnight; the segment runs until
	// the next one starts.
	ProfileStartTime          int32           `pumpx2:"profileStartTime"`
	ProfileProcessedStartTime models.MinsTime `pumpx2:"profileProcessedStartTime"`
	// ProfileBasalRate is in milliunits per hour.
	ProfileBasalRate int32 `pumpx2:"profileBasalRate"`
	// ProfileCarbRatio is grams per unit, times 1000.
	ProfileCarbRatio int64 `pumpx2:"profileCarbRatio"`
	// ProfileTargetBG is in mg/dL.
	ProfileTargetBG int32 `pumpx2:"profileTargetBG"`
	// ProfileISF is mg/dL per unit.
	ProfileISF  int32              `pumpx2:"profileISF"`
	IDPStatusID int32              `pumpx2:"idpStatusId"`
	IDPStatus   []IDPSegmentStatus `pumpx2:"idpStatus"`
}

var idpSegmentResponseProps = messages.Props{OpCode: 67, Size: 15, Type: messages.Response, Request: "IDPSegmentRequest"}

func init() {
	messages.Register("IDPSegmentResponse", &IDPSegmentResponse{}, idpSegmentResponseProps,
		messages.Constructor{
			Params: "int idpId, int segmentIndex, int profileStartTime, int profileBasalRate, long profileCarbRatio, int profileTargetBG, int profileISF, int statusId",
			Func:   NewIDPSegmentResponse,
		},
	)
}

// NewIDPSegmentResponse builds the response; statusID is an IDPSegmentStatus
// bitmask of the fields the segment sets.
func NewIDPSegmentResponse(idpID, segmentIndex, profileStartTime, profileBasalRate int32, profileCarbRatio int64, profileTargetBG, profileISF, statusID int32) *IDPSegmentResponse {
	m := &IDPSegmentResponse{
		IDPID:                     idpID,
		SegmentIndex:              segmentIndex,
		ProfileStartTime:          profileStartTime,
		ProfileProcessedStartTime: models.NewMinsTime(profileStartTime),
		ProfileBasalRate:          profileBasalRate,
		ProfileCarbRatio:          profileCarbRatio,
		ProfileTargetBG:           profileTargetBG,
		ProfileISF:                profileISF,
		IDPStatusID:               statusID,
		IDPStatus:                 IDPSegmentStatusFromBitmask(statusID),
	}
	m.SetCargo(helpers.Combine(
		[]byte{byte(idpID), byte(segmentIndex)},
		helpers.FirstTwoBytesLittleEndian(profileStartTime),
		helpers.FirstTwoBytesLittleEndian(profileBasalRate),
		helpers.ToUint32(profileCarbRatio),
		helpers.FirstTwoBytesLittleEndian(profileTargetBG),
		helpers.FirstTwoBytesLittleEndian(profileISF),
		[]byte{byte(statusID)},
	))
	return m
}

// Parse implements messages.Message.
func (m *IDPSegmentResponse) Parse(raw []byte) error {
	if len(raw) != idpSegmentResponseProps.Size {
		return messages.InvalidLength("IDPSegmentResponse", len(raw), idpSegmentResponseProps.Size)
	}
	m.SetCargo(raw)
	m.IDPID = int32(int8(raw[0]))
	m.SegmentIndex = int32(int8(raw[1]))
	m.ProfileStartTime = helpers.ReadShort(raw, 2)
	m.ProfileProcessedStartTime = models.NewMinsTime(m.ProfileStartTime)
	m.ProfileBasalRate = helpers.ReadShort(raw, 4)
	m.ProfileCarbRatio = helpers.ReadUint32(raw, 6)
	m.ProfileTargetBG = helpers.ReadShort(raw, 10)
	m.ProfileISF = helpers.ReadShort(raw, 12)
	m.IDPStatusID = int32(int8(raw[14]))
	m.IDPStatus = IDPSegmentStatusFromBitmask(m.IDPStatusID)
	return nil
}

// IDPSegmentStatus is one field a profile segment sets, as a bit of its
// status.
type IDPSegmentStatus int32

// The segment fields, by their bits.
const (
	IDPSegmentStatusBasalRate        IDPSegmentStatus = 1
	IDPSegmentStatusCarbRatio        IDPSegmentStatus = 2
	IDPSegmentStatusTargetBG         IDPSegmentStatus = 4
	IDPSegmentStatusCorrectionFactor IDPSegmentStatus = 8
	IDPSegmentStatusStartTime        IDPSegmentStatus = 16
)

// IDPSegmentStatuses are the segment fields in pumpX2's declaration order.
var IDPSegmentStatuses = []IDPSegmentStatus{
	IDPSegmentStatusBasalRate, IDPSegmentStatusCarbRatio, IDPSegmentStatusTargetBG, IDPSegmentStatusCorrectionFactor, IDPSegmentStatusStartTime,
}

// IDPSegmentStatusFromBitmask is the fields whose bits mask sets, in
// declaration order; never nil.
func IDPSegmentStatusFromBitmask(mask int32) []IDPSegmentStatus {
	out := []IDPSegmentStatus{}
	for _, s := range IDPSegmentStatuses {
		if mask&int32(s) != 0 {
			out = append(out, s)
		}
	}
	return out
}

// IDPSegmentStatusToBitmask sums the fields' bits.
func IDPSegmentStatusToBitmask(statuses ...IDPSegmentStatus) int32 {
	var mask int32
	for _, s := range statuses {
		mask += int32(s)
	}
	return mask
}

func (s IDPSegmentStatus) String() string {
	switch s {
	case IDPSegmentStatusBasalRate:
		return "BASAL_RATE"
	case IDPSegmentStatusCarbRatio:
		return "CARB_RATIO"
	case IDPSegmentStatusTargetBG:
		return "TARGET_BG"
	case IDPSegmentStatusCorrectionFactor:
		return "CORRECTION_FACTOR"
	case IDPSegmentStatusStartTime:
		return "START_TIME"
	}
	return fmt.Sprintf("IDPSegmentStatus(%d)", int32(s))
}
