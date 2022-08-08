package wasm

import (
	"testing"
)

func TestNameDefaulting(t *testing.T) {
	config := &ModuleConfig{
		Module: "abce/foo.wasm",
	}
	config.Default()

	if config.Name != "foo.wasm" {
		t.Fatalf("want=foo.wasm, got='%s'", config.Name)
	}
}

func TestValidation(t *testing.T) {
	for _, test := range []struct {
		name      string
		config    ModuleConfig
		shouldErr bool
	}{
		{
			name: "correct",
			config: ModuleConfig{
				Module: "abc.wasm",
			},
			shouldErr: false,
		},
		{
			name:      "empty module",
			config:    ModuleConfig{},
			shouldErr: true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			err := test.config.Validate()
			if err == nil && test.shouldErr {
				t.Error("config validation should fail")
			}
			if err != nil && !test.shouldErr {
				t.Error("config validation shoud not fail")
			}
		})
	}
}
