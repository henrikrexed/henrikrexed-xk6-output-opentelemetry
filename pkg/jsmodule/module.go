// Package jsmodule exposes a k6 JavaScript module at "k6/x/otel" that allows
// scripts to set custom OTel baggage and span attributes from JS.
//
// Usage in k6 scripts:
//
//	import otel from "k6/x/otel";
//	export default function() {
//	    otel.setBaggage("user.type", "premium");
//	    otel.setAttribute("business.flow", "checkout");
//	    http.get("http://frontend:8080/api/products");
//	}
package jsmodule

import (
	"sync"

	"go.k6.io/k6/js/modules"
)

func init() {
	modules.Register("k6/x/otel", new(RootModule))
}

// BaggageStore is a global store for custom baggage entries set from JS.
// The output extension reads from this to inject into W3C baggage headers.
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

// Exports returns the module's default export.
func (mi *ModuleInstance) Exports() modules.Exports {
	return modules.Exports{
		Default: &OtelAPI{},
	}
}

// OtelAPI is the object exposed as the default export of k6/x/otel.
type OtelAPI struct{}

// SetBaggage sets a custom W3C Baggage entry that will be injected into
// outgoing HTTP requests.
func (api *OtelAPI) SetBaggage(key, value string) {
	BaggageStore.Set(key, value)
}

// SetAttribute sets a custom span attribute that will be added to the
// current iteration span.
func (api *OtelAPI) SetAttribute(key, value string) {
	AttributeStore.Set(key, value)
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
