package envtest

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	k8senvtest "sigs.k8s.io/controller-runtime/pkg/envtest"
)

func Test_findMakefile(t *testing.T) {
	actual, err := findMakefile(".")
	assert.NoError(t, err)

	expected, err := filepath.Abs("../../../Makefile")
	assert.NoError(t, err)
	assert.Equal(t, expected, actual)
}

func Test_Install(t *testing.T) {
	assert.NoError(t, Install())
	testEnv := &k8senvtest.Environment{}
	_, err := testEnv.Start()
	assert.NoError(t, err)
	assert.NoError(t, testEnv.Stop())
}
