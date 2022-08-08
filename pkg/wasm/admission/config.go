package admission

import (
	"fmt"

	v1 "k8s.io/api/admissionregistration/v1"
	"k8s.io/kubernetes/pkg/wasm"
)

type ModuleType string

const (
	ModuleTypeWASI       = "wasi"
	ModuleTypeKubewarden = "kubewarden"
)

func (mt *ModuleType) Validate() error {
	switch *mt {
	case ModuleTypeWASI, ModuleTypeKubewarden:
		return nil
	default:
		return fmt.Errorf("unknown module type: '%v'", mt)
	}
}

type Config struct {
	Modules []ModuleConfig `json:"modules"`
}

func (c *Config) Default() {
	for i := range c.Modules {
		c.Modules[i].Default()
	}
}

func (c *Config) Validate() error {
	for i := range c.Modules {
		err := c.Modules[i].Validate()
		if err != nil {
			return err
		}
	}
	return nil
}

type ModuleConfig struct {
	wasm.ModuleConfig
	Type     ModuleType              `json:"type,omitempty"`
	Mutating bool                    `json:"mutating,omitempty"`
	Rules    []v1.RuleWithOperations `json:"rules,omitempty"`
}

func (mc *ModuleConfig) Default() {
	mc.ModuleConfig.Default()
	if mc.Type == "" {
		mc.Type = ModuleTypeWASI
	}
}

func (mc *ModuleConfig) Validate() error {
	err := mc.ModuleConfig.Validate()
	if err != nil {
		return err
	}
	err = mc.Type.Validate()
	if err != nil {
		return err
	}
	return nil
}
