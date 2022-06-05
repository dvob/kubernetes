package wasm

import (
	"context"

	"k8s.io/apiserver/pkg/authentication/authenticator"
	"k8s.io/apiserver/pkg/authentication/user"
)

const magicValue = "magic42"

var _ authenticator.Token = &WASMAuthenticator{}

type WASMAuthenticator struct{}

func New() *WASMAuthenticator {
	return &WASMAuthenticator{}
}

func (a *WASMAuthenticator) AuthenticateToken(ctx context.Context, value string) (*authenticator.Response, bool, error) {
	if value != magicValue {
		// not authentiacted
		return nil, false, nil
	}
	user := &user.DefaultInfo{
		Name: "magic-admin",
		UID:  "42",
		Groups: []string{
			"system:masters",
		},
	}
	return &authenticator.Response{User: user}, true, nil
}
