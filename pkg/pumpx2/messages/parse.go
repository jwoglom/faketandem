package messages

import (
	"bytes"
	"encoding/hex"
	"regexp"
	"strings"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/helpers"
)

var rawHexSeparator = regexp.MustCompile(`[,\s]+`)

// splitRawHexPackets is Main.splitRawHexPackets.
func splitRawHexPackets(entries []string) []string {
	var out []string
	for _, entry := range entries {
		for _, token := range rawHexSeparator.Split(entry, -1) {
			if token != "" {
				out = append(out, token)
			}
		}
	}
	return out
}

// decodeHex is commons-codec's Hex.decodeHex.
func decodeHex(s string) ([]byte, error) {
	if len(s)%2 != 0 {
		return nil, exception("org.apache.commons.codec.DecoderException", "Odd number of characters.")
	}
	b, err := hex.DecodeString(s)
	if err != nil {
		return nil, exception("org.apache.commons.codec.DecoderException", "%v", err)
	}
	return b, nil
}

// lengthHeuristics are CharacteristicGuesser.filterKnownPossibilities's
// guesses from the joined argument's length, used when no characteristic is
// named.
var lengthHeuristics = []struct {
	opcode, length int
	characteristic Characteristic
}{
	{32, 14, CurrentStatus}, {33, 22, CurrentStatus}, {35, 30, CurrentStatus}, {37, 22, CurrentStatus},
	{37, 148, Control}, {-92, 78, Control}, {-91, 70, Control}, {-90, 66, Control}, {-89, 68, Control},
	{32, 384, Authorization}, {33, 384, Authorization}, {34, 384, Authorization}, {35, 384, Authorization},
	{36, 384, Authorization}, {37, 354, Authorization},
}

// resolveCharacteristic is the characteristic choice in Main.parse.
func resolveCharacteristic(t *opcodeTables, opcode int, btChar, rawHex string) (Characteristic, error) {
	possible := possibleCharacteristics(t, opcode)
	if len(possible) > 1 {
		if btChar != "" {
			if _, ok := CharacteristicUUID(btChar); !ok {
				return "", exception("java.lang.IllegalArgumentException", "No enum constant Characteristic.%s", btChar)
			}
			possible = map[Characteristic]bool{Characteristic(btChar): true}
		} else {
			for _, h := range lengthHeuristics {
				if h.opcode == opcode && h.length == len(rawHex) {
					possible = map[Characteristic]bool{h.characteristic: true}
					break
				}
			}
		}
		switch {
		case possible[Control]:
			possible = map[Characteristic]bool{Control: true}
		case possible[Authorization]:
			possible = map[Characteristic]bool{Authorization: true}
		case possible[CurrentStatus]:
			possible = map[Characteristic]bool{CurrentStatus: true}
		default:
			return "", exception("faketandem.NoMessage", "no single characteristic for opcode %d among %v", opcode, possible)
		}
	}
	if len(possible) == 0 {
		return "", exception("faketandem.NoMessage", "Unknown opcode %d", opcode)
	}
	var chosen Characteristic
	for c := range possible {
		chosen = c
	}
	return chosen, nil
}

// ParseFragments is cliparser's `parse` command as faketandem runs it: the
// raw BLE fragments of one message, in receive order, and the characteristic
// (a pumpX2 Characteristic name, or "" for none) they arrived on. It returns
// the message cliparser would print, or an error wherever cliparser would print
// something other than a message (an exception, or no message at all).
func ParseFragments(btChar string, rawPacketsHex []string, env Env) (Message, error) {
	t, err := messageTables()
	if err != nil {
		return nil, err
	}
	rawHex := strings.Join(rawPacketsHex, " ")
	packets := splitRawHexPackets([]string{rawHex})
	if len(packets) == 0 {
		return nil, exception("java.lang.IndexOutOfBoundsException", "no packets")
	}
	first, err := decodeHex(packets[0])
	if err != nil {
		return nil, err
	}
	if len(first) < 3 {
		return nil, exception("java.lang.ArrayIndexOutOfBoundsException", "%d", len(first))
	}
	txID := first[1]
	opcode := int(int8(first[2]))

	characteristic, err := resolveCharacteristic(t, opcode, btChar, rawHex)
	if err != nil {
		return nil, err
	}
	expectedClass, ok := FromOpcode(opcode, characteristic)
	if !ok {
		return nil, exception("java.lang.NullPointerException", "no class for opcode %d on %s", opcode, characteristic)
	}
	expected, err := expectedClass.New()
	if err != nil {
		return nil, err
	}
	expected.SetCargo([]byte{})
	return testRequest(expected, txID, characteristic, packets, env)
}

