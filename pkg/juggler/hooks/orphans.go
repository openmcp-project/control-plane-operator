package hooks

import (
	"context"
	"fmt"

	"github.com/openmcp-project/control-plane-operator/pkg/utils"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// checkForResources checks if there are any resources of the given GroupVersionKind remaining in the cluster
func checkForResources(gvk schema.GroupVersionKind, ctx context.Context, c client.Client) error {
	list := &unstructured.UnstructuredList{}
	list.SetGroupVersionKind(gvk)

	err := c.List(ctx, list, client.Limit(1))
	if utils.IsCRDNotFound(err) {
		// CRD not found, so no resources can exist.
		return nil
	}
	if err != nil {
		return err
	}

	if len(list.Items) > 0 {
		return fmt.Errorf("cannot uninstall because there is a least one object of %s remaining", gvk)
	}

	return nil
}

// PreventOrphanedResources can be used as a pre-uninstall hook to prevent CRDs from being deleted
// before any corresponding resources are deleted first.
func PreventOrphanedResources(gvks []schema.GroupVersionKind) func(ctx context.Context, c client.Client) error {
	return func(ctx context.Context, c client.Client) error {
		for _, gvk := range gvks {
			if err := checkForResources(gvk, ctx, c); err != nil {
				return err
			}
		}
		return nil
	}
}
