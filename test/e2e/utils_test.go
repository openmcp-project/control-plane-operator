//go:build e2e

package e2e

import (
	"context"
	"fmt"
	"testing"
	"time"

	xpres "github.com/crossplane-contrib/xp-testing/pkg/resources"
	"github.com/openmcp-project/control-plane-operator/api/v1beta1"
	"github.com/openmcp-project/control-plane-operator/pkg/juggler"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/e2e-framework/klient/k8s"
	res "sigs.k8s.io/e2e-framework/klient/k8s/resources"
	"sigs.k8s.io/e2e-framework/klient/wait"
	"sigs.k8s.io/e2e-framework/klient/wait/conditions"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
)

// isControlPlaneReady validates that a v1beta1.ControlPlane was created and is ready
func isControlPlaneReady(o *v1beta1.ControlPlane) bool {
	return o.CreationTimestamp.Size() > 0 && meta.IsStatusConditionTrue(o.Status.Conditions, "Ready")
}

// IsStatusDeploymentConditionPresentAndEqual checks if a DeploymentCondition is present and has the given status
func IsStatusDeploymentConditionPresentAndEqual(conditions []v1.DeploymentCondition, conditionType string, status corev1.ConditionStatus) bool {
	for _, condition := range conditions {
		if string(condition.Type) == conditionType {
			return condition.Status == status
		}
	}
	return false
}

// SetUpControlPlaneResources imports the ControlPlane resources
func SetUpControlPlaneResources(path, cpName string) func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
	return func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
		xpres.ImportResources(ctx, t, cfg, path)
		_ = getClientFor(cfg)
		waitForControlPlaneResource(cfg, t, cpName)
		return ctx
	}
}

// waitForControlPlaneResource waits for the ControlPlane to be created
func waitForControlPlaneResource(cfg *envconf.Config, t *testing.T, cpName string) {
	client := cfg.Client()

	// create new Control Plane
	cp := newControlPlaneResource(cfg, cpName)
	err := wait.For(conditions.New(client.Resources()).ResourceMatch(cp, func(object k8s.Object) bool {
		d := object.(*v1beta1.ControlPlane)
		return d.CreationTimestamp.Size() > 0
	}))

	if err != nil {
		t.Error(err)
	}
}

// newControlPlaneResource creates a new v1beta1.ControlPlane resource with a given name
func newControlPlaneResource(cfg *envconf.Config, cpName string) *v1beta1.ControlPlane {
	return &v1beta1.ControlPlane{
		ObjectMeta: metav1.ObjectMeta{
			Name: cpName, Namespace: cfg.Namespace(),
		},
	}
}

// UpdateControlPlaneOrError updates a v1beta1.ControlPlane resource
func UpdateControlPlaneOrError(ctx context.Context, t *testing.T, cfg *envconf.Config, updateCP *v1beta1.ControlPlane) {
	client := cfg.Client()
	r, _ := res.New(client.RESTConfig())

	// update the ControlPlane resource
	err := r.Update(ctx, updateCP)
	if err != nil {
		t.Fatal(err)
	}
}

// GetControlPlaneOrError returns the v1beta1.ControlPlane under the given cpName
func GetControlPlaneOrError(t *testing.T, cfg *envconf.Config, cpName string) *v1beta1.ControlPlane {
	ct := &v1beta1.ControlPlane{}
	namespace := cfg.Namespace()
	res := cfg.Client().Resources()

	err := res.Get(context.TODO(), cpName, namespace, ct)
	if err != nil {
		t.Error("failed to get ControlPlane resource. error: ", err)
	}
	return ct
}

// UpdateControlPlaneSpec updates the Control Plane spec with the given v1beta1.ComponentsConfig
func UpdateControlPlaneSpec(cp *v1beta1.ControlPlane, config *v1beta1.ComponentsConfig) *v1beta1.ControlPlane {
	updateCP := cp.DeepCopy()
	updateCP.Spec.ComponentsConfig = *config
	return updateCP
}

// WaitForComponentStatusToBeHealthy waits for the given components to be healthy
func WaitForComponentStatusToBeHealthy(t *testing.T, cfg *envconf.Config, cpName string, componentNames ...string) {

	cpObserved := GetControlPlaneOrError(t, cfg, cpName)
	err := wait.For(conditions.New(cfg.Client().Resources()).ResourceMatch(cpObserved, func(object k8s.Object) bool {
		return checkControlPlaneComponentHealthiness(cpObserved, componentNames...)
	}), wait.WithTimeout(time.Minute*10))

	if err != nil {
		fmt.Println("Control Plane ", cpName, " components are not healthy:", componentNames)
		t.Error(err)
	}
}

// checkControlPlaneComponentHealthiness checks if the given components are healthy
func checkControlPlaneComponentHealthiness(cp *v1beta1.ControlPlane, componentNames ...string) bool {
	if componentNames == nil {
		return false
	}

	for _, c := range cp.Status.Conditions {
		for _, componentName := range componentNames {
			if c.Reason == juggler.StatusUnhealthy.Name && c.Type == componentName+"Ready" {
				return false
			}
		}
	}
	return true
}

// getClientFor returns a Resources client that can be used to interact with the API. It also adds the v1beta1 scheme to the client.
func getClientFor(config *envconf.Config) *res.Resources {
	_ = v1beta1.AddToScheme(config.Client().Resources().GetScheme())
	return config.Client().Resources()
}