// testRequest is Main.testRequest: the fragments fed through BTResponseParser
// against a PacketArrayList built for the expected message.
func testRequest(expected Message, txID byte, characteristic Characteristic, packets []string, env Env) (Message, error) {
	if _, err := tronPackets(expected, txID, env); err != nil {
		return nil, err
	}
	pal, err := buildPacketArrayList(ClassOf(expected), txID)
	if err != nil {
		return nil, err
	}
	if characteristic == "QUALIFYING_EVENTS" {
		return nil, exception("org.apache.commons.lang3.NotImplementedException", "Qualifying event characteristic")
	}

	var result Message
	for _, p := range packets {
		data, err := decodeHex(p)
		if err != nil {
			return nil, err
		}
		result, err = btResponseParse(pal, data, characteristic, env)
		if err != nil {
			return nil, err
		}
	}
	if result == nil {
		return nil, exception("faketandem.NoMessage", "Message not fully formed")
	}
	return result, nil
}

// packetArrayList is pumpX2's PacketArrayList (stream=false) or
// StreamPacketArrayList (stream=true).
type packetArrayList struct {
	stream             bool
	expectedOpCode     int8
	expectedCargoSize  int8
	actualExpectedSize int
	expectedTxID       int8
	isSigned           bool
	tolerateShortCargo bool
	fullCargo          []byte
	messageData        []byte
	firstByteMod15     int8
	opCode             int8
	empty              bool
}

var shortCargoTolerant = map[string]bool{
	"Jpake1aResponse": true,
	"Jpake1bResponse": true,
	"Jpake2Response":  true,
}

// buildPacketArrayList is TronMessageWrapper.buildPacketArrayList.
func buildPacketArrayList(c *Class, txID byte) (*packetArrayList, error) {
	props := c.Props
	var opcode, size int
	if props.Type == Request {
		if props.Response == "" {
			return nil, exception("java.lang.NullPointerException", "%s has no response class", c.Name)
		}
		opcode = props.OpCode
		size = props.Size
		if props.Signed {
			size += 24
		}
	} else {
		opcode = props.OpCode
		size = props.Size
	}

	listClass, ok := FromOpcode(int(int8(opcode)), props.Characteristic)
	if !ok {
		return nil, exception("java.lang.NullPointerException", "no class for opcode %d on %s", opcode, props.Characteristic)
	}
	if int8(opcode) == 0 {
		return nil, illegalArgument("The validated expression is false")
	}
	p := &packetArrayList{
		stream:            listClass.Props.Stream,
		expectedOpCode:    int8(opcode),
		expectedCargoSize: int8(size),
		expectedTxID:      int8(txID),
		isSigned:          props.Signed,
		messageData:       make([]byte, 3),
		empty:             true,
	}
	p.actualExpectedSize = int(p.expectedCargoSize)
	if p.expectedCargoSize < 0 && p.expectedCargoSize > -128 {
		p.actualExpectedSize = 256 + int(p.expectedCargoSize)
	}
	if p.actualExpectedSize+2 < 0 {
		return nil, exception("java.lang.NegativeArraySizeException", "%d", p.actualExpectedSize+2)
	}
	p.fullCargo = make([]byte, p.actualExpectedSize+2)
	if !p.stream {
		p.tolerateShortCargo = shortCargoTolerant[listClass.Name]
	}
	return p, nil
}

func (p *packetArrayList) needsMorePacket() bool { return p.firstByteMod15 >= 0 }

