package simulation

import (
	"math"
	"sync/atomic"
	"unsafe"
)

// atomicAddFloat64 atomically adds a delta to a float64 value.
// It's implemented using atomic operations on uint64.
func atomicAddFloat64(val *float64, delta float64) {
	for {
		old := atomic.LoadUint64((*uint64)(unsafe.Pointer(val)))
		new := math.Float64bits(math.Float64frombits(old) + delta)
		if atomic.CompareAndSwapUint64((*uint64)(unsafe.Pointer(val)), old, new) {
			return
		}
	}
}
