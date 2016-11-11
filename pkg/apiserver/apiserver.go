package apiserver

import (
	"time"

	"k8s.io/kubernetes/pkg/api/rest"
	"k8s.io/kubernetes/pkg/api/unversioned"
	"k8s.io/kubernetes/pkg/genericapiserver"
	"k8s.io/kubernetes/pkg/registry/generic"
	"k8s.io/kubernetes/pkg/util/wait"

	"github.com/openshift/kube-aggregator/pkg/apis/apifederation"
	discoveryapiv1beta1 "github.com/openshift/kube-aggregator/pkg/apis/apifederation/v1beta1"
	clientset "github.com/openshift/kube-aggregator/pkg/client/clientset_generated/internalclientset/typed/apifederation/internalversion"
	"github.com/openshift/kube-aggregator/pkg/client/informers"
	apiserverstorage "github.com/openshift/kube-aggregator/pkg/registry/apiserver"
)

// TODO move to genericapiserver or something like that
type RESTOptionsGetter interface {
	NewFor(resource unversioned.GroupResource) generic.RESTOptions
}

type Config struct {
	GenericConfig *genericapiserver.Config

	RESTOptionsGetter RESTOptionsGetter
}

// APIDiscoveryServer contains state for a Kubernetes cluster master/api server.
type APIDiscoveryServer struct {
	GenericAPIServer *genericapiserver.GenericAPIServer

	// proxyHandlers tracks all of the proxyHandler's we've built so that we can update them in place when necessary
	proxyHandlers map[string]*ProxyHandler
}

type completedConfig struct {
	*Config
}

// Complete fills in any fields not set that are required to have valid data. It's mutating the receiver.
func (c *Config) Complete() completedConfig {
	c.GenericConfig.Complete()

	return completedConfig{c}
}

// SkipComplete provides a way to construct a server instance without config completion.
func (c *Config) SkipComplete() completedConfig {
	return completedConfig{c}
}

// New returns a new instance of APIDiscoveryServer from the given config.
func (c completedConfig) New() (*APIDiscoveryServer, error) {
	genericServer, err := c.Config.GenericConfig.SkipComplete().New() // completion is done in Complete, no need for a second time
	if err != nil {
		return nil, err
	}

	s := &APIDiscoveryServer{
		GenericAPIServer: genericServer,
		proxyHandlers:    map[string]*ProxyHandler{},
	}

	apiGroupInfo := genericapiserver.NewDefaultAPIGroupInfo(apifederation.GroupName)
	apiGroupInfo.GroupMeta.GroupVersion = discoveryapiv1beta1.SchemeGroupVersion

	v1beta1storage := map[string]rest.Storage{}
	v1beta1storage["apiservers"] = apiserverstorage.NewREST(c.RESTOptionsGetter.NewFor(apifederation.Resource("apiservers")))

	apiGroupInfo.VersionedResourcesStorageMap[discoveryapiv1beta1.SchemeGroupVersion.Version] = v1beta1storage

	if err := s.GenericAPIServer.InstallAPIGroup(&apiGroupInfo); err != nil {
		return nil, err
	}

	informerFactory := informers.NewSharedInformerFactory(
		clientset.NewForConfigOrDie(c.Config.GenericConfig.LoopbackClientConfig),
		5*time.Minute, // this is effectively used as a refresh interval right now.  Might want to do something nicer later on.
	)
	proxyRegistrationController := NewProxyRegistrationController(informerFactory.APIServers(), s)

	s.GenericAPIServer.AddPostStartHook("start-informers", func(context genericapiserver.PostStartHookContext) error {
		informerFactory.Start(wait.NeverStop)
		return nil
	})
	s.GenericAPIServer.AddPostStartHook("proxy-registration-controller", func(context genericapiserver.PostStartHookContext) error {
		proxyRegistrationController.Run(1, wait.NeverStop)
		return nil
	})

	return s, nil
}

func (s *APIDiscoveryServer) AddProxy(apiServer *apifederation.APIServer) {
	if handler, exists := s.proxyHandlers[apiServer.Name]; exists {
		handler.SetDestinationHost(apiServer.Spec.InternalHost)
		handler.SetEnabled(true)
		return
	}

	path := "/apis/" + apiServer.Spec.Group + "/" + apiServer.Spec.Version
	// v1. is a special case for the legacy API.  It proxies to a wider set of endpoints.
	if apiServer.Name == "v1." {
		path = "/api"
	}

	proxyHandler := &ProxyHandler{
		enabled:         true,
		destinationHost: apiServer.Spec.InternalHost,
	}

	s.GenericAPIServer.HandlerContainer.SecretRoutes.Handle(path, proxyHandler)
	s.GenericAPIServer.HandlerContainer.SecretRoutes.Handle(path+"/", proxyHandler)

	s.proxyHandlers[apiServer.Name] = proxyHandler
}

func (s *APIDiscoveryServer) RemoveProxy(apiServerName string) {
	handler, exists := s.proxyHandlers[apiServerName]
	if !exists {
		return
	}

	handler.SetEnabled(false)
}
