package historylog_test

import (
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestNonexistentHistoryLogStreamRequestMatchesJar(t *testing.T) {
	parity.CheckClass(t, "NonexistentHistoryLogStreamRequest")
}
