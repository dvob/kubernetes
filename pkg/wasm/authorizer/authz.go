package authorizer

import (
	"context"
	"fmt"
	"os"

	authorizationv1 "k8s.io/api/authorization/v1"
	"k8s.io/apiserver/pkg/authorization/authorizer"
	k8s "k8s.io/apiserver/pkg/authorization/authorizer"
	"k8s.io/klog/v2"
	"k8s.io/kubernetes/pkg/wasm/internal/wasi"
)

var _ k8s.Authorizer = (*Authorizer)(nil)

type Authorizer struct {
	runner          wasi.Runner
	decisionOnError authorizer.Decision
}

func NewAuthorizerWithConfig(config *AuthorizationModuleConfig) (*Authorizer, error) {
	source, err := os.ReadFile(config.File)
	if err != nil {
		return nil, err
	}

	exec, err := wasi.NewWASIDefaultRunner(source, "authz", config.Settings)
	if err != nil {
		return nil, err
	}

	return &Authorizer{
		runner:          exec,
		decisionOnError: authorizer.DecisionNoOpinion,
	}, nil
}

func (a *Authorizer) Authorize(ctx context.Context, attr authorizer.Attributes) (decision authorizer.Decision, reason string, err error) {
	// NOTE: implementation is based on webhook implementation
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
	err = a.runner.Run(ctx, req, resp)
	if err != nil {
		klog.Errorf("Failed to run wasm exec: %v", err)
		return a.decisionOnError, "", err
	}

	if resp == nil {
		klog.Errorf("response from wasm exec is nil")
		return a.decisionOnError, "", err
	}

	switch {
	case resp.Status.Denied && resp.Status.Allowed:
		fmt.Println(1)
		return authorizer.DecisionDeny, resp.Status.Reason, fmt.Errorf("wasm subject access review returned both allow and deny response")
	case resp.Status.Denied:
		return authorizer.DecisionDeny, resp.Status.Reason, nil
	case resp.Status.Allowed:
		return authorizer.DecisionAllow, resp.Status.Reason, nil
	default:
		return authorizer.DecisionNoOpinion, resp.Status.Reason, nil
	}
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
