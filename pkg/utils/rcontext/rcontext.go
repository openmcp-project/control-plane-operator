package rcontext

import (
	"context"

	"github.com/fluxcd/pkg/apis/meta"
	"github.com/openmcp-project/control-plane-operator/api/v1beta1"
	"github.com/openmcp-project/control-plane-operator/pkg/controlplane/secretresolver"
	corev1 "k8s.io/api/core/v1"
)

type tenantNamespaceKey struct{}

func WithTenantNamespace(ctx context.Context, namespace string) context.Context {
	return context.WithValue(ctx, tenantNamespaceKey{}, namespace)
}

func TenantNamespace(ctx context.Context) string {
	return ctx.Value(tenantNamespaceKey{}).(string)
}

//
// -----------------------
//

type fluxKubeconfigKey struct{}

func WithFluxKubeconfigRef(ctx context.Context, ref *corev1.SecretReference) context.Context {
	return context.WithValue(ctx, fluxKubeconfigKey{}, &meta.KubeConfigReference{
		SecretRef: meta.SecretKeyReference{
			Name: ref.Name,
			Key:  "kubeconfig",
		},
	})
}

func FluxKubeconfigRef(ctx context.Context) *meta.KubeConfigReference {
	return ctx.Value(fluxKubeconfigKey{}).(*meta.KubeConfigReference)
}

//
// -----------------------
//

type versionResolverFnKey struct{}

func WithVersionResolver(ctx context.Context, fn v1beta1.VersionResolverFn) context.Context {
	return context.WithValue(ctx, versionResolverFnKey{}, fn)
}

func VersionResolver(ctx context.Context) v1beta1.VersionResolverFn {
	return ctx.Value(versionResolverFnKey{}).(v1beta1.VersionResolverFn)
}

//
// -----------------------
//

type secretRefResolverFnKey struct{}

func WithSecretRefResolver(ctx context.Context, fn secretresolver.ResolveFunc) context.Context {
	return context.WithValue(ctx, secretRefResolverFnKey{}, fn)
}

func SecretRefResolver(ctx context.Context) secretresolver.ResolveFunc {
	return ctx.Value(secretRefResolverFnKey{}).(secretresolver.ResolveFunc)
}
