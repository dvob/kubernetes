package admission

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"time"

	admissionv1 "k8s.io/api/admission/v1"
	v1 "k8s.io/api/admissionregistration/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/uuid"
	k8s "k8s.io/apiserver/pkg/admission"
	"k8s.io/apiserver/pkg/admission/plugin/webhook/generic"
	"k8s.io/apiserver/pkg/admission/plugin/webhook/request"
	"k8s.io/apiserver/pkg/admission/plugin/webhook/rules"
	"k8s.io/klog/v2"
	"k8s.io/kubernetes/pkg/wasm/internal/wasi"
)

var _ k8s.MutationInterface = (*Module)(nil)
var _ k8s.ValidationInterface = (*Module)(nil)

type AdmissionReviewFunc func(context.Context, *admissionv1.AdmissionReview) (*admissionv1.AdmissionReview, error)

type Module struct {
	name     string
	review   AdmissionReviewFunc
	Mutating bool
	Rules    []v1.RuleWithOperations
}

func NewModuleFromFn(fn AdmissionReviewFunc, mut bool, rules []v1.RuleWithOperations) *Module {
	return &Module{
		review:   fn,
		Mutating: mut,
		Rules:    rules,
	}
}

func NewModule(config *ModuleConfig) (*Module, error) {
	switch config.Type {
	case ModuleTypeWASI:
		return NewWASIModule(config)
	case ModuleTypeKubewarden:
		return NewKubewardenModule(config)
	default:
		return nil, fmt.Errorf("unknown module type '%s'", config.Type)
	}
}

func NewWASIModule(config *ModuleConfig) (*Module, error) {
	source, err := os.ReadFile(config.Module)
	if err != nil {
		return nil, err
	}
	runtime, err := wasi.NewRuntime(source)
	if err != nil {
		return nil, err
	}

	fnName := "validate"
	if !runtime.HasFunction(fnName) {
		return nil, fmt.Errorf("missing function '%s' in module '%s'", fnName, config.Name)
	}
	rawRunner := runtime.RawRunner(fnName)

	if config.Debug {
		rawRunner = wasi.DebugRawRunner(rawRunner)
	}

	runner := wasi.NewEnvelopeRunner(rawRunner, config.Settings)

	reviewFn := func(ctx context.Context, in *admissionv1.AdmissionReview) (*admissionv1.AdmissionReview, error) {
		out := &admissionv1.AdmissionReview{}
		err := runner.Run(ctx, in, out)
		return out, err

	}

	return &Module{
		name:     config.Name,
		review:   reviewFn,
		Mutating: config.Mutating,
		Rules:    config.Rules,
	}, nil

}

func NewKubewardenModule(config *ModuleConfig) (*Module, error) {
	moduleSource, err := os.ReadFile(config.Module)
	if err != nil {
		return nil, err
	}

	mod, err := wasi.NewKubewardenModule(moduleSource, config.Debug)
	if err != nil {
		return nil, err
	}

	err = mod.ValidateSettings(context.Background(), config.Settings)
	if err != nil {
		return nil, err
	}

	reviewFn := func(ctx context.Context, ar *admissionv1.AdmissionReview) (*admissionv1.AdmissionReview, error) {
		return mod.Validate(ctx, ar, config.Settings)
	}
	return &Module{
		name:     config.Name,
		review:   reviewFn,
		Mutating: config.Mutating,
		Rules:    config.Rules,
	}, nil
}

func (m *Module) Handles(operation k8s.Operation) bool {
	for _, rule := range m.Rules {
		for _, op := range rule.Operations {
			if op == v1.OperationAll {
				return true
			}
			// The constants are the same such that this is a valid cast (and this
			// is tested).
			if op == v1.OperationType(operation) {
				return true
			}
		}
	}
	return false
}

func (m *Module) Validate(ctx context.Context, attr k8s.Attributes, o k8s.ObjectInterfaces) (err error) {
	if m.Mutating {
		return nil
	}

	if !m.matchRequest(attr) {
		return nil
	}

	start := time.Now()
	defer func() { klog.InfoS("run validation", "duration", time.Now().Sub(start)) }()

	versionedAttr, err := generic.NewVersionedAttributes(attr, attr.GetKind(), o)
	if err != nil {
		return err
	}

	uid := types.UID(uuid.NewUUID())
	req := m.toAdmissionReview(uid, versionedAttr, o)

	resp, err := m.review(ctx, req)
	if err != nil {
		return err
	}

	result, err := request.VerifyAdmissionResponse(uid, m.Mutating, resp)
	if err != nil {
		return err
	}

	if result.Allowed {
		return nil
	}

	return toRejectErr(m.name, result.Result)
}

func (m *Module) Admit(ctx context.Context, attr k8s.Attributes, o k8s.ObjectInterfaces) (err error) {
	return errWithName(m.name, m.admit(ctx, attr, o))
}

func (m *Module) admit(ctx context.Context, attr k8s.Attributes, o k8s.ObjectInterfaces) (err error) {
	if !m.Mutating {
		return nil
	}

	if !m.matchRequest(attr) {
		return nil
	}

	start := time.Now()
	defer func() { klog.InfoS("run mutation", "duration", time.Now().Sub(start)) }()

	versionedAttr, err := generic.NewVersionedAttributes(attr, attr.GetKind(), o)
	if err != nil {
		return err
	}

	uid := types.UID(uuid.NewUUID())
	req := m.toAdmissionReview(uid, versionedAttr, o)

	// DEBUG
	typeOrig := reflect.TypeOf(attr.GetObject()).Elem()
	typeVersioned := reflect.TypeOf(req.Request.Object.Object).Elem()
	klog.InfoS("TYPE INFO", "orig", typeOrig.PkgPath()+"."+typeOrig.Name(), "versioned", typeVersioned.PkgPath()+"."+typeVersioned.Name())

	resp, err := m.review(ctx, req)
	if err != nil {
		return err
	}

	result, err := request.VerifyAdmissionResponse(uid, m.Mutating, resp)
	if err != nil {
		return err
	}

	if !result.Allowed {
		return toRejectErr(m.name, result.Result)
	}

	if result.PatchType != "Full" {
		return fmt.Errorf("patch type not supported")
	}

	if len(result.Patch) == 0 {
		return nil
	}

	// reset obj
	v := reflect.ValueOf(versionedAttr.VersionedObject)
	v.Elem().Set(reflect.Zero(v.Elem().Type()))

	err = json.Unmarshal(result.Patch, versionedAttr.VersionedObject)
	if err != nil {
		return fmt.Errorf("failed to apply changes: %w", err)
	}

	return o.GetObjectConvertor().Convert(versionedAttr.VersionedObject, attr.GetObject(), nil)
}

func (m *Module) toAdmissionReview(uid types.UID, versionedAttr *generic.VersionedAttributes, o k8s.ObjectInterfaces) *admissionv1.AdmissionReview {
	invocation := &generic.WebhookInvocation{
		Webhook:     nil,
		Resource:    versionedAttr.GetResource(),
		Subresource: versionedAttr.GetSubresource(),
		Kind:        versionedAttr.GetKind(),
	}

	return request.CreateV1AdmissionReview(uid, versionedAttr, invocation)
}

func (m *Module) matchRequest(attr k8s.Attributes) bool {
	for _, rule := range m.Rules {
		if (&rules.Matcher{Attr: attr, Rule: rule}).Matches() {
			return true
		}
	}
	return false
}

func errWithName(name string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("module=%s, err=%w", name, err)
}
