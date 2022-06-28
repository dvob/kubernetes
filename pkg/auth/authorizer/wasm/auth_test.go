package wasm

import (
	"context"
	"testing"
)

func TestAuth(t *testing.T) {

	wasmAuthenticator, err := New("testdata/auth.wasm")
	if err != nil {
		t.Fatal(err)
	}

	resp, ok, err := wasmAuthenticator.AuthenticateToken(context.TODO(), "mySecret")
	if !ok {
		t.Fatal("auth not ok", err)
	}

	if err != nil {
		t.Fatal(err)
	}

	wantUser := "wasm-admin"
	if resp.User.GetName() != wantUser {
		t.Errorf("username want='%s', got='%s'", wantUser, resp.User.GetName())
	}

	if len(resp.User.GetGroups()) != 1 {
		t.Fatalf("numbers of groups: want=1, got=%d", len(resp.User.GetGroups()))
	}

	wantGroup := "system:masters"
	if resp.User.GetGroups()[0] != wantGroup {
		t.Fatalf("groups want='%s', got='%s'", wantGroup, resp.User.GetGroups()[0])
	}
}
