package admission

import (
	"context"
	"encoding/json"
	"fmt"
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
)

type AdmissionReviewFunc func(context.Context, *admissionv1.AdmissionReview) (*admissionv1.AdmissionReview, error)

var _ k8s.MutationInterface = (*Controller)(nil)
var _ k8s.ValidationInterface = (*Controller)(nil)

type Controller struct {
	review   AdmissionReviewFunc
	mutating bool
	rules    []v1.RuleWithOperations
}

func (c *Controller) Handles(operation k8s.Operation) bool {
	for _, rule := range c.rules {
		for _, op := range rule.Operations {
			if op == v1.OperationAll {
				return true
			}
			if op == v1.OperationType(operation) {
				return true
			}
		}
	}
	return false
}

func (c *Controller) Validate(ctx context.Context, attr k8s.Attributes, o k8s.ObjectInterfaces) (err error) {
	if c.mutating {
		return nil
	}

	if !c.matchRequest(attr) {
		return nil
	}

	start := time.Now()
	defer func() { klog.InfoS("run validation", "duration", time.Now().Sub(start)) }()

	versionedAttr, err := generic.NewVersionedAttributes(attr, attr.GetKind(), o)
	if err != nil {
		return err
	}

	uid := types.UID(uuid.NewUUID())
	req := toAdmissionReview(uid, versionedAttr, o)

	resp, err := c.review(ctx, req)
	if err != nil {
		return err
	}

	result, err := request.VerifyAdmissionResponse(uid, c.mutating, resp)
	if err != nil {
		return err
	}

	if result.Allowed {
		return nil
	}

	return toRejectErr(result.Result)
}

func (c *Controller) Admit(ctx context.Context, attr k8s.Attributes, o k8s.ObjectInterfaces) (err error) {
	if !c.mutating {
		return nil
	}

	if !c.matchRequest(attr) {
		return nil
	}

	start := time.Now()
	defer func() { klog.InfoS("run mutation", "duration", time.Now().Sub(start)) }()

	versionedAttr, err := generic.NewVersionedAttributes(attr, attr.GetKind(), o)
	if err != nil {
		return err
	}

	uid := types.UID(uuid.NewUUID())
	req := toAdmissionReview(uid, versionedAttr, o)

	resp, err := c.review(ctx, req)
	if err != nil {
		return err
	}

	result, err := request.VerifyAdmissionResponse(uid, c.mutating, resp)
	if err != nil {
		return err
	}

	if !result.Allowed {
		return toRejectErr(result.Result)
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

func (c *Controller) matchRequest(attr k8s.Attributes) bool {
	for _, rule := range c.rules {
		if (&rules.Matcher{Attr: attr, Rule: rule}).Matches() {
			return true
		}
	}
	return false
}

func toAdmissionReview(uid types.UID, versionedAttr *generic.VersionedAttributes, o k8s.ObjectInterfaces) *admissionv1.AdmissionReview {
	invocation := &generic.WebhookInvocation{
		Webhook:     nil,
		Resource:    versionedAttr.GetResource(),
		Subresource: versionedAttr.GetSubresource(),
		Kind:        versionedAttr.GetKind(),
	}

	return request.CreateV1AdmissionReview(uid, versionedAttr, invocation)
}
