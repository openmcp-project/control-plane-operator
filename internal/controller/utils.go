package controller

import (
	"fmt"
	"hash/fnv"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

var _ predicate.Predicate = filterObjectPredicate{}

type filterObjectPredicate struct {
	filterFunc func(o client.Object) bool
}

// Create implements predicate.Predicate.
func (p filterObjectPredicate) Create(evt event.CreateEvent) bool {
	return p.filterFunc(evt.Object)
}

// Delete implements predicate.Predicate.
func (p filterObjectPredicate) Delete(evt event.DeleteEvent) bool {
	return p.filterFunc(evt.Object)
}

// Generic implements predicate.Predicate.
func (p filterObjectPredicate) Generic(evt event.GenericEvent) bool {
	return p.filterFunc(evt.Object)
}

// Update implements predicate.Predicate.
func (p filterObjectPredicate) Update(evt event.UpdateEvent) bool {
	return p.filterFunc(evt.ObjectNew) || p.filterFunc(evt.ObjectOld)
}

func shortenToXCharacters(input string, maxLen int) string {
	if len(input) <= maxLen {
		return input
	}

	hash := fnv.New32a()
	hash.Write([]byte(input))

	suffix := fmt.Sprintf("--%x", hash.Sum32())
	trimLength := maxLen - len(suffix)

	return input[:trimLength] + suffix
}
