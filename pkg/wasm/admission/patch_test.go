package admission

import (
	"reflect"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestPatch(t *testing.T) {
	origPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "pod1",
			Namespace: "ns1",
			Annotations: map[string]string{
				"puzzle.ch/key1": "bla",
				"puzzle.ch/key2": "bla",
			},
		},
	}

	patchPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "pod1",
			Namespace: "ns1",
			Annotations: map[string]string{
				"puzzle.ch/key1": "bla",
			},
		},
	}

	patch := func(in interface{}, patch interface{}) error {
		v := reflect.ValueOf(in)
		v.Elem().Set(reflect.ValueOf(patch).Elem())
		//v.Elem().Set(reflect.Zero(v.Elem().Type()))
		return nil
	}

	patch(origPod, patchPod)

	if _, ok := origPod.GetAnnotations()["puzzle.ch/key2"]; ok {
		t.Fatal("annotation not removed")
	}
}
