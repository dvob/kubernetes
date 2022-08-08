package admission

import (
	"context"
	"fmt"
	"os"

	"k8s.io/api/admission/v1"
	"k8s.io/kubernetes/pkg/wasm/internal/wasi"
)

// newWASIAdmissionReviewFunc returns a AdmissionReviewFunc which performs the proper
// encoding and decoding of AdmissionReview and settings to a RawRunner.
func newWASIAdmissionReviewFunc(name string, settings interface{}, rawRunner wasi.RawRunner) AdmissionReviewFunc {
	runner := wasi.NewEnvelopeRunner(rawRunner, settings)
	return func(ctx context.Context, ar *v1.AdmissionReview) (*v1.AdmissionReview, error) {
		resp := &v1.AdmissionReview{}
		err := runner.Run(ctx, ar, resp)
		if err != nil {
			return nil, fmt.Errorf("wasm module '%s' admission review failed: %w", name, err)
		}
		return resp, nil
	}
}

func newWASIAdmissionReviewFuncFromConfig(config *ModuleConfig) (AdmissionReviewFunc, error) {
	source, err := os.ReadFile(config.Module)
	if err != nil {
		return nil, err
	}

	runtime, err := wasi.NewRuntime(source)
	if err != nil {
		return nil, err
	}

	fnName := "validate"
	if !runtime.HasFunction(fnName) {
		return nil, fmt.Errorf("missing function '%s' in module '%s'", fnName, config.Name)
	}

	rawRunner := runtime.RawRunner("validate")
	if config.Debug {
		rawRunner = wasi.DebugRawRunner(rawRunner)
	}
	return newWASIAdmissionReviewFunc(config.Name, config.Settings, rawRunner), nil
}

func newKubewardenAdmissionReviewFuncFromConfig(config *ModuleConfig) (AdmissionReviewFunc, error) {
	moduleSource, err := os.ReadFile(config.Module)
	if err != nil {
		return nil, err
	}

	mod, err := wasi.NewKubewardenModule(moduleSource, config.Debug)
	if err != nil {
		return nil, err
	}

	err = mod.ValidateSettings(context.Background(), config.Settings)
	if err != nil {
		return nil, err
	}

	reviewFn := func(ctx context.Context, ar *v1.AdmissionReview) (*v1.AdmissionReview, error) {
		return mod.Validate(ctx, ar, config.Settings)
	}
	return reviewFn, nil
}

func newAdmissionReviewFunc(config *ModuleConfig) (AdmissionReviewFunc, error) {
	switch config.Type {
	case ModuleTypeWASI:
		return newWASIAdmissionReviewFuncFromConfig(config)
	case ModuleTypeKubewarden:
		return newKubewardenAdmissionReviewFuncFromConfig(config)
	default:
		return nil, fmt.Errorf("unknown module type '%s'", config.Type)
	}
}

func NewController(config *ModuleConfig) (*Controller, error) {
	reviewFunc, err := newAdmissionReviewFunc(config)
	if err != nil {
		return nil, err
	}
	return &Controller{
		review:   reviewFunc,
		mutating: config.Mutating,
		rules:    config.Rules,
	}, nil
}
