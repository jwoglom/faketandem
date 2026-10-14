package currentstatus_test

import (
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/currentstatus"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestStreamDataReadinessResponseReadsTheSparseReadinessIDs(t *testing.T) {
	for _, tc := range []struct {
		id   int32
		want currentstatus.FreestyleLibre2Readiness
	}{
		{0, currentstatus.FreestyleLibre2ReadinessPumpReadyForFSL2ContextTransfer},
		{5, currentstatus.FreestyleLibre2ReadinessNotReadyPumpTimeIsInvalid},
		{11, currentstatus.FreestyleLibre2ReadinessNotReadyThePumpIsUnableToFulfillTheRequestDueToAnUnexpectedError},
		{1, currentstatus.FreestyleLibre2ReadinessUnknown},
		{200, currentstatus.FreestyleLibre2ReadinessUnknown},
	} {
		m := &currentstatus.StreamDataReadinessResponse{}
		if err := m.Parse([]byte{byte(tc.id), 0}); err != nil {
			t.Fatal(err)
		}
		if got := m.FreestyleLibre2Readiness(); got != tc.want {
			t.Errorf("readiness %d is %v, want %v", tc.id, got, tc.want)
		}
		if m.StreamDataType() != currentstatus.StreamDataTypeFreestyleLibre2SensorContext {
			t.Errorf("stream data type %v", m.StreamDataType())
		}
	}
}

func TestStreamDataReadinessResponseMatchesJar(t *testing.T) {
	parity.CheckClass(t, "StreamDataReadinessResponse")
}
