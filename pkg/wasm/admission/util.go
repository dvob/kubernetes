package admission

import (
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func toRejectErr(moduleName string, status *metav1.Status) error {
	deniedBy := fmt.Sprintf("admission WASM module %q denied the request", moduleName)

	switch {
	case status == nil:
		return fmt.Errorf("%s %s", deniedBy, "without explanation")
	case len(status.Message) > 0:
		return fmt.Errorf("%s: %s", deniedBy, status.Message)
	case len(status.Reason) > 0:
		return fmt.Errorf("%s: %s", deniedBy, status.Reason)
	default:
		return fmt.Errorf("%s %s", deniedBy, "without explanation")
	}
}
