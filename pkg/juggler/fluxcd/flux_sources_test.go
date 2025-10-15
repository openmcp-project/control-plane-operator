//nolint:dupl
package fluxcd

import (
	"testing"

	fluxmeta "github.com/fluxcd/pkg/apis/meta"
	sourcev1 "github.com/fluxcd/source-controller/api/v1"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/openmcp-project/control-plane-operator/pkg/juggler"
)

func TestHelmRepositoryAdapter_GetHealthiness(t *testing.T) {
	tests := []struct {
		name     string
		adapter  HelmRepositoryAdapter
		expected juggler.ResourceHealthiness
	}{
		{
			name: "HelmRepositoryAdapter - Status Condition nil - Ready condition not present",
			adapter: HelmRepositoryAdapter{
				Source: &sourcev1.HelmRepository{
					Status: sourcev1.HelmRepositoryStatus{
						Conditions: nil,
					},
				},
			},
			expected: juggler.ResourceHealthiness{
				Healthy: false,
				Message: msgReadyNotPresent,
			},
		},
		{
			name: "HelmRepositoryAdapter - Status Condition Ready not found",
			adapter: HelmRepositoryAdapter{
				Source: &sourcev1.HelmRepository{
					Status: sourcev1.HelmRepositoryStatus{
						Conditions: []metav1.Condition{
							{
								Type:    "NotReady", // can not be found
								Status:  metav1.ConditionTrue,
								Message: "The release is ready",
							},
						},
					},
				},
			},
			expected: juggler.ResourceHealthiness{
				Healthy: false,
				Message: msgReadyNotPresent,
			},
		},
		{
			name: "HelmRepositoryAdapter - Status Condition Ready = True",
			adapter: HelmRepositoryAdapter{
				Source: &sourcev1.HelmRepository{
					Status: sourcev1.HelmRepositoryStatus{
						Conditions: []metav1.Condition{
							{
								Type:    fluxmeta.ReadyCondition,
								Status:  metav1.ConditionTrue,
								Message: "The release is ready",
							},
						},
					},
				},
			},
			expected: juggler.ResourceHealthiness{
				Healthy: true,
				Message: "The release is ready",
			},
		},
		{
			name: "HelmRepositoryAdapter - Status Condition Ready = False",
			adapter: HelmRepositoryAdapter{
				Source: &sourcev1.HelmRepository{
					Status: sourcev1.HelmRepositoryStatus{
						Conditions: []metav1.Condition{
							{
								Type:    fluxmeta.ReadyCondition,
								Status:  metav1.ConditionFalse,
								Message: "The release is not ready",
							},
						},
					},
				},
			},
			expected: juggler.ResourceHealthiness{
				Healthy: false,
				Message: "The release is not ready",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := tt.adapter.GetHealthiness()
			if !assert.Equal(t, tt.expected, actual) {
				t.Errorf("HelmRepositoryAdapter.GetHealthiness() = %v, want %v", actual, tt.expected)
			}
		})
	}
}

