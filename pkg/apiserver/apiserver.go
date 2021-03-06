package apiserver

import (
	"net/http"
	"os"
	"time"

	kapi "k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/api/rest"
	"k8s.io/kubernetes/pkg/api/unversioned"
	apiserverfilters "k8s.io/kubernetes/pkg/apiserver/filters"
	authhandlers "k8s.io/kubernetes/pkg/auth/handlers"
	"k8s.io/kubernetes/pkg/client/restclient"
	"k8s.io/kubernetes/pkg/genericapiserver"
	genericfilters "k8s.io/kubernetes/pkg/genericapiserver/filters"
	"k8s.io/kubernetes/pkg/registry/generic"
	"k8s.io/kubernetes/pkg/util/wait"

	"github.com/openshift/kube-aggregator/pkg/apis/apifederation"
	discoveryapiv1beta1 "github.com/openshift/kube-aggregator/pkg/apis/apifederation/v1beta1"
	"github.com/openshift/kube-aggregator/pkg/apiserver/bootstraproutes"
	clientset "github.com/openshift/kube-aggregator/pkg/client/clientset_generated/internalclientset/typed/apifederation/internalversion"
	"github.com/openshift/kube-aggregator/pkg/client/informers"
	listers "github.com/openshift/kube-aggregator/pkg/client/listers/apifederation/internalversion"
	apiserverstorage "github.com/openshift/kube-aggregator/pkg/registry/apiserver"
)

// TODO move to genericapiserver or something like that
type RESTOptionsGetter interface {
	NewFor(resource unversioned.GroupResource) generic.RESTOptions
}

type Config struct {
	GenericConfig *genericapiserver.Config

	RESTOptionsGetter RESTOptionsGetter

	ProxyTLSConfig restclient.TLSClientConfig

	AuthUser string
}

