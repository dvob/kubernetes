package authenticator

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"

	authv1 "k8s.io/api/authentication/v1"
	authn "k8s.io/apiserver/pkg/authentication/authenticator"
	"k8s.io/apiserver/pkg/authentication/token/union"
	"k8s.io/apiserver/pkg/authentication/user"
	"k8s.io/klog/v2"
	"k8s.io/kubernetes/pkg/wasm/internal/wasi"
)

var _ authn.Token = (*Module)(nil)

type Config struct {
	Modules []ModuleConfig `json:"modules"`
}

func NewAuthenticatorFromConfigFile(configFile string, auds authn.Audiences) (authn.Token, error) {
	file, err := os.Open(configFile)
	if err != nil {
		return nil, fmt.Errorf("failed to open WASM authenticator configuratio: %w", err)
	}
	return NewAuthenticatorFromReader(file, auds)
}

func NewAuthenticatorFromReader(configInput io.Reader, auds authn.Audiences) (authn.Token, error) {
	data, err := io.ReadAll(configInput)
	if err != nil {
		return nil, fmt.Errorf("failed to read WASM authenticator configuration: %w", err)
	}
	config := &Config{}
	err = json.Unmarshal(data, config)
	if err != nil {
		return nil, fmt.Errorf("failed to read WASM authenticator configuration: %w", err)
	}
	return NewAuthenticator(config, auds)
}

func NewAuthenticator(config *Config, auds authn.Audiences) (authn.Token, error) {
	authenticators := []authn.Token{}
	for i, moduleConfig := range config.Modules {
		m, err := NewModuleFromConfig(&moduleConfig, auds)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize WASM authenticator module %d: %w", i, err)
		}
		authenticators = append(authenticators, m)
	}
	return union.New(authenticators...), nil
}

type ModuleConfig struct {
	Name     string      `json:"name,omitempty"`
	Module   string      `json:"module"`
	Settings interface{} `json:"settings,omitempty"`
	Debug    bool        `json:"debug,omitempty"`
}

type Module struct {
	name         string
	runner       wasi.Runner
	implicitAuds authn.Audiences
	settings     interface{}
}

func NewModuleFromConfig(config *ModuleConfig, auds authn.Audiences) (*Module, error) {
	source, err := os.ReadFile(config.Module)
	if err != nil {
		return nil, err
	}

	runtime, err := wasi.NewRuntime(source)
	if err != nil {
		return nil, err
	}

	rawRunner := runtime.RawRunner("authn")
	if config.Debug {
		rawRunner = wasi.DebugRawRunner(rawRunner)
	}

	runner := wasi.NewEnvelopeRunner(rawRunner, config.Settings)
	return &Module{
		name:         config.Name,
		runner:       runner,
		settings:     config.Settings,
		implicitAuds: auds,
	}, nil
}

func (m *Module) AuthenticateToken(ctx context.Context, token string) (*authn.Response, bool, error) {
	wantAuds, checkAuds := authn.AudiencesFrom(ctx)

	req := &authv1.TokenReview{
		Spec: authv1.TokenReviewSpec{
			Token:     token,
			Audiences: wantAuds,
		},
	}

	resp := &authv1.TokenReview{}
	err := m.runner.Run(ctx, req, resp)
	if err != nil {
		klog.ErrorS(err, "failed to run wasm authentication module", "module_name", m.name)
		return nil, false, err
	}

	tr := resp

	var auds authn.Audiences
	if checkAuds {
		gotAuds := m.implicitAuds
		if len(tr.Status.Audiences) > 0 {
			gotAuds = tr.Status.Audiences
		}
		auds = wantAuds.Intersect(gotAuds)
		if len(auds) == 0 {
			klog.V(4).InfoS("no matching audiences", "want_auds", wantAuds, "got_auds", gotAuds)
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
