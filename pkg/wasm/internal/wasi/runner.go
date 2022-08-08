package wasi

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
)

func DebugRawRunner(rawRunner RawRunner) RawRunner {
	return RawRunnerFunc(func(ctx context.Context, in []byte) ([]byte, error) {
		log.Printf("in: '%s'\n", in)
		out, err := rawRunner.Run(ctx, in)
		if err != nil {
			log.Printf("err: '%s'", err)
			return nil, err
		}
		log.Printf("out: '%s'\n", out)
		return out, nil
	})
}

type RawRunner interface {
	Run(ctx context.Context, in []byte) ([]byte, error)
}

type RawRunnerFunc func(context.Context, []byte) ([]byte, error)

func (fn RawRunnerFunc) Run(ctx context.Context, in []byte) ([]byte, error) {
	return fn(ctx, in)
}

type Runner interface {
	Run(ctx context.Context, in interface{}, out interface{}) error
}

type RunnerFunc func(context.Context, interface{}, interface{}) error

func (fn RunnerFunc) Run(ctx context.Context, in interface{}, out interface{}) error {
	return fn(ctx, in, out)
}

type JSONRunner struct {
	rawRunner RawRunner
}

func NewJSONRunner(rawRunner RawRunner) *JSONRunner {
	return &JSONRunner{
		rawRunner: rawRunner,
	}
}

func (jr *JSONRunner) Run(ctx context.Context, in interface{}, out interface{}) error {
	req, err := json.Marshal(in)
	if err != nil {
		return err
	}

	resp, err := jr.rawRunner.Run(ctx, req)
	if err != nil {
		return err
	}
	return json.Unmarshal(resp, out)
}

type Request struct {
	Request  interface{} `json:"request"`
	Settings interface{} `json:"settings,omitempty"`
}

type Response struct {
	Response json.RawMessage `json:"response,omitempty"`
	Error    *string         `json:"error,omitempty"`
}

type EnvelopeRunner struct {
	runner   Runner
	settings interface{}
}

func NewEnvelopeRunner(rawRunner RawRunner, settings interface{}) *EnvelopeRunner {
	return &EnvelopeRunner{
		runner:   NewJSONRunner(rawRunner),
		settings: settings,
	}
}

func (er *EnvelopeRunner) Run(ctx context.Context, input interface{}, output interface{}) error {
	req := &Request{
		Request:  input,
		Settings: er.settings,
	}

	resp := &Response{}
	err := er.runner.Run(ctx, req, resp)
	if err != nil {
		return err
	}
	if resp.Error != nil && len(*resp.Error) > 0 {
		return fmt.Errorf("envelope error: '%s'", *resp.Error)
	}
	return json.Unmarshal(resp.Response, output)
}
