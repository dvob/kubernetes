package authenticator

import (
	"context"
	"fmt"
	"io"
	"os"

	authv1 "k8s.io/api/authentication/v1"
	"k8s.io/apimachinery/pkg/util/yaml"
	authn "k8s.io/apiserver/pkg/authentication/authenticator"
	"k8s.io/apiserver/pkg/authentication/token/union"
	"k8s.io/kubernetes/pkg/wasm"
	"k8s.io/kubernetes/pkg/wasm/internal/wasi"
)

// newWASITokenReviewFunc returns a TokenReviewFunc which performs the proper
// encoding and decoding of TokenReview and settings to a RawRunner.
func newWASITokenReviewFunc(name string, settings interface{}, rawRunner wasi.RawRunner) TokenReviewFunc {
	runner := wasi.NewEnvelopeRunner(rawRunner, settings)
	return func(ctx context.Context, tr *authv1.TokenReview) (*authv1.TokenReview, error) {
		resp := &authv1.TokenReview{}
		err := runner.Run(ctx, tr, resp)
		if err != nil {
			return nil, fmt.Errorf("wasm module '%s' token review failed: %w", name, err)
		}
		return resp, nil
	}
}

func newWASITokenReviewFuncFromConfig(config *wasm.ModuleConfig) (TokenReviewFunc, error) {
	source, err := os.ReadFile(config.Module)
	if err != nil {
		return nil, err
	}

	runtime, err := wasi.NewRuntime(source)
	if err != nil {
		return nil, err
	}

	rawRunner := runtime.RawRunner("authn")
	if config.Debug {
		rawRunner = wasi.DebugRawRunner(rawRunner)
	}
	return newWASITokenReviewFunc(config.Name, config.Settings, rawRunner), nil
}

func NewAuthenticator(config *wasm.Config, auds authn.Audiences) (authn.Token, error) {
	authenticators := []authn.Token{}
	for i, moduleConfig := range config.Modules {
		m, err := NewAuthenticatorFromConfig(&moduleConfig, auds)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize WASM authenticator module %d: %w", i, err)
		}
		authenticators = append(authenticators, m)
	}
	return union.New(authenticators...), nil
}

func NewAuthenticatorFromConfig(config *wasm.ModuleConfig, auds authn.Audiences) (*Authenticator, error) {
	reviewFunc, err := newWASITokenReviewFuncFromConfig(config)
	if err != nil {
		return nil, err
	}
	return &Authenticator{
		review:       reviewFunc,
		implicitAuds: auds,
	}, nil
}

func NewAuthenticatorFromConfigFile(configFile string, auds authn.Audiences) (authn.Token, error) {
	file, err := os.Open(configFile)
	if err != nil {
		return nil, fmt.Errorf("failed to open WASM authenticator configuratio: %w", err)
	}
	defer file.Close()
	return NewAuthenticatorFromReader(file, auds)
}

func NewAuthenticatorFromReader(configInput io.Reader, auds authn.Audiences) (authn.Token, error) {
	config := &wasm.Config{}
	decoder := yaml.NewYAMLOrJSONDecoder(configInput, 4096)
	err := decoder.Decode(config)
	if err != nil {
		return nil, fmt.Errorf("failed to read WASM authenticator configuration: %w", err)
	}
	config.Default()
	err = config.Validate()
	if err != nil {
		return nil, fmt.Errorf("invalid module configuration: %w", err)
	}
	return NewAuthenticator(config, auds)
}
