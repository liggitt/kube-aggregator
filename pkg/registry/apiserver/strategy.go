package apiserver

import (
	"fmt"

	kapi "k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/fields"
	"k8s.io/kubernetes/pkg/labels"
	"k8s.io/kubernetes/pkg/registry/generic"
	"k8s.io/kubernetes/pkg/runtime"
	"k8s.io/kubernetes/pkg/storage"
	"k8s.io/kubernetes/pkg/util/validation/field"

	discoveryapi "github.com/openshift/kube-aggregator/pkg/api"
	"github.com/openshift/kube-aggregator/pkg/api/validation"
)

type apiServerStrategy struct {
	runtime.ObjectTyper
	kapi.NameGenerator
}

var strategy = apiServerStrategy{kapi.Scheme, kapi.SimpleNameGenerator}

func (apiServerStrategy) NamespaceScoped() bool {
	return false
}

func (apiServerStrategy) PrepareForCreate(ctx kapi.Context, obj runtime.Object) {
	_ = obj.(*discoveryapi.APIServer)
}

func (apiServerStrategy) PrepareForUpdate(ctx kapi.Context, obj, old runtime.Object) {
	newAPIServer := obj.(*discoveryapi.APIServer)
	oldAPIServer := old.(*discoveryapi.APIServer)
	newAPIServer.Status = oldAPIServer.Status
}

func (apiServerStrategy) Validate(ctx kapi.Context, obj runtime.Object) field.ErrorList {
	return validation.ValidateAPIServer(obj.(*discoveryapi.APIServer))
}

func (apiServerStrategy) AllowCreateOnUpdate() bool {
	return false
}

func (apiServerStrategy) AllowUnconditionalUpdate() bool {
	return false
}

func (apiServerStrategy) Canonicalize(obj runtime.Object) {
}

func (apiServerStrategy) ValidateUpdate(ctx kapi.Context, obj, old runtime.Object) field.ErrorList {
	return validation.ValidateAPIServerUpdate(obj.(*discoveryapi.APIServer), old.(*discoveryapi.APIServer))
}

// MatchAPIServer is the filter used by the generic etcd backend to watch events
// from etcd to clients of the apiserver only interested in specific labels/fields.
func MatchAPIServer(label labels.Selector, field fields.Selector) storage.SelectionPredicate {
	return storage.SelectionPredicate{
		Label: label,
		Field: field,
		GetAttrs: func(obj runtime.Object) (labels.Set, fields.Set, error) {
			apiserver, ok := obj.(*discoveryapi.APIServer)
			if !ok {
				return nil, nil, fmt.Errorf("given object is not a APIServer.")
			}
			return labels.Set(apiserver.ObjectMeta.Labels), APIServerToSelectableFields(apiserver), nil
		},
	}
}

// APIServerToSelectableFields returns a field set that represents the object.
func APIServerToSelectableFields(obj *discoveryapi.APIServer) fields.Set {
	return generic.ObjectMetaFieldsSet(&obj.ObjectMeta, true)
}
