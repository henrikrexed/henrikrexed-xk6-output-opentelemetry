// Package jsmodule exposes a k6 JavaScript module at "k6/x/otel" that allows
// scripts to set custom OTel baggage/attributes and use high-level helpers
// (step, request, check) that automatically manage spans and baggage.
package jsmodule

import (
	"fmt"
	"sync"

	"github.com/grafana/sobek"
	"go.k6.io/k6/js/modules"
)

func init() {
	modules.Register("k6/x/otel", new(RootModule))
}

// BaggageStore is a global store for custom baggage entries set from JS.
var BaggageStore = &baggageStore{entries: make(map[string]string)}

// AttributeStore is a global store for custom span attributes set from JS.
var AttributeStore = &attributeStore{attrs: make(map[string]string)}

// RootModule implements modules.Module.
type RootModule struct{}

func (rm *RootModule) NewModuleInstance(vu modules.VU) modules.Instance {
	return &ModuleInstance{vu: vu}
}

// ModuleInstance is the per-VU instance.
type ModuleInstance struct {
	vu modules.VU
}

// Exports returns the module's default export with all API methods.
func (mi *ModuleInstance) Exports() modules.Exports {
	return modules.Exports{
		Default: &OtelAPI{vu: mi.vu},
	}
}

// OtelAPI is the object exposed as the default export of k6/x/otel.
type OtelAPI struct {
	vu modules.VU
}

// SetBaggage sets a custom W3C Baggage entry.
func (api *OtelAPI) SetBaggage(key, value string) {
	BaggageStore.Set(key, value)
}

// SetAttribute sets a custom span attribute on the current iteration span.
func (api *OtelAPI) SetAttribute(key, value string) {
	AttributeStore.Set(key, value)
}

// Step wraps a k6 group() call: sets k6.test.step baggage to the step name,
// executes the callback inside a k6 group, then restores the previous step.
// Usage: otel.step("Browse Products", function() { ... })
func (api *OtelAPI) Step(name string, fn sobek.Callable) (sobek.Value, error) {
	rt := api.vu.Runtime()

	// Set baggage for this step
	BaggageStore.Set("k6.test.step", name)

	// Call k6's group(name, fn) via the runtime
	groupFn := rt.Get("group")
	if groupFn == nil || sobek.IsUndefined(groupFn) {
		// Fallback: just call fn directly if group isn't available
		return fn(sobek.Undefined(), rt.ToValue(name))
	}

	groupCallable, ok := sobek.AssertFunction(groupFn)
	if !ok {
		// group isn't callable, just call fn
		return fn(sobek.Undefined())
	}

	result, err := groupCallable(sobek.Undefined(), rt.ToValue(name), rt.ToValue(fn))

	// Restore step to "default"
	BaggageStore.Set("k6.test.step", "default")

	return result, err
}

// Request wraps an HTTP request: sets baggage, makes the request via k6 http module,
// returns the response.
// Usage: otel.request("list-products", "GET", "http://frontend/api/products")
// Usage: otel.request("add-to-cart", "POST", url, { body: ..., headers: ... })
func (api *OtelAPI) Request(name, method, url string, params ...sobek.Value) (sobek.Value, error) {
	rt := api.vu.Runtime()

	// Set request name as baggage
	BaggageStore.Set("k6.request.name", name)

	// Get the http module from the runtime global scope
	httpObj := rt.Get("http")
	if httpObj == nil || sobek.IsUndefined(httpObj) {
		return sobek.Undefined(), fmt.Errorf("http module not available; add: import http from 'k6/http'")
	}

	// Determine method
	var methodFnName string
	switch method {
	case "GET":
		methodFnName = "get"
	case "POST":
		methodFnName = "post"
	case "PUT":
		methodFnName = "put"
	case "PATCH":
		methodFnName = "patch"
	case "DELETE":
		methodFnName = "del"
	default:
		methodFnName = "request"
	}

	httpObject := httpObj.ToObject(rt)
	fnVal := httpObject.Get(methodFnName)
	if fnVal == nil || sobek.IsUndefined(fnVal) {
		return sobek.Undefined(), fmt.Errorf("http.%s not available", methodFnName)
	}

	callable, ok := sobek.AssertFunction(fnVal)
	if !ok {
		return sobek.Undefined(), fmt.Errorf("http.%s is not callable", methodFnName)
	}

	// Build args: for get/del it's (url, params?), for post/put/patch it's (url, body?, params?)
	args := []sobek.Value{rt.ToValue(url)}
targs = append(args, params...)
	result, err := callable(httpObj, args...)

	// Clear request name
	BaggageStore.Set("k6.request.name", "")

	return result, err
}

// Check wraps k6's check() function with the response and checks object.
// Usage: otel.check("products-ok", response, { "status is 200": (r) => r.status === 200 })
func (api *OtelAPI) Check(name string, response sobek.Value, checks sobek.Value) (sobek.Value, error) {
	rt := api.vu.Runtime()

	// Set check group name as attribute
	AttributeStore.Set("k6.check.group", name)

	// Call k6's check(response, checks)
	checkFn := rt.Get("check")
	if checkFn == nil || sobek.IsUndefined(checkFn) {
		return sobek.Undefined(), fmt.Errorf("check function not available; add: import { check } from 'k6'")
	}

	callable, ok := sobek.AssertFunction(checkFn)
	if !ok {
		return sobek.Undefined(), fmt.Errorf("check is not callable")
	}

	return callable(sobek.Undefined(), response, checks)
}

// --- thread-safe stores ---

type baggageStore struct {
	mu      sync.RWMutex
	entries map[string]string
}

func (s *baggageStore) Set(key, value string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.entries[key] = value
}

func (s *baggageStore) GetAll() map[string]string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	cp := make(map[string]string, len(s.entries))
	for k, v := range s.entries {
		cp[k] = v
	}
	return cp
}

func (s *baggageStore) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.entries = make(map[string]string)
}

type attributeStore struct {
	mu    sync.RWMutex
	attrs map[string]string
}

func (s *attributeStore) Set(key, value string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.attrs[key] = value
}

func (s *attributeStore) GetAll() map[string]string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	cp := make(map[string]string, len(s.attrs))
	for k, v := range s.attrs {
		cp[k] = v
	}
	return cp
}
