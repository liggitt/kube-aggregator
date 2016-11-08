package apiserver

import (
	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/registry/generic"
	"k8s.io/kubernetes/pkg/registry/generic/registry"
	"k8s.io/kubernetes/pkg/runtime"
	"k8s.io/kubernetes/pkg/storage"

	discoveryapi "github.com/openshift/kube-aggregator/pkg/api"
)

// rest implements a RESTStorage for network policies against etcd
type REST struct {
	*registry.Store
}

// NewREST returns a RESTStorage object that will work against network policies.
func NewREST(opts generic.RESTOptions) *REST {
	prefix := "/" + opts.ResourcePrefix

	newListFunc := func() runtime.Object { return &discoveryapi.APIServerList{} }
	storageInterface, dFunc := opts.Decorator(
		opts.StorageConfig,
		1000, // cache size
		&discoveryapi.APIServer{},
		prefix,
		strategy,
		newListFunc,
		storage.NoTriggerPublisher,
	)

	store := &registry.Store{
		NewFunc: func() runtime.Object { return &discoveryapi.APIServer{} },

		// NewListFunc returns an object capable of storing results of an etcd list.
		NewListFunc: newListFunc,
		// Produces a APIServer that etcd understands, to the root of the resource
		// by combining the namespace in the context with the given prefix
		KeyRootFunc: func(ctx api.Context) string {
			return prefix
		},
		// Produces a APIServer that etcd understands, to the resource by combining
		// the namespace in the context with the given prefix
		KeyFunc: func(ctx api.Context, name string) (string, error) {
			return registry.NoNamespaceKeyFunc(ctx, prefix, name)
		},
		// Retrieve the name field of an apiserver
		ObjectNameFunc: func(obj runtime.Object) (string, error) {
			return obj.(*discoveryapi.APIServer).Name, nil
		},
		// Used to match objects based on labels/fields for list and watch
		PredicateFunc:           MatchAPIServer,
		QualifiedResource:       discoveryapi.Resource("apiservers"),
		EnableGarbageCollection: opts.EnableGarbageCollection,
		DeleteCollectionWorkers: opts.DeleteCollectionWorkers,

		// Used to validate controller creation
		CreateStrategy: strategy,

		// Used to validate controller updates
		UpdateStrategy: strategy,
		DeleteStrategy: strategy,

		Storage:     storageInterface,
		DestroyFunc: dFunc,
	}
	return &REST{store}
}
