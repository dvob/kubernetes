package admission

type AdmissionConfig struct {
	Modules []AdmissionModuleConfig `json:"modules"`
}

type AdmissionModuleConfig struct {
	File     string `json:"file"`
	Mutating bool   `json:"mutating"`
	Settings interface{}
}
