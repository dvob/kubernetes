package admission

import (
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func toRejectErr(status *metav1.Status) error {
	deniedBy := fmt.Sprintf("admission denied the request")

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
