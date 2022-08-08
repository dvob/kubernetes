package authorizer

import (
	"context"
	"fmt"

	authorizationv1 "k8s.io/api/authorization/v1"
	"k8s.io/apiserver/pkg/authentication/user"
	"k8s.io/apiserver/pkg/authorization/authorizer"
	k8s "k8s.io/apiserver/pkg/authorization/authorizer"
)

type SubjectAccessReviewFunc func(context.Context, *authorizationv1.SubjectAccessReview) (*authorizationv1.SubjectAccessReview, error)

var _ k8s.Authorizer = (*Authorizer)(nil)

type Authorizer struct {
	review          SubjectAccessReviewFunc
	decisionOnError authorizer.Decision
}

func (m *Authorizer) Authorize(ctx context.Context, attr authorizer.Attributes) (decision authorizer.Decision, reason string, err error) {
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

	resp, err := m.review(ctx, req)
	if err != nil {
		return m.decisionOnError, "", err
	}

	if resp == nil {
		return m.decisionOnError, "", fmt.Errorf("review func did not return response")
	}

	switch {
	case resp.Status.Denied && resp.Status.Allowed:
		return authorizer.DecisionDeny, resp.Status.Reason, fmt.Errorf("subject access review returned both allow and deny response")
	case resp.Status.Denied:
		return authorizer.DecisionDeny, resp.Status.Reason, nil
	case resp.Status.Allowed:
		return authorizer.DecisionAllow, resp.Status.Reason, nil
	default:
		return authorizer.DecisionNoOpinion, resp.Status.Reason, nil
	}
}

func (m *Authorizer) RulesFor(_ user.Info, _ string) ([]authorizer.ResourceRuleInfo, []authorizer.NonResourceRuleInfo, bool, error) {
	var (
		resourceRules    []authorizer.ResourceRuleInfo
		nonResourceRules []authorizer.NonResourceRuleInfo
	)
	incomplete := true
	return resourceRules, nonResourceRules, incomplete, fmt.Errorf("authorizer does not support user rule resolution")
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
