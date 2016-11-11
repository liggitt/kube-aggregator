package apiserver

import (
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	kapierrors "k8s.io/kubernetes/pkg/api/errors"
	"k8s.io/kubernetes/pkg/api/rest"
	"k8s.io/kubernetes/pkg/api/unversioned"
	"k8s.io/kubernetes/pkg/apiserver"
	"k8s.io/kubernetes/pkg/client/restclient"
	"k8s.io/kubernetes/pkg/client/typed/discovery"
	"k8s.io/kubernetes/pkg/genericapiserver"
	"k8s.io/kubernetes/pkg/genericapiserver/mux"
	"k8s.io/kubernetes/pkg/registry/generic"
	"k8s.io/kubernetes/pkg/util/wait"

	discoveryapi "github.com/openshift/kube-aggregator/pkg/apis/apifederation"
	discoveryapiv1beta1 "github.com/openshift/kube-aggregator/pkg/apis/apifederation/v1beta1"
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
}

// APIDiscoveryServer contains state for a Kubernetes cluster master/api server.
type APIDiscoveryServer struct {
	GenericAPIServer *genericapiserver.GenericAPIServer
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
	s, err := c.Config.GenericConfig.SkipComplete().New() // completion is done in Complete, no need for a second time
	if err != nil {
		return nil, err
	}

	m := &APIDiscoveryServer{
		GenericAPIServer: s,
	}

	apiGroupInfo := genericapiserver.NewDefaultAPIGroupInfo(discoveryapi.GroupName)
	apiGroupInfo.GroupMeta.GroupVersion = discoveryapiv1beta1.SchemeGroupVersion

	v1beta1storage := map[string]rest.Storage{}
	v1beta1storage["apiservers"] = apiserverstorage.NewREST(c.RESTOptionsGetter.NewFor(discoveryapi.Resource("apiservers")))

	apiGroupInfo.VersionedResourcesStorageMap[discoveryapiv1beta1.SchemeGroupVersion.Version] = v1beta1storage

	if err := m.GenericAPIServer.InstallAPIGroup(&apiGroupInfo); err != nil {
		return nil, err
	}

	informerFactory := informers.NewSharedInformerFactory(
		clientset.NewForConfigOrDie(c.Config.GenericConfig.LoopbackClientConfig),
		5*time.Minute, // this is effectively used as a refresh interval right now.  Might want to do something nicer later on.
	)

	discoveryHandler := &discoveryHandler{
		apiserverLister:            informerFactory.APIServers().Lister(),
		versionsToDiscoveryClients: map[string]discovery.DiscoveryInterface{},
	}
	discoveryHandler.Install(m.GenericAPIServer.HandlerContainer)

	m.GenericAPIServer.AddPostStartHook("start-informers", func(context genericapiserver.PostStartHookContext) error {
		informerFactory.Start(wait.NeverStop)
		return nil
	})

	return m, nil
}

type discoveryHandler struct {
	apiserverLister listers.APIServerLister

	// discoveryConfig is used to stamp out new clients for use in running discovery against other
	// API servers
	discoveryConfig restclient.Config

	discoveryClientLock sync.RWMutex
	// versionsToDiscoveryClients maps the apiserver.Name to the discovery client we use to run discovery
	// checks on it.  This needs to be cleaned up when deletions happen
	versionsToDiscoveryClients map[string]discovery.DiscoveryInterface
}

func (h *discoveryHandler) Install(c *mux.APIContainer) {
	// c.SecretRoutes.HandleFunc("/apis", func(w http.ResponseWriter, r *http.Request) {

	// 	status := http.StatusOK
	// 	if r.URL.Path != "/" && r.URL.Path != "/index.html" {
	// 		// Since "/" matches all paths, handleIndex is called for all paths for which there is no handler registered.
	// 		// We want to return a 404 status with a list of all valid paths, incase of an invalid URL request.
	// 		status = http.StatusNotFound
	// 	}
	// 	var handledPaths []string
	// 	// Extract the paths handled using restful.WebService
	// 	for _, ws := range c.RegisteredWebServices() {
	// 		handledPaths = append(handledPaths, ws.RootPath())
	// 	}
	// 	// Extract the paths handled using mux handler.
	// 	handledPaths = append(handledPaths, c.NonSwaggerRoutes.HandledPaths()...)
	// 	sort.Strings(handledPaths)
	// 	apiserver.WriteRawJSON(status, unversioned.RootPaths{Paths: handledPaths}, w)
	// })

	c.SecretRoutes.HandleFunc("/api", func(w http.ResponseWriter, r *http.Request) {
		fmt.Printf("#### 1a\n")
		// if we've registered "v1.", then just proxy straight there
		apiServer, err := h.apiserverLister.Get("v1.")
		if statusErr, ok := err.(*kapierrors.StatusError); ok && err != nil {
			apiserver.WriteRawJSON(int(statusErr.Status().Code), statusErr.Status(), w)
			return
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		urlStr := "https://" + apiServer.Spec.InternalHost + r.RequestURI
		fmt.Printf("Outbound URL= %q\n", urlStr)
		proxyRequest, err := http.NewRequest(r.Method, urlStr, r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		copyHeader(proxyRequest.Header, r.Header)

		client := &http.Client{}
		proxyResponse, err := client.Do(proxyRequest)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer proxyResponse.Body.Close()
		copyHeader(w.Header(), proxyResponse.Header)

		buffer := make([]byte, 0, 1024)
		for {
			readBytes, err := proxyResponse.Body.Read(buffer)
			if err != nil && err != io.EOF {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			if _, err := w.Write(buffer[:readBytes]); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			if err == io.EOF {
				break
			}
		}
	})
}

func copyHeader(dst, src http.Header) {
	for k, vv := range src {
		for _, v := range vv {
			dst.Add(k, v)
		}
	}
}

func (h *discoveryHandler) getClient(apiServer *discoveryapi.APIServer) (discovery.DiscoveryInterface, error) {
	var ret discovery.DiscoveryInterface

	func() {
		h.discoveryClientLock.RLock()
		defer h.discoveryClientLock.RLock()

		if existingClient, ok := h.versionsToDiscoveryClients[apiServer.Name]; ok {
			ret = existingClient
			return
		}
	}()

	if ret != nil {
		return ret, nil
	}

	h.discoveryClientLock.Lock()
	defer h.discoveryClientLock.Lock()

	cfg := h.discoveryConfig
	cfg.Host = apiServer.Spec.InternalHost
	cfg.Insecure = true

	client, err := discovery.NewDiscoveryClientForConfig(&cfg)
	if err != nil {
		return nil, err
	}
	h.versionsToDiscoveryClients[apiServer.Name] = client

	return client, nil
}
