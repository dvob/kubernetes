package authorizer

import (
	"context"
	"fmt"
	"io"
	"os"

	v1 "k8s.io/api/authorization/v1"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/apiserver/pkg/authorization/authorizer"
	k8s "k8s.io/apiserver/pkg/authorization/authorizer"
	"k8s.io/apiserver/pkg/authorization/union"
	"k8s.io/kubernetes/pkg/wasm"
	"k8s.io/kubernetes/pkg/wasm/internal/wasi"
)

// newWASISubjectAccessReviewFunc returns a SubjectAccessReviewFunc which performs the proper
// encoding and decoding of the SubjectAccessReview and settings to a RawRunner.
func newWASISubjectAccessReviewFunc(name string, settings interface{}, rawRunner wasi.RawRunner) SubjectAccessReviewFunc {
	runner := wasi.NewEnvelopeRunner(rawRunner, settings)
	return func(ctx context.Context, tr *v1.SubjectAccessReview) (*v1.SubjectAccessReview, error) {
		resp := &v1.SubjectAccessReview{}
		err := runner.Run(ctx, tr, resp)
		if err != nil {
			return nil, fmt.Errorf("wasm module '%s' subject access review failed: %w", name, err)
		}
		return resp, nil
	}
}

func newWASISubjectAccessReviewFuncFromConfig(config *wasm.ModuleConfig) (SubjectAccessReviewFunc, error) {
	source, err := os.ReadFile(config.Module)
	if err != nil {
		return nil, err
	}

	runtime, err := wasi.NewRuntime(source)
	if err != nil {
		return nil, err
	}

	rawRunner := runtime.RawRunner("authz")
	if config.Debug {
		rawRunner = wasi.DebugRawRunner(rawRunner)
	}
	return newWASISubjectAccessReviewFunc(config.Name, config.Settings, rawRunner), nil
}

func NewAuthorizer(config *wasm.ModuleConfig) (*Authorizer, error) {
	reviewFunc, err := newWASISubjectAccessReviewFuncFromConfig(config)
	if err != nil {
		return nil, err
	}
	return &Authorizer{
		review:          reviewFunc,
		decisionOnError: authorizer.DecisionNoOpinion,
	}, nil
}

func New(config *wasm.Config) (k8s.Authorizer, k8s.RuleResolver, error) {
	authorizers := []k8s.Authorizer{}
	ruleResolvers := []k8s.RuleResolver{}
	for _, moduleConfig := range config.Modules {
		module, err := NewAuthorizer(&moduleConfig)
		if err != nil {
			return nil, nil, err
		}
		authorizers = append(authorizers, module)
		ruleResolvers = append(ruleResolvers, module)
	}
	return union.New(authorizers...), union.NewRuleResolvers(ruleResolvers...), nil
}

func NewAuthorizerFormConfigFile(configFile string) (k8s.Authorizer, k8s.RuleResolver, error) {
	file, err := os.Open(configFile)
	if err != nil {
		return nil, nil, err
	}
	return NewAuthorizerFromReader(file)
}

func NewAuthorizerFromReader(configInput io.Reader) (k8s.Authorizer, k8s.RuleResolver, error) {
	config := &wasm.Config{}
	decoder := yaml.NewYAMLOrJSONDecoder(configInput, 4096)
	err := decoder.Decode(config)
	if err != nil {
		return nil, nil, err
	}
	config.Default()
	err = config.Validate()
	if err != nil {
		return nil, nil, fmt.Errorf("invalid module configuration: %w", err)
	}
	return New(config)
}
