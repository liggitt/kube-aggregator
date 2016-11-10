package apiserver

import (
	"time"

	"k8s.io/kubernetes/pkg/api/rest"
	"k8s.io/kubernetes/pkg/api/unversioned"
	"k8s.io/kubernetes/pkg/genericapiserver"
	"k8s.io/kubernetes/pkg/registry/generic"
	"k8s.io/kubernetes/pkg/util/wait"

	discoveryapi "github.com/openshift/kube-aggregator/pkg/api"
	discoveryapiv1beta1 "github.com/openshift/kube-aggregator/pkg/api/v1beta1"
	clientset "github.com/openshift/kube-aggregator/pkg/client/clientset_generated/internalclientset"
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

	m.GenericAPIServer.AddPostStartHook("start-informers", func(context genericapiserver.PostStartHookContext) error {
		informerFactory.Start(wait.NeverStop)
		return nil
	})

	return m, nil
}
