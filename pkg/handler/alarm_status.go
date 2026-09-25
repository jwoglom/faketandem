package handler

import (
	"fmt"

	"github.com/jwoglom/faketandem/pkg/protocol"
	"github.com/jwoglom/faketandem/pkg/pumpx2"
	"github.com/jwoglom/faketandem/pkg/state"

	log "github.com/sirupsen/logrus"
)

// AlarmStatusHandler answers AlarmStatusRequest with the pump's live
// active-alarm bitmask.
//
// This is one of the handlers that has to bypass the cliparser: pumpX2's
// AlarmStatusResponse "data" constructor takes an AlarmResponseType varargs
// array, which the JSON bridge cannot build, so cliparser silently falls back
// to the no-argument constructor and emits a zero-length cargo. The driver
// reads eight bytes of bitmask from that cargo, so the fallback is not merely
// static, it is malformed. pkg/protocol builds the real eight-byte message
// instead.
type AlarmStatusHandler struct{}

// NewAlarmStatusHandler creates a new alarm status handler.
func NewAlarmStatusHandler() *AlarmStatusHandler {
	return &AlarmStatusHandler{}
}

// MessageType returns the message type this handler processes.
func (h *AlarmStatusHandler) MessageType() string {
	return "AlarmStatusRequest"
}

// RequiresAuth returns true if this message requires authentication.
func (h *AlarmStatusHandler) RequiresAuth() bool {
	return true
}

// HandleMessage processes an AlarmStatusRequest.
func (h *AlarmStatusHandler) HandleMessage(msg *pumpx2.ParsedMessage, pumpState *state.PumpState) (*Response, error) {
	mask := pumpState.GetAlarmBitmask()
	log.Infof("Handling AlarmStatusRequest: txID=%d, alarmBitmask=0x%016x", msg.TxID, mask)

	response, err := protocol.BuildAlarmStatusResponse(uint8(msg.TxID), mask)
	if err != nil {
		return nil, fmt.Errorf("failed to build AlarmStatusResponse: %w", err)
	}

	return &Response{
		NativeResponse: response,
		Immediate:      true,
	}, nil
}
