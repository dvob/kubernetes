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
	"k8s.io/kubernetes/pkg/apis/core"

	// adds core scheme and conversions to scheme
	_ "k8s.io/kubernetes/pkg/apis/core/install"
)

const (
	admissionTestModuleFile    = "../testmodules/target/wasm32-wasi/debug/test_admission.wasm"
	admissionMutTestModuleFile = "../testmodules/target/wasm32-wasi/debug/test_admission_mut.wasm"
	safeAnnotationsModule      = "../testmodules/kubewarden/safe-annotations_v0.2.0.wasm"
	allowPrivilegeModule       = "../testmodules/kubewarden/allow-privilege-escalation-psp-policy_v0.1.11.wasm"
)

var (
	scheme *runtime.Scheme = runtime.NewScheme()
)

func init() {
	corev1.AddToScheme(scheme)
}

func TestWASIValidate(t *testing.T) {
	config := &ModuleConfig{
		Type:     ModuleTypeWASI,
		Module:   admissionTestModuleFile,
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
	admissionController, err := NewModule(config)
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()

	//s := runtime.NewScheme()
	objInterface := admission.NewObjectInterfacesFromScheme(scheme)
	ns := "default"
	podName := "not-allowed"
	pod := &core.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      podName,
			Namespace: ns,
		},
		Spec: core.PodSpec{},
	}
	var obj runtime.Object = pod
	attr := admission.NewAttributesRecord(obj, nil, schema.GroupVersionKind{"", "v1", "Pod"}, ns, podName, schema.GroupVersionResource{"", "v1", "pods"}, "", admission.Create, &metav1.CreateOptions{}, false, &user.DefaultInfo{})

	err = admissionController.Validate(ctx, attr, objInterface)
	if err == nil {
		t.Fatalf("request should fail")
	}
	if !strings.Contains(err.Error(), "denied") {
		t.Fatal("not rejected", err)
	}
}

func TestWASIMutate(t *testing.T) {
	config := &ModuleConfig{
		Module:   admissionMutTestModuleFile,
		Type:     ModuleTypeWASI,
		Debug:    false,
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
	}
	admissionController, err := NewModule(config)
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()

	//s := runtime.NewScheme()
	// corev1.AddToScheme(s)
	// core.AddToScheme(s)
	objInterface := admission.NewObjectInterfacesFromScheme(scheme)
	ns := "default"
	podName := "foo"
	var obj runtime.Object = &core.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      podName,
			Namespace: ns,
		},
		Spec: core.PodSpec{
			Containers: []core.Container{
				{
					Name: "bla",
				},
			},
		},
	}
	attr := admission.NewAttributesRecord(obj, nil, schema.GroupVersionKind{"", "v1", "Pod"}, ns, podName, schema.GroupVersionResource{"", "v1", "pods"}, "", admission.Create, &metav1.CreateOptions{}, false, &user.DefaultInfo{})

	err = admissionController.Admit(ctx, attr, objInterface)
	if err != nil {
		t.Fatal(err)
	}

	expectedAnnotationKey := "puzzle.ch/test-annotation"
	expectedAnnotationValue := "foo"

	pod, ok := attr.GetObject().(*core.Pod)
	if !ok {
		t.Fatalf("obj is not pod but %T", pod)
	}

	val, ok := pod.GetAnnotations()[expectedAnnotationKey]
	if !ok {
		t.Fatalf("annotation '%s' missing on pod", expectedAnnotationKey)
	}
	if val != expectedAnnotationValue {
		t.Fatalf("annotation '%s' has wrong value: want=%s, got=%s", expectedAnnotationKey, expectedAnnotationValue, val)
	}
}

