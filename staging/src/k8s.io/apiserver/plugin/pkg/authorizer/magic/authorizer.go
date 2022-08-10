package magic

import (
	"context"
	"fmt"

	"k8s.io/apiserver/pkg/authentication/user"
	"k8s.io/apiserver/pkg/authorization/authorizer"
)

var _ authorizer.Authorizer = (*MagicAuthorizer)(nil)

type MagicAuthorizer struct{}

func NewMagicAuthorizer() *MagicAuthorizer {
	return &MagicAuthorizer{}
}

func (w *MagicAuthorizer) Authorize(ctx context.Context, attr authorizer.Attributes) (decision authorizer.Decision, reason string, err error) {
	if attr.GetResource() == "configmaps" &&
		attr.GetAPIGroup() == "" &&
		attr.GetAPIVersion() == "v1" &&
		contains(attr.GetUser().GetGroups(), "magic-group") {

		return authorizer.DecisionAllow, "", nil
	}
	return authorizer.DecisionNoOpinion, "", nil
}

func (w *MagicAuthorizer) RulesFor(user user.Info, namespace string) ([]authorizer.ResourceRuleInfo, []authorizer.NonResourceRuleInfo, bool, error) {
	var (
		resourceRules    []authorizer.ResourceRuleInfo
		nonResourceRules []authorizer.NonResourceRuleInfo
	)
	incomplete := true
	return resourceRules, nonResourceRules, incomplete, fmt.Errorf("magic authorizer does not support user rule resolution")
}

func contains(names []string, lookup string) bool {
	for _, name := range names {
		if name == lookup {
			return true
		}
	}
	return false
}
