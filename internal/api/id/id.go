package id

import (
	"sync"
	"sync/atomic"
)

var mu sync.RWMutex
var counters map[string]*uint32 = make(map[string]*uint32)

func NextId(counterKey string) uint32 {
	mu.RLock()
	counter, exists := counters[counterKey]
	mu.RUnlock()

	if !exists {
		mu.Lock()
		if _, exists := counters[counterKey]; !exists {
			var initial uint32
			counter = &initial
			counters[counterKey] = counter
		}
		mu.Unlock()
	}

	return atomic.AddUint32(counter, 1)
}

func Skip(counterKey string, delta uint32) uint32 {
	mu.RLock()
	counter, exists := counters[counterKey]
	mu.RUnlock()

	if !exists {
		mu.Lock()
		if _, exists := counters[counterKey]; !exists {
			var initial uint32
			counter = &initial
			counters[counterKey] = counter
		}
		mu.Unlock()
	}

	return atomic.AddUint32(counter, delta)
}
