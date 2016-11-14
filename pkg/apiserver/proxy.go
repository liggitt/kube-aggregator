package apiserver

import (
	"net/http"
	"net/url"
	"sync"

	kapi "k8s.io/kubernetes/pkg/api"
	kapierrors "k8s.io/kubernetes/pkg/api/errors"
	"k8s.io/kubernetes/pkg/apiserver"
	"k8s.io/kubernetes/pkg/client/restclient"
	genericrest "k8s.io/kubernetes/pkg/registry/generic/rest"
	"k8s.io/kubernetes/pkg/runtime"
	"k8s.io/kubernetes/pkg/util/httpstream/spdy"
)

// ProxyHandler provides a http.Handler which will proxy traffic to locations
// specified by items implementing Redirector.
type ProxyHandler struct {
	// lock protects us for updates.
	lock sync.RWMutex

	// enabled tracks whether we should return anything.  There's no "remove from mux" function
	enabled bool

	destinationHost string
	// TODO add TLS options of some kind

	contextMapper kapi.RequestContextMapper

	proxyUserIdentification UserIdentification
}

func (r *ProxyHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if !r.isEnabled() {
		http.Error(w, "", http.StatusNotFound)
		return
	}

	ctx, ok := r.contextMapper.Get(req)
	if !ok {
		http.Error(w, "missing context", http.StatusInternalServerError)
		return
	}
	user, ok := kapi.UserFrom(ctx)
	if !ok {
		http.Error(w, "missing user", http.StatusInternalServerError)
		return
	}

	location := &url.URL{}
	location.Scheme = "https"
	location.Host = r.getDestinationHost()
	location.Path = req.URL.Path
	values := location.Query()
	for k, vs := range req.URL.Query() {
		for _, v := range vs {
			values.Add(k, v)
		}
	}
	location.RawQuery = values.Encode()

	newReq, err := http.NewRequest(req.Method, location.String(), req.Body)
	if statusErr, ok := err.(*kapierrors.StatusError); ok && err != nil {
		apiserver.WriteRawJSON(int(statusErr.Status().Code), statusErr.Status(), w)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	copyHeader(newReq.Header, req.Header)
	newReq.ContentLength = req.ContentLength
	// Copy the TransferEncoding is for future-proofing. Currently Go only supports "chunked" and
	// it can determine the TransferEncoding based on ContentLength and the Body.
	newReq.TransferEncoding = req.TransferEncoding

	// TODO: work out a way to re-use most of the transport for a given server while
	cfg := &restclient.Config{
		Insecure:        true,
		BearerToken:     r.proxyUserIdentification.BearerToken,
		TLSClientConfig: r.proxyUserIdentification.TLSClientConfig,
		Impersonate: restclient.ImpersonationConfig{
			UserName: user.GetName(),
			Groups:   user.GetGroups(),
			Extra:    user.GetExtra(),
		},
	}
	roundTripper, err := restclient.TransportFor(cfg)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	upgrade := false
	if connectionHeader := req.Header.Get("Connection"); len(connectionHeader) > 0 {
		upgrade = true
		tlsConfig, err := restclient.TLSConfigFor(cfg)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		upgradeRoundTripper := spdy.NewRoundTripper(tlsConfig)
		roundTripper, err = restclient.HTTPWrappersForConfig(cfg, upgradeRoundTripper)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	handler := genericrest.NewUpgradeAwareProxyHandler(location, roundTripper, false, upgrade, &responder{w: w})
	handler.ServeHTTP(w, newReq)
}

func copyHeader(dst, src http.Header) {
	for k, vv := range src {
		for _, v := range vv {
			dst.Add(k, v)
		}
	}
}

// responder implements rest.Responder for assisting a connector in writing objects or errors.
type responder struct {
	w http.ResponseWriter
}

func (r *responder) Object(statusCode int, obj runtime.Object) {
	apiserver.WriteRawJSON(statusCode, obj, r.w)
}

func (r *responder) Error(err error) {
	http.Error(r.w, err.Error(), http.StatusInternalServerError)
}

func (r *ProxyHandler) getDestinationHost() string {
	r.lock.RLock()
	defer r.lock.RUnlock()
	return r.destinationHost
}
func (r *ProxyHandler) SetDestinationHost(destinationHost string) {
	r.lock.Lock()
	defer r.lock.Unlock()
	r.destinationHost = destinationHost
}

func (r *ProxyHandler) isEnabled() bool {
	r.lock.RLock()
	defer r.lock.RUnlock()
	return r.enabled
}

func (r *ProxyHandler) SetEnabled(enabled bool) {
	r.lock.Lock()
	defer r.lock.Unlock()
	r.enabled = enabled
}
