package wasm

import (
	"encoding/json"
	"fmt"
	"path/filepath"
)

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
	Name     string      `json:"name,omitempty"`
	Module   string      `json:"module"`
	Settings interface{} `json:"settings,omitempty"`
	Debug    bool        `json:"debug,omitempty"`
}

func (mc *ModuleConfig) Default() {
	if mc.Name == "" {
		mc.Name = filepath.Base(mc.Module)
	}
}

func (mc *ModuleConfig) Validate() error {
	if mc.Module == "" {
		return fmt.Errorf("no module path specified")
	}
	_, err := json.Marshal(mc.Settings)
	if err != nil {
		return fmt.Errorf("cannot serialize settings of module '%s': %w", mc.Name, err)
	}
	return nil
}