func TestKubewardenAnnotationMutate(t *testing.T) {
	config := &ModuleConfig{
		Module:   "../testmodules/target/wasm32-wasi/debug/test_kubewarden_mut.wasm",
		Type:     ModuleTypeKubewarden,
		Mutating: true,
		Debug:    false,
		Settings: struct{}{},
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
	admissionController, err := NewModule(config)
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()

	objInterface := admission.NewObjectInterfacesFromScheme(scheme)
	ns := "default"
	podName := "foo"
	var obj runtime.Object = &core.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      podName,
			Namespace: ns,
		},
		Spec: core.PodSpec{
			Containers: []core.Container{
				{
					Name: "bla",
				},
			},
		},
	}
	attr := admission.NewAttributesRecord(obj, nil, schema.GroupVersionKind{"", "v1", "Pod"}, ns, podName, schema.GroupVersionResource{"", "v1", "pods"}, "", admission.Create, &metav1.CreateOptions{}, false, &user.DefaultInfo{})

	err = admissionController.Admit(ctx, attr, objInterface)
	if err != nil {
		t.Fatal(err)
	}

	expectedAnnotationKey := "puzzle.ch/test-annotation"
	expectedAnnotationValue := "foo"

	pod, ok := attr.GetObject().(*core.Pod)
	if !ok {
		t.Fatalf("obj is not pod but %T", pod)
	}

	val, ok := pod.GetAnnotations()[expectedAnnotationKey]
	if !ok {
		t.Fatalf("annotation '%s' missing on pod", expectedAnnotationKey)
	}
	if val != expectedAnnotationValue {
		t.Fatalf("annotation '%s' has wrong value: want=%s, got=%s", expectedAnnotationKey, expectedAnnotationValue, val)
	}
}

func TestKubewardenValidate(t *testing.T) {
	moduleConfig := &ModuleConfig{
		Name:     "safe-annotations",
		Type:     "kubewarden",
		Module:   safeAnnotationsModule,
		Mutating: false,
		Settings: struct {
			DeniedAnnotations []string `json:"denied_annotations"`
		}{
			DeniedAnnotations: []string{
				"invalid-annotation",
			},
		},
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

	admissionController, err := NewModule(moduleConfig)
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()

	//s := runtime.NewScheme()
	objInterface := admission.NewObjectInterfacesFromScheme(scheme)
	ns := "default"
	podName := "not-allowed"
	pod := &core.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      podName,
			Namespace: ns,
			Annotations: map[string]string{
				"invalid-annotation": "bla",
			},
		},
		Spec: core.PodSpec{},
	}
	var obj runtime.Object = pod
	attr := admission.NewAttributesRecord(obj, nil, schema.GroupVersionKind{"", "v1", "Pod"}, ns, podName, schema.GroupVersionResource{"", "v1", "pods"}, "", admission.Create, &metav1.CreateOptions{}, false, &user.DefaultInfo{})

	err = admissionController.Validate(ctx, attr, objInterface)
	if err == nil {
		t.Fatalf("request should fail")
	}

	expectedErrText := "\"safe-annotations\" denied the request"
	if !strings.Contains(err.Error(), expectedErrText) {
		t.Fatalf("text '%s' expected in error. got=%s", expectedErrText, err.Error())
	}
}

func TestKubewardenMutate(t *testing.T) {
	moduleConfig := &ModuleConfig{
		Name:     "privilege-escalation",
		Type:     "kubewarden",
		Module:   allowPrivilegeModule,
		Mutating: true,
		Debug:    false,
		Settings: struct {
			DefaultAllowPrivilegeEscalation bool `json:"default_allow_privilege_escalation"`
		}{
			DefaultAllowPrivilegeEscalation: false,
		},
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

	admissionController, err := NewModule(moduleConfig)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()

	objInterface := admission.NewObjectInterfacesFromScheme(scheme)
	ns := "default"
	podName := "not-allowed"
	pod := &core.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      podName,
			Namespace: ns,
		},
		Spec: core.PodSpec{
			NodeName: "bla",
			Containers: []core.Container{
				{
					Name: "test",
				},
			},
		},
	}

	var obj runtime.Object = pod
	attr := admission.NewAttributesRecord(obj, nil, schema.GroupVersionKind{"", "v1", "Pod"}, ns, podName, schema.GroupVersionResource{"", "v1", "pods"}, "", admission.Create, &metav1.CreateOptions{}, false, &user.DefaultInfo{})

	err = admissionController.Admit(ctx, attr, objInterface)
	if err != nil {
		t.Fatal(err)
	}

	sc := pod.Spec.Containers[0].SecurityContext

	if sc == nil {
		t.Fatal("security context got not set in pod")
	}
	if sc.AllowPrivilegeEscalation == nil {
		t.Fatal("allow privilege escalation not set in security context")
	}

	if *sc.AllowPrivilegeEscalation != false {
		t.Fatal("allowPrivilegeEscalation is not false")
	}
}
