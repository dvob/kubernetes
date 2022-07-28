package admission

import (
	"context"
	"strings"
	"testing"

	v1 "k8s.io/api/admissionregistration/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apiserver/pkg/admission"
	"k8s.io/apiserver/pkg/authentication/user"
)

const (
	admissionTestModuleFile    = "../testdata/target/wasm32-wasi/debug/test_admission.wasm"
	admissionMutTestModuleFile = "../testdata/target/wasm32-wasi/debug/test_admission_mut.wasm"
)

func newTestAdmissionController(t *testing.T) *AdmissionController {
	config := &AdmissionModuleConfig{
		File:     admissionTestModuleFile,
		Mutating: false,
		Rules: []v1.RuleWithOperations{
			{
				Operations: []v1.OperationType{"CREATE"},
				Rule: v1.Rule{
					APIGroups:   []string{"*"},
					APIVersions: []string{"*"},
					Resources:   []string{"*"},
				},
			},
		},
	}
	admissionController, err := NewAdmissionControllerWithConfig(config)
	if err != nil {
		t.Fatal(err)
	}
	return admissionController
}

func TestAdmissionReject(t *testing.T) {
	admissionController := newTestAdmissionController(t)
	ctx := context.Background()

	s := runtime.NewScheme() // admission.NewObjectInterfacesFromScheme(runtime.NewScheme())
	corev1.AddToScheme(s)
	objInterface := admission.NewObjectInterfacesFromScheme(s)
	ns := "default"
	podName := "not-allowed"
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      podName,
			Namespace: ns,
		},
		Spec: corev1.PodSpec{},
	}
	attr := admission.NewAttributesRecord(pod, nil, schema.GroupVersionKind{"", "v1", "Pod"}, ns, podName, schema.GroupVersionResource{"", "v1", "pods"}, "", admission.Create, &metav1.CreateOptions{}, false, &user.DefaultInfo{})

	err := admissionController.Validate(ctx, attr, objInterface)
	if err == nil {
		t.Fatalf("request should fail")
	}
	if !strings.Contains(err.Error(), "reject") {
		t.Fatal("not rejected", err)
	}
}

func newTestAdmissionControllerMut(t *testing.T) *AdmissionController {
	config := &AdmissionModuleConfig{
		File:     admissionMutTestModuleFile,
		Mutating: true,
		Rules: []v1.RuleWithOperations{
			{
				Operations: []v1.OperationType{"CREATE"},
				Rule: v1.Rule{
					APIGroups:   []string{"*"},
					APIVersions: []string{"*"},
					Resources:   []string{"*"},
				},
			},
		},
		Debug: false,
	}
	admissionController, err := NewAdmissionControllerWithConfig(config)
	if err != nil {
		t.Fatal(err)
	}
	return admissionController
}

func TestAdmissionMutating(t *testing.T) {
	admissionController := newTestAdmissionControllerMut(t)
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

	err := admissionController.Admit(ctx, attr, objInterface)
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
