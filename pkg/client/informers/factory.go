package informers

import (
	"reflect"
	"sync"
	"time"

	kapi "k8s.io/kubernetes/pkg/api"
	kapiv1 "k8s.io/kubernetes/pkg/api/v1"
	"k8s.io/kubernetes/pkg/client/cache"
	"k8s.io/kubernetes/pkg/runtime"
	"k8s.io/kubernetes/pkg/watch"

	discoveryapi "github.com/openshift/kube-aggregator/pkg/api"
	clientset "github.com/openshift/kube-aggregator/pkg/client/clientset_generated/internalclientset"
	listers "github.com/openshift/kube-aggregator/pkg/client/listers/core/internalversion"
)

type SharedInformerFactory interface {
	// Start starts informers that can start AFTER the API server and controllers have started
	Start(stopCh <-chan struct{})

	APIServers() APIServerInformer
}

type sharedInformerFactory struct {
	client        clientset.Interface
	lock          sync.Mutex
	defaultResync time.Duration

	informers map[reflect.Type]cache.SharedIndexInformer
	// startedInformers is used for tracking which informers have been started
	// this allows calling of Start method multiple times
	startedInformers map[reflect.Type]bool
}

// NewSharedInformerFactory constructs a new instance of sharedInformerFactory
func NewSharedInformerFactory(client clientset.Interface, defaultResync time.Duration) SharedInformerFactory {
	return &sharedInformerFactory{
		client:           client,
		defaultResync:    defaultResync,
		informers:        make(map[reflect.Type]cache.SharedIndexInformer),
		startedInformers: make(map[reflect.Type]bool),
	}
}

// Start initializes all requested informers.
func (f *sharedInformerFactory) Start(stopCh <-chan struct{}) {
	f.lock.Lock()
	defer f.lock.Unlock()

	for informerType, informer := range f.informers {
		if !f.startedInformers[informerType] {
			go informer.Run(stopCh)
			f.startedInformers[informerType] = true
		}
	}
}

func (f *sharedInformerFactory) APIServers() APIServerInformer {
	return &apiServerInformer{sharedInformerFactory: f}
}

type APIServerInformer interface {
	Informer() cache.SharedIndexInformer
	Lister() listers.APIServerLister
}

type apiServerInformer struct {
	*sharedInformerFactory
}

func (f *apiServerInformer) Informer() cache.SharedIndexInformer {
	f.lock.Lock()
	defer f.lock.Unlock()

	informerType := reflect.TypeOf(&discoveryapi.APIServer{})
	informer, exists := f.informers[informerType]
	if exists {
		return informer
	}
	informer = cache.NewSharedIndexInformer(
		&cache.ListWatch{
			// TODO something weird is happening with the client generator
			// we never specify options, so lose them for now
			ListFunc: func(options kapi.ListOptions) (runtime.Object, error) {
				versioned := kapiv1.ListOptions{}
				err := kapi.Scheme.Convert(&options, &versioned, nil)
				if err != nil {
					return nil, err
				}
				return f.client.Pkg().APIServers().List(versioned)
			},
			WatchFunc: func(options kapi.ListOptions) (watch.Interface, error) {
				versioned := kapiv1.ListOptions{}
				err := kapi.Scheme.Convert(&options, &versioned, nil)
				if err != nil {
					return nil, err
				}
				return f.client.Pkg().APIServers().Watch(versioned)
			},
		},
		&discoveryapi.APIServer{},
		f.defaultResync,
		cache.Indexers{},
	)
	f.informers[informerType] = informer

	return informer
}

func (f *apiServerInformer) Lister() listers.APIServerLister {
	return listers.NewAPIServerLister(f.Informer().GetIndexer())
}
