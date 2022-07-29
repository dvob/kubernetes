package admission

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"reflect"

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

var _ k8s.MutationInterface = (*AdmissionController)(nil)
var _ k8s.ValidationInterface = (*AdmissionController)(nil)

type AdmissionReviewFunc func(context.Context, *admissionv1.AdmissionReview) (*admissionv1.AdmissionReview, error)

type AdmissionController struct {
	review   AdmissionReviewFunc
	Mutating bool
	Rules    []v1.RuleWithOperations
}

func NewAdmissionControllerWithFn(fn AdmissionReviewFunc, mut bool, rules []v1.RuleWithOperations) *AdmissionController {
	return &AdmissionController{
		review:   fn,
		Mutating: mut,
		Rules:    rules,
	}
}

func NewAdmissionControllerWithConfig(config *ModuleConfig) (*AdmissionController, error) {
	source, err := os.ReadFile(config.Module)
	if err != nil {
		return nil, err
	}

	var runner wasi.Runner
	if config.Mutating {
		runner, err = wasi.NewWASIDefaultRunner(source, "mutate", config.Settings)
	} else {
		runner, err = wasi.NewWASIDefaultRunner(source, "validate", config.Settings)
	}

	if err != nil {
		return nil, err
	}

	reviewFn := func(ctx context.Context, in *admissionv1.AdmissionReview) (*admissionv1.AdmissionReview, error) {
		out := &admissionv1.AdmissionReview{}
		err := runner.Run(ctx, in, out)
		return out, err

	}

	return &AdmissionController{
		review:   reviewFn,
		Mutating: config.Mutating,
		Rules:    config.Rules,
	}, nil
}

func (a *AdmissionController) Handles(operation k8s.Operation) bool {
	// we run admission for all request. later in Admit and Validate we check if we
	// run the request through the WASM stuff by checking the rules
	return true
}

func (a *AdmissionController) Validate(ctx context.Context, attr k8s.Attributes, o k8s.ObjectInterfaces) (err error) {
	if a.Mutating {
		return nil
	}

	if !a.matchRequest(attr) {
		fmt.Println("skip")
		return nil
	}

	uid := types.UID(uuid.NewUUID())
	req, err := a.toAdmissionReview(uid, attr, o)
	if err != nil {
		return err
	}

	resp, err := a.review(ctx, req)
	if err != nil {
		return err
	}

	result, err := request.VerifyAdmissionResponse(uid, a.Mutating, resp)
	if err != nil {
		return err
	}

	if result.Allowed {
		return nil
	}

	return toRejectErr("none", result.Result)
}

func (a *AdmissionController) Admit(ctx context.Context, attr k8s.Attributes, o k8s.ObjectInterfaces) (err error) {
	// TODO: use custom error with module name

	if !a.Mutating {
		return nil
	}

	if !a.matchRequest(attr) {
		klog.Info("skip")
		return nil
	}

	uid := types.UID(uuid.NewUUID())
	req, err := a.toAdmissionReview(uid, attr, o)
	if err != nil {
		return err
	}

	resp, err := a.review(ctx, req)
	if err != nil {
		return err
	}

	result, err := request.VerifyAdmissionResponse(uid, a.Mutating, resp)
	if err != nil {
		return err
	}

	if !result.Allowed {
		return toRejectErr("none", result.Result)
	}

	if result.PatchType != "Full" {
		return fmt.Errorf("patch type not supported")
	}

	if len(result.Patch) == 0 {
		return nil
	}

	// reset obj
	v := reflect.ValueOf(attr.GetObject())
	v.Elem().Set(reflect.Zero(v.Elem().Type()))

	err = json.Unmarshal(result.Patch, attr.GetObject())
	if err != nil {
		return fmt.Errorf("failed to apply changes: %w", err)
	}

	return nil
}

func (a *AdmissionController) toAdmissionReview(uid types.UID, attr k8s.Attributes, o k8s.ObjectInterfaces) (*admissionv1.AdmissionReview, error) {
	versionedAttr, err := generic.NewVersionedAttributes(attr, attr.GetKind(), o)
	if err != nil {
		return nil, err
	}

	invocation := &generic.WebhookInvocation{
		Webhook:     nil,
		Resource:    attr.GetResource(),
		Subresource: attr.GetSubresource(),
		Kind:        attr.GetKind(),
	}

	req := request.CreateV1AdmissionReview(uid, versionedAttr, invocation)
	return req, nil
}

func (a *AdmissionController) matchRequest(attr k8s.Attributes) bool {
	for _, rule := range a.Rules {
		if (&rules.Matcher{Attr: attr, Rule: rule}).Matches() {
			return true
		}
	}
	return false
}
