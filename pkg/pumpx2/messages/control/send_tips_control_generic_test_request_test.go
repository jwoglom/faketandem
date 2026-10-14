package control_test

import (
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestSendTipsControlGenericTestRequestMatchesJar(t *testing.T) {
	parity.CheckClass(t, "SendTipsControlGenericTestRequest")
}
