package controlstream_test

import (
	"reflect"
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/controlstream"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestLoadCartridgeStateStreamResponseParsesAsDetectingCartridge(t *testing.T) {
	built := controlstream.NewLoadCartridgeStateStreamResponse(-3)
	parsed, err := controlstream.NewDetectingCartridgeStateStreamResponseFromCargo(built.Cargo())
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(*parsed, built.DetectingCartridgeStateStreamResponse) || !parsed.LoadCartridgeState || parsed.StateID != -3 {
		t.Errorf("parsed %+v, built %+v", parsed, built)
	}
}

func TestLoadCartridgeStateStreamResponseMatchesJar(t *testing.T) {
	parity.CheckClass(t, "LoadCartridgeStateStreamResponse")
}
