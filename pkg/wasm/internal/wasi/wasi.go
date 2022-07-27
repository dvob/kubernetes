package wasi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/mitchellh/mapstructure"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	wasi "github.com/tetratelabs/wazero/wasi_snapshot_preview1"
)

type Request struct {
	Request  interface{} `json:"request"`
	Settings interface{} `json:"settings,omitempty"`
}

type Response struct {
	Response interface{} `json:"response,omitempty"`
	Error    *string     `json:"settings,omitempty"`
}

type Wrapper struct {
	run func(context.Context, []byte) ([]byte, error)
}

func NewWrapper(fn func(context.Context, []byte) ([]byte, error)) *Wrapper {
	return &Wrapper{
		run: fn,
	}
}

func (w *Wrapper) Run(ctx context.Context, input interface{}, settings interface{}, output interface{}) error {
	if input == nil {
		panic("missing input")
	}
	req := &Request{
		Request:  input,
		Settings: settings,
	}
	reqData, err := json.Marshal(req)
	if err != nil {
		return err
	}
	respData, err := w.run(ctx, reqData)
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

type Executor struct {
	mu       *sync.Mutex
	runtime  wazero.Runtime
	code     wazero.CompiledModule
	instance api.Module
	stdin    bytes.Buffer
	stdout   bytes.Buffer
	stderr   bytes.Buffer
}

func NewExecutor(moduleSource []byte) (*Executor, error) {
	ctx := context.Background()

	runtime := wazero.NewRuntime()

	// Instantiate WASI, which implements system I/O such as console output.
	if _, err := wasi.Instantiate(ctx, runtime); err != nil {
		return nil, err
	}

	// Compile the WebAssembly module using the default configuration.
	code, err := runtime.CompileModule(ctx, moduleSource, wazero.NewCompileConfig())
	if err != nil {
		return nil, err
	}

	return &Executor{
		mu:      &sync.Mutex{},
		runtime: runtime,
		code:    code,
	}, nil
}

func (e *Executor) HasFunction(fnName string) bool {
	exportedFunctions := e.code.ExportedFunctions()
	_, ok := exportedFunctions[fnName]
	return ok
}

func (e *Executor) Close(ctx context.Context) error {
	return e.runtime.Close(ctx)
}

func (e *Executor) Run(ctx context.Context, fnName string, input []byte) ([]byte, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.stdin.Reset()
	e.stdout.Reset()
	e.stderr.Reset()

	e.stdin.Write(input)

	config := wazero.NewModuleConfig().WithStdin(&e.stdin).WithStdout(&e.stdout).WithStderr(&e.stderr).WithStartFunctions()

	instance, err := e.runtime.InstantiateModule(ctx, e.code, config)
	if err != nil {
		return nil, fmt.Errorf("failed with stderr '%s': %w)", e.stderr.String(), err)
	}
	defer instance.Close(ctx)

	fn := instance.ExportedFunction(fnName)
	if fn == nil {
		return nil, fmt.Errorf("function '%s' missing", fnName)
	}

	_, err = fn.Call(ctx)
	if err != nil {
		errOut := e.stderr.String()
		if errOut != "" {
			return nil, fmt.Errorf("call to %s failed. stderr: '%s', err: %w", fnName, errOut, err)
		}
		return nil, err
	}

	output := make([]byte, e.stdout.Len())
	copy(output, e.stdout.Bytes())

	return output, nil
}
