package messages

import (
	"crypto/hmac"
	"crypto/sha1" //nolint:gosec // HMAC-SHA1 is what the Tandem wire protocol specifies
	"strconv"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/helpers"
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
	if c.Props.Characteristic == Control && c.Props.Type == Request {
		return controlMaxChunkSize
	}
	return defaultMaxChunkSize
}

// Packets frames m for the wire as transaction txID: the BLE writes or
// notifications that carry it, signed with env's key if m is signed.
func Packets(m Message, txID byte, env Env) ([][]byte, error) {
	return tronPackets(m, txID, env)
}

// tronPackets is new TronMessageWrapper(message, txId).packets(), with the
// signing inputs cliparser takes from its environment.
func tronPackets(m Message, txID byte, env Env) ([][]byte, error) {
	c := ClassOf(m)
	chunk := determineMaxChunkSize(c)
	if env.MaxChunkSize != "" {
		v, err := strconv.ParseInt(env.MaxChunkSize, 10, 32)
		if err != nil {
			return nil, exception("java.lang.NumberFormatException", "For input string: %q", env.MaxChunkSize)
		}
		chunk = int(v)
	}
	var key []byte
	if c.Props.Signed {
		key = env.authKey()
	}
	tsr, err := env.timeSinceReset()
	if err != nil {
		return nil, err
	}
	return packetize(c, m.Cargo(), key, txID, chunk, tsr)
}

// packetize is Packetize.packetize.
func packetize(c *Class, cargo, key []byte, txID byte, maxChunkSize int, timeSinceReset int64) ([][]byte, error) {
	if cargo == nil {
		return nil, exception("java.lang.NullPointerException", "packetize requires cargo, messageName=%s", c.Name)
	}
	length := 3 + len(cargo)
	if c.Props.Signed {
		length += 24
	}
	packet := make([]byte, length)
	packet[0] = byte(int8(c.Props.OpCode))
	packet[1] = txID
	packet[2] = byte(length - 3)
	copy(packet[3:], cargo)

	if c.Props.Signed {
		i := length - 20
		messageData := append([]byte{}, packet[:i]...)
		copy(messageData[length-24:], helpers.ToUint32(timeSinceReset))
		if len(key) == 0 {
			return nil, illegalArgument("Empty key")
		}
		mac := hmacSha1(messageData, key)
		copy(packet, messageData)
		copy(packet[i:], mac)
	}

	packetWithCRC := append(packet, helpers.CRC16(packet)...)

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
