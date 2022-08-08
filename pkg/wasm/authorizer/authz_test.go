package authorizer

import (
	"bytes"
	"context"
	"testing"

	"k8s.io/apiserver/pkg/authentication/user"
	"k8s.io/apiserver/pkg/authorization/authorizer"
	"k8s.io/kubernetes/pkg/wasm"
)

const (
	authzTestModuleFile = "../testmodules/target/wasm32-wasi/debug/test_authz.wasm"
)

func TestConfigYAML(t *testing.T) {
	config := `
modules: 
- module: ../testmodules/target/wasm32-wasi/debug/test_authz.wasm
  debug: false
  settings:
    allow_all: true
`
	sut, _, err := NewAuthorizerFromReader(bytes.NewBufferString(config))
	if err != nil {
		t.Fatal(err)
	}
	attrs := &authorizer.AttributesRecord{
		ResourceRequest: true,
		User:            &user.DefaultInfo{Groups: []string{"foo-group"}},
		Name:            "foo",
	}

	ctx := context.Background()
	decision, _, err := sut.Authorize(ctx, attrs)
	if err != nil {
		t.Fatal(err)
	}
	if decision != authorizer.DecisionAllow {
		t.Fatal("requests should be allowed")
	}
}

func TestConfig(t *testing.T) {
	config := `{
  "modules": [
    {
      "module": "../testmodules/target/wasm32-wasi/debug/test_authz.wasm",
      "debug": false,
      "settings": {
        "allow_all": false,
	"magic_group": "foo-group",
	"magic_name": "foo"
      }
    }
  ]
}`
	sut, _, err := NewAuthorizerFromReader(bytes.NewBufferString(config))
	if err != nil {
		t.Fatal(err)
	}

	for _, test := range []struct {
		name     string
		attrs    authorizer.Attributes
		decision authorizer.Decision
	}{
		{
			name: "allow",
			attrs: &authorizer.AttributesRecord{
				ResourceRequest: true,
				User:            &user.DefaultInfo{Groups: []string{"foo-group"}},
				Name:            "foo",
			},
			decision: authorizer.DecisionAllow,
		},
		{
			name: "no_opinion_wrong_group",
			attrs: &authorizer.AttributesRecord{
				ResourceRequest: true,
				User: &user.DefaultInfo{
					Groups: []string{"wrong-group"},
				},
				Name: "foo",
			},
			decision: authorizer.DecisionNoOpinion,
		},
		{
			name: "no_opinion_wrong_name",
			attrs: &authorizer.AttributesRecord{
				ResourceRequest: true,
				User:            &user.DefaultInfo{Groups: []string{"foo-group"}},
				Name:            "wrong-name",
			},
			decision: authorizer.DecisionNoOpinion,
		},
		{
			name: "no_opinion_all_wrong",
			attrs: &authorizer.AttributesRecord{
				ResourceRequest: true,
				User:            &user.DefaultInfo{Groups: []string{"wrong-group"}},
				Name:            "wrong-name",
			},
			decision: authorizer.DecisionNoOpinion,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			ctx := context.Background()
			decision, reason, err := sut.Authorize(ctx, test.attrs)
			_ = reason
			if err != nil {
				t.Fatal(err)
			}

			if decision != test.decision {
				t.Errorf("wrong decision: want=%v, got=%v", test.decision, decision)
			}
		})
	}
}

func newTestAuthorizer(t *testing.T) *Module {
	config := &wasm.ModuleConfig{
		Module: authzTestModuleFile,
	}
	authorizer, err := NewModule(config)
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
