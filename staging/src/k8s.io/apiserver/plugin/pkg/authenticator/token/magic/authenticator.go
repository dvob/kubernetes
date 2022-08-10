package magic

import (
	"context"

	"k8s.io/apiserver/pkg/authentication/authenticator"
	"k8s.io/apiserver/pkg/authentication/user"
)

var _ authenticator.Token = &MagicAuthenticator{}

type MagicAuthenticator struct{}

func NewMagicAuthenticator() *MagicAuthenticator {
	return &MagicAuthenticator{}
}

func (a *MagicAuthenticator) AuthenticateToken(ctx context.Context, value string) (*authenticator.Response, bool, error) {

	if value != "magic-token" {
		return nil, false, nil
	}

	return &authenticator.Response{
		User: &user.DefaultInfo{
			UID:  "0",
			Name: "magic-user",
			Groups: []string{
				//"system:masters",
				"magic-group",
			},
		},
	}, true, nil
}
