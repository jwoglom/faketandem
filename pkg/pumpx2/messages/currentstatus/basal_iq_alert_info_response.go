package currentstatus

import (
	"fmt"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/helpers"
)

// BasalIQAlertInfoResponse is the Basal-IQ alert the pump is showing, by
// its BasalIQAlert ID.
type BasalIQAlertInfoResponse struct {
	messages.Base
	AlertID int64 `pumpx2:"alertId"`
}

var basalIQAlertInfoResponseProps = messages.Props{OpCode: 103, Size: 4, Type: messages.Response, Request: "BasalIQAlertInfoRequest"}

func init() {
	messages.Register("BasalIQAlertInfoResponse", &BasalIQAlertInfoResponse{}, basalIQAlertInfoResponseProps,
		messages.Constructor{Params: "long alertId", Func: NewBasalIQAlertInfoResponse},
	)
}

// NewBasalIQAlertInfoResponse builds the response.
func NewBasalIQAlertInfoResponse(alertID int64) *BasalIQAlertInfoResponse {
	m := &BasalIQAlertInfoResponse{AlertID: alertID}
	m.SetCargo(helpers.ToUint32(alertID))
	return m
}

// Parse implements messages.Message.
func (m *BasalIQAlertInfoResponse) Parse(raw []byte) error {
	if len(raw) != basalIQAlertInfoResponseProps.Size {
		return messages.InvalidLength("BasalIQAlertInfoResponse", len(raw), basalIQAlertInfoResponseProps.Size)
	}
	m.SetCargo(raw)
	m.AlertID = helpers.ReadUint32(raw, 0)
	return nil
}

// Alert is the alert AlertID names, or nil.
func (m *BasalIQAlertInfoResponse) Alert() *BasalIQAlert { return BasalIQAlertFromID(m.AlertID) }

// BasalIQAlert is a Basal-IQ alert, by its ID.
type BasalIQAlert int64

// The Basal-IQ alerts.
const (
	BasalIQAlertNoAlert               BasalIQAlert = 0
	BasalIQAlertInsulinSuspendedAlert BasalIQAlert = 24576
	BasalIQAlertInsulinResumedAlert   BasalIQAlert = 24577
	BasalIQAlertInsulinResumedTimeout BasalIQAlert = 24578
)

// BasalIQAlertFromID is the alert with ID id, or nil.
func BasalIQAlertFromID(id int64) *BasalIQAlert {
	a := BasalIQAlert(id)
	switch a {
	case BasalIQAlertNoAlert, BasalIQAlertInsulinSuspendedAlert, BasalIQAlertInsulinResumedAlert, BasalIQAlertInsulinResumedTimeout:
		return &a
	}
	return nil
}

func (a BasalIQAlert) String() string {
	switch a {
	case BasalIQAlertNoAlert:
		return "NO_ALERT"
	case BasalIQAlertInsulinSuspendedAlert:
		return "INSULIN_SUSPENDED_ALERT"
	case BasalIQAlertInsulinResumedAlert:
		return "INSULIN_RESUMED_ALERT"
	case BasalIQAlertInsulinResumedTimeout:
		return "INSULIN_RESUMED_TIMEOUT"
	}
	return fmt.Sprintf("BasalIQAlert(%d)", int64(a))
}
