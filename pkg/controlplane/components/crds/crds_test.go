package crds

import (
	"embed"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"

	"github.com/openmcp-project/control-plane-operator/pkg/juggler"
)

var (
	//go:embed testdata
	crdFiles embed.FS
)

func Test_readAllCRDs(t *testing.T) {
	crds, err := readAllCRDs(crdFiles)
	assert.NoError(t, err)
	assert.Len(t, crds, 2)
	assert.Equal(t, "ControlPlane", crds[0].Spec.Names.Kind)
	assert.Equal(t, "CrossplanePackageRestriction", crds[1].Spec.Names.Kind)
}

func Test_RegisterAsComponents(t *testing.T) {
	testCases := []struct {
		desc               string
		names              []string
		expectedErr        error
		expectedComponents int
	}{
		{
			desc:               "should register CrossplanePackageRestriction",
			names:              []string{"crossplanepackagerestrictions.core.orchestrate.cloud.sap"},
			expectedComponents: 1,
			expectedErr:        nil,
		},
		{
			desc:               "should not register anything",
			names:              []string{"doesnotexist"},
			expectedComponents: 0,
			expectedErr:        nil,
		},
	}
	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			j := juggler.NewJuggler(logr.Logger{}, nil)
			actualErr := RegisterAsComponents(j, crdFiles, true, tC.names...)
			assert.Equal(t, tC.expectedErr, actualErr)
			assert.Equal(t, tC.expectedComponents, j.RegisteredComponents())
		})
	}
}
