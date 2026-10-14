package control_test

import (
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/control"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestFillCannulaRequest(t *testing.T) {
	expected, err := control.NewFillCannulaRequest(300)
	if err != nil {
		t.Fatal(err)
	}
	parity.TestMessage(t, "016d986d1a2c010f152820b9273d0fb99c1241f1", 109, 1, messages.Control, expected, "006db9da764426a0aa99bb708571f4")
}

func TestFillCannulaRequestRefusesAPrimeOutsideItsBounds(t *testing.T) {
	for _, milliunits := range []int32{0, -1, 3001} {
		if _, err := control.NewFillCannulaRequest(milliunits); err == nil {
			t.Errorf("a prime of %d milliunits was built", milliunits)
		}
	}
	if _, err := control.NewFillCannulaRequest(3000); err != nil {
		t.Errorf("a prime of 3000 milliunits: %v", err)
	}
}

func TestFillCannulaRequestMatchesJar(t *testing.T) {
	parity.CheckClass(t, "FillCannulaRequest")
}
