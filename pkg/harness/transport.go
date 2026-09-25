package harness

import (
	"net/http"

	"github.com/jwoglom/faketandem/pkg/bluetooth"
)

// transportBody is the request and response shape of /api/transport.
type transportBody struct {
	// ATTMTU is the ATT MTU in force on the link. Setting it changes how
	// responses are fragmented: at the spec minimum of 23 every notification
	// carries 20 bytes (18 of message), which is how pumpX2's Packetize and
	// the cliparser jar frame everything; a real Mobi paired with the official
	// iOS app negotiates a much larger MTU, so most responses reach the phone
	// as a single notification with remaining=0.
	//
	// Exposed so a conformance test can exercise a driver's reassembler
	// against both, rather than only against the fragmented case the emulator
	// used to be hard-wired to.
	ATTMTU int `json:"att_mtu"`
	// MaxNotificationBytes is how many bytes one notification can carry at
	// that MTU (MTU minus the 3-byte ATT header), read-only.
	MaxNotificationBytes int `json:"max_notification_bytes"`
	// MaxMessageBytesPerNotification is how many message bytes that leaves
	// after this protocol's own 2-byte [remaining][txId] framing, read-only.
	// A response whose framed body fits in this many bytes goes out as a
	// single notification.
	MaxMessageBytesPerNotification int `json:"max_message_bytes_per_notification"`
	// Supported reports whether this transport can have its MTU changed at
	// all. The Linux GATT transport cannot: gatt negotiates the MTU with the
	// connecting central and does not expose the result.
	Supported bool `json:"mtu_configurable"`
}

// handleTransport reads and sets link-level parameters that are not pump state
// -- currently just the ATT MTU.
func (h *Harness) handleTransport(w http.ResponseWriter, r *http.Request) {
	if h.transport == nil {
		writeError(w, http.StatusInternalServerError, "no transport attached")
		return
	}

	negotiator, supported := h.transport.(bluetooth.MTUNegotiator)

	switch r.Method {
	case http.MethodGet:
		writeJSON(w, http.StatusOK, transportSnapshot(negotiator, supported))

	case http.MethodPut, http.MethodPatch:
		if !supported {
			writeError(w, http.StatusNotImplemented, "this transport does not expose its ATT MTU")
			return
		}

		// Start from the current value so a PUT that omits att_mtu is a no-op
		// rather than an attempt to set 0.
		body := transportBody{ATTMTU: negotiator.ATTMTU()}
		if err := decodeBody(r, &body); err != nil {
			writeError(w, http.StatusBadRequest, "%v", err)
			return
		}
		if err := negotiator.SetATTMTU(body.ATTMTU); err != nil {
			writeError(w, http.StatusBadRequest, "%v", err)
			return
		}
		writeJSON(w, http.StatusOK, transportSnapshot(negotiator, supported))

	default:
		writeError(w, http.StatusMethodNotAllowed, "method %s not allowed on /api/transport", r.Method)
	}
}

// transportSnapshot renders the current link parameters.
func transportSnapshot(negotiator bluetooth.MTUNegotiator, supported bool) transportBody {
	mtu := bluetooth.DefaultATTMTU
	if supported {
		mtu = negotiator.ATTMTU()
	}
	return transportBody{
		ATTMTU:                         mtu,
		MaxNotificationBytes:           bluetooth.MaxNotificationBytes(mtu),
		MaxMessageBytesPerNotification: bluetooth.MaxNotificationBytes(mtu) - 2,
		Supported:                      supported,
	}
}
