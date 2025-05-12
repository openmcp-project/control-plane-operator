package policies

import (
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"

	"github.com/openmcp-project/control-plane-operator/pkg/juggler"
)

func Test_RegisterAsComponents(t *testing.T) {
	j := juggler.NewJuggler(logr.Logger{}, nil)
	err := RegisterAsComponents(j, nil, true)
	assert.NoError(t, err)
	// 1 * CrossplanePackageRestriction + (3 PackageTypes * 2 GenericObjectComponent) = 7
	assert.Equal(t, 7, j.RegisteredComponents())
}

func TestRegisterDeploymentRuntimeConfigProtection(t *testing.T) {
	j := juggler.NewJuggler(logr.Logger{}, nil)
	err := RegisterDeploymentRuntimeConfigProtection(j, nil, true)
	assert.NoError(t, err)
	// Policy + Policybinding = 2 in total
	assert.Equal(t, 2, j.RegisteredComponents())
}
