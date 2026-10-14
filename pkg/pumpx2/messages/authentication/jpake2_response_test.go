package authentication_test

import (
	"bytes"
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/authentication"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestJpake2Response(t *testing.T) {
	round := decodeHex(t, "0300174104634f4cfde021623a609c98311ae7508b41126542de029b76b1781de8f3db6e723950a8e64715c97cf44d80a6976b91ab846cdbf732bf160c1c0223f0a674a4734104732ff948fed2dbb06806502a7ae7ca0afdf1991eab79c866ff53e9d23bd4d1886f3cb55663b7979947f6a96d52511ac843b9d28b20805f263365bc50d1b93d7620bec12ade679a6b482ba5bfbe973e91dcdcd87bc3ab04090803db2945cca39e4c")
	expected, err := authentication.NewJpake2Response(0, round)
	if err != nil {
		t.Fatal(err)
	}
	parsed := parity.TestMessage(t, "00022502aa00000300174104634f4cfde021623a609c98311ae7508b41126542de029b76b1781de8f3db6e723950a8e64715c97cf44d80a6976b91ab846cdbf732bf160c1c0223f0a674a4734104732ff948fed2dbb06806502a7ae7ca0afdf1991eab79c866ff53e9d23bd4d1886f3cb55663b7979947f6a96d52511ac843b9d28b20805f263365bc50d1b93d7620bec12ade679a6b482ba5bfbe973e91dcdcd87bc3ab04090803db2945cca39e4c9be8",
		2, 10, messages.Authorization, expected).(*authentication.Jpake2Response)
	if !bytes.Equal(parsed.CentralChallengeHash, round) {
		t.Errorf("round %x, want %x", parsed.CentralChallengeHash, round)
	}
}

func TestJpake2ResponseMatchesJar(t *testing.T) {
	parity.CheckClass(t, "Jpake2Response")
}
