package wasi

import (
	"context"
	"encoding/json"
	"fmt"
)

type Request struct {
	Request  interface{} `json:"request"`
	Settings interface{} `json:"settings,omitempty"`
}

type RawResponse []byte

func (r *RawResponse) UnmarshalJSON(data []byte) error {
	*r = data
	return nil
}

func (r *RawResponse) MarshalJSON() ([]byte, error) {
	return []byte(*r), nil
}

type Response struct {
	Response RawResponse `json:"response,omitempty"`
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
	return json.Unmarshal(resp.Response, output)
}
