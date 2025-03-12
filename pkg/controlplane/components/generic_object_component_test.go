package components

import (
	"context"
	"testing"

	"github.com/openmcp-project/control-plane-operator/pkg/juggler"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func genComponent(enabled, healthy bool, reconcileErr error) *GenericObjectComponent {
	return &GenericObjectComponent{
		NamespacedName: types.NamespacedName{
			Name:      "example",
			Namespace: "some-namespace",
		},
		Enabled: enabled,
		Type:    &corev1.Secret{},
		IsObjectHealthyFunc: func(obj client.Object) juggler.ResourceHealthiness {
			return juggler.ResourceHealthiness{Healthy: healthy}
		},
		ReconcileObjectFunc: func(ctx context.Context, obj client.Object) error {
			return reconcileErr
		},
	}
}

func Test_GenericObjectComponent(t *testing.T) {
	testCases := []struct {
		desc            string
		comp            *GenericObjectComponent
		validationFuncs []validationFunc
	}{
		{
			desc: "should be disabled",
			comp: genComponent(false, true, nil),
			validationFuncs: []validationFunc{
				hasName("SecretExample"),
				isEnabled(false),
			},
		},
		{
			desc: "should be enabled",
			comp: genComponent(true, true, nil),
			validationFuncs: []validationFunc{
				hasName("SecretExample"),
				isEnabled(true),
				isAllowed(true),
				hasDependencies(0),
				hasNoHooks(),
				isTargetComponent(
					hasNamespace("some-namespace"),
				),
				isObjectComponent(
					objectIsType(&corev1.Secret{}),
					canBuildAndReconcile(nil),
					canCheckHealthiness(nil, juggler.ResourceHealthiness{
						Healthy: true,
					}),
				),
			},
		},
		{
			desc: "should fail to reconcile and is unhealthy",
			comp: genComponent(true, false, errFake),
			validationFuncs: []validationFunc{
				hasName("SecretExample"),
				isObjectComponent(
					canBuildAndReconcile(errFake),
					canCheckHealthiness(nil, juggler.ResourceHealthiness{
						Healthy: false,
					}),
				),
			},
		},
		{
			desc: "should use name override",
			comp: &GenericObjectComponent{
				NamespacedName: types.NamespacedName{
					Name: "example",
				},
				Type:         &corev1.Secret{},
				NameOverride: "SomeCustomName",
			},
			validationFuncs: []validationFunc{
				hasName("SomeCustomName"),
			},
		},
		{
			desc: "should use type name override",
			comp: &GenericObjectComponent{
				NamespacedName: types.NamespacedName{
					Name: "example",
				},
				Type:             &corev1.Secret{},
				TypeNameOverride: "SomeCustomType",
			},
			validationFuncs: []validationFunc{
				hasName("SomeCustomTypeExample"),
			},
		},
	}
	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			ctx := newContext(nil, nil)
			for _, vfn := range tC.validationFuncs {
				vfn(t, ctx, tC.comp)
			}
		})
	}
}
