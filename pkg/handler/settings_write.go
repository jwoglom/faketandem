package handler

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"

	"github.com/jwoglom/faketandem/pkg/pumpx2"
	"github.com/jwoglom/faketandem/pkg/settings"
	"github.com/jwoglom/faketandem/pkg/state"

	log "github.com/sirupsen/logrus"
)

// settingsWriteResponseParamsOverrides provides response params for the
// response classes whose real constructor doesn't take a plain int status
// (some only have a raw byte[] constructor).
var settingsWriteResponseParamsOverrides = map[string]map[string]interface{}{
	// ChangeControlIQSettingsResponse has no int-status constructor, only a raw byte[] one (size=3).
	"ChangeControlIQSettingsResponse": {"raw": "000000"},
	// SetQuickBolusSettingsResponse has no int-status constructor, only a raw byte[] one (size=1).
	"SetQuickBolusSettingsResponse": {"raw": "00"},
}

// SettingsWriteHandler handles settings write requests by updating the settings
// manager so subsequent reads reflect the new values, and returns success.
type SettingsWriteHandler struct {
	bridge          *pumpx2.Bridge
	settingsManager *settings.Manager
	msgType         string
	responseType    string
	readbackKey     string // which GenericSettings key to update on write
	// readbackValues maps the request onto readbackKey's fields; nil copies the parsed cargo as-is.
	readbackValues func(msg *pumpx2.ParsedMessage) (map[string]interface{}, error)
}

// NewSettingsWriteHandler creates a settings write handler
func NewSettingsWriteHandler(bridge *pumpx2.Bridge, sm *settings.Manager, msgType, readbackKey string) *SettingsWriteHandler {
	responseType := msgType[:len(msgType)-7] + "Response"
	return &SettingsWriteHandler{
		bridge:          bridge,
		settingsManager: sm,
		msgType:         msgType,
		responseType:    responseType,
		readbackKey:     readbackKey,
	}
}

// NewQuickBolusSettingsHandler handles SetQuickBolusSettingsRequest, reflecting
// the write in PumpGlobalsResponse the way the pump does.
func NewQuickBolusSettingsHandler(bridge *pumpx2.Bridge, sm *settings.Manager) *SettingsWriteHandler {
	h := NewSettingsWriteHandler(bridge, sm, "SetQuickBolusSettingsRequest", "PumpGlobalsRequest")
	h.readbackValues = quickBolusReadback
	return h
}

// SetQuickBolusSettingsRequest cargo: enabled, mode, units increment (mU, LE),
// carbs increment (mg, LE), then a bitmask of the fields the pump applies. The
// unflagged fields carry the pump's current values and are ignored (TandemKit#428).
const (
	quickBolusChangedEnabled        = 0x01
	quickBolusChangedMode           = 0x02
	quickBolusChangedIncrementUnits = 0x04
	quickBolusChangedIncrementCarbs = 0x08
)

// quickBolusReadback decodes the raw cargo rather than the named fields, which
// differ between pumpX2 versions.
func quickBolusReadback(msg *pumpx2.ParsedMessage) (map[string]interface{}, error) {
	raw, ok := cargoValue(msg, "cargo")
	if !ok {
		return nil, fmt.Errorf("no cargo field")
	}
	rawHex, ok := raw.(string)
	if !ok {
		return nil, fmt.Errorf("cargo is %T, not a hex string", raw)
	}
	b, err := hex.DecodeString(rawHex)
	if err != nil {
		return nil, fmt.Errorf("cargo %q is not hex: %w", rawHex, err)
	}
	if len(b) != 7 {
		return nil, fmt.Errorf("cargo %q is %d bytes, want 7", rawHex, len(b))
	}

	changed := b[6]
	values := map[string]interface{}{}
	if changed&quickBolusChangedEnabled != 0 {
		values["quickBolusEnabledRaw"] = int(b[0])
	}
	if changed&quickBolusChangedMode != 0 {
		values["quickBolusEntryType"] = int(b[1])
	}
	if changed&quickBolusChangedIncrementUnits != 0 {
		values["quickBolusIncrementUnits"] = int(binary.LittleEndian.Uint16(b[2:4]))
	}
	if changed&quickBolusChangedIncrementCarbs != 0 {
		values["quickBolusIncrementCarbs"] = int(binary.LittleEndian.Uint16(b[4:6]))
	}
	return values, nil
}

// MessageType returns the message type
func (h *SettingsWriteHandler) MessageType() string { return h.msgType }

// RequiresAuth returns true
func (h *SettingsWriteHandler) RequiresAuth() bool { return true }

// HandleMessage processes a settings write request
func (h *SettingsWriteHandler) HandleMessage(msg *pumpx2.ParsedMessage, pumpState *state.PumpState) (*Response, error) {
	log.Infof("Handling %s: txID=%d cargo=%v", h.msgType, msg.TxID, msg.Cargo)

	h.updateReadback(msg)

	params, ok := settingsWriteResponseParamsOverrides[h.responseType]
	if !ok {
		params = map[string]interface{}{"status": 0}
	}

	response, err := h.bridge.EncodeMessage(msg.TxID, h.responseType, params)
	if err != nil {
		return nil, fmt.Errorf("failed to encode %s: %w", h.responseType, err)
	}

	return &Response{
		ResponseMessage: response,
		Immediate:       true,
	}, nil
}

// updateReadback makes subsequent reads of readbackKey reflect the write.
func (h *SettingsWriteHandler) updateReadback(msg *pumpx2.ParsedMessage) {
	if h.readbackKey == "" || msg.Cargo == nil {
		return
	}
	values := msg.Cargo
	if h.readbackValues != nil {
		var err error
		if values, err = h.readbackValues(msg); err != nil {
			log.Warnf("Failed to decode %s for %s readback: %v", h.msgType, h.readbackKey, err)
			return
		}
	}
	if err := h.settingsManager.UpdateConstant(h.readbackKey, values); err != nil {
		log.Warnf("Failed to update settings readback for %s: %v", h.readbackKey, err)
	}
}

// SetModesHandler handles SetModesRequest
type SetModesHandler struct {
	bridge *pumpx2.Bridge
}

// NewSetModesHandler creates a new SetModes handler
func NewSetModesHandler(bridge *pumpx2.Bridge) *SetModesHandler {
	return &SetModesHandler{bridge: bridge}
}

// MessageType returns the message type
func (h *SetModesHandler) MessageType() string { return "SetModesRequest" }

// RequiresAuth returns true
func (h *SetModesHandler) RequiresAuth() bool { return true }

// HandleMessage processes a SetModesRequest
func (h *SetModesHandler) HandleMessage(msg *pumpx2.ParsedMessage, pumpState *state.PumpState) (*Response, error) {
	log.Infof("Handling SetModesRequest: txID=%d cargo=%v", msg.TxID, msg.Cargo)

	// pumpX2's SetModesRequest field is "bitmap" (it also prints a derived
	// "command" enum name); there is no "mode" field, so the mode was never
	// actually applied.
	if bitmap, ok := cargoInt(msg, "bitmap", "mode"); ok {
		pumpState.SetControlIQMode(int(bitmap))
	}

	// SetModesResponse has no int-status constructor, only a raw byte[] one (size=1).
	response, err := h.bridge.EncodeMessage(msg.TxID, "SetModesResponse", map[string]interface{}{
		"raw": "00",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to encode SetModesResponse: %w", err)
	}

	return &Response{
		ResponseMessage: response,
		Immediate:       true,
	}, nil
}
