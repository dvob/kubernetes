package wasm

type Config struct {
	Modules []ModuleConfig `json:"modules"`
}

type ModuleConfig struct {
	Name     string      `json:"name,omitempty"`
	Module   string      `json:"module"`
	Settings interface{} `json:"settings,omitempty"`
	Debug    bool        `json:"debug,omitempty"`
}
