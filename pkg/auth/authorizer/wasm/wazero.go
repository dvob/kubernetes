package wasm

import (
	"bytes"
	"context"
	"fmt"
	"strconv"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/wasi"
)

type WasiEngine struct {
	runtime  wazero.Runtime
	code     wazero.CompiledModule
	instance api.Module
	id       int
}

func NewWasiEngine(moduleSource []byte) (*WasiEngine, error) {

	ctx := context.Background()

	runtime := wazero.NewRuntime()
	//TODO: defer r.Close(ctx)

	// Instantiate WASI, which implements system I/O such as console output.
	if _, err := wasi.InstantiateSnapshotPreview1(ctx, runtime); err != nil {
		return nil, err
	}

	// Compile the WebAssembly module using the default configuration.
	code, err := runtime.CompileModule(ctx, moduleSource, wazero.NewCompileConfig())
	if err != nil {
		return nil, err
	}

	return &WasiEngine{
		runtime: runtime,
		code:    code,
	}, nil
}

func (e *WasiEngine) MemorySize() uint64 {
	return 0
}

func (e *WasiEngine) Run(input []byte) ([]byte, error) {
	ctx := context.Background()

	stdin := bytes.NewBuffer(input)
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	config := wazero.NewModuleConfig().WithStdin(stdin).WithStdout(stdout).WithStderr(stderr).WithStartFunctions()

	instance, err := e.runtime.InstantiateModule(ctx, e.code, config.WithName(strconv.Itoa(e.id)))
	if err != nil {
		return nil, fmt.Errorf("failed with stderr '%s': %w)", stderr.String(), err)
	}
	defer instance.Close(ctx)

	fn := instance.ExportedFunction("auth")
	if fn == nil {
		return nil, fmt.Errorf("function auth missing")
	}

	_, err = fn.Call(ctx)
	errOut := stderr.String()
	if err != nil {
		if errOut != "" {
			return nil, fmt.Errorf("call to auth failed. stderr: '%s', err: %w", errOut, err)
		}
		return nil, err
	}

	return stdout.Bytes(), nil
}
