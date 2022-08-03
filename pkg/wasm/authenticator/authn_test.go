package authenticator

import (
	"bytes"
	"context"
	"reflect"
	"testing"
)

const (
	authnTestModuleFile = "../testmodules/target/wasm32-wasi/debug/test_authn.wasm"
	testToken           = "my-test-token"
	testUser            = "my-user"
	testUID             = "1337"
)

var (
	testGroups = []string{"system:masters"}
)

func TestConfig(t *testing.T) {
	config := `{
  "modules": [
    {
      "module": "../testmodules/target/wasm32-wasi/debug/test_authn.wasm",
      "debug": false,
      "settings": {
        "token": "magic-token",
        "user": "magic-user",
        "uid": "1",
        "groups": ["mygroup1", "mygroup2"]
      }
    }
  ]
}`
	testToken := "magic-token"
	testUser := "magic-user"
	testUID := "1"
	testGroups := []string{"mygroup1", "mygroup2"}

	authenticator, err := NewAuthenticatorFromReader(bytes.NewBufferString(config), nil)
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	resp, ok, err := authenticator.AuthenticateToken(ctx, testToken)

	if !ok {
		t.Fatalf("token '%s' should be authenticated", testToken)
	}

	if resp.User.GetName() != testUser {
		t.Errorf("wrong username: want=%s, got=%s", testUser, resp.User.GetName())
	}

	if resp.User.GetUID() != testUID {
		t.Errorf("wrong UID: want=%s, got=%s", testUID, resp.User.GetUID())
	}

	if !reflect.DeepEqual(resp.User.GetGroups(), testGroups) {
		t.Errorf("wrong groups: want=%s, got=%s", testGroups, resp.User.GetGroups())
	}
}

func newTestAuthenticator(t *testing.T) *Module {
	config := &ModuleConfig{
		Module: authnTestModuleFile,
	}
	authenticator, err := NewModuleFromConfig(config, nil)
	if err != nil {
		t.Fatal(err)
	}
	return authenticator
}

func TestAuthenticatorSuccess(t *testing.T) {
	authenticator := newTestAuthenticator(t)
	ctx := context.Background()

	resp, ok, err := authenticator.AuthenticateToken(ctx, testToken)
	if err != nil {
		t.Fatal(err)
	}

	if !ok {
		t.Fatalf("token '%s' should be authenticated", testToken)
	}

	if resp.User.GetName() != testUser {
		t.Errorf("wrong username: want=%s, got=%s", testUser, resp.User.GetName())
	}

	if resp.User.GetUID() != testUID {
		t.Errorf("wrong UID: want=%s, got=%s", testUID, resp.User.GetUID())
	}

	if !reflect.DeepEqual(resp.User.GetGroups(), testGroups) {
		t.Errorf("wrong groups: want=%s, got=%s", testGroups, resp.User.GetGroups())
	}
}
