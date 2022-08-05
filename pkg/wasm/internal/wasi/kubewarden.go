package wasi

import (
	"context"
	"encoding/json"
	"fmt"

	v1 "k8s.io/api/admission/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

type SettingsValidationResponse struct {
	Valid   bool   `json:"valid"`
	Message string `json:"message"`
}

type ValidationRequest struct {
	Request  *v1.AdmissionRequest `json:"request"`
	Settings interface{}          `json:"settings"`
}

// ValidationResponse is the data which is returned by a Kubewarden module.
// The definition is taken from https://github.com/kubewarden/policy-sdk-rust/blob/9fec79b066ef35dd9dda7025fc40b4e9e429fb14/src/response.rs#L7
type ValidationResponse struct {
	// True if the request has been accepted, false otherwise
	Accepted bool `json:"accepted"`

	// Message shown to the user when the request is rejected
	Message *string `json:"message"`

	// Code shown to the user when the request is rejected
	Code uint16 `json:"code"`

	// Mutated Object - used only by mutation policies
	MutatedObject json.RawMessage `json:"mutated_object"`

	// AuditAnnotations is an unstructured key value map set by remote admission controller (e.g. error=image-blacklisted).
	// MutatingAdmissionWebhook and ValidatingAdmissionWebhook admission controller will prefix the keys with
	// admission webhook name (e.g. imagepolicy.example.com/error=image-blacklisted). AuditAnnotations will be provided by
	// the admission webhook to add additional context to the audit log for this request.
	AuditAnnotations map[string]string `json:"audit_annotations"`

	// warnings is a list of warning messages to return to the requesting API client.
	// Warning messages describe a problem the client making the API request should correct or be aware of.
	// Limit warnings to 120 characters if possible.
	// Warnings over 256 characters and large numbers of warnings may be truncated.
	Warnings []string `json:"warnings"`
}

type KubewardenModule struct {
	runtime          *WAPCRuntime
	validate         Runner
	validateSettings Runner
}

func NewKubewardenModule(moduleSource []byte, debug bool) (*KubewardenModule, error) {
	runtime, err := NewWAPCRuntime(moduleSource)
	if err != nil {
		return nil, err
	}

	validateRawRunner := runtime.RawRunner("validate")
	validateSettingsRawRunner := runtime.RawRunner("validate_settings")

	if debug {
		validateRawRunner = DebugRawRunner(validateRawRunner)
		validateSettingsRawRunner = DebugRawRunner(validateSettingsRawRunner)
	}

	return &KubewardenModule{
		runtime:          runtime,
		validateSettings: NewJSONRunner(validateSettingsRawRunner),
		validate:         NewJSONRunner(validateRawRunner),
	}, nil
}

func (k *KubewardenModule) ValidateSettings(ctx context.Context, settings interface{}) error {
	resp := &SettingsValidationResponse{}
	err := k.validateSettings.Run(ctx, settings, resp)
	if err != nil {
		return err
	}

	if !resp.Valid {
		return fmt.Errorf("invalid settings: %s", resp.Message)
	}
	return nil
}

func (k *KubewardenModule) Validate(ctx context.Context, ar *v1.AdmissionReview, settings interface{}) (*v1.AdmissionReview, error) {
	if ar.Request == nil {
		return nil, fmt.Errorf("request not set in admission review")
	}

	req := ValidationRequest{
		Request:  ar.Request,
		Settings: settings,
	}
	resp := &ValidationResponse{}
	err := k.validate.Run(ctx, req, resp)
	if err != nil {
		return nil, err
	}
	admissionResponse, err := convertResponse(ar.Request.UID, resp)
	if err != nil {
		return nil, fmt.Errorf("failed to convert to admission response: %w", err)
	}
	arOut := &v1.AdmissionReview{
		Response: admissionResponse,
	}
	// TODO: can we to this better?
	arOut.SetGroupVersionKind(v1.SchemeGroupVersion.WithKind("AdmissionReview"))
	return arOut, nil
}

func convertResponse(uid types.UID, resp *ValidationResponse) (*v1.AdmissionResponse, error) {
	var (
		status    *metav1.Status
		patchType *v1.PatchType
	)
	if !resp.Accepted && resp.Message != nil {
		status = &metav1.Status{
			Message: *resp.Message,
			Code:    int32(resp.Code),
		}
	}
	if len(resp.MutatedObject) > 0 {
		fullPatch := v1.PatchType("Full")
		patchType = &fullPatch
	}
	return &v1.AdmissionResponse{
		UID:              uid,
		Allowed:          resp.Accepted,
		Result:           status,
		Patch:            []byte(resp.MutatedObject),
		PatchType:        patchType,
		AuditAnnotations: resp.AuditAnnotations,
		Warnings:         resp.Warnings,
	}, nil
}
