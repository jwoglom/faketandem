package messages

import (
	"crypto/hmac"
	"crypto/sha1" //nolint:gosec // HMAC-SHA1 is what the Tandem wire protocol specifies
	"strconv"
)

// Packetize chunk sizes: pumpX2's Packetize.DEFAULT_MAX_CHUNK_SIZE and
// CONTROL_MAX_CHUNK_SIZE.
const (
	defaultMaxChunkSize = 18
	controlMaxChunkSize = 40
)

func hmacSha1(data, key []byte) []byte {
	mac := hmac.New(sha1.New, key)
	mac.Write(data)
	return mac.Sum(nil)
}

// determineMaxChunkSize is Packetize.determineMaxChunkSize.
func determineMaxChunkSize(c *Class) int {
	if c.Props.Characteristic == "CONTROL" && c.Props.Type == "REQUEST" {
		return controlMaxChunkSize
	}
	return defaultMaxChunkSize
}

// tronPackets is new TronMessageWrapper(message, txId).packets(), with the
// signing inputs cliparser takes from its environment.
func tronPackets(m *Message, txID byte, env Env) ([][]byte, error) {
	chunk := determineMaxChunkSize(m.Class)
	if env.MaxChunkSize != "" {
		v, err := strconv.ParseInt(env.MaxChunkSize, 10, 32)
		if err != nil {
			return nil, exception("java.lang.NumberFormatException", "For input string: %q", env.MaxChunkSize)
		}
		chunk = int(v)
	}
	var key []byte
	if m.Class.Props.Signed {
		key = env.authKey()
	}
	tsr, err := env.timeSinceReset()
	if err != nil {
		return nil, err
	}
	return packetize(m, key, txID, chunk, tsr)
}

// packetize is Packetize.packetize.
func packetize(m *Message, key []byte, txID byte, maxChunkSize int, timeSinceReset int64) ([][]byte, error) {
	if m.Cargo == nil {
		return nil, exception("java.lang.NullPointerException", "packetize requires cargo, messageName=%s", m.Class.Name)
	}
	length := 3 + len(m.Cargo)
	if m.Class.Props.Signed {
		length += 24
	}
	packet := make([]byte, length)
	packet[0] = byte(int8(m.Class.Props.OpCode))
	packet[1] = txID
	packet[2] = byte(length - 3)
	copy(packet[3:], m.Cargo)

	if m.Class.Props.Signed {
		i := length - 20
		messageData := append([]byte{}, packet[:i]...)
		copy(messageData[length-24:], toUint32(timeSinceReset))
		if len(key) == 0 {
			return nil, illegalArgument("Empty key")
		}
		mac := hmacSha1(messageData, key)
		copy(packet, messageData)
		copy(packet[i:], mac)
	}

	packetWithCRC := append(packet, crc16(packet)...)

	if maxChunkSize <= 0 {
		return nil, illegalArgument("partitionSize must be positive, got %d", maxChunkSize)
	}
	var chunks [][]byte
	for start := 0; start < len(packetWithCRC); start += maxChunkSize {
		end := start + maxChunkSize
		if end > len(packetWithCRC) {
			end = len(packetWithCRC)
		}
		chunks = append(chunks, packetWithCRC[start:end])
	}

	out := make([][]byte, len(chunks))
	for n, chunk := range chunks {
		remaining := byte(len(chunks) - 1 - n)
		out[n] = append([]byte{remaining, txID}, chunk...)
	}
	return out, nil
}
