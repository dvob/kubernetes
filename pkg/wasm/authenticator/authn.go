package authenticator

import (
	"context"
	"encoding/json"
	"errors"
	"os"

	authv1 "k8s.io/api/authentication/v1"
	authn "k8s.io/apiserver/pkg/authentication/authenticator"
	"k8s.io/apiserver/pkg/authentication/token/union"
	"k8s.io/apiserver/pkg/authentication/user"
	"k8s.io/kubernetes/pkg/wasm/internal/wasi"
)

var _ authn.Token = (*Module)(nil)

type Config struct {
	Modules []ModuleConfig `json:"modules"`
}

func NewAuthenticatorFromConfigFile(configFile string) (authn.Token, error) {
	data, err := os.ReadFile(configFile)
	if err != nil {
		return nil, err
	}
	config := &Config{}
	err = json.Unmarshal(data, config)
	if err != nil {
		return nil, err
	}
	return NewAuthenticator(config)
}

func NewAuthenticator(config *Config) (authn.Token, error) {
	authenticators := []authn.Token{}
	for _, moduleConfig := range config.Modules {
		m, err := NewModuleFromConfig(&moduleConfig)
		if err != nil {
			return nil, err
		}
		authenticators = append(authenticators, m)
	}
	return union.New(authenticators...), nil
}

type ModuleConfig struct {
	File      string `json:"file"`
	Settings  interface{}
	Audiences []string
}

type Module struct {
	runner       wasi.Runner
	implicitAuds authn.Audiences
	settings     interface{}
}

func NewModuleFromConfig(config *ModuleConfig) (*Module, error) {
	source, err := os.ReadFile(config.File)
	if err != nil {
		return nil, err
	}

	runner, err := wasi.NewWASIDefaultRunner(source, "authn", config.Settings)
	if err != nil {
		return nil, err
	}

	return &Module{
		runner:       runner,
		settings:     config.Settings,
		implicitAuds: config.Audiences,
	}, nil
}

func (a *Module) AuthenticateToken(ctx context.Context, token string) (*authn.Response, bool, error) {
	wantAuds, checkAuds := authn.AudiencesFrom(ctx)

	req := &authv1.TokenReview{
		Spec: authv1.TokenReviewSpec{
			Token:     token,
			Audiences: wantAuds,
		},
	}

	resp := &authv1.TokenReview{}
	err := a.runner.Run(ctx, req, resp)
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
