/*
Copyright 2014 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package apiserver

import (
	"fmt"
	"net/http"
	"net/url"
	"sync"

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
}

func (r *ProxyHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if !r.isEnabled() {
		http.Error(w, "{}", http.StatusNotFound)
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
	fmt.Printf("Outbound URL=%q from path %q\n", location.String(), req.URL.Path)

	newReq, err := http.NewRequest(req.Method, location.String(), req.Body)
	if statusErr, ok := err.(*kapierrors.StatusError); ok && err != nil {
		apiserver.WriteRawJSON(int(statusErr.Status().Code), statusErr.Status(), w)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	newReq.Header = req.Header
	newReq.ContentLength = req.ContentLength
	// Copy the TransferEncoding is for future-proofing. Currently Go only supports "chunked" and
	// it can determine the TransferEncoding based on ContentLength and the Body.
	newReq.TransferEncoding = req.TransferEncoding

	cfg := &restclient.Config{
		Insecure:    true,
		BearerToken: "deads/system:masters",
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
