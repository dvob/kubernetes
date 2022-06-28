package wasm

import (
	"context"
	"fmt"
	"os"

	authn "k8s.io/api/authentication/v1beta1"
	"k8s.io/apiserver/pkg/authentication/authenticator"
	"k8s.io/apiserver/pkg/authentication/user"
)

var _ authenticator.Token = &WASMAuthenticator{}

type WASMAuthenticator struct {
	Authenticator Authenticator
	Token         string
}

func New(moduleFile string) (*WASMAuthenticator, error) {
	moduleBytes, err := os.ReadFile(moduleFile)
	if err != nil {
		return nil, err
	}
	engine, err := NewWasiEngine(moduleBytes)
	if err != nil {
		return nil, err
	}

	authenticator := NewWasmAuthenticator(engine)
	return &WASMAuthenticator{
		Authenticator: authenticator,
	}, nil
}

func (a *WASMAuthenticator) AuthenticateToken(ctx context.Context, value string) (*authenticator.Response, bool, error) {
	tr := &authn.TokenReview{
		Spec: authn.TokenReviewSpec{
			Token: value,
		},
	}

	resp, err := a.Authenticator.Authenticate(tr)
	if err != nil {
		return nil, false, err
	}

	if !resp.Status.Authenticated {
		return nil, false, fmt.Errorf("not authenticated: %s", resp.Status.Error)
	}

	user := &user.DefaultInfo{
		Name:   resp.Status.User.Username,
		UID:    resp.Status.User.UID,
		Groups: resp.Status.User.Groups,
	}

	return &authenticator.Response{
		User:      user,
		Audiences: resp.Status.Audiences,
	}, true, nil
}
