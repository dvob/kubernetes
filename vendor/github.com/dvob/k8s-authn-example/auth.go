package auth

import (
	"context"

	"k8s.io/apiserver/pkg/authentication/authenticator"
	"k8s.io/apiserver/pkg/authentication/user"
)

var _ authenticator.Token = (*Authenticator)(nil)

type Authenticator struct{}

func NewAuthenticator() (*Authenticator, error) {
	return &Authenticator{}, nil
}

func (a *Authenticator) AuthenticateToken(ctx context.Context, token string) (*authenticator.Response, bool, error) {

	if token != "magicString" {
		return nil, false, nil
	}

	return &authenticator.Response{
		User: &user.DefaultInfo{
			Name: "test-user",
			UID:  "42",
			Groups: []string{
				"system:masters",
			},
		},
	}, true, nil
}
