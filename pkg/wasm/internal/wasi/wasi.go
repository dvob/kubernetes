package wasi

import (
	"bytes"
	"context"
	"fmt"
	"sync"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	wasi "github.com/tetratelabs/wazero/wasi_snapshot_preview1"
)

type Runtime struct {
	mu       *sync.Mutex
	runtime  wazero.Runtime
	code     wazero.CompiledModule
	instance api.Module
	stdin    bytes.Buffer
	stdout   bytes.Buffer
	stderr   bytes.Buffer
}

func NewWASIDefaultRunner(moduleSource []byte, fnName string, settings interface{}) (*EnvelopeRunner, error) {
	r, err := NewRuntime(moduleSource)
	if err != nil {
		return nil, err
	}

	return NewEnvelopeRunner(r.RawRunner(fnName), settings), nil
}

func NewRuntime(moduleSource []byte) (*Runtime, error) {
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

	return &Runtime{
		mu:      &sync.Mutex{},
		runtime: runtime,
		code:    code,
	}, nil
}

func (r *Runtime) RawRunner(fnName string) RawRunner {
	return RawRunnerFunc(func(ctx context.Context, in []byte) ([]byte, error) {
		return r.Run(ctx, fnName, in)
	})
}

func (r *Runtime) HasFunction(fnName string) bool {
	exportedFunctions := r.code.ExportedFunctions()
	_, ok := exportedFunctions[fnName]
	return ok
}

func (r *Runtime) Close(ctx context.Context) error {
	return r.runtime.Close(ctx)
}

func (r *Runtime) Run(ctx context.Context, fnName string, input []byte) ([]byte, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.stdin.Reset()
	r.stdout.Reset()
	r.stderr.Reset()

	r.stdin.Write(input)

	config := wazero.NewModuleConfig().WithStdin(&r.stdin).WithStdout(&r.stdout).WithStderr(&r.stderr).WithStartFunctions()

	instance, err := r.runtime.InstantiateModule(ctx, r.code, config)
	if err != nil {
		return nil, fmt.Errorf("failed with stderr '%s': %w)", r.stderr.String(), err)
	}
	defer instance.Close(ctx)

	fn := instance.ExportedFunction(fnName)
	if fn == nil {
		return nil, fmt.Errorf("function '%s' missing", fnName)
	}

	_, err = fn.Call(ctx)
	if err != nil {
		errOut := r.stderr.String()
		if errOut != "" {
			return nil, fmt.Errorf("call to %s failed. stderr: '%s', err: %w", fnName, errOut, err)
		}
		return nil, err
	}

	output := make([]byte, r.stdout.Len())
	copy(output, r.stdout.Bytes())

	return output, nil
}
