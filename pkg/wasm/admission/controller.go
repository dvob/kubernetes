package admission

import (
	"context"
	"encoding/json"
	"io"

	"k8s.io/apiserver/pkg/admission"
)

// PluginName indicates the name of the admission plugin.
const PluginName = "WASM"

// Register registers a plugin
func Register(plugins *admission.Plugins) {
	plugins.Register(PluginName, func(config io.Reader) (admission.Interface, error) {
		return NewControllerFromReader(config)
	})
}

func NewControllerFromReader(configInput io.Reader) (*Controller, error) {
	data, err := io.ReadAll(configInput)
	if err != nil {
		return nil, err
	}
	config := &Config{}
	err = json.Unmarshal(data, config)
	if err != nil {
		return nil, err
	}
	return NewController(config)
}

func NewController(config *Config) (*Controller, error) {
	controller := &Controller{}
	for _, moduleConfig := range config.Modules {
		module, err := NewModule(&moduleConfig)
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

type Controller struct {
	mutator   []admission.MutationInterface
	validator []admission.ValidationInterface
}

func (c *Controller) Handles(operation admission.Operation) bool {
	// we run admission for all request. later in Admit and Validate we check if we
	// run the request through the WASM logic by checking the rules
	return true
}

func (c *Controller) Validate(ctx context.Context, attr admission.Attributes, o admission.ObjectInterfaces) (err error) {
	for _, validator := range c.validator {
		err := validator.Validate(ctx, attr, o)
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *Controller) Admit(ctx context.Context, attr admission.Attributes, o admission.ObjectInterfaces) (err error) {
	for _, mutator := range c.mutator {
		err := mutator.Admit(ctx, attr, o)
		if err != nil {
			return err
		}
	}
	return nil
}