func (p *packetArrayList) validatePacket(data []byte) error {
	if len(data) == 0 {
		return illegalArgument("Empty data")
	}
	if len(data) < 3 {
		return illegalArgument("Invalid data size: %d", len(data))
	}
	firstByteMod15 := int8(data[0] & 15)
	if int8(data[1]) != p.expectedTxID {
		if p.stream {
			return illegalArgument("Unexpected transaction ID 1: %d, expecting %d", int8(data[1]), p.expectedTxID)
		}
		return exception("com.jwoglom.pumpx2.pump.messages.models.UnexpectedTransactionIdException", "found %d expected %d", int8(data[1]), p.expectedTxID)
	}
	switch {
	case p.empty:
		p.firstByteMod15 = firstByteMod15
		if err := p.parse(data); err != nil {
			return err
		}
	case firstByteMod15 == 0:
		if p.firstByteMod15 != firstByteMod15 {
			return illegalArgument("Unexpected packets remaining 3: %d, expected %d", firstByteMod15, p.firstByteMod15)
		}
		p.fullCargo = append(p.fullCargo, data[2:]...)
	case p.firstByteMod15 == firstByteMod15:
		p.fullCargo = append(p.fullCargo, data[2:]...)
	default:
		return illegalArgument("Unexpected packets remaining 2: %d, expected %d", firstByteMod15, p.firstByteMod15)
	}
	if p.stream && !p.needsMorePacket() {
		p.empty = true
	} else if !p.stream {
		p.empty = false
	}
	p.firstByteMod15--
	return nil
}

func (p *packetArrayList) parse(b []byte) error {
	if len(b) < 5 {
		return exception("java.lang.ArrayIndexOutOfBoundsException", "Index 4 out of bounds for length %d", len(b))
	}
	if p.stream {
		return p.parseStream(b)
	}
	opCode := int8(b[2])
	cargoSize := int(int8(b[4]))
	if cargoSize < 0 && cargoSize > -128 {
		cargoSize += 256
	}
	p.applySizeExceptions(opCode, cargoSize)

	if opCode != p.expectedOpCode {
		return exception("com.jwoglom.pumpx2.pump.messages.models.UnexpectedOpCodeException", "%d != %d", opCode, p.expectedOpCode)
	}
	p.firstByteMod15 = int8(b[0] & 15)
	p.opCode = opCode
	if int8(b[3]) != p.expectedTxID {
		return exception("com.jwoglom.pumpx2.pump.messages.models.UnexpectedTransactionIdException", "found %d expected %d", int8(b[3]), p.expectedTxID)
	}
	if cargoSize != p.actualExpectedSize {
		switch {
		case cargoSize == p.actualExpectedSize+24 && p.isSigned:
			p.expectedCargoSize += 24
			p.actualExpectedSize += 24
		case p.tolerateShortCargo && cargoSize < p.actualExpectedSize && cargoSize >= p.actualExpectedSize-4:
			p.expectedCargoSize = int8(cargoSize)
			p.actualExpectedSize = cargoSize
		default:
			return illegalArgument("Unexpected cargo size: %d, expecting %d for opCode=%d", cargoSize, p.actualExpectedSize, opCode)
		}
	}
	p.fullCargo = append([]byte{}, b[5:]...)
	return nil
}

// sizeExceptions are the opcodes PacketArrayList.parse allows another cargo
// size for: a declared size of any of sizes is taken as the expected size set.
var sizeExceptions = []struct {
	opcode int8
	sizes  []int
	set    int
}{
	{32, []int{0}, 0},
	{-114, []int{1}, 1},
	{-113, []int{51}, 51},
	{121, []int{10}, 10},
	{-29, []int{1, 25}, 1},
	{-29, []int{2, 26}, 2},
	{-23, []int{1, 25}, 1},
	{-23, []int{5, 29}, 5},
}

// applySizeExceptions is the cargo-size special-casing at the top of
// PacketArrayList.parse. ErrorResponse (77) also takes over the expected
// opcode, since it answers any request.
func (p *packetArrayList) applySizeExceptions(opCode int8, cargoSize int) {
	if opCode == 77 && (cargoSize == 2 || cargoSize == 26) {
		p.expectedOpCode = 77
		p.expectedCargoSize = int8(cargoSize)
		p.actualExpectedSize = cargoSize
	}
	for _, e := range sizeExceptions {
		if e.opcode != opCode {
			continue
		}
		for _, size := range e.sizes {
			if size == cargoSize {
				p.expectedCargoSize = int8(e.set)
				p.actualExpectedSize = e.set
				return
			}
		}
	}
}

// historyLogStreamOpcode is StreamPacketArrayList.HISTORY_LOG_STREAM_OPCODE.
const historyLogStreamOpcode = -127

