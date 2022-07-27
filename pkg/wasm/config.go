package wasm

type AdmissionConfig struct {
	Modules []AdmissionModuleConfig `json:"modules"`
}

type AdmissionModuleConfig struct {
	File     string `json:"file"`
	Mutating bool   `json:"mutating"`
	Settings interface{}
}

type AuthorizationConfig struct {
	Modules []AuthorizationModuleConfig `json:"modules"`
}

type AuthorizationModuleConfig struct {
	File     string `json:"file"`
	Settings interface{}
}
