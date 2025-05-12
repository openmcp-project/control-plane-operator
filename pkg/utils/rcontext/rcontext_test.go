package rcontext

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/net/context"
	corev1 "k8s.io/api/core/v1"

	"github.com/openmcp-project/control-plane-operator/api/v1beta1"
	"github.com/openmcp-project/control-plane-operator/pkg/controlplane/secretresolver"
)

func TestTenantNamespace(t *testing.T) {
	val := "some-namespace"
	ctx := WithTenantNamespace(context.TODO(), val)
	actual := TenantNamespace(ctx)
	assert.Equal(t, val, actual)
}

func TestFluxKubeconfigRef(t *testing.T) {
	val := &corev1.SecretReference{Name: "some-secret", Namespace: "some-namespace"}
	ctx := WithFluxKubeconfigRef(context.TODO(), val)
	actual := FluxKubeconfigRef(ctx)
	assert.Equal(t, val.Name, actual.SecretRef.Name)
	assert.Equal(t, "kubeconfig", actual.SecretRef.Key)
}

func TestWithVersionResolver(t *testing.T) {
	fn := func(componentName string, channelName string) (v1beta1.ComponentVersion, error) {
		return v1beta1.ComponentVersion{}, nil
	}
	ctx := WithVersionResolver(context.TODO(), fn)
	actual := VersionResolver(ctx)
	if reflect.ValueOf(actual).Pointer() != reflect.ValueOf(fn).Pointer() {
		t.Error("Functions are not equal")
	}
}

func TestSecretRefResolver(t *testing.T) {
	fn := func(urlType secretresolver.UrlSecretType) (*corev1.LocalObjectReference, error) {
		return nil, nil
	}
	ctx := WithSecretRefResolver(context.TODO(), fn)
	actual := SecretRefResolver(ctx)
	if reflect.ValueOf(actual).Pointer() != reflect.ValueOf(fn).Pointer() {
		t.Error("Functions are not equal")
	}
}