func TestGitRepositoryAdapter_GetHealthiness(t *testing.T) {
	tests := []struct {
		name     string
		adapter  GitRepositoryAdapter
		expected juggler.ResourceHealthiness
	}{
		{
			name: "GitRepositoryAdapter - Status Condition nil - Ready condition not present",
			adapter: GitRepositoryAdapter{
				Source: &sourcev1.GitRepository{
					Status: sourcev1.GitRepositoryStatus{
						Conditions: nil,
					},
				},
			},
			expected: juggler.ResourceHealthiness{
				Healthy: false,
				Message: msgReadyNotPresent,
			},
		},
		{
			name: "GitRepositoryAdapter - Status Condition Ready not found",
			adapter: GitRepositoryAdapter{
				Source: &sourcev1.GitRepository{
					Status: sourcev1.GitRepositoryStatus{
						Conditions: []metav1.Condition{
							{
								Type:    "NotReady", // can not be found
								Status:  metav1.ConditionTrue,
								Message: "The release is ready",
							},
						},
					},
				},
			},
			expected: juggler.ResourceHealthiness{
				Healthy: false,
				Message: msgReadyNotPresent,
			},
		},
		{
			name: "GitRepositoryAdapter - Status Condition Ready = True",
			adapter: GitRepositoryAdapter{
				Source: &sourcev1.GitRepository{
					Status: sourcev1.GitRepositoryStatus{
						Conditions: []metav1.Condition{
							{
								Type:    fluxmeta.ReadyCondition,
								Status:  metav1.ConditionTrue,
								Message: "The release is ready",
							},
						},
					},
				},
			},
			expected: juggler.ResourceHealthiness{
				Healthy: true,
				Message: "The release is ready",
			},
		},
		{
			name: "GitRepositoryAdapter - Status Condition Ready = False",
			adapter: GitRepositoryAdapter{
				Source: &sourcev1.GitRepository{
					Status: sourcev1.GitRepositoryStatus{
						Conditions: []metav1.Condition{
							{
								Type:    fluxmeta.ReadyCondition,
								Status:  metav1.ConditionFalse,
								Message: "The release is not ready",
							},
						},
					},
				},
			},
			expected: juggler.ResourceHealthiness{
				Healthy: false,
				Message: "The release is not ready",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := tt.adapter.GetHealthiness()
			if !assert.Equal(t, tt.expected, actual) {
				t.Errorf("GitRepositoryAdapter.GetHealthiness() = %v, want %v", actual, tt.expected)
			}
		})
	}
}

func TestOCIRepositoryAdapter_GetHealthiness(t *testing.T) {
	tests := []struct {
		name     string
		adapter  OCIRepositoryAdapter
		expected juggler.ResourceHealthiness
	}{
		{
			name: "OCIRepositoryAdapter - Status Condition nil - Ready condition not present",
			adapter: OCIRepositoryAdapter{
				Source: &sourcev1.OCIRepository{
					Status: sourcev1.OCIRepositoryStatus{
						Conditions: nil,
					},
				},
			},
			expected: juggler.ResourceHealthiness{
				Healthy: false,
				Message: msgReadyNotPresent,
			},
		},
		{
			name: "OCIRepositoryAdapter - Status Condition Ready not found",
			adapter: OCIRepositoryAdapter{
				Source: &sourcev1.OCIRepository{
					Status: sourcev1.OCIRepositoryStatus{
						Conditions: []metav1.Condition{
							{
								Type:    "NotReady", // can not be found
								Status:  metav1.ConditionTrue,
								Message: "The release is ready",
							},
						},
					},
				},
			},
			expected: juggler.ResourceHealthiness{
				Healthy: false,
				Message: msgReadyNotPresent,
			},
		},
		{
			name: "OCIRepositoryAdapter - Status Condition Ready = True",
			adapter: OCIRepositoryAdapter{
				Source: &sourcev1.OCIRepository{
					Status: sourcev1.OCIRepositoryStatus{
						Conditions: []metav1.Condition{
							{
								Type:    fluxmeta.ReadyCondition,
								Status:  metav1.ConditionTrue,
								Message: "The release is ready",
							},
						},
					},
				},
			},
			expected: juggler.ResourceHealthiness{
				Healthy: true,
				Message: "The release is ready",
			},
		},
		{
			name: "OCIRepositoryAdapter - Status Condition Ready = False",
			adapter: OCIRepositoryAdapter{
				Source: &sourcev1.OCIRepository{
					Status: sourcev1.OCIRepositoryStatus{
						Conditions: []metav1.Condition{
							{
								Type:    fluxmeta.ReadyCondition,
								Status:  metav1.ConditionFalse,
								Message: "The release is not ready",
							},
						},
					},
				},
			},
			expected: juggler.ResourceHealthiness{
				Healthy: false,
				Message: "The release is not ready",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := tt.adapter.GetHealthiness()
			if !assert.Equal(t, tt.expected, actual) {
				t.Errorf("OCIRepositoryAdapter.GetHealthiness() = %v, want %v", actual, tt.expected)
			}
		})
	}
}
