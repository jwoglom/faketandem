package authentication_test

import (
	"bytes"
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/authentication"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestJpake1bRequest(t *testing.T) {
	round := decodeHex(t, "4104faf1c1c7737041a88872ba435c79fe67b14fd793f452b4e1a41cf4e8569ed1b67c9ab0523d97f79a536ab498d23d0d941443b319402f9406393c1c800c203cac41042dd19aad7f25eabd064b6f2ee474ac900fd61550aef781eca29b4e9fdb921a1ecd395e88cc0ee6f09c961d49e3e31d22cb178ea63ce560ebe71127521b6f7d3320303b8172c67d855dbdeeac9a32f1f067ea4924b95701563021874c2f78785e6a")
	expected := authentication.NewJpake1bRequest(0, round)
	parsed := parity.TestMessage(t, "09012201a700004104faf1c1c7737041a88872ba", 1, 10, messages.Authorization, expected,
		"0801435c79fe67b14fd793f452b4e1a41cf4e856",
		"07019ed1b67c9ab0523d97f79a536ab498d23d0d",
		"0601941443b319402f9406393c1c800c203cac41",
		"0501042dd19aad7f25eabd064b6f2ee474ac900f",
		"0401d61550aef781eca29b4e9fdb921a1ecd395e",
		"030188cc0ee6f09c961d49e3e31d22cb178ea63c",
		"0201e560ebe71127521b6f7d3320303b8172c67d",
		"0101855dbdeeac9a32f1f067ea4924b957015630",
		"000121874c2f78785e6afd38",
	).(*authentication.Jpake1bRequest)
	if parsed.AppInstanceID != 0 || !bytes.Equal(parsed.CentralChallenge, round) {
		t.Errorf("app instance ID %d, round %x", parsed.AppInstanceID, parsed.CentralChallenge)
	}
}

func TestJpake1bRequestMatchesJar(t *testing.T) {
	parity.CheckClass(t, "Jpake1bRequest")
}
