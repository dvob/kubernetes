package wasm

import (
	"context"
	"io"
	"fmt"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apiserver/pkg/admission"
	api "k8s.io/kubernetes/pkg/apis/core"
)

// PluginName indicates name of admission plugin.
const PluginName = "WASMAdmission"

// Register registers a plugin
func Register(plugins *admission.Plugins) {
	plugins.Register(PluginName, func(config io.Reader) (admission.Interface, error) {
		return NewWASMAdmission(), nil
	})
}

// AlwaysPullImages is an implementation of admission.Interface.
// It looks at all new pods and overrides each container's image pull policy to Always.
type WASMAdmission struct {
	*admission.Handler
}

var _ admission.MutationInterface = &WASMAdmission{}
var _ admission.ValidationInterface = &WASMAdmission{}

// Admit makes an admission decision based on the request attributes
func (a *WASMAdmission) Admit(ctx context.Context, attributes admission.Attributes, o admission.ObjectInterfaces) (err error) {
	// Ignore all calls to subresources or resources other than pods.
	if shouldIgnore(attributes) {
		return nil
	}
	cm, ok := attributes.GetObject().(*api.ConfigMap)
	if !ok {
		return apierrors.NewBadRequest("Resource was marked with kind ConfigMap but was unable to be converted")
	}

	cm.Data["hello-from-admission"] = "none"

	return nil
}

// Validate makes sure that all containers are set to always pull images
func (*WASMAdmission) Validate(ctx context.Context, attributes admission.Attributes, o admission.ObjectInterfaces) (err error) {
	if shouldIgnore(attributes) {
		return nil
	}

	cm, ok := attributes.GetObject().(*api.ConfigMap)
	if !ok {
		return apierrors.NewBadRequest("Resource was marked with kind ConfigMap but was unable to be converted")
	}

	_, ok = cm.Data["hello-from-admission"]
	if !ok {
		return admission.NewForbidden(attributes, fmt.Errorf("field not found"))
	}
	return nil
}

func shouldIgnore(attributes admission.Attributes) bool {
	// Ignore all calls to subresources or resources other than configmaps
	if len(attributes.GetSubresource()) != 0 || attributes.GetResource().GroupResource() != api.Resource("configmaps") {
		return true
	}
	return false
}

func NewWASMAdmission() *WASMAdmission {
	return &WASMAdmission{
		Handler: admission.NewHandler(admission.Create, admission.Update),
	}
}
