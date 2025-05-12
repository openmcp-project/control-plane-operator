//nolint:dupl
package fluxcd

import (
	"testing"

	helmv2 "github.com/fluxcd/helm-controller/api/v2"
	fluxmeta "github.com/fluxcd/pkg/apis/meta"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/openmcp-project/control-plane-operator/pkg/juggler"
)

func TestHelmReleaseManifesto_GetHealthiness(t *testing.T) {
	tests := []struct {
		name      string
		manifesto HelmReleaseManifesto
		expected  juggler.ResourceHealthiness
	}{
		{
			name: "HelmReleaseManifesto - Status Condition nil - Ready condition not present",
			manifesto: HelmReleaseManifesto{
				Manifest: &helmv2.HelmRelease{
					Status: helmv2.HelmReleaseStatus{
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
			name: "HelmReleaseManifesto - Status Condition Ready not found",
			manifesto: HelmReleaseManifesto{
				Manifest: &helmv2.HelmRelease{
					Status: helmv2.HelmReleaseStatus{
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
			name: "HelmReleaseManifesto - Status Condition Ready = True",
			manifesto: HelmReleaseManifesto{
				Manifest: &helmv2.HelmRelease{
					Status: helmv2.HelmReleaseStatus{
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
			name: "HelmReleaseManifesto - Status Condition Ready = False",
			manifesto: HelmReleaseManifesto{
				Manifest: &helmv2.HelmRelease{
					Status: helmv2.HelmReleaseStatus{
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
			actual := tt.manifesto.GetHealthiness()
			if !assert.Equal(t, tt.expected, actual) {
				t.Errorf("HelmReleaseManifesto.GetHealthiness() = %v, want %v", actual, tt.expected)
			}
		})
	}
}
