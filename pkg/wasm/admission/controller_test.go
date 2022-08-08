package admission

import (
	"bytes"
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apiserver/pkg/admission"
	"k8s.io/apiserver/pkg/authentication/user"
)

func TestConfig(t *testing.T) {
	config := `{
  "modules": [
    {
      "module": "../testmodules/target/wasm32-wasi/debug/test_admission_mut.wasm",
      "type": "wasi",
      "debug": false,
      "mutating": true,
      "rules": [
        {
          "operations": ["CREATE"],
          "apiGroups": [""],
          "apiVersions": ["v1"],
          "resources": ["pods"]
	}
      ]
    }
  ]
}`

	admissionController, err := NewControllerFromReader(bytes.NewBufferString(config))
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("mutator=%d, validator=%d", len(admissionController.mutator), len(admissionController.validator))
	ctx := context.Background()

	s := runtime.NewScheme() // admission.NewObjectInterfacesFromScheme(runtime.NewScheme())
	corev1.AddToScheme(s)
	objInterface := admission.NewObjectInterfacesFromScheme(s)
	ns := "default"
	podName := "foo"
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      podName,
			Namespace: ns,
		},
		Spec: corev1.PodSpec{},
	}
	attr := admission.NewAttributesRecord(pod, nil, schema.GroupVersionKind{"", "v1", "Pod"}, ns, podName, schema.GroupVersionResource{"", "v1", "pods"}, "", admission.Create, &metav1.CreateOptions{}, false, &user.DefaultInfo{})

	err = admissionController.Validate(ctx, attr, objInterface)
	if err != nil {
		t.Fatal("")
	}

	err = admissionController.Admit(ctx, attr, objInterface)
	if err != nil {
		t.Fatal(err)
	}

	expectedAnnotationKey := "puzzle.ch/test-annotation"
	expectedAnnotationValue := "foo"

	val, ok := pod.GetAnnotations()[expectedAnnotationKey]
	if !ok {
		t.Fatalf("annotation '%s' missing on pod", expectedAnnotationKey)
	}
	if val != expectedAnnotationValue {
		t.Fatalf("annotation '%s' has wrong value: want=%s, got=%s", expectedAnnotationKey, expectedAnnotationValue, val)
	}
}

func TestMultipleModules(t *testing.T) {
	config := `
modules:
- module: ../testmodules/target/wasm32-wasi/debug/test_admission_mut_helper.wasm
  name: add-annotation-a
  type: wasi
  debug: true
  mutating: true
  settings:
    annotations:
      annotation-a: "true"
  rules:
  - operations: ["CREATE"]
    apiGroups: [""]
    apiVersions: ["v1"]
    resources: ["pods"]
- module: ../testmodules/target/wasm32-wasi/debug/test_admission_mut_helper.wasm
  name: add-annotation-b
  type: wasi
  debug: true
  mutating: true
  settings:
    annotations:
      annotation-b: "true"
  rules:
  - operations: ["CREATE"]
    apiGroups: [""]
    apiVersions: ["v1"]
    resources: ["pods"]
`

	admissionController, err := NewControllerFromReader(bytes.NewBufferString(config))
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("mutator=%d, validator=%d", len(admissionController.mutator), len(admissionController.validator))
	ctx := context.Background()

	s := runtime.NewScheme()
	corev1.AddToScheme(s)
	objInterface := admission.NewObjectInterfacesFromScheme(s)
	ns := "default"
	podName := "foo"
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      podName,
			Namespace: ns,
		},
		Spec: corev1.PodSpec{},
	}
	attr := admission.NewAttributesRecord(pod, nil, schema.GroupVersionKind{"", "v1", "Pod"}, ns, podName, schema.GroupVersionResource{"", "v1", "pods"}, "", admission.Create, &metav1.CreateOptions{}, false, &user.DefaultInfo{})

	err = admissionController.Admit(ctx, attr, objInterface)
	if err != nil {
		t.Fatal(err)
	}

	t.Log(pod.GetAnnotations())

	expectedAnnotations := []string{"annotation-a", "annotation-b"}

	for _, key := range expectedAnnotations {
		if _, ok := pod.GetAnnotations()[key]; !ok {
			t.Fatalf("annotation '%s' not found on pod", key)
		}
	}
}
