package authorizer

type AuthorizationConfig struct {
	Modules []AuthorizationModuleConfig `json:"modules"`
}

type AuthorizationModuleConfig struct {
	File     string `json:"file"`
	Settings interface{}
}
