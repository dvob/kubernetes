package wasi

import (
	"context"
	"os"
	"strings"
	"testing"
)

var (
	safeAnnotationsModule = "../../testmodules/kubewarden/safe-annotations_v0.2.0.wasm"
)

func TestKubewarden(t *testing.T) {
	ctx := context.Background()

	if _, err := os.Stat(safeAnnotationsModule); err != nil {
		t.Skip("safe-annotations module not available")
	}

	moduleSource, err := os.ReadFile(safeAnnotationsModule)
	if err != nil {
		t.Fatal(err)
	}

	mod, err := NewKubewardenModule(moduleSource)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("invalid settings", func(t *testing.T) {
		// in the invalidSettings the DeniedAnnotations are an int instead of a map[string]string
		invalidSettings := struct {
			DeniedAnnotations int `json:"denied_annotations"`
		}{}

		err := mod.ValidateSettings(ctx, invalidSettings)
		if err == nil {
			t.Fatalf("expected error because of invalid settings")
		}

		expectedErrorString := "invalid settings"
		if !strings.Contains(err.Error(), expectedErrorString) {
			t.Fatalf("error shold contain string '%s', got=%s", expectedErrorString, err.Error())
		}
	})
	t.Run("valid settings", func(t *testing.T) {
		validSettings := struct {
			DeniedAnnotations []string `json:"denied_annotations"`
		}{
			DeniedAnnotations: []string{
				"foo",
			},
		}

		err := mod.ValidateSettings(ctx, validSettings)
		if err != nil {
			t.Fatal(err)
		}
	})
}
