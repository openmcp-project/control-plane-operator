package utils

import (
	"errors"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
)

// IsCRDNotFound checks if the given error is a CRD not found error.
func IsCRDNotFound(err error) bool {
	// check if err tree contains a "NoKindMatchError" error.
	if errors.Is(err, &meta.NoKindMatchError{}) {
		return true
	}

	// check if err tree contains a "ErrResourceDiscoveryFailed" error.
	var rdfErr *apiutil.ErrResourceDiscoveryFailed
	if !errors.As(err, &rdfErr) {
		return false
	}

	// all wrapped errors must be "NotFound" errors.
	// only then the entire "ErrResourceDiscoveryFailed" is considered as "CRD not found".
	for _, wrappedErr := range *rdfErr {
		if !apierrors.IsNotFound(wrappedErr) {
			return false
		}
	}

	return true
}
