package handler

import (
	"fmt"

	"github.com/jwoglom/faketandem/pkg/pumpx2"
	"github.com/jwoglom/faketandem/pkg/state"

	log "github.com/sirupsen/logrus"
)

// TimeSinceResetHandler handles TimeSinceResetRequest messages
type TimeSinceResetHandler struct {
	bridge *pumpx2.Bridge
}

// NewTimeSinceResetHandler creates a new time since reset handler
func NewTimeSinceResetHandler(bridge *pumpx2.Bridge) *TimeSinceResetHandler {
	return &TimeSinceResetHandler{
		bridge: bridge,
	}
}

// MessageType returns the message type this handler processes
func (h *TimeSinceResetHandler) MessageType() string {
	return "TimeSinceResetRequest"
}

// RequiresAuth returns true if this message requires authentication
func (h *TimeSinceResetHandler) RequiresAuth() bool {
	return false // TimeSinceReset can be sent before authentication
}

// HandleMessage processes a TimeSinceResetRequest
func (h *TimeSinceResetHandler) HandleMessage(msg *pumpx2.ParsedMessage, pumpState *state.PumpState) (*Response, error) {
	log.Infof("Handling TimeSinceResetRequest: txID=%d", msg.TxID)

	// Update the time since reset
	pumpState.UpdateTimeSinceReset()
	timeSinceReset := pumpState.GetTimeSinceReset()

	log.Debugf("Responding with time since reset: %d seconds", timeSinceReset)

	// Update bridge with current time since reset
	h.bridge.SetTimeSinceReset(timeSinceReset)

	// Build response using pumpX2 bridge. TimeSinceResetResponse's real
	// constructor is (long currentTime, long pumpTimeSinceReset).
	//
	// currentTime is the pump's clock in PUMP-EPOCH seconds (seconds since
	// 2008-01-01), not Unix seconds: both pumpX2 and TandemKit decode it via
	// their Jan-1-2008 helpers, and drivers compare it against the phone clock
	// to detect pump-time drift. Sending time.Now().Unix() reported a pump
	// clock roughly 38 years fast.
	response, err := h.bridge.EncodeMessage(
		msg.TxID,
		"TimeSinceResetResponse",
		map[string]interface{}{
			"currentTime":        pumpState.PumpTimeNow(),
			"pumpTimeSinceReset": timeSinceReset,
		},
	)

	if err != nil {
		return nil, fmt.Errorf("failed to encode TimeSinceResetResponse: %w", err)
	}

	return &Response{
		ResponseMessage: response,
		Immediate:       true,
		StateChanges: []StateChange{
			{
				Type: StateChangeTime,
			},
		},
	}, nil
}

// ChangeTimeDateHandler handles ChangeTimeDateRequest: it sets the pump's clock
// to the requested date and time and writes the TimeChanged/DateChange records
// a pump writes for it.
type ChangeTimeDateHandler struct {
	bridge *pumpx2.Bridge
}

// NewChangeTimeDateHandler creates a change time/date handler.
func NewChangeTimeDateHandler(bridge *pumpx2.Bridge) *ChangeTimeDateHandler {
	return &ChangeTimeDateHandler{bridge: bridge}
}

// MessageType returns the message type this handler processes
func (h *ChangeTimeDateHandler) MessageType() string {
	return "ChangeTimeDateRequest"
}

// RequiresAuth returns true if this message requires authentication
func (h *ChangeTimeDateHandler) RequiresAuth() bool {
	return true
}

// HandleMessage processes a ChangeTimeDateRequest. tandemEpochTime is the new
// clock reading in pump-epoch seconds, as local time.
func (h *ChangeTimeDateHandler) HandleMessage(msg *pumpx2.ParsedMessage, pumpState *state.PumpState) (*Response, error) {
	requested, ok := cargoInt(msg, "tandemEpochTime")
	if !ok || requested <= 0 {
		return nil, fmt.Errorf("ChangeTimeDateRequest without a tandemEpochTime: %v", msg.Cargo)
	}
	prior, after := pumpState.SetPumpWallClock(uint32(requested))
	log.Infof("Handling ChangeTimeDateRequest: txID=%d pump clock %d -> %d (%+d s)",
		msg.TxID, prior, after, int64(after)-int64(prior))

	response, err := h.bridge.EncodeMessage(msg.TxID, "ChangeTimeDateResponse", map[string]interface{}{"status": 0})
	if err != nil {
		return nil, fmt.Errorf("failed to encode ChangeTimeDateResponse: %w", err)
	}
	return &Response{ResponseMessage: response, Immediate: true}, nil
}
