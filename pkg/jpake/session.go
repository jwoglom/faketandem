package jpake

import (
	"crypto/hmac"
	"crypto/sha256"
)

// PairingCodeToBytes is pumpX2's JpakeAuthBuilder.pairingCodeToBytes: each
// digit as its ASCII byte, anything else as 0xFF. The result is the EC-JPAKE
// password.
func PairingCodeToBytes(code string) []byte {
	out := make([]byte, 0, len(code))
	for _, c := range code {
		if c >= '0' && c <= '9' {
			out = append(out, byte(c))
		} else {
			out = append(out, 0xFF)
		}
	}
	return out
}

// HMACSHA256 is HMAC-SHA256(key, data), pumpX2's HmacSha256.hmacSha256(data, key).
func HMACSHA256(key, data []byte) []byte {
	mac := hmac.New(sha256.New, key)
	mac.Write(data)
	return mac.Sum(nil)
}

// HKDF is pumpX2's Hkdf.build(nonce, keyMaterial): HMAC-SHA256 keyed on the
// nonce over the key material, expanded to one 32-byte block.
func HKDF(nonce, keyMaterial []byte) []byte {
	prk := HMACSHA256(nonce, keyMaterial)
	return HMACSHA256(prk, []byte{0x01})
}

// KeyConfirmation is the round-4 digest each side sends over its own nonce:
// HMAC-SHA256 keyed on HKDF(serverNonce3, derivedSecret).
func KeyConfirmation(serverNonce3, derivedSecret, nonce []byte) []byte {
	return HMACSHA256(HKDF(serverNonce3, derivedSecret), nonce)
}
