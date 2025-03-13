package utils

import (
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// UpdateConditions merges the "newConditions" into "conditions" while keeping the "LastTransitionTime" of unchanged
// conditions and removing stale conditions that are not present in "newConditions" anymore.
func UpdateConditions(conditions *[]metav1.Condition, newConditions []metav1.Condition) bool {
	changed := false
	for _, newCond := range newConditions {
		setChanged := meta.SetStatusCondition(conditions, newCond)
		if setChanged {
			changed = true
		}
	}

	for _, cond := range *conditions {
		if meta.FindStatusCondition(newConditions, cond.Type) == nil {
			removeChanged := meta.RemoveStatusCondition(conditions, cond.Type)
			if removeChanged {
				changed = true
			}
		}
	}

	return changed
}
