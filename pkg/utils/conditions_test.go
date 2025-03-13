package utils

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	condType1 = "Condition1"
	condType2 = "Condition2"
	condType3 = "Condition3"
	condType4 = "Condition4"
)

var (
	transitionTime      = metav1.Now()
	transitionTimeLater = metav1.Time{Time: transitionTime.Add(1 * time.Hour)}
)

func Test_UpdateConditions(t *testing.T) {
	testCases := []struct {
		desc                                          string
		conditions, newConditions, expectedConditions []metav1.Condition
		expectedChanged                               bool
	}{
		{
			desc: "should add conditions to empty list",
			newConditions: []metav1.Condition{
				{
					Type:               condType1,
					Status:             metav1.ConditionFalse,
					LastTransitionTime: transitionTime,
				},
			},
			expectedConditions: []metav1.Condition{
				{
					Type:               condType1,
					Status:             metav1.ConditionFalse,
					LastTransitionTime: transitionTime,
				},
			},
			expectedChanged: true,
		},
		{
			desc: "should add conditions to list with existing items",
			conditions: []metav1.Condition{
				{
					Type:               condType1,
					Status:             metav1.ConditionFalse,
					LastTransitionTime: transitionTime,
				},
			},
			newConditions: []metav1.Condition{
				{
					Type:               condType1,
					Status:             metav1.ConditionFalse,
					LastTransitionTime: transitionTimeLater,
				},
				{
					Type:               condType2,
					Status:             metav1.ConditionTrue,
					LastTransitionTime: transitionTimeLater,
				},
			},
			expectedConditions: []metav1.Condition{
				{
					Type:               condType1,
					Status:             metav1.ConditionFalse,
					LastTransitionTime: transitionTime,
				},
				{
					Type:               condType2,
					Status:             metav1.ConditionTrue,
					LastTransitionTime: transitionTimeLater,
				},
			},
			expectedChanged: true,
		},
		{
			desc: "should override existing condition with same type",
			conditions: []metav1.Condition{
				{
					Type:               condType1,
					Status:             metav1.ConditionFalse,
					LastTransitionTime: transitionTime,
				},
			},
			newConditions: []metav1.Condition{
				{
					Type:               condType1,
					Status:             metav1.ConditionTrue,
					LastTransitionTime: transitionTimeLater,
				},
			},
			expectedConditions: []metav1.Condition{
				{
					Type:               condType1,
					Status:             metav1.ConditionTrue,
					LastTransitionTime: transitionTimeLater,
				},
			},
			expectedChanged: true,
		},
		{
			desc: "should remove stale condition",
			conditions: []metav1.Condition{
				{
					Type:               condType1,
					Status:             metav1.ConditionFalse,
					LastTransitionTime: transitionTime,
				},
				{
					Type:               condType2,
					Status:             metav1.ConditionTrue,
					LastTransitionTime: transitionTime,
				},
			},
			newConditions: []metav1.Condition{
				{
					Type:               condType2,
					Status:             metav1.ConditionTrue,
					LastTransitionTime: transitionTimeLater,
				},
			},
			expectedConditions: []metav1.Condition{
				{
					Type:               condType2,
					Status:             metav1.ConditionTrue,
					LastTransitionTime: transitionTime,
				},
			},
			expectedChanged: true,
		},
		{
			desc: "should do everything at once",
			conditions: []metav1.Condition{
				{
					Type:               condType1,
					Status:             metav1.ConditionFalse,
					LastTransitionTime: transitionTime,
				},
				{
					Type:               condType2,
					Status:             metav1.ConditionTrue,
					LastTransitionTime: transitionTime,
				},
				{
					Type:               condType3,
					Status:             metav1.ConditionFalse,
					LastTransitionTime: transitionTime,
				},
			},
			newConditions: []metav1.Condition{
				{
					Type:               condType1,
					Status:             metav1.ConditionFalse,
					LastTransitionTime: transitionTimeLater,
				},
				{
					Type:               condType2,
					Status:             metav1.ConditionFalse,
					LastTransitionTime: transitionTimeLater,
				},
				{
					Type:               condType4,
					Status:             metav1.ConditionFalse,
					LastTransitionTime: transitionTimeLater,
				},
			},
			expectedConditions: []metav1.Condition{
				{
					Type:               condType1,
					Status:             metav1.ConditionFalse,
					LastTransitionTime: transitionTime,
				},
				{
					Type:               condType2,
					Status:             metav1.ConditionFalse,
					LastTransitionTime: transitionTimeLater,
				},
				{
					Type:               condType4,
					Status:             metav1.ConditionFalse,
					LastTransitionTime: transitionTimeLater,
				},
			},
			expectedChanged: true,
		},
		{
			desc: "should not do anything",
			conditions: []metav1.Condition{
				{
					Type:               condType1,
					Status:             metav1.ConditionFalse,
					LastTransitionTime: transitionTime,
				},
			},
			newConditions: []metav1.Condition{
				{
					Type:               condType1,
					Status:             metav1.ConditionFalse,
					LastTransitionTime: transitionTimeLater,
				},
			},
			expectedConditions: []metav1.Condition{
				{
					Type:               condType1,
					Status:             metav1.ConditionFalse,
					LastTransitionTime: transitionTime,
				},
			},
			expectedChanged: false,
		},
	}
	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			actualChanged := UpdateConditions(&tC.conditions, tC.newConditions)
			assert.Equal(t, tC.expectedChanged, actualChanged)
			assert.EqualValues(t, tC.expectedConditions, tC.conditions)
		})
	}
}
