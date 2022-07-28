package wasi

import (
	"context"
	"testing"
)

func newDummyRawRunner(out string) RawRunnerFunc {
	return func(_ context.Context, _ []byte) ([]byte, error) {
		return []byte(out), nil
	}
}

func debugRawRunner(rawRunner RawRunner, t *testing.T) RawRunner {
	return RawRunnerFunc(func(ctx context.Context, in []byte) ([]byte, error) {
		t.Logf("in: '%s'\n", in)
		out, err := rawRunner.Run(ctx, in)
		if err != nil {
			t.Logf("err: '%s'", err)
			return nil, err
		}
		t.Logf("out: '%s'\n", out)
		return out, nil
	})
}

func TestEnvelopeRunner(t *testing.T) {
	data := `{"response":34}`
	er := NewEnvelopeRunner(debugRawRunner(newDummyRawRunner(data), t), map[string]string{"val1": "22"})
	ctx := context.Background()

	var i int
	err := er.Run(ctx, 1, &i)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(i)
}
