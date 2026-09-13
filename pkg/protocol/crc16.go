package protocol

// crc16Table is the standard CRC-CCITT (polynomial 0x1021) lookup table used by
// every Tandem pump message. It is generated once at init rather than written
// out as a 256-entry literal so it cannot drift from the polynomial.
var crc16Table = func() [256]uint16 {
	var table [256]uint16
	for i := 0; i < 256; i++ {
		crc := uint16(i) << 8
		for bit := 0; bit < 8; bit++ {
			if crc&0x8000 != 0 {
				crc = (crc << 1) ^ 0x1021
			} else {
				crc <<= 1
			}
		}
		table[i] = crc
	}
	return table
}()

// CalculateCRC16 computes the two trailing CRC bytes a Tandem pump message
// carries, in the order they appear on the wire (low byte first).
//
// This is the same routine as TandemKit's CalculateCRC16
// (Sources/TandemCore/Common/CRC16.swift) and pumpX2's Packetize: seed 0xFFFF,
// table-driven CRC-CCITT, emitted little-endian. The CRC covers the message
// header (opcode, txId, cargo length) plus the cargo and, for signed messages,
// the 24-byte trailer -- but NOT the per-fragment [remaining][txId] framing
// bytes, which are added afterwards.
func CalculateCRC16(data []byte) []byte {
	crc := uint16(0xFFFF)
	for _, b := range data {
		crc = (crc << 8) ^ crc16Table[b^byte(crc>>8)]
	}
	return []byte{byte(crc & 0xFF), byte(crc >> 8)}
}
