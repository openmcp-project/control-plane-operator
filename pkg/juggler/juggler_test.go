package juggler

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/openmcp-project/control-plane-operator/api/v1beta1"
	"k8s.io/client-go/tools/record"

	"github.com/go-logr/logr"
	"github.com/go-logr/logr/testr"
	"github.com/stretchr/testify/assert"
)

var errBoom = errors.New("boom")

// Test for function Reconcile
func TestJuggler_Reconcile(t *testing.T) {
	type fields struct {
		components  []Component
		reconcilers []ComponentReconciler
	}
	tests := []struct {
		name   string
		fields fields
		want   []ComponentResult
	}{
		{
			name: "Success, one component, one result",
			fields: fields{
				components: []Component{FakeComponent{Enabled: true, Allowed: true}},
				reconcilers: []ComponentReconciler{
					FakeReconciler{
						KnownTypesFunc: knowsAll(),
						ObserverFunc: func(ctx context.Context, component Component) (ComponentObservation, error) {
							return ComponentObservation{
								ResourceExists: true,
								ResourceHealthiness: ResourceHealthiness{
									Healthy: true,
								},
							}, nil
						},
					}},
			},
			want: []ComponentResult{
				{
					Component: FakeComponent{Enabled: true, Allowed: true},
					Result:    StatusHealthy,
					Message:   "FakeComponent is healthy.",
				},
			},
		},
		{
			name: "Component is not healthy",
			fields: fields{
				components: []Component{FakeComponent{Enabled: true, Allowed: true}},
				reconcilers: []ComponentReconciler{
					FakeReconciler{
						KnownTypesFunc: knowsAll(),
						ObserverFunc: func(ctx context.Context, component Component) (ComponentObservation, error) {
							return ComponentObservation{
								ResourceExists: true,
								ResourceHealthiness: ResourceHealthiness{
									Healthy: false,
									Message: "not healthy",
								},
							}, nil
						},
					},
				},
			},
			want: []ComponentResult{
				{
					Component: FakeComponent{Enabled: true, Allowed: true},
					Result:    StatusUnhealthy,
					Message:   "not healthy",
				},
			},
		},
		{
			name: "error - too many reconcilers for one type",
			fields: fields{
				components: []Component{FakeComponent{Enabled: true, Allowed: true}},
				reconcilers: []ComponentReconciler{
					FakeReconciler{
						KnownTypesFunc: knowsAll(),
					},
					FakeReconciler{
						KnownTypesFunc: knowsAll(),
					}},
			},
			want: []ComponentResult{
				{
					Component: FakeComponent{Enabled: true, Allowed: true},
					Result:    StatusReconcilerNotFound,
					Message:   errTooManyReconcilers.Error(),
				},
			},
		},
		{
			name: "error, no reconciler found",
			fields: fields{
				components: []Component{
					FakeComponent{Enabled: true, Allowed: true},
				},
				reconcilers: []ComponentReconciler{
					FakeReconciler{
						KnownTypesFunc: func() []reflect.Type { return []reflect.Type{} },
					}},
			},
			want: []ComponentResult{
				{
					Component: FakeComponent{Enabled: true, Allowed: true},
					Result:    StatusReconcilerNotFound,
					Message:   errNoReconcilerForComponentType.Error(),
				},
			},
		},
		{
			name: "error, dependency not registered",
			fields: fields{
				components: []Component{
					FakeComponent{Enabled: true, Dependencies: []Component{FakeComponent2{}}, Allowed: true},
				},
				reconcilers: []ComponentReconciler{
					FakeReconciler{
						KnownTypesFunc: knowsAll(),
						ObserverFunc: func(ctx context.Context, component Component) (ComponentObservation, error) {
							return ComponentObservation{ResourceExists: false}, nil
						},
					}},
			},
			want: []ComponentResult{
				{
					Component: FakeComponent{Enabled: true, Dependencies: []Component{FakeComponent2{}}, Allowed: true},
					Result:    StatusDependencyCheckFailed,
					Message:   errDependencyNotRegistered.Error(),
				},
			},
		},
		{
			name: "error, dependency not enabled",
			fields: fields{
				components: []Component{
					FakeComponent{Enabled: true, Dependencies: []Component{FakeComponent2{}}, Allowed: true},
					FakeComponent2{Enabled: false, Allowed: true},
				},
				reconcilers: []ComponentReconciler{
					FakeReconciler{
						KnownTypesFunc: knowsAll(),
						ObserverFunc: func(ctx context.Context, component Component) (ComponentObservation, error) {
							return ComponentObservation{ResourceExists: false}, nil
						},
					}},
			},
			want: []ComponentResult{
				{
					Component: FakeComponent{Enabled: true, Dependencies: []Component{FakeComponent2{}}, Allowed: true},
					Result:    StatusDependencyCheckFailed,
					Message:   errDependencyNotEnabled.Error(),
				},
				{
					Component: FakeComponent2{Enabled: false, Allowed: true},
					Result:    StatusDisabled,
					Message:   "FakeComponent2 is not enabled.",
				},
			},
		},
		{
			name: "Success - dependencies satisfied",
			fields: fields{
				components: []Component{
					FakeComponent{Enabled: true, Dependencies: []Component{FakeComponent2{}}, Allowed: true},
					FakeComponent2{Enabled: true, Allowed: true},
				},
				reconcilers: []ComponentReconciler{
					FakeReconciler{
						KnownTypesFunc: knowsAll(),
						ObserverFunc: func(ctx context.Context, component Component) (ComponentObservation, error) {
							return ComponentObservation{
								ResourceExists: true,
								ResourceHealthiness: ResourceHealthiness{
									Healthy: true,
								},
							}, nil
						},
					}},
			},
			want: []ComponentResult{
				{
					Component: FakeComponent{Enabled: true, Dependencies: []Component{FakeComponent2{}}, Allowed: true},
					Result:    StatusHealthy,
					Message:   "FakeComponent is healthy.",
				},
				{
					Component: FakeComponent2{Enabled: true, Allowed: true},
					Result:    StatusHealthy,
					Message:   "FakeComponent2 is healthy.",
				},
			},
		},
		{
			name: "error, component is not allowed to be installed",
			fields: fields{
				components: []Component{
					FakeComponent{Enabled: true, Allowed: false},
				},
				reconcilers: []ComponentReconciler{
					FakeReconciler{
						KnownTypesFunc: knowsAll(),
						ObserverFunc: func(ctx context.Context, component Component) (ComponentObservation, error) {
							return ComponentObservation{ResourceExists: false}, nil
						},
					},
				},
			},
			want: []ComponentResult{
				{
					Component: FakeComponent{Enabled: true, Allowed: false},
					Result:    StatusComponentNotAllowed,
					Message:   "FakeComponent not installable.",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cp := v1beta1.ControlPlane{}
			am := NewJuggler(testr.New(t), &ObjectEventRecorder{
				recorder: record.NewFakeRecorder(3),
				object:   &cp,
			})
			am.RegisterComponent(tt.fields.components...)
			am.RegisterReconciler(tt.fields.reconcilers...)
			result := am.Reconcile(context.TODO())
			assert.Equal(t, tt.want, result)
		})
	}
}

func knowsAll() func() []reflect.Type {
	return func() []reflect.Type {
		return []reflect.Type{
			reflect.TypeOf(FakeComponent{}),
			reflect.TypeOf(FakeComponent2{}),
		}
	}
}

// Test for function RegisterComponent
func TestJuggler_RegisterComponent(t *testing.T) {
	componentList1 := []Component{
		FakeComponent{
			Name:         "FakeComponent1",
			Dependencies: nil,
			Enabled:      true,
		},
	}

	componentList2 := []Component{
		FakeComponent{
			Name:         "FakeComponent1",
			Dependencies: nil,
			Enabled:      true,
		},
		FakeComponent{
			Name:         "FakeComponent2",
			Dependencies: nil,
			Enabled:      false,
		},
	}

	tests := []struct {
		name   string
		fields []Component
		want   []Component
	}{
		{name: "Register one component", fields: componentList1, want: componentList1},
		{name: "Register more components", fields: componentList2, want: componentList2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cp := v1beta1.ControlPlane{}
			am := NewJuggler(testr.New(t), &ObjectEventRecorder{
				recorder: record.NewFakeRecorder(3),
				object:   &cp,
			})
			am.RegisterComponent(tt.fields...)
			assert.Equal(t, tt.want, am.components)
		})
	}
}

