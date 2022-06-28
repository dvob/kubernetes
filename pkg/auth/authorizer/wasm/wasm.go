package wasm

import (
	"encoding/json"
	"fmt"

	admission "k8s.io/api/admission/v1beta1"
	authn "k8s.io/api/authentication/v1beta1"
	authz "k8s.io/api/authorization/v1beta1"
)

type Engine interface {
	Run([]byte) ([]byte, error)
	MemorySize() uint64
}

type Authenticator interface {
	Authenticate(*authn.TokenReview) (*authn.TokenReview, error)
}

type Authorizer interface {
	Authorize(*authz.SubjectAccessReview) (*authz.SubjectAccessReview, error)
}

type Admiter interface {
	Admit(*admission.AdmissionReview) (*admission.AdmissionReview, error)
}

func runJSON(engine Engine, input interface{}, result interface{}) error {
	rawInput, err := json.Marshal(input)
	if err != nil {
		return err
	}
	//fmt.Printf("input: '%s'", rawInput)
	rawResult, err := engine.Run(rawInput)
	if err != nil {
		return err
	}
	return json.Unmarshal(rawResult, result)
}

// AuthN
var _ Authenticator = &WasmAuthenticator{}

type WasmAuthenticator struct {
	engine Engine
}

func NewWasmAuthenticator(engine Engine) *WasmAuthenticator {
	return &WasmAuthenticator{
		engine: engine,
	}
}

func (wa *WasmAuthenticator) Authenticate(tr *authn.TokenReview) (*authn.TokenReview, error) {
	resp := &AuthResult{}
	req := AuthRequest{
		Request:  tr,
		Settings: Settings{},
	}
	err := runJSON(wa.engine, req, resp)
	if err != nil {
		return nil, err
	}

	if resp.Error != nil {
		return nil, fmt.Errorf("auth failed: %s", *resp.Error)
	}
	return resp.Response, err
}

type AuthResult struct {
	Response *authn.TokenReview `json:"response"`
	Error    *string            `json:"error"`
}

type Settings struct {
	Token string `json:"token"`
}
type AuthRequest struct {
	Request  *authn.TokenReview `json:"request"`
	Settings Settings           `json:"settings"`
}
