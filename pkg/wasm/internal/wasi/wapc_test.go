package wasi

import (
	"context"
	"os"
	"reflect"
	"testing"
)

const (
	wapcTestModuleFile = "../../testmodules/target/wasm32-wasi/debug/test_wapc.wasm"
)

func newWAPCRuntime(t *testing.T) *WAPCRuntime {
	source, err := os.ReadFile(wapcTestModuleFile)
	if err != nil {
		t.Fatal(err)
	}
	runtime, err := NewWAPCRuntime(source)
	if err != nil {
		t.Fatal(err)
	}

	return runtime
}

func TestWAPCRuntime(t *testing.T) {
	wasiExec := newWAPCRuntime(t)
	ctx := context.Background()

	input := []byte("input str")
	expectedOutput := []byte("INPUT STR")

	output, err := wasiExec.Run(ctx, "run", input)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(output, expectedOutput) {
		t.Fatalf("want=%s, got=%s", expectedOutput, output)
	}
}
