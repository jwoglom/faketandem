package pumpx2

// ParsedMessage represents a parsed pump message from cliparser
type ParsedMessage struct {
	Opcode      int                    `json:"opcode"`
	MessageType string                 `json:"messageType"`
	TxID        int                    `json:"txId"`
	Cargo       map[string]interface{} `json:"cargo"`
	IsSigned    bool                   `json:"isSigned"`
	IsValid     bool                   `json:"isValid"`
	Raw         string                 `json:"raw,omitempty"`

	// RawPacketsHex holds the original, unstripped BLE fragments (including
	// framing) exactly as received, in order. Kept around so callers that need
	// to re-forward the message verbatim (e.g. the pumpX2 JPAKE bridge, which
	// forwards the client's request bytes to jpake-server as-is) don't have to
	// re-encode it through a second cliparser invocation.
	RawPacketsHex []string `json:"-"`
}

// EncodedMessage represents an encoded message ready to send
type EncodedMessage struct {
	Characteristic string   `json:"characteristic"`
	Packets        []string `json:"packets"` // Hex strings
	MessageType    string   `json:"messageType"`
	TxID           int      `json:"txId"`
	Opcode         int      `json:"opcode"`
}

// OpcodeInfo represents information about an opcode
type OpcodeInfo struct {
	Opcode      int    `json:"opcode"`
	MessageType string `json:"messageType"`
	Direction   string `json:"direction"` // "request" or "response"
}
