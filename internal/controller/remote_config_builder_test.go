package controller

import (
	"testing"

	"github.com/openmcp-project/control-plane-operator/api/v1beta1"
	"github.com/openmcp-project/controller-utils/pkg/api"
	"github.com/openmcp-project/controller-utils/pkg/clientconfig"
	"github.com/stretchr/testify/assert"
	"k8s.io/client-go/rest"
)

func TestNewRemoteConfigBuilder(t *testing.T) {
	fn := NewRemoteConfigBuilder()
	assert.NotNil(t, fn)
	testCases := []struct {
		name               string
		target             v1beta1.Target
		expectedError      error
		validateReloadFunc func(t *testing.T, reloadFunc clientconfig.ReloadFunc) error
		validateTarget     func(t *testing.T, config *rest.Config) error
	}{
		{
			name: "not valid target",
			target: v1beta1.Target{
				Target: api.Target{
					Kubeconfig:     nil,
					KubeconfigRef:  nil,
					ServiceAccount: nil,
				},
			},
			expectedError: clientconfig.ErrInvalidConnectionMethod,
			validateTarget: func(t *testing.T, config *rest.Config) error {
				assert.Nil(t, config)
				return nil
			},
			validateReloadFunc: func(t *testing.T, reloadFunc clientconfig.ReloadFunc) error {
				assert.Nil(t, reloadFunc)
				return nil
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			target, reloadFunc, err := fn(tc.target)
			assert.Equal(t, tc.expectedError, err)
			if tc.validateTarget != nil {
				err := tc.validateTarget(t, target)
				assert.NoErrorf(t, err, "validation failed unexpectedly")
			}
			if tc.validateReloadFunc != nil {
				err := tc.validateReloadFunc(t, reloadFunc)
				assert.NoErrorf(t, err, "validation failed unexpectedly")
			}
		})
	}
}
