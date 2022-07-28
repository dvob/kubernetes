package admission

import v1 "k8s.io/api/admissionregistration/v1"

type AdmissionConfig struct {
	Modules []AdmissionModuleConfig `json:"modules"`
}

type AdmissionModuleConfig struct {
	File     string `json:"file"`
	Mutating bool   `json:"mutating"`
	Rules    []v1.RuleWithOperations
	Settings interface{}
	Debug    bool
}
