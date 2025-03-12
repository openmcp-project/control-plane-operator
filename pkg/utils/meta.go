package utils

import (
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	labelManagedBy      = "app.kubernetes.io/managed-by"
	labelManagedByValue = "control-plane-operator"
)

func SetLabel(obj v1.Object, label string, value string) {
	labels := obj.GetLabels()
	if labels == nil {
		labels = map[string]string{}
	}
	labels[label] = value
	obj.SetLabels(labels)
}

func SetManagedBy(obj v1.Object) {
	SetLabel(obj, labelManagedBy, labelManagedByValue)
}

func IsManaged() client.MatchingLabels {
	return client.MatchingLabels{labelManagedBy: labelManagedByValue}
}
