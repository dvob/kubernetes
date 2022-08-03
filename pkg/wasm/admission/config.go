package admission

import v1 "k8s.io/api/admissionregistration/v1"

type Config struct {
	Modules []ModuleConfig `json:"modules"`
}

type ModuleType string

const (
	ModuleTypeWASI       = "wasi"
	ModuleTypeKubewarden = "kubewarden"
)

type ModuleConfig struct {
	Name     string                  `json:"name"`
	Type     ModuleType              `json:"type"`
	Module   string                  `json:"file"`
	Mutating bool                    `json:"mutating"`
	Rules    []v1.RuleWithOperations `json:"rules"`
	Settings interface{}             `json:"settings"`
}
