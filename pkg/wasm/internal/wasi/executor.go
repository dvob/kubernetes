package wasi

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
)

type Request struct {
	Request  interface{} `json:"request"`
	Settings interface{} `json:"settings,omitempty"`
}

type Response struct {
	Response json.RawMessage `json:"response,omitempty"`
	Error    *string         `json:"settings,omitempty"`
}

type Executor struct {
	run      func(context.Context, []byte) ([]byte, error)
	settings interface{}
	debug    io.Writer
}

func NewExecutorWithFn(fn func(context.Context, []byte) ([]byte, error)) *Executor {
	return &Executor{
		run:   fn,
		debug: nil,
	}
}

func (e *Executor) SetDebugOut(out io.Writer) {
	e.debug = out
}

func (e *Executor) Run(ctx context.Context, input interface{}, output interface{}) error {
	req := &Request{
		Request:  input,
		Settings: e.settings,
	}
	reqData, err := json.Marshal(req)
	if err != nil {
		return err
	}

	if e.debug != nil {
		fmt.Fprintf(e.debug, "request: '%s'", reqData)
	}

	respData, err := e.run(ctx, reqData)
	if err != nil {
		return err
	}

	if e.debug != nil {
		fmt.Fprintf(e.debug, "response: '%s'", respData)
	}

	resp := &Response{}
	err = json.Unmarshal(respData, resp)
	if err != nil {
		return err
	}
	if resp.Error != nil && len(*resp.Error) > 0 {
		return fmt.Errorf("returned error: '%s'", *resp.Error)
	}
	return json.Unmarshal(resp.Response, output)
}
