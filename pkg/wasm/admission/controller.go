package admission

import (
	"context"
	"fmt"
	"io"

	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/apiserver/pkg/admission"
	k8s "k8s.io/apiserver/pkg/admission"
)

// PluginName indicates the name of the admission plugin.
const PluginName = "WASM"

// Register registers a plugin
func Register(plugins *admission.Plugins) {
	plugins.Register(PluginName, func(config io.Reader) (admission.Interface, error) {
		return NewControllerFromReader(config)
	})
}

var _ k8s.MutationInterface = (*ControllerChain)(nil)
var _ k8s.ValidationInterface = (*ControllerChain)(nil)

type ControllerChain struct {
	mutator   []admission.MutationInterface
	validator []admission.ValidationInterface
}

func (c *ControllerChain) Handles(operation admission.Operation) bool {
	// we run admission for all request. later in Admit and Validate we check if we
	// run the request through the WASM logic by checking the rules
	return true
}

func (c *ControllerChain) Validate(ctx context.Context, attr admission.Attributes, o admission.ObjectInterfaces) (err error) {
	for _, validator := range c.validator {
		err := validator.Validate(ctx, attr, o)
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *ControllerChain) Admit(ctx context.Context, attr admission.Attributes, o admission.ObjectInterfaces) (err error) {
	for _, mutator := range c.mutator {
		err := mutator.Admit(ctx, attr, o)
		if err != nil {
			return err
		}
	}
	return nil
}

func New(config *Config) (*ControllerChain, error) {
	controller := &ControllerChain{}
	for _, moduleConfig := range config.Modules {
		module, err := NewController(&moduleConfig)
		if err != nil {
			return nil, err
		}

		if moduleConfig.Mutating {
			controller.mutator = append(controller.mutator, module)
		} else {
			controller.validator = append(controller.validator, module)
		}
	}
	return controller, nil
}

func NewControllerFromReader(configInput io.Reader) (*ControllerChain, error) {
	config := &Config{}
	decoder := yaml.NewYAMLOrJSONDecoder(configInput, 4096)
	err := decoder.Decode(config)
	if err != nil {
		return nil, err
	}
	config.Default()
	err = config.Validate()
	if err != nil {
		return nil, fmt.Errorf("invalid module configuration: %w", err)
	}
	return New(config)
}