func TestJuggler_reconcileComponent(t *testing.T) {
	type args struct {
		component  Component
		reconciler ComponentReconciler
	}
	tests := []struct {
		name string
		args args
		want ComponentResult
	}{
		{
			name: "error Observing",
			args: args{
				component: FakeComponent{Enabled: true, Allowed: true},
				reconciler: FakeReconciler{
					KnownTypesFunc: knowsAll(),
					ObserverFunc: func(ctx context.Context, component Component) (ComponentObservation, error) {
						return ComponentObservation{}, errBoom
					},
				},
			},
			want: ComponentResult{
				Component: FakeComponent{Enabled: true, Allowed: true},
				Result:    StatusObservationFailed,
				Message:   errBoom.Error(),
			},
		},
		{
			name: "needs uninstall, uninstall error", args: args{
				component: FakeComponent{Enabled: false, Allowed: true},
				reconciler: FakeReconciler{
					KnownTypesFunc: knowsAll(),
					ObserverFunc: func(ctx context.Context, component Component) (ComponentObservation, error) {
						return ComponentObservation{
							ResourceExists: true,
						}, nil
					},
					UninstallFunc: func(ctx context.Context, component Component) error {
						return errBoom
					},
					PreUninstallFunc: func(ctx context.Context, component Component) error { return nil },
				},
			},
			want: ComponentResult{
				Component: FakeComponent{Enabled: false, Allowed: true},
				Result:    StatusUninstallFailed,
				Message:   errBoom.Error(),
			},
		},
		{
			name: "needs uninstall, pre-uninstall hook failed", args: args{
				component: FakeComponent{Enabled: false, Allowed: true},
				reconciler: FakeReconciler{
					KnownTypesFunc: knowsAll(),
					ObserverFunc: func(ctx context.Context, component Component) (ComponentObservation, error) {
						return ComponentObservation{
							ResourceExists: true,
						}, nil
					},
					PreUninstallFunc: func(ctx context.Context, component Component) error { return errBoom },
				},
			},
			want: ComponentResult{
				Component: FakeComponent{Enabled: false, Allowed: true},
				Result:    StatusUninstallFailed,
				Message:   errors.Join(errHookFailed, errBoom).Error(),
			},
		},
		{
			name: "need uninstall, uninstall ok", args: args{
				component: FakeComponent{Enabled: false, Allowed: true},
				reconciler: FakeReconciler{
					KnownTypesFunc: knowsAll(),
					ObserverFunc: func(ctx context.Context, component Component) (ComponentObservation, error) {
						return ComponentObservation{ResourceExists: true}, nil
					},
					UninstallFunc: func(ctx context.Context, component Component) error {
						return nil
					},
					PreUninstallFunc: func(ctx context.Context, component Component) error { return nil },
				},
			},
			want: ComponentResult{
				Component: FakeComponent{Enabled: false, Allowed: true},
				Result:    StatusUninstalled,
				Message:   "FakeComponent has been uninstalled successfully.",
			},
		},
		{
			name: "need uninstall, keep on uninstall enabled", args: args{
				component: FakeComponent{Enabled: false, Allowed: true, KeepInstalled: true},
				reconciler: FakeReconciler{
					KnownTypesFunc: knowsAll(),
					ObserverFunc: func(ctx context.Context, component Component) (ComponentObservation, error) {
						return ComponentObservation{ResourceExists: true}, nil
					},
					UninstallFunc: func(ctx context.Context, component Component) error {
						return nil
					},
					PreUninstallFunc: func(ctx context.Context, component Component) error { return nil },
				},
			},
			want: ComponentResult{
				Component: FakeComponent{Enabled: false, Allowed: true, KeepInstalled: true},
				Result:    StatusUninstalled,
				Message:   "FakeComponent is marked as 'keep on uninstall'.",
			},
		},
		{
			name: "error - component is not allowed to be installed",
			args: args{
				component: FakeComponent{Enabled: true, Allowed: false},
				reconciler: FakeReconciler{
					KnownTypesFunc: knowsAll(),
					ObserverFunc: func(ctx context.Context, component Component) (ComponentObservation, error) {
						return ComponentObservation{ResourceExists: false}, nil
					},
				},
			},
			want: ComponentResult{
				Component: FakeComponent{Enabled: true, Allowed: false},
				Result:    StatusComponentNotAllowed,
				Message:   "FakeComponent not installable.",
			},
		},
		{
			name: "needs install, install error", args: args{
				component: FakeComponent{Enabled: true, Allowed: true},
				reconciler: FakeReconciler{
					KnownTypesFunc: knowsAll(),
					ObserverFunc: func(ctx context.Context, component Component) (ComponentObservation, error) {
						return ComponentObservation{ResourceExists: false}, nil
					},
					InstallFunc: func(ctx context.Context, component Component) error {
						return errBoom
					},
				},
			},
			want: ComponentResult{
				Component: FakeComponent{Enabled: true, Allowed: true},
				Result:    StatusInstallFailed,
				Message:   errBoom.Error(),
			},
		},
		{
			name: "needs install, pre-install hook failed", args: args{
				component: FakeComponent{Enabled: true, Allowed: true},
				reconciler: FakeReconciler{
					KnownTypesFunc: knowsAll(),
					ObserverFunc: func(ctx context.Context, component Component) (ComponentObservation, error) {
						return ComponentObservation{ResourceExists: false}, nil
					},
					PreInstallFunc: func(ctx context.Context, component Component) error {
						return errBoom
					},
				},
			},
			want: ComponentResult{
				Component: FakeComponent{Enabled: true, Allowed: true},
				Result:    StatusInstallFailed,
				Message:   errors.Join(errHookFailed, errBoom).Error(),
			},
		},
		{
			name: "needs update, update error", args: args{
				component: FakeComponent{Enabled: true, Allowed: true},
				reconciler: FakeReconciler{
					KnownTypesFunc: knowsAll(),
					ObserverFunc: func(ctx context.Context, component Component) (ComponentObservation, error) {
						return ComponentObservation{ResourceExists: true}, nil
					},
					UpdateFunc: func(ctx context.Context, component Component) error {
						return errBoom
					},
				},
			},
			want: ComponentResult{
				Component: FakeComponent{Enabled: true, Allowed: true},
				Result:    StatusUpdateFailed,
				Message:   errBoom.Error(),
			},
		},
		{
			name: "needs update, pre-update hook failed", args: args{
				component: FakeComponent{Enabled: true, Allowed: true},
				reconciler: FakeReconciler{
					KnownTypesFunc: knowsAll(),
					ObserverFunc: func(ctx context.Context, component Component) (ComponentObservation, error) {
						return ComponentObservation{ResourceExists: true}, nil
					},
					PreUpdateFunc: func(ctx context.Context, component Component) error {
						return errBoom
					},
				},
			},
			want: ComponentResult{
				Component: FakeComponent{Enabled: true, Allowed: true},
				Result:    StatusUpdateFailed,
				Message:   errors.Join(errHookFailed, errBoom).Error(),
			},
		},
		{
			name: "is disabled, no-op", args: args{
				component: FakeComponent{Enabled: false, Allowed: true},
				reconciler: FakeReconciler{
					KnownTypesFunc: knowsAll(),
					ObserverFunc: func(ctx context.Context, component Component) (ComponentObservation, error) {
						return ComponentObservation{
							ResourceExists: false,
						}, nil
					},
				},
			},
			want: ComponentResult{
				Component: FakeComponent{Enabled: false, Allowed: true},
				Result:    StatusDisabled,
				Message:   "FakeComponent is not enabled.",
			},
		},
		{
			name: "is not healthy",
			args: args{
				component: FakeComponent{Enabled: true, Allowed: true},
				reconciler: FakeReconciler{
					KnownTypesFunc: knowsAll(),
					ObserverFunc: func(ctx context.Context, component Component) (ComponentObservation, error) {
						return ComponentObservation{
							ResourceExists: true,
							ResourceHealthiness: ResourceHealthiness{
								Healthy: false,
								Message: "not healthy",
							},
						}, nil
					},
				},
			},
			want: ComponentResult{
				Component: FakeComponent{Enabled: true, Allowed: true},
				Result:    StatusUnhealthy,
				Message:   "not healthy",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cp := v1beta1.ControlPlane{}
			am := NewJuggler(testr.New(t), &ObjectEventRecorder{
				recorder: record.NewFakeRecorder(3),
				object:   &cp,
			})
			am.RegisterReconciler(tt.args.reconciler)
			result := am.reconcileComponent(context.TODO(), tt.args.component)
			assert.Equal(t, tt.want, result)
		})
	}
}

func Test_Juggler_RegisterOrphanedComponents(t *testing.T) {
	juggler := NewJuggler(logr.Logger{}, nil)

	r1 := FakeReconciler{
		KnownTypesFunc: func() []reflect.Type { return []reflect.Type{reflect.TypeOf(FakeComponent{})} },
		DetectOrphanedComponentsFunc: func(_ context.Context, configuredComponents []Component) ([]Component, error) {
			assert.Len(t, configuredComponents, 1)
			return []Component{FakeComponent{}}, nil
		},
	}

	r2 := FakeReconciler{
		KnownTypesFunc: func() []reflect.Type { return []reflect.Type{reflect.TypeOf(FakeComponent2{})} },
		DetectOrphanedComponentsFunc: func(_ context.Context, configuredComponents []Component) ([]Component, error) {
			assert.Len(t, configuredComponents, 2)
			return []Component{FakeComponent2{}, FakeComponent2{}}, nil
		},
	}

	juggler.RegisterComponent(FakeComponent{}, FakeComponent2{}, FakeComponent2{})
	juggler.RegisterReconciler(r1, r2)

	err := juggler.RegisterOrphanedComponents(context.Background())
	assert.NoError(t, err)
	assert.Len(t, juggler.components, 6)
}
