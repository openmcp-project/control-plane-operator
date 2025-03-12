package clusterroles

import (
	"testing"

	"github.com/go-logr/logr"
	"github.com/openmcp-project/control-plane-operator/pkg/juggler"
	"github.com/stretchr/testify/assert"
)

func Test_RegisterAsComponents(t *testing.T) {
	j := juggler.NewJuggler(logr.Logger{}, nil)
	RegisterAsComponents(j, []juggler.Component{}, true)
	assert.Equal(t, 2, j.RegisteredComponents())
}
