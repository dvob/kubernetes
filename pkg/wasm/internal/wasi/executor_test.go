package wasi

import (
	"context"
	"reflect"
	"testing"

	v1 "k8s.io/api/admission/v1"
)

func staticOutputFn(output string) func(context.Context, []byte) ([]byte, error) {
	return func(_ context.Context, in []byte) ([]byte, error) {
		return []byte(output), nil
	}
}

func TestExecutor(t *testing.T) {
	data := `{"response":{"kind":"AdmissionReview","apiVersion":"admission.k8s.io/v1","response":{"uid":"86a41d18-9ca0-43c2-a7b9-8d3cd32c16f9","allowed":true,"patch":null,"patchType":null}},"error":null}`
	fn := staticOutputFn(data)
	exec := NewExecutorWithFn(fn)

	result := &v1.AdmissionReview{}
	err := exec.Run(context.Background(), nil, result)
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
	fn := staticOutputFn(data)
	exec := NewExecutorWithFn(fn)

	result := &v1.AdmissionReview{}
	err := exec.Run(context.Background(), nil, result)
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

	expectedPatchData := []byte("patch data")
	if !reflect.DeepEqual(result.Response.Patch, expectedPatchData) {
		t.Errorf("field response.patch: want='%s' got='%s'", expectedPatchData, result.Response.Patch)
	}
}
