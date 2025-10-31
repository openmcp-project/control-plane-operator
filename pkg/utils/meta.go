package utils

import (
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	LabelManagedBy      = "app.kubernetes.io/managed-by"
	LabelManagedByValue = "control-plane-operator"
	LabelComponentName  = "controlplane.core.orchestrate.cloud.sap/component"
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
	SetLabel(obj, LabelManagedBy, LabelManagedByValue)
}

func IsManaged() client.MatchingLabels {
	return client.MatchingLabels{LabelManagedBy: LabelManagedByValue}
}

func HasComponentLabel() client.ListOption {
	return client.HasLabels{LabelComponentName}
}

func SetLabels(obj v1.Object, labels map[string]string) {
	for k, v := range labels {
		SetLabel(obj, k, v)
	}
}
