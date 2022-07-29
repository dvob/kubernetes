package wasi

import (
	"context"
	"time"

	"github.com/wapc/wapc-go"
	"github.com/wapc/wapc-go/engines/wazero"
)

type WAPCRuntime struct {
	pool *wapc.Pool
}

func NewWAPCRuntime(moduleSource []byte) (*WAPCRuntime, error) {
	ctx := context.Background()

	engine := wazero.Engine()

	module, err := engine.New(ctx, moduleSource, nil)
	if err != nil {
		return nil, err
	}

	pool, err := wapc.NewPool(ctx, module, 1)
	if err != nil {
		return nil, err
	}

	return &WAPCRuntime{
		pool: pool,
	}, nil
}

func (r *WAPCRuntime) RawRunner(fnName string) RawRunner {
	return RawRunnerFunc(func(ctx context.Context, in []byte) ([]byte, error) {
		return r.Run(ctx, fnName, in)
	})
}

func (r *WAPCRuntime) Close(ctx context.Context) {
	r.pool.Close(ctx)
}

func (r *WAPCRuntime) Run(ctx context.Context, fnName string, input []byte) ([]byte, error) {
	i, err := r.pool.Get(time.Second * 30)
	if err != nil {
		return nil, err
	}
	defer r.pool.Return(i)
	return i.Invoke(ctx, fnName, input)
}
