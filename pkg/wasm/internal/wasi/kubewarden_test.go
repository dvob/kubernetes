package wasi

import (
	"context"
	"os"
	"testing"
)

var (
	safeAnnotationsModule = "../../testmodules/kubewarden/safe-annotations_v0.2.0.wasm"
)

func TestKubewarden(t *testing.T) {
	ctx := context.Background()

	moduleSource, err := os.ReadFile(safeAnnotationsModule)
	if err != nil {
		t.Fatal(err)
	}
	runtime, err := NewWAPCRuntime(moduleSource)
	if err != nil {
		t.Fatal(err)
	}

	validateSettings := NewJSONRunner(runtime.RawRunner("validate_settings"))

	settings := struct {
		Foo               string `json:"foo"`
		DeniedAnnotations int    `json:"denied_annotations"`
	}{
		Foo:               "bla",
		DeniedAnnotations: 12,
	}

	resp := &SettingsValidationResponse{}
	err = validateSettings.Run(ctx, settings, resp)
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("%#+v", resp)
}
