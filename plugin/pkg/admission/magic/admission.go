package magic

import (
	"context"
	"fmt"
	"io"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apiserver/pkg/admission"
	"k8s.io/kubernetes/pkg/apis/core"
)

// PluginName indicates name of admission plugin.
const PluginName = "MagicAdmission"

// Register registers a plugin
func Register(plugins *admission.Plugins) {
	plugins.Register(PluginName, func(config io.Reader) (admission.Interface, error) {
		return NewMagicAdmission(), nil
	})
}

var _ admission.MutationInterface = &MagicAdmission{}
var _ admission.ValidationInterface = &MagicAdmission{}

type MagicAdmission struct {
	*admission.Handler
}

func NewMagicAdmission() *MagicAdmission {
	return &MagicAdmission{
		Handler: admission.NewHandler(admission.Create, admission.Update),
	}
}

func (a *MagicAdmission) Admit(ctx context.Context, attributes admission.Attributes, o admission.ObjectInterfaces) (err error) {
	if shouldIgnore(attributes) {
		return nil
	}
	cm, ok := attributes.GetObject().(*core.ConfigMap)
	if !ok {
		return apierrors.NewBadRequest("Resource was marked with kind ConfigMap but was unable to be converted")
	}

	if cm.Data == nil {
		cm.Data = map[string]string{}
	}
	cm.Data["magic-value"] = "foobar"

	return nil
}

func (*MagicAdmission) Validate(ctx context.Context, attributes admission.Attributes, o admission.ObjectInterfaces) (err error) {
	if shouldIgnore(attributes) {
		return nil
	}

	cm, ok := attributes.GetObject().(*core.ConfigMap)
	if !ok {
		return apierrors.NewBadRequest("Resource was marked with kind ConfigMap but was unable to be converted")
	}

	if _, ok = cm.Data["not-allowed-value"]; ok {
		return admission.NewForbidden(attributes, fmt.Errorf("value 'not-allowed-value' not allowed in configmap"))
	}
	return nil
}

func shouldIgnore(attributes admission.Attributes) bool {
	// Ignore all calls to subresources or resources other than configmaps
	if len(attributes.GetSubresource()) != 0 || attributes.GetResource().GroupResource() != core.Resource("configmaps") {
		return true
	}
	return false
}
