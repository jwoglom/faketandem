package authentication_test

import (
	"bytes"
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/authentication"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestJpake1bResponse(t *testing.T) {
	round := decodeHex(t, "4104dc82c0b7f60e601ebed41ebafac79dac6b23055d6c2949e3bd7643acd951c400ca60513dbff125da5238e0a7eee27ff4533afded0725ad2804987c90646ade0f41048177dda93af133fcfcc3a78408af82370d76af3ecfe78bc16c732310ed00b188ae0b4c2769876ac29d6c65a205c96dd518e3166aa57d61bca1a6756aabbe4f6920c472ac523abdd69e678149c128daa073861c6c9371f04254d158b6481f2226ba")
	expected, err := authentication.NewJpake1bResponse(0, round)
	if err != nil {
		t.Fatal(err)
	}
	parsed := parity.TestMessage(t, "00012301a700004104dc82c0b7f60e601ebed41ebafac79dac6b23055d6c2949e3bd7643acd951c400ca60513dbff125da5238e0a7eee27ff4533afded0725ad2804987c90646ade0f41048177dda93af133fcfcc3a78408af82370d76af3ecfe78bc16c732310ed00b188ae0b4c2769876ac29d6c65a205c96dd518e3166aa57d61bca1a6756aabbe4f6920c472ac523abdd69e678149c128daa073861c6c9371f04254d158b6481f2226ba3927",
		1, 10, messages.Authorization, expected).(*authentication.Jpake1bResponse)
	if !bytes.Equal(parsed.CentralChallengeHash, round) {
		t.Errorf("round %x, want %x", parsed.CentralChallengeHash, round)
	}
}

func TestJpake1bResponseMatchesJar(t *testing.T) {
	parity.CheckClass(t, "Jpake1bResponse")
}
