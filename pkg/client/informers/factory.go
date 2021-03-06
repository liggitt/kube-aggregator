package informers

import (
	"reflect"
	"sync"
	"time"

	kapi "k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/client/cache"
	"k8s.io/kubernetes/pkg/runtime"
	"k8s.io/kubernetes/pkg/watch"

	discoveryapi "github.com/openshift/kube-aggregator/pkg/apis/apifederation"
	clientset "github.com/openshift/kube-aggregator/pkg/client/clientset_generated/internalclientset/typed/apifederation/internalversion"
	listers "github.com/openshift/kube-aggregator/pkg/client/listers/apifederation/internalversion"
)

type SharedInformerFactory interface {
	// Start starts informers that can start AFTER the API server and controllers have started
	Start(stopCh <-chan struct{})

	APIServers() APIServerInformer
}

type sharedInformerFactory struct {
	client        clientset.ApifederationInterface
	lock          sync.Mutex
	defaultResync time.Duration

	informers map[reflect.Type]cache.SharedIndexInformer
	// startedInformers is used for tracking which informers have been started
	// this allows calling of Start method multiple times
	startedInformers map[reflect.Type]bool
}

// NewSharedInformerFactory constructs a new instance of sharedInformerFactory
func NewSharedInformerFactory(client clientset.ApifederationInterface, defaultResync time.Duration) SharedInformerFactory {
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
			ListFunc: func(options kapi.ListOptions) (runtime.Object, error) {
				return f.client.APIServers().List(options)
			},
			WatchFunc: func(options kapi.ListOptions) (watch.Interface, error) {
				return f.client.APIServers().Watch(options)
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
