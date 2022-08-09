package admission

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"k8s.io/api/admission/v1"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/klog/v2"
	"k8s.io/kubernetes/pkg/wasm/internal/wasi"
)

// newWASIAdmissionReviewFunc returns a AdmissionReviewFunc which performs the proper
// encoding and decoding of AdmissionReview and settings to a RawRunner.
func newWASIAdmissionReviewFunc(settings interface{}, rawRunner wasi.RawRunner) AdmissionReviewFunc {
	runner := wasi.NewEnvelopeRunner(rawRunner, settings)
	return func(ctx context.Context, ar *v1.AdmissionReview) (*v1.AdmissionReview, error) {
		resp := &v1.AdmissionReview{}
		err := runner.Run(ctx, ar, resp)
		if err != nil {
			return nil, err
		}
		return resp, nil
	}
}

// newWASIAdmissionReviewFuncFromConfig initializes a review function which passes
// serialized reviews to a WASM runtime.
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
	return newWASIAdmissionReviewFunc(config.Settings, rawRunner), nil
}

// newKubewardenAdmissionReviewFuncFromConfig initializes a review function
// which passes reviews to a Kubewarden module.
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

// newAdmissionReviewFunc returns a new review function bases on the
// ModuleConfiguration
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

// withLogger wraps a AdmissionReviewFunc and logs
func withLogger(name string, fn AdmissionReviewFunc) AdmissionReviewFunc {
	return func(ctx context.Context, ar *v1.AdmissionReview) (*v1.AdmissionReview, error) {
		start := time.Now()
		defer func() {
			klog.InfoS("run admission review", "module_name", name, "duration", time.Now().Sub(start))
		}()
		return fn(ctx, ar)
	}
}

// withNamedErr wraps a AdmissionReviewFunc and adds context information to a
// potential error.
func withNamedErr(name string, fn AdmissionReviewFunc) AdmissionReviewFunc {
	return func(ctx context.Context, ar *v1.AdmissionReview) (*v1.AdmissionReview, error) {
		ar, err := fn(ctx, ar)
		if err != nil {
			return nil, fmt.Errorf("admission review failed in wasm module '%s': %w", name, err)
		}
		return ar, nil
	}
}

func NewController(config *ModuleConfig) (*Controller, error) {
	reviewFunc, err := newAdmissionReviewFunc(config)
	if err != nil {
		return nil, err
	}
	reviewFunc = withNamedErr(config.Name, reviewFunc)
	reviewFunc = withLogger(config.Name, reviewFunc)
	return &Controller{
		review:   reviewFunc,
		mutating: config.Mutating,
		rules:    config.Rules,
	}, nil
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
