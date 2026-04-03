package handler

import (
	"fmt"

	"github.com/jwoglom/faketandem/pkg/pumpx2"
	"github.com/jwoglom/faketandem/pkg/state"

	log "github.com/sirupsen/logrus"
)

// CartridgeHandler handles cartridge change mode requests
type CartridgeHandler struct {
	bridge       *pumpx2.Bridge
	msgType      string
	responseType string
}

// NewCartridgeHandler creates a new cartridge mode handler
func NewCartridgeHandler(bridge *pumpx2.Bridge, msgType string) *CartridgeHandler {
	responseType := msgType
	if len(responseType) > 7 && responseType[len(responseType)-7:] == "Request" {
		responseType = responseType[:len(responseType)-7] + "Response"
	}
	return &CartridgeHandler{
		bridge:       bridge,
		msgType:      msgType,
		responseType: responseType,
	}
}

// MessageType returns the message type this handler processes
func (h *CartridgeHandler) MessageType() string {
	return h.msgType
}

// RequiresAuth returns true if this message requires authentication
func (h *CartridgeHandler) RequiresAuth() bool {
	return true
}

// HandleMessage processes a cartridge mode request
func (h *CartridgeHandler) HandleMessage(msg *pumpx2.ParsedMessage, pumpState *state.PumpState) (*Response, error) {
	log.Infof("Handling %s: txID=%d", h.msgType, msg.TxID)

	response, err := h.bridge.EncodeMessage(
		msg.TxID,
		h.responseType,
		map[string]interface{}{
			"status": 0,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to encode %s: %w", h.responseType, err)
	}

	return &Response{
		ResponseMessage: response,
		Immediate:       true,
	}, nil
}
