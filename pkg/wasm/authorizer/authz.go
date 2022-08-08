package authorizer

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"

	authorizationv1 "k8s.io/api/authorization/v1"
	"k8s.io/apiserver/pkg/authentication/user"
	"k8s.io/apiserver/pkg/authorization/authorizer"
	k8s "k8s.io/apiserver/pkg/authorization/authorizer"
	"k8s.io/apiserver/pkg/authorization/union"
	"k8s.io/klog/v2"
	"k8s.io/kubernetes/pkg/wasm"
	"k8s.io/kubernetes/pkg/wasm/internal/wasi"
)

var _ k8s.Authorizer = (*Module)(nil)

type noRulesImpl struct{}

func NewAuthorizer(config *wasm.Config) (k8s.Authorizer, k8s.RuleResolver, error) {
	authorizers := []k8s.Authorizer{}
	ruleResolvers := []k8s.RuleResolver{}
	for _, moduleConfig := range config.Modules {
		module, err := NewModule(&moduleConfig)
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
	data, err := io.ReadAll(configInput)
	if err != nil {
		return nil, nil, err
	}
	config := &wasm.Config{}
	err = json.Unmarshal(data, config)
	if err != nil {
		return nil, nil, err
	}
	return NewAuthorizer(config)
}

type Module struct {
	name            string
	runner          wasi.Runner
	decisionOnError authorizer.Decision
}

func NewModule(config *wasm.ModuleConfig) (*Module, error) {
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

	runner := wasi.NewEnvelopeRunner(rawRunner, config.Settings)

	return &Module{
		name:            config.Name,
		runner:          runner,
		decisionOnError: authorizer.DecisionNoOpinion,
	}, nil
}

func (m *Module) Authorize(ctx context.Context, attr authorizer.Attributes) (decision authorizer.Decision, reason string, err error) {
	req := &authorizationv1.SubjectAccessReview{}
	if user := attr.GetUser(); user != nil {
		req.Spec = authorizationv1.SubjectAccessReviewSpec{
			User:   user.GetName(),
			UID:    user.GetUID(),
			Groups: user.GetGroups(),
			Extra:  convertToSARExtra(user.GetExtra()),
		}
	}

	if attr.IsResourceRequest() {
		req.Spec.ResourceAttributes = &authorizationv1.ResourceAttributes{
			Namespace:   attr.GetNamespace(),
			Verb:        attr.GetVerb(),
			Group:       attr.GetAPIGroup(),
			Version:     attr.GetAPIVersion(),
			Resource:    attr.GetResource(),
			Subresource: attr.GetSubresource(),
			Name:        attr.GetName(),
		}
	} else {
		req.Spec.NonResourceAttributes = &authorizationv1.NonResourceAttributes{
			Path: attr.GetPath(),
			Verb: attr.GetVerb(),
		}
	}

	resp := &authorizationv1.SubjectAccessReview{}
	err = m.runner.Run(ctx, req, resp)
	if err != nil {
		klog.ErrorS(err, "failed to run wasm authorization module", "module_name", m.name)
		return m.decisionOnError, "", err
	}

	if resp == nil {
		klog.Errorf("response from wasm exec is nil")
		return m.decisionOnError, "", err
	}

	switch {
	case resp.Status.Denied && resp.Status.Allowed:
		return authorizer.DecisionDeny, resp.Status.Reason, fmt.Errorf("wasm subject access review returned both allow and deny response")
	case resp.Status.Denied:
		return authorizer.DecisionDeny, resp.Status.Reason, nil
	case resp.Status.Allowed:
		return authorizer.DecisionAllow, resp.Status.Reason, nil
	default:
		return authorizer.DecisionNoOpinion, resp.Status.Reason, nil
	}
}

func (m *Module) RulesFor(_ user.Info, _ string) ([]authorizer.ResourceRuleInfo, []authorizer.NonResourceRuleInfo, bool, error) {
	var (
		resourceRules    []authorizer.ResourceRuleInfo
		nonResourceRules []authorizer.NonResourceRuleInfo
	)
	incomplete := true
	return resourceRules, nonResourceRules, incomplete, fmt.Errorf("wasm authorizer does not support user rule resolution")
}

func convertToSARExtra(extra map[string][]string) map[string]authorizationv1.ExtraValue {
	if extra == nil {
		return nil
	}
	ret := map[string]authorizationv1.ExtraValue{}
	for k, v := range extra {
		ret[k] = authorizationv1.ExtraValue(v)
	}

	return ret
}
