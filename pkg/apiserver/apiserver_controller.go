package apiserver

import (
	"fmt"
	"time"

	"github.com/golang/glog"

	kapierrors "k8s.io/kubernetes/pkg/api/errors"
	"k8s.io/kubernetes/pkg/client/cache"
	"k8s.io/kubernetes/pkg/controller"
	utilruntime "k8s.io/kubernetes/pkg/util/runtime"
	"k8s.io/kubernetes/pkg/util/wait"
	"k8s.io/kubernetes/pkg/util/workqueue"

	"github.com/openshift/kube-aggregator/pkg/apis/apifederation"
	"github.com/openshift/kube-aggregator/pkg/client/informers"
	listers "github.com/openshift/kube-aggregator/pkg/client/listers/apifederation/internalversion"
)

type APIHandlerManager interface {
	AddAPIServer(apiServer *apifederation.APIServer)
	RemoveAPIServer(apiServerName string)
}

type APIServerRegistrationController struct {
	apiHandlerManager APIHandlerManager

	apiServerLister listers.APIServerLister

	// To allow injection for testing.
	syncFn func(key string) error

	queue workqueue.RateLimitingInterface
}

func NewAPIServerRegistrationController(apiServerInformer informers.APIServerInformer, apiHandlerManager APIHandlerManager) *APIServerRegistrationController {
	c := &APIServerRegistrationController{
		apiHandlerManager: apiHandlerManager,
		apiServerLister:   apiServerInformer.Lister(),
		queue:             workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "APIServerRegistrationController"),
	}

	apiServerInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    c.addAPIServer,
		DeleteFunc: c.deleteAPIServer,
	})

	c.syncFn = c.sync

	return c
}

func (c *APIServerRegistrationController) sync(key string) error {
	apiServer, err := c.apiServerLister.Get(key)
	if kapierrors.IsNotFound(err) {
		c.apiHandlerManager.RemoveAPIServer(key)
		return nil
	}
	if err != nil {
		return err
	}

	c.apiHandlerManager.AddAPIServer(apiServer)
	return nil
}

func (c *APIServerRegistrationController) Run(workers int, stopCh <-chan struct{}) {
	defer utilruntime.HandleCrash()
	defer c.queue.ShutDown()
	defer glog.Infof("Shutting down APIServerRegistrationController")

	glog.Infof("Starting APIServerRegistrationController")

	for i := 0; i < workers; i++ {
		go wait.Until(c.runWorker, time.Second, stopCh)
	}

	<-stopCh
}

func (c *APIServerRegistrationController) runWorker() {
	for c.processNextWorkItem() {
	}
}

// processNextWorkItem deals with one key off the queue.  It returns false when it's time to quit.
func (c *APIServerRegistrationController) processNextWorkItem() bool {
	key, quit := c.queue.Get()
	if quit {
		return false
	}
	defer c.queue.Done(key)

	err := c.syncFn(key.(string))
	if err == nil {
		c.queue.Forget(key)
		return true
	}

	utilruntime.HandleError(fmt.Errorf("%v failed with : %v", key, err))
	c.queue.AddRateLimited(key)

	return true
}

func (c *APIServerRegistrationController) enqueue(obj *apifederation.APIServer) {
	key, err := controller.KeyFunc(obj)
	if err != nil {
		glog.Errorf("Couldn't get key for object %#v: %v", obj, err)
		return
	}

	c.queue.Add(key)
}

func (c *APIServerRegistrationController) addAPIServer(obj interface{}) {
	castObj := obj.(*apifederation.APIServer)
	glog.V(4).Infof("Adding daemon set %s", castObj.Name)
	c.enqueue(castObj)
}

func (c *APIServerRegistrationController) deleteAPIServer(obj interface{}) {
	castObj, ok := obj.(*apifederation.APIServer)
	if !ok {
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			glog.Errorf("Couldn't get object from tombstone %#v", obj)
			return
		}
		castObj, ok = tombstone.Obj.(*apifederation.APIServer)
		if !ok {
			glog.Errorf("Tombstone contained object that is not expected %#v", obj)
			return
		}
	}
	glog.V(4).Infof("Deleting %q", castObj.Name)
	c.enqueue(castObj)
}