// APIDiscoveryServer contains state for a Kubernetes cluster master/api server.
type APIDiscoveryServer struct {
	GenericAPIServer *genericapiserver.GenericAPIServer

	// proxyHandlers tracks all of the proxyHandler's we've built so that we can update them in place when necessary
	proxyHandlers map[string]*proxyHandler

	lister listers.APIServerLister

	proxyTLSConfig restclient.TLSClientConfig

	unprotectedMux *http.ServeMux
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
	informerFactory := informers.NewSharedInformerFactory(
		clientset.NewForConfigOrDie(c.Config.GenericConfig.LoopbackClientConfig),
		5*time.Minute, // this is effectively used as a refresh interval right now.  Might want to do something nicer later on.
	)
	unprotectedMux := http.NewServeMux()

	c.Config.GenericConfig.BuildHandlerChainsFunc = (&handlerChainConfig{
		informers:      informerFactory,
		unprotectedMux: unprotectedMux,
	}).handlerChain

	genericServer, err := c.Config.GenericConfig.SkipComplete().New() // completion is done in Complete, no need for a second time
	if err != nil {
		return nil, err
	}

	s := &APIDiscoveryServer{
		GenericAPIServer: genericServer,
		proxyHandlers:    map[string]*proxyHandler{},
		lister:           informerFactory.APIServers().Lister(),
		proxyTLSConfig:   c.ProxyTLSConfig,
		unprotectedMux:   unprotectedMux,
	}

	bootstraproutes.RoleBindings{AuthUser: c.AuthUser}.Install(unprotectedMux)

	apiGroupInfo := genericapiserver.NewDefaultAPIGroupInfo(apifederation.GroupName)
	apiGroupInfo.GroupMeta.GroupVersion = discoveryapiv1beta1.SchemeGroupVersion

	v1beta1storage := map[string]rest.Storage{}
	v1beta1storage["apiservers"] = apiserverstorage.NewREST(c.RESTOptionsGetter.NewFor(apifederation.Resource("apiservers")))

	apiGroupInfo.VersionedResourcesStorageMap[discoveryapiv1beta1.SchemeGroupVersion.Version] = v1beta1storage

	if err := s.GenericAPIServer.InstallAPIGroup(&apiGroupInfo); err != nil {
		return nil, err
	}

	proxyRegistrationController := NewAPIServerRegistrationController(informerFactory.APIServers(), s)

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

type handlerChainConfig struct {
	informers      informers.SharedInformerFactory
	unprotectedMux *http.ServeMux
}

func (h *handlerChainConfig) handlerChain(apiHandler http.Handler, c *genericapiserver.Config) (secure, insecure http.Handler) {
	attributeGetter := apiserverfilters.NewRequestAttributeGetter(c.RequestContextMapper)

	// add this as a filter so that we never collide with "already registered" failures on `/apis`
	handler := WithAPIs(apiHandler, h.informers)

	handler = apiserverfilters.WithAuthorization(handler, attributeGetter, c.Authorizer)

	// this mux is NOT protected by authorization, but DOES have authentication information
	// this is so that everyone can hit these endpoints, but we have the user information for proxy cases
	handler = WithUnprotectedMux(handler, h.unprotectedMux)

	handler = apiserverfilters.WithImpersonation(handler, c.RequestContextMapper, c.Authorizer)
	handler = apiserverfilters.WithAudit(handler, attributeGetter, os.Stdout)
	handler = authhandlers.WithAuthentication(handler, c.RequestContextMapper, c.Authenticator, authhandlers.Unauthorized(c.SupportsBasicAuth))

	handler = genericfilters.WithCORS(handler, c.CorsAllowedOriginList, nil, nil, nil, "true")
	handler = genericfilters.WithPanicRecovery(handler, c.RequestContextMapper)
	handler = apiserverfilters.WithRequestInfo(handler, genericapiserver.NewRequestInfoResolver(c), c.RequestContextMapper)
	handler = kapi.WithRequestContext(handler, c.RequestContextMapper)
	handler = genericfilters.WithTimeoutForNonLongRunningRequests(handler, c.LongRunningFunc)
	handler = genericfilters.WithMaxInFlightLimit(handler, c.MaxRequestsInFlight, c.LongRunningFunc)

	return handler, nil
}

func (s *APIDiscoveryServer) AddAPIServer(apiServer *apifederation.APIServer) {
	// make copy so we don't mess with the original
	tlsConfig := s.proxyTLSConfig
	tlsConfig.CAData = apiServer.Spec.CABundle

	if handler, exists := s.proxyHandlers[apiServer.Name]; exists {
		handler.SetDestinationHost(apiServer.Spec.InternalHost)
		handler.SetEnabled(true)
		handler.SetTLSConfig(tlsConfig)
		handler.SetInsecureSkipTLSVerify(apiServer.Spec.InsecureSkipTLSVerify)
		return
	}

	path := "/apis/" + apiServer.Spec.Group + "/" + apiServer.Spec.Version
	// v1. is a special case for the legacy API.  It proxies to a wider set of endpoints.
	if apiServer.Name == "v1." {
		path = "/api"
	}

	proxyHandler := &proxyHandler{
		enabled:               true,
		destinationHost:       apiServer.Spec.InternalHost,
		contextMapper:         s.GenericAPIServer.RequestContextMapper(),
		proxyTLSConfig:        tlsConfig,
		insecureSkipTLSVerify: apiServer.Spec.InsecureSkipTLSVerify,
	}
	// proxies are unprotected by
	s.unprotectedMux.Handle(path, proxyHandler)
	s.unprotectedMux.Handle(path+"/", proxyHandler)
	s.proxyHandlers[apiServer.Name] = proxyHandler

	// if we're dealing with the legacy group, we're done here
	if apiServer.Name == "v1." {
		return
	}

	// otherwise, its time to register the group discovery endpoint
	groupPath := "/apis/" + apiServer.Spec.Group
	groupDiscoveryHandler := &apiGroupHandler{
		groupName: apiServer.Spec.Group,
		lister:    s.lister,
	}
	// discovery is protected
	s.GenericAPIServer.HandlerContainer.SecretRoutes.Handle(groupPath, groupDiscoveryHandler)
	s.GenericAPIServer.HandlerContainer.SecretRoutes.Handle(groupPath+"/", groupDiscoveryHandler)

}

func (s *APIDiscoveryServer) RemoveAPIServer(apiServerName string) {
	handler, exists := s.proxyHandlers[apiServerName]
	if !exists {
		return
	}

	handler.SetEnabled(false)
}

func WithAPIs(handler http.Handler, informers informers.SharedInformerFactory) http.Handler {
	apisHandler := &apisHandler{
		lister:   informers.APIServers().Lister(),
		delegate: handler,
	}
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		apisHandler.ServeHTTP(w, req)
	})
}

func WithUnprotectedMux(handler http.Handler, mux *http.ServeMux) http.Handler {
	if mux == nil {
		return handler
	}

	// register the handler at this stage against everything under slash.  More specific paths that get registered will take precedence
	mux.Handle("/", handler)

	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		mux.ServeHTTP(w, req)
	})
}
