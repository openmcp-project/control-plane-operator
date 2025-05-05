package envtest

import (
	"testing"

	"github.com/stretchr/testify/assert"
	k8senvtest "sigs.k8s.io/controller-runtime/pkg/envtest"
)

func Test_Install(t *testing.T) {
	testEnv := &k8senvtest.Environment{}
	_, err := testEnv.Start()
	assert.NoError(t, err)
	assert.NoError(t, testEnv.Stop())
}
