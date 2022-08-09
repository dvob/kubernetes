package wasm

import (
	"context"
	"reflect"
	"testing"

	v1 "k8s.io/api/admission/v1"
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

func TestEnvelopeWithNestedFields(t *testing.T) {
	data := `{"response":{"kind":"AdmissionReview","apiVersion":"admission.k8s.io/v1","response":{"uid":"86a41d18-9ca0-43c2-a7b9-8d3cd32c16f9","allowed":true,"patch":null,"patchType":null}},"error":null}`
	rawRunner := newDummyRawRunner(data)
	runner := NewEnvelopeRunner(rawRunner, nil)

	result := &v1.AdmissionReview{}
	err := runner.Run(context.Background(), nil, result)
	if err != nil {
		t.Fatal(err)
	}

	expectedKind := "AdmissionReview"
	if result.Kind != expectedKind {
		t.Errorf("field kind: want='%s', got='%s'", expectedKind, result.Kind)
	}
	expectedAPIVersion := "admission.k8s.io/v1"
	if result.APIVersion != expectedAPIVersion {
		t.Errorf("field apiVersion: want='%s', got='%s'", expectedAPIVersion, result.APIVersion)
	}

	if result.Response == nil {
		t.Fatal("response is nil")
	}

	expectedAllowed := true
	if result.Response.Allowed != expectedAllowed {
		t.Errorf("field response.allowed: want='%v', got='%v'", expectedAllowed, result.Response.Allowed)
	}
}

func TestUnmarshalPatch(t *testing.T) {
	data := `{
		"response":{
			"kind":"AdmissionReview",
			"apiVersion":"admission.k8s.io/v1",
			"response":{
				"uid":"86a41d18-9ca0-43c2-a7b9-8d3cd32c16f9",
				"allowed":true,
				"patch":"cGF0Y2ggZGF0YQ==",
				"patchType":"ownType"
			}
		}
	}`
	rawRunner := newDummyRawRunner(data)
	runner := NewEnvelopeRunner(rawRunner, nil)

	result := &v1.AdmissionReview{}
	err := runner.Run(context.Background(), nil, result)
	if err != nil {
		t.Fatal(err)
	}

	expectedPatchData := []byte("patch data")
	if !reflect.DeepEqual(result.Response.Patch, expectedPatchData) {
		t.Errorf("field response.patch: want='%s' got='%s'", expectedPatchData, result.Response.Patch)
	}
}
