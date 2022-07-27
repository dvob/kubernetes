package wasi

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mitchellh/mapstructure"
)

type Request struct {
	Request  interface{} `json:"request"`
	Settings interface{} `json:"settings,omitempty"`
}

type Response struct {
	Response interface{} `json:"response,omitempty"`
	Error    *string     `json:"settings,omitempty"`
}

type Executor struct {
	run      func(context.Context, []byte) ([]byte, error)
	settings interface{}
}

func NewExecutorWithFn(fn func(context.Context, []byte) ([]byte, error)) *Executor {
	return &Executor{
		run: fn,
	}
}

func (e *Executor) Run(ctx context.Context, input interface{}, output interface{}) error {
	if input == nil {
		panic("missing input")
	}
	req := &Request{
		Request:  input,
		Settings: e.settings,
	}
	reqData, err := json.Marshal(req)
	if err != nil {
		return err
	}
	respData, err := e.run(ctx, reqData)
	if err != nil {
		return err
	}
	resp := &Response{}
	err = json.Unmarshal(respData, resp)
	if err != nil {
		return err
	}
	if resp.Error != nil && len(*resp.Error) > 0 {
		return fmt.Errorf("returned error: '%s'", *resp.Error)
	}
	return mapstructure.Decode(resp.Response, output)
}
