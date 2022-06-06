package wasm

import (
	"context"

	"k8s.io/apiserver/pkg/authentication/authenticator"
	"k8s.io/apiserver/pkg/authentication/user"
)

var _ authenticator.Token = &WASMAuthenticator{}

type WASMAuthenticator struct{}

func New() *WASMAuthenticator {
	return &WASMAuthenticator{}
}

func (a *WASMAuthenticator) AuthenticateToken(ctx context.Context, value string) (*authenticator.Response, bool, error) {
	if value == "admin42" {
		user := &user.DefaultInfo{
			Name: "magic-admin",
			UID:  "242",
			Groups: []string{
				"system:masters",
			},
		}
		return &authenticator.Response{User: user}, true, nil
	}
	if value == "user42" {
		user := &user.DefaultInfo{
			Name:   "magic-user",
			UID:    "142",
			Groups: []string{"user"},
		}
		return &authenticator.Response{User: user}, true, nil
	}
	// not authentiacted
	return nil, false, nil
}
