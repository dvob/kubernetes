package authenticator

import (
	"context"
	"errors"
	"os"

	authv1 "k8s.io/api/authentication/v1"
	authn "k8s.io/apiserver/pkg/authentication/authenticator"
	"k8s.io/apiserver/pkg/authentication/user"
	"k8s.io/kubernetes/pkg/wasm/internal/wasi"
)

var _ authn.Token = (*Authenticator)(nil)

type AuthenticationConfig struct {
	Modules []AuthenticationModuleConfig `json:"modules"`
}

type AuthenticationModuleConfig struct {
	File      string `json:"file"`
	Settings  interface{}
	Audiences []string
}

type Authenticator struct {
	//exec         *wasi.Executor
	wrapper      *wasi.Wrapper
	implicitAuds authn.Audiences
	settings     interface{}
}

func NewAuthenticatorWithConfig(config *AuthenticationModuleConfig) (*Authenticator, error) {
	source, err := os.ReadFile(config.File)
	if err != nil {
		return nil, err
	}

	wasiExecutor, err := wasi.NewExecutor(source)
	if err != nil {
		return nil, err
	}

	runFn := func(ctx context.Context, in []byte) ([]byte, error) {
		return wasiExecutor.Run(ctx, "authn", in)
	}

	return &Authenticator{
		//exec:         wasiExecutor,
		wrapper:      wasi.NewWrapper(runFn),
		settings:     config.Settings,
		implicitAuds: config.Audiences,
	}, nil
}

type WASIAuthenticationRequest struct {
	Request  *authv1.TokenReview `json:"request,omitempty"`
	Settings interface{}         `json:"settings,omitempty"`
}

type WASIAuthenticationResponse struct {
	Response *authv1.TokenReview `json:"response,omitempty"`
	Error    *string             `json:"error,omitempty"`
}

func (a *Authenticator) AuthenticateToken(ctx context.Context, token string) (*authn.Response, bool, error) {
	wantAuds, checkAuds := authn.AudiencesFrom(ctx)

	req := WASIAuthenticationRequest{
		Request: &authv1.TokenReview{
			Spec: authv1.TokenReviewSpec{
				Token:     token,
				Audiences: wantAuds,
			},
		},
		Settings: a.settings,
	}

	// reqPayload, err := json.Marshal(req)
	// if err != nil {
	// 	return nil, false, err
	// }

	resp := &authv1.TokenReview{}
	err := a.wrapper.Run(ctx, req.Request, a.settings, resp)
	if err != nil {
		return nil, false, err
	}

	tr := resp

	var auds authn.Audiences
	if checkAuds {
		gotAuds := a.implicitAuds
		if len(tr.Status.Audiences) > 0 {
			gotAuds = tr.Status.Audiences
		}
		auds = wantAuds.Intersect(gotAuds)
		if len(auds) == 0 {
			return nil, false, nil
		}
	}

	if !tr.Status.Authenticated {
		if tr.Status.Error != "" {
			return nil, false, errors.New(tr.Status.Error)
		}
		return nil, false, nil
	}

	u := &user.DefaultInfo{
		Name:   tr.Status.User.Username,
		UID:    tr.Status.User.UID,
		Groups: tr.Status.User.Groups,
	}
	for key, value := range tr.Status.User.Extra {
		u.Extra[key] = value
	}

	return &authn.Response{
		Audiences: auds,
		User:      u,
	}, true, nil
}
