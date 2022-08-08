package admission

import (
	v1 "k8s.io/api/admissionregistration/v1"
	"k8s.io/kubernetes/pkg/wasm"
)

type ModuleType string

const (
	ModuleTypeWASI       = "wasi"
	ModuleTypeKubewarden = "kubewarden"
)

type Config struct {
	Modules []ModuleConfig `json:"modules"`
}

type ModuleConfig struct {
	wasm.ModuleConfig
	Type     ModuleType              `json:"type"`
	Mutating bool                    `json:"mutating"`
	Rules    []v1.RuleWithOperations `json:"rules"`
}