// parseStream is StreamPacketArrayList.parse.
func (p *packetArrayList) parseStream(b []byte) error {
	opCode := int8(b[2])
	cargoSize := int8(b[4])
	if opCode != p.expectedOpCode {
		return exception("com.jwoglom.pumpx2.pump.messages.models.UnexpectedOpCodeException", "%d != %d", opCode, p.expectedOpCode)
	}
	p.firstByteMod15 = int8(b[0] & 15)
	p.opCode = opCode
	if int8(b[3]) != p.expectedTxID {
		return exception("com.jwoglom.pumpx2.pump.messages.models.UnexpectedTransactionIdException", "found %d expected %d", int8(b[3]), p.expectedTxID)
	}
	if int(cargoSize) != p.actualExpectedSize {
		switch {
		case int(cargoSize) == p.actualExpectedSize+24 && p.isSigned:
			p.expectedCargoSize += 24
			p.actualExpectedSize += 24
		case opCode != historyLogStreamOpcode:
			return illegalArgument("Unexpected cargo size: %d, expecting %d for opCode=%d", cargoSize, p.actualExpectedSize, opCode)
		}
	}
	p.fullCargo = append([]byte{}, b[5:]...)
	if opCode == historyLogStreamOpcode {
		if len(b) < 6 {
			return exception("java.lang.ArrayIndexOutOfBoundsException", "Index 5 out of bounds for length %d", len(b))
		}
		numHistoryLogs := int8(b[5])
		if int(cargoSize) != int(numHistoryLogs)*26+2 {
			return illegalArgument("Cargo size doesn't match number of history logs requested")
		}
		p.expectedCargoSize = cargoSize
	}
	return nil
}

// validate is PacketArrayList.validate.
func (p *packetArrayList) validate(key []byte) (bool, error) {
	if p.needsMorePacket() {
		return false, nil
	}
	if len(p.fullCargo) < 2 {
		return false, illegalArgument("cannot drop last 2 of %d bytes", len(p.fullCargo))
	}
	body := p.fullCargo[:len(p.fullCargo)-2]
	header := []byte{byte(p.expectedOpCode), byte(p.expectedTxID), byte(p.expectedCargoSize)}
	p.messageData = helpers.Combine(header, body)
	expectedCrc := []byte{0, 0}
	if len(p.fullCargo) >= 2 {
		expectedCrc = p.fullCargo[len(p.fullCargo)-2:]
	}

	if !bytes.Equal(helpers.CRC16(p.messageData), expectedCrc) {
		if shouldIgnoreInvalidHmac(key) {
			return false, nil
		}
		return false, exception("com.jwoglom.pumpx2.pump.messages.PacketArrayList$InvalidCRCException", "CRC validation failed for: %d", p.expectedOpCode)
	}
	if p.isSigned {
		if len(p.messageData) < 20 {
			return false, illegalArgument("cannot drop last 20 of %d bytes", len(p.messageData))
		}
		signedPart := p.messageData[:len(p.messageData)-20]
		got := p.messageData[len(p.messageData)-20:]
		if !bytes.Equal(got, hmacSha1(signedPart, key)) && !shouldIgnoreInvalidHmac(key) {
			return false, exception("com.jwoglom.pumpx2.pump.messages.PacketArrayList$InvalidSignedMessageHMACSignatureException", "Pump response invalid: SIGNATURE")
		}
	}
	return true, nil
}

// btResponseParse is BTResponseParser.parse for one fragment: the message
// once the last fragment is in, else nil.
func btResponseParse(p *packetArrayList, data []byte, characteristic Characteristic, env Env) (Message, error) {
	if err := p.validatePacket(data); err != nil {
		return nil, err
	}
	if p.needsMorePacket() {
		return nil, nil
	}
	ok, err := p.validate(env.authKey())
	if err != nil || !ok {
		return nil, err
	}
	cargo := append([]byte{}, p.messageData[3:]...)
	c, found := FromOpcode(int(p.opCode), characteristic)
	if !found {
		return nil, nil
	}
	m, err := Parse(c, cargo)
	if err != nil {
		// Messages.parse logs the exception and returns null.
		return nil, nil
	}
	return m, nil
}

// Parse is Messages.parse once the packets are joined: cargo (with its
// signed trailer, when the message has one) parsed into a new instance of c.
func Parse(c *Class, cargo []byte) (m Message, err error) {
	m, err = c.New()
	if err != nil {
		return nil, err
	}
	defer func() {
		if r := recover(); r != nil {
			err = recovered(r)
		}
	}()
	if err := m.Parse(cargo); err != nil {
		return nil, err
	}
	return m, nil
}
