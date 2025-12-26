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

// CliParserOutput represents the raw output from cliparser
type CliParserOutput struct {
	Success bool                   `json:"success"`
	Error   string                 `json:"error,omitempty"`
	Data    map[string]interface{} `json:"data,omitempty"`
}
