package control_test

import (
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/control"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestStartDexcomG6SensorSessionRequest(t *testing.T) {
	tester := parity.MessageTester{Env: messages.Env{TimeSinceReset: "1"}}
	tester.Test(t, "012eb22e1a5f24210af31f6806b4cc57d79bb047", 46, 1, messages.Control, control.NewStartDexcomG6SensorSessionRequest(9311),
		"002e5f4261e734aaa05941d57f7901")
}

func TestStartDexcomG6SensorSessionRequestMatchesJar(t *testing.T) {
	parity.CheckClass(t, "StartDexcomG6SensorSessionRequest")
}
