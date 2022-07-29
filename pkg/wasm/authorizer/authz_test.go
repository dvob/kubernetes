package authorizer

import (
	"context"
	"testing"

	"k8s.io/apiserver/pkg/authentication/user"
	"k8s.io/apiserver/pkg/authorization/authorizer"
)

const (
	authzTestModuleFile = "../testmodules/target/wasm32-wasi/debug/test_authz.wasm"
)

func newTestAuthorizer(t *testing.T) *Authorizer {
	config := &AuthorizationModuleConfig{
		File: authzTestModuleFile,
	}
	authorizer, err := NewAuthorizerWithConfig(config)
	if err != nil {
		t.Fatal(err)
	}
	return authorizer
}

func TestAuthorizerSuccess(t *testing.T) {
	authroizer := newTestAuthorizer(t)
	ctx := context.Background()

	attr := &authorizer.AttributesRecord{User: &user.DefaultInfo{}}
	decision, reason, err := authroizer.Authorize(ctx, attr)
	if err != nil {
		t.Fatal(err)
	}

	if decision != authorizer.DecisionAllow {
		t.Errorf("decision is not allow but '%v'", decision)
	}

	if len(reason) != 0 {
		t.Errorf("reason is not empty")
	}
}
