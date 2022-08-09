package admission

import (
	"context"

	"k8s.io/apiserver/pkg/admission"
	k8s "k8s.io/apiserver/pkg/admission"
)

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
